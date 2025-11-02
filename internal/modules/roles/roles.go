package roles

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

const (
	moduleName  = "pgb_instance_roles"
	channelName = "pgb_instance_roles"
)

// RolesModule manages automatic role discovery for database instances
type RolesModule struct {
	centralPool *pgxpool.Pool
	logger      *logger.Logger
}

// NewRolesModule creates a new instance roles module
func NewRolesModule(centralPool *pgxpool.Pool, log *logger.Logger) *RolesModule {
	return &RolesModule{
		centralPool: centralPool,
		logger:      log,
	}
}

// Name returns the module name
func (r *RolesModule) Name() string {
	return moduleName
}

// GetChannelName returns the channel name to listen on
func (r *RolesModule) GetChannelName() string {
	return channelName
}

// Initialize sets up the module
func (r *RolesModule) Initialize(ctx context.Context, pool *pgxpool.Pool) error {
	if r.logger != nil {
		r.logger.LogSystemf(logger.LevelInfo, moduleName, "Initializing %s module", moduleName)
	}

	// Note: This module doesn't create tables on the source database
	// It only operates on the central pansoinco_suite database

	if r.logger != nil {
		r.logger.LogSystemf(logger.LevelInfo, moduleName, "Module %s initialized successfully", moduleName)
	}

	return nil
}

// Start begins the module's background operations
func (r *RolesModule) Start(ctx context.Context) error {
	// No background tasks needed - role discovery happens on-demand via LISTEN/NOTIFY
	return nil
}

// Stop stops the module's background operations
func (r *RolesModule) Stop() error {
	// No background tasks to stop
	return nil
}

// ProcessNotification processes a notification containing an instance_id
func (r *RolesModule) ProcessNotification(ctx context.Context, payload string) error {
	// Parse instance_id from payload
	instanceID, err := strconv.Atoi(payload)
	if err != nil {
		return fmt.Errorf("invalid instance_id in payload: %w", err)
	}

	if r.logger != nil {
		r.logger.LogSystemf(logger.LevelInfo, moduleName, "Processing role discovery for instance_id: %d", instanceID)
	}

	// Get instance connection details
	instance, err := r.getInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance details: %w", err)
	}

	// Discover roles from the target database
	roles, err := r.discoverRoles(ctx, instance.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to discover roles: %w", err)
	}

	// Insert discovered roles into sw_instance_roles
	inserted, err := r.insertRoles(ctx, instanceID, roles)
	if err != nil {
		return fmt.Errorf("failed to insert roles: %w", err)
	}

	if r.logger != nil {
		r.logger.LogSystemf(logger.LevelInfo, moduleName, "Discovered %d roles for instance %d (%s), inserted %d new roles",
			len(roles), instanceID, instance.Name, inserted)
	}

	return nil
}

// ProcessQueue processes any queued items (not applicable for this module)
func (r *RolesModule) ProcessQueue(ctx context.Context) error {
	// This module doesn't have a queue - it only responds to notifications
	return nil
}

// Instance represents a database instance
type Instance struct {
	ID               int
	Name             string
	ConnectionString string
}

// getInstance retrieves instance details from sw_instance
func (r *RolesModule) getInstance(ctx context.Context, instanceID int) (*Instance, error) {
	query := `
		SELECT
			si.id,
			ps.short_name || ' ' || si.db_name as name,
			si.db_connection_string
		FROM sw_instance si
		LEFT JOIN ps_sw ps ON ps.id = si.sw_id
		WHERE si.id = $1
	`

	instance := &Instance{}
	err := r.centralPool.QueryRow(ctx, query, instanceID).Scan(
		&instance.ID,
		&instance.Name,
		&instance.ConnectionString,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query instance: %w", err)
	}

	if instance.ConnectionString == "" {
		return nil, fmt.Errorf("instance %d has no connection string", instanceID)
	}

	return instance, nil
}

// discoverRoles queries the target database for all non-system roles
func (r *RolesModule) discoverRoles(ctx context.Context, connString string) ([]string, error) {
	// Create a connection to the target database
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer conn.Close(ctx)

	// Query for non-system roles
	query := `
		SELECT rolname
		FROM pg_roles
		WHERE rolname NOT LIKE 'pg_%'
		  AND rolname NOT IN ('postgres')
		ORDER BY rolname
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, roleName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}

// insertRoles inserts discovered roles into sw_instance_roles
func (r *RolesModule) insertRoles(ctx context.Context, instanceID int, roles []string) (int, error) {
	if len(roles) == 0 {
		return 0, nil
	}

	insertQuery := `
		INSERT INTO sw_instance_roles (instance_id, role)
		VALUES ($1, $2)
		ON CONFLICT (instance_id, role) DO NOTHING
	`

	inserted := 0
	for _, role := range roles {
		result, err := r.centralPool.Exec(ctx, insertQuery, instanceID, role)
		if err != nil {
			if r.logger != nil {
				r.logger.LogSystemf(logger.LevelWarn, moduleName, "Failed to insert role %s for instance %d: %v", role, instanceID, err)
			}
			continue
		}

		if result.RowsAffected() > 0 {
			inserted++
		}
	}

	return inserted, nil
}

// CreateTrigger creates the trigger on sw_instance to automatically notify on INSERT
func CreateTrigger(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) error {
	if log != nil {
		log.LogSystemf(logger.LevelInfo, moduleName, "Creating trigger for automatic role discovery notifications")
	}

	// Create the trigger function (in public schema)
	functionSQL := `
		CREATE OR REPLACE FUNCTION trg_instance_roles_notify()
		RETURNS TRIGGER AS $$
		BEGIN
			-- Send NOTIFY with the new instance ID
			PERFORM pg_notify('pgb_instance_roles', NEW.id::text);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`

	if _, err := pool.Exec(ctx, functionSQL); err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create the trigger on sw_instance
	triggerSQL := `
		DROP TRIGGER IF EXISTS S01_instance_roles_notify ON sw_instance;

		CREATE TRIGGER S01_instance_roles_notify
		AFTER INSERT ON sw_instance
		FOR EACH ROW
		EXECUTE FUNCTION pgb.trg_instance_roles_notify();
	`

	if _, err := pool.Exec(ctx, triggerSQL); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	if log != nil {
		log.LogSystemf(logger.LevelInfo, moduleName, "Trigger created successfully")
	}

	return nil
}
