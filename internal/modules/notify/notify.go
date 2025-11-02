package notify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

const (
	moduleName  = "pgb_notify"
	channelName = "pgb_notify"
)

// NotifyModule handles notification forwarding between databases
type NotifyModule struct {
	sourcePool  *pgxpool.Pool  // Connection to source database
	centralPool *pgxpool.Pool  // Connection to central notification database
	sourceName  string         // Name of the source database
	logger      *logger.Logger // Logger instance
	mu          sync.RWMutex
	shutdown    chan struct{}
	wg          sync.WaitGroup
}

// Notification represents a notification in the source database
type Notification struct {
	ID          int
	UserEmail   string
	SenderDB    string
	Message     string
	MessageLink string
	Criticality int
	IsSent      bool
	SentTS      *time.Time
	CreatedAt   time.Time
}

// NewNotifyModule creates a new notify module instance
func NewNotifyModule(sourcePool *pgxpool.Pool, centralPool *pgxpool.Pool, sourceName string, log *logger.Logger) *NotifyModule {
	return &NotifyModule{
		sourcePool:  sourcePool,
		centralPool: centralPool,
		sourceName:  sourceName,
		logger:      log,
		shutdown:    make(chan struct{}),
	}
}

// Name returns the module name
func (n *NotifyModule) Name() string {
	return moduleName
}

// GetChannelName returns the NOTIFY channel name for new notifications
func (n *NotifyModule) GetChannelName() string {
	return channelName
}

