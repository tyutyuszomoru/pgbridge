package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

const (
	// Schema creation SQL
	createSchemaSQL = `CREATE SCHEMA IF NOT EXISTS pgb;`

	// pgb_log table creation SQL
	createLogTableSQL = `
		CREATE TABLE IF NOT EXISTS pgb.pgb_log (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			service_name VARCHAR(50) DEFAULT 'pgbridge',
			event_type VARCHAR(50) NOT NULL,
			database_name VARCHAR(100),
			module_name VARCHAR(50),
			message TEXT,
			details JSONB
		);`

	// Indexes for pgb_log table
	createLogTimestampIndexSQL = `
		CREATE INDEX IF NOT EXISTS idx_pgb_log_timestamp
		ON pgb.pgb_log(timestamp);`

	createLogEventTypeIndexSQL = `
		CREATE INDEX IF NOT EXISTS idx_pgb_log_event_type
		ON pgb.pgb_log(event_type);`

	// Service startup log entry
	insertStartupLogSQL = `
		INSERT INTO pgb.pgb_log (event_type, database_name, message, details)
		VALUES ($1, $2, $3, $4);`
)

// SchemaInitializer handles database schema initialization
type SchemaInitializer struct {
	pool         *pgxpool.Pool
	databaseName string
	logger       *logger.Logger
}

// NewSchemaInitializer creates a new schema initializer
func NewSchemaInitializer(pool *pgxpool.Pool, databaseName string, log *logger.Logger) *SchemaInitializer {
	return &SchemaInitializer{
		pool:         pool,
		databaseName: databaseName,
		logger:       log,
	}
}

// Initialize creates the pgb schema and all required tables
func (si *SchemaInitializer) Initialize(ctx context.Context) error {
	if si.logger != nil {
		si.logger.LogSystemf(logger.LevelInfo, "schema", "Initializing database schema for: %s", si.databaseName)
	}

	// Create schema
	if err := si.createSchema(ctx); err != nil {
		schemaErr := fmt.Errorf("failed to create schema: %w", err)
		if si.logger != nil {
			si.logger.LogModuleError(si.databaseName, "schema", "create schema", schemaErr)
		}
		return schemaErr
	}

	// Create pgb_log table
	if err := si.createLogTable(ctx); err != nil {
		tableErr := fmt.Errorf("failed to create log table: %w", err)
		if si.logger != nil {
			si.logger.LogModuleError(si.databaseName, "schema", "create log table", tableErr)
		}
		return tableErr
	}

	// Create indexes
	if err := si.createIndexes(ctx); err != nil {
		indexErr := fmt.Errorf("failed to create indexes: %w", err)
		if si.logger != nil {
			si.logger.LogModuleError(si.databaseName, "schema", "create indexes", indexErr)
		}
		return indexErr
	}

	// Log service startup
	if err := si.logServiceStartup(ctx); err != nil {
		logErr := fmt.Errorf("failed to log service startup: %w", err)
		if si.logger != nil {
			si.logger.LogModuleError(si.databaseName, "schema", "log startup", logErr)
		}
		return logErr
	}

	if si.logger != nil {
		si.logger.LogSystemf(logger.LevelInfo, "schema", "Schema initialization completed for: %s", si.databaseName)
	}

	return nil
}

// createSchema creates the pgb schema if it doesn't exist
func (si *SchemaInitializer) createSchema(ctx context.Context) error {
	_, err := si.pool.Exec(ctx, createSchemaSQL)
	return err
}

// createLogTable creates the pgb_log table if it doesn't exist
func (si *SchemaInitializer) createLogTable(ctx context.Context) error {
	_, err := si.pool.Exec(ctx, createLogTableSQL)
	return err
}

// createIndexes creates indexes on the pgb_log table
func (si *SchemaInitializer) createIndexes(ctx context.Context) error {
	// Create timestamp index
	if _, err := si.pool.Exec(ctx, createLogTimestampIndexSQL); err != nil {
		return err
	}

	// Create event type index
	if _, err := si.pool.Exec(ctx, createLogEventTypeIndexSQL); err != nil {
		return err
	}

	return nil
}

// logServiceStartup logs the service startup event
func (si *SchemaInitializer) logServiceStartup(ctx context.Context) error {
	details := map[string]interface{}{
		"initialized_at": time.Now().Format(time.RFC3339),
		"version":        "1.0.0",
	}

	_, err := si.pool.Exec(ctx, insertStartupLogSQL,
		"SERVICE_START",
		si.databaseName,
		"pgbridge service connected and initialized",
		details,
	)
	return err
}

// InitializeWithConnectionManager is a convenience method that initializes
// the schema using a connection manager
func InitializeWithConnectionManager(ctx context.Context, cm *ConnectionManager) error {
	pool := cm.GetPool()
	if pool == nil {
		return fmt.Errorf("connection pool is nil")
	}

	initializer := NewSchemaInitializer(pool, cm.config.Name, cm.logger)
	return initializer.Initialize(ctx)
}