// Initialize creates the required tables and indexes
func (n *NotifyModule) Initialize(ctx context.Context, pool *pgxpool.Pool) error {
	if n.logger != nil {
		n.logger.LogModuleInit(n.sourceName, moduleName)
	}

	// Create notification table in source database
	if err := n.createNotificationTable(ctx); err != nil {
		initErr := fmt.Errorf("failed to create notification table: %w", err)
		if n.logger != nil {
			n.logger.LogModuleError(n.sourceName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	// Create indexes
	if err := n.createIndexes(ctx); err != nil {
		initErr := fmt.Errorf("failed to create indexes: %w", err)
		if n.logger != nil {
			n.logger.LogModuleError(n.sourceName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	// Create trigger function and trigger for automatic NOTIFY
	if err := n.createTriggers(ctx); err != nil {
		initErr := fmt.Errorf("failed to create triggers: %w", err)
		if n.logger != nil {
			n.logger.LogModuleError(n.sourceName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	// Verify central database table exists (we don't create it, just verify)
	if err := n.verifyCentralTable(ctx); err != nil {
		initErr := fmt.Errorf("failed to verify central table: %w", err)
		if n.logger != nil {
			n.logger.LogModuleError(n.sourceName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	if n.logger != nil {
		n.logger.LogSystemf(logger.LevelInfo, "notify", "Notify module initialized successfully for database: %s", n.sourceName)
	}

	return nil
}

// createNotificationTable creates the pgb.pgb_notify table in source database
func (n *NotifyModule) createNotificationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pgb.pgb_notify (
			id SERIAL PRIMARY KEY,
			user_email VARCHAR(255) NOT NULL,
			sender_db VARCHAR(100) NOT NULL,
			message TEXT,
			message_link VARCHAR(500),
			criticality SMALLINT DEFAULT 1 NOT NULL,
			is_sent BOOLEAN DEFAULT false NOT NULL,
			sent_ts TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT valid_criticality CHECK (criticality >= 1 AND criticality <= 5)
		);

		COMMENT ON TABLE pgb.pgb_notify IS 'Notification queue for pgbridge notify module';
		COMMENT ON COLUMN pgb.pgb_notify.criticality IS 'Criticality level: 1=Info, 2=Low, 3=Medium, 4=High, 5=Critical';
		COMMENT ON COLUMN pgb.pgb_notify.is_sent IS 'Whether notification was sent to central database';
	`

	_, err := n.sourcePool.Exec(ctx, query)
	return err
}

// createIndexes creates indexes for performance
func (n *NotifyModule) createIndexes(ctx context.Context) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_pgb_notify_is_sent
			ON pgb.pgb_notify(is_sent) WHERE is_sent = false;`,
		`CREATE INDEX IF NOT EXISTS idx_pgb_notify_user_email
			ON pgb.pgb_notify(user_email);`,
		`CREATE INDEX IF NOT EXISTS idx_pgb_notify_created_at
			ON pgb.pgb_notify(created_at);`,
	}

	for _, index := range indexes {
		if _, err := n.sourcePool.Exec(ctx, index); err != nil {
			return err
		}
	}

	return nil
}

// createTriggers creates the trigger function and trigger for automatic NOTIFY
func (n *NotifyModule) createTriggers(ctx context.Context) error {
	// Create the trigger function
	functionSQL := `
		CREATE OR REPLACE FUNCTION pgb.trg_pgb_send_notification()
		RETURNS TRIGGER AS $$
		BEGIN
			-- Send NOTIFY with the new notification ID
			PERFORM pg_notify('pgb_notify', NEW.id::text);

			-- Mark as sent (will be handled by pgbridge)
			-- Note: We don't set is_sent here as pgbridge will do it after forwarding

			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		COMMENT ON FUNCTION pgb.trg_pgb_send_notification() IS 'Automatically sends NOTIFY when notification is inserted';
	`

	if _, err := n.sourcePool.Exec(ctx, functionSQL); err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create the trigger
	triggerSQL := `
		DROP TRIGGER IF EXISTS S01_send_notification ON pgb.pgb_notify;

		CREATE TRIGGER S01_send_notification
		AFTER INSERT ON pgb.pgb_notify
		FOR EACH ROW
		EXECUTE FUNCTION pgb.trg_pgb_send_notification();

		COMMENT ON TRIGGER S01_send_notification ON pgb.pgb_notify IS 'Automatically triggers notification forwarding on insert';
	`

	if _, err := n.sourcePool.Exec(ctx, triggerSQL); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	return nil
}

// verifyCentralTable verifies the central database table exists
func (n *NotifyModule) verifyCentralTable(ctx context.Context) error {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'ps_notifications'
		)
	`

	var exists bool
	err := n.centralPool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check central table: %w", err)
	}

	if !exists {
		return fmt.Errorf("central table public.ps_notifications does not exist")
	}

	return nil
}

// Start begins the module operation
func (n *NotifyModule) Start(ctx context.Context) error {
	// No background tasks needed - notifications are forwarded on-demand via LISTEN/NOTIFY
	return nil
}

// Stop gracefully stops the module
func (n *NotifyModule) Stop() error {
	close(n.shutdown)
	n.wg.Wait()
	return nil
}

// ProcessNotification handles incoming NOTIFY messages with notification IDs
func (n *NotifyModule) ProcessNotification(ctx context.Context, payload string) error {
	// Parse the payload as a notification ID
	var notifyID int
	if _, err := fmt.Sscanf(payload, "%d", &notifyID); err != nil {
		return fmt.Errorf("invalid notification ID in payload '%s': %w", payload, err)
	}

	// Forward the notification
	return n.forwardNotification(ctx, notifyID)
}

// ProcessQueue processes unsent notifications from the queue (called at startup)
func (n *NotifyModule) ProcessQueue(ctx context.Context) error {
	// Find all unsent notifications
	query := `
		SELECT id FROM pgb.pgb_notify
		WHERE is_sent = false
		ORDER BY created_at ASC
	`

	rows, err := n.sourcePool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query unsent notifications: %w", err)
	}
	defer rows.Close()

	var notifyIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan notification ID: %w", err)
		}
		notifyIDs = append(notifyIDs, id)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating notification rows: %w", err)
	}

	// Process each unsent notification
	for _, notifyID := range notifyIDs {
		if err := n.forwardNotification(ctx, notifyID); err != nil {
			// Log error but continue processing other notifications
			continue
		}
	}

	return nil
}

// forwardNotification retrieves a notification and sends it to central database
func (n *NotifyModule) forwardNotification(ctx context.Context, notifyID int) error {
	// Retrieve notification from source database
	notification, err := n.getNotification(ctx, notifyID)
	if err != nil {
		notifyErr := fmt.Errorf("failed to retrieve notification %d: %w", notifyID, err)
		if n.logger != nil {
			n.logger.LogNotifyFailed(n.sourceName, notifyID, notifyErr)
		}
		return notifyErr
	}

	// Check if already sent
	if notification.IsSent {
		return nil // Already sent
	}

	// Insert into central database
	centralID, err := n.insertToCentral(ctx, notification)
	if err != nil {
		insertErr := fmt.Errorf("failed to insert to central database: %w", err)
		if n.logger != nil {
			n.logger.LogNotifyFailed(n.sourceName, notifyID, insertErr)
		}
		return insertErr
	}

	// Mark as sent in source database
	if err := n.markAsSent(ctx, notifyID); err != nil {
		markErr := fmt.Errorf("failed to mark as sent: %w", err)
		if n.logger != nil {
			n.logger.LogNotifyFailed(n.sourceName, notifyID, markErr)
		}
		return markErr
	}

	if n.logger != nil {
		n.logger.LogNotifyForwarded(n.sourceName, notifyID, centralID)
	}

	return nil
}

// getNotification retrieves a notification from the source database
func (n *NotifyModule) getNotification(ctx context.Context, notifyID int) (*Notification, error) {
	query := `
		SELECT id, user_email, sender_db, COALESCE(message, ''),
		       COALESCE(message_link, ''), criticality,
		       is_sent, sent_ts, created_at
		FROM pgb.pgb_notify
		WHERE id = $1
	`

	notification := &Notification{}
	err := n.sourcePool.QueryRow(ctx, query, notifyID).Scan(
		&notification.ID,
		&notification.UserEmail,
		&notification.SenderDB,
		&notification.Message,
		&notification.MessageLink,
		&notification.Criticality,
		&notification.IsSent,
		&notification.SentTS,
		&notification.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query notification: %w", err)
	}

	return notification, nil
}

// insertToCentral inserts a notification into the central database
func (n *NotifyModule) insertToCentral(ctx context.Context, notification *Notification) (int, error) {
	query := `
		INSERT INTO public.ps_notifications (
			user_email,
			sender_db,
			original_id,
			message,
			message_link,
			criticality,
			received_ts
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var centralID int
	err := n.centralPool.QueryRow(ctx, query,
		notification.UserEmail,
		notification.SenderDB,
		notification.ID,
		notification.Message,
		notification.MessageLink,
		notification.Criticality,
	).Scan(&centralID)

	if err != nil {
		return 0, fmt.Errorf("failed to insert into central database: %w", err)
	}

	return centralID, nil
}

// markAsSent updates the notification record as successfully sent
func (n *NotifyModule) markAsSent(ctx context.Context, notifyID int) error {
	query := `
		UPDATE pgb.pgb_notify
		SET is_sent = true,
		    sent_ts = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := n.sourcePool.Exec(ctx, query, notifyID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	return nil
}

