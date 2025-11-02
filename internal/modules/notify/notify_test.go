package notify

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/pgbridge_test?sslmode=disable"
	}

	sourcePool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("Failed to create source pool: %v", err)
	}

	// For testing, use the same database as "central"
	// In production, these would be different databases
	centralPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		sourcePool.Close()
		t.Fatalf("Failed to create central pool: %v", err)
	}

	return sourcePool, centralPool
}

func cleanupTables(t *testing.T, sourcePool, centralPool *pgxpool.Pool) {
	ctx := context.Background()

	// Drop tables in source database
	queries := []string{
		"DROP TABLE IF EXISTS pgb.pgb_notify CASCADE",
		"DROP TABLE IF EXISTS public.ps_notifications CASCADE",
	}

	for _, query := range queries {
		sourcePool.Exec(ctx, query)
		centralPool.Exec(ctx, query)
	}
}

func createCentralTable(t *testing.T, centralPool *pgxpool.Pool) {
	ctx := context.Background()
	query := `
		CREATE TABLE IF NOT EXISTS public.ps_notifications (
			id SERIAL PRIMARY KEY,
			user_email VARCHAR NOT NULL,
			received_ts TIMESTAMP,
			sender_db VARCHAR NOT NULL,
			original_id INT NOT NULL,
			message TEXT,
			message_link VARCHAR,
			is_seen BOOLEAN DEFAULT false NOT NULL,
			seen_ts TIMESTAMP,
			criticality SMALLINT DEFAULT 1 NOT NULL
		)
	`
	_, err := centralPool.Exec(ctx, query)
	if err != nil {
		t.Fatalf("Failed to create central table: %v", err)
	}
}

func TestNotifyModule_Name(t *testing.T) {
	module := NewNotifyModule(nil, nil, "test_db", nil)
	if module.Name() != "pgb_notify" {
		t.Errorf("Expected name 'pgb_notify', got '%s'", module.Name())
	}
}

func TestNotifyModule_GetChannelName(t *testing.T) {
	module := NewNotifyModule(nil, nil, "test_db", nil)
	if module.GetChannelName() != "pgb_notify" {
		t.Errorf("Expected channel 'pgb_notify', got '%s'", module.GetChannelName())
	}
}

func TestNotifyModule_Initialize(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	// Create central table first
	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify pgb_notify table exists in source database
	var exists bool
	err = sourcePool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'pgb' AND table_name = 'pgb_notify'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("pgb_notify table was not created")
	}
}

func TestNotifyModule_InitializeIdempotent(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize once
	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("First initialize failed: %v", err)
	}

	// Initialize again - should not error
	err = module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Errorf("Second initialize failed: %v", err)
	}
}

func TestNotifyModule_CreateNotification(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create a notification
	var notifyID int
	err = sourcePool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_notify (user_email, sender_db, message, criticality)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "user@example.com", "test_db", "Test notification", 2).Scan(&notifyID)

	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	if notifyID <= 0 {
		t.Error("Expected positive notification ID")
	}

	// Retrieve the notification
	notification, err := module.getNotification(ctx, notifyID)
	if err != nil {
		t.Fatalf("Failed to get notification: %v", err)
	}

	if notification.UserEmail != "user@example.com" {
		t.Errorf("Expected user_email 'user@example.com', got '%s'", notification.UserEmail)
	}
	if notification.SenderDB != "test_db" {
		t.Errorf("Expected sender_db 'test_db', got '%s'", notification.SenderDB)
	}
	if notification.Criticality != 2 {
		t.Errorf("Expected criticality 2, got %d", notification.Criticality)
	}
	if notification.IsSent {
		t.Error("Expected is_sent to be false")
	}
}

func TestNotifyModule_ForwardNotification(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create a notification in source database
	var notifyID int
	err = sourcePool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_notify (user_email, sender_db, message, message_link, criticality)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, "user@example.com", "test_db", "Test notification", "http://example.com", 3).Scan(&notifyID)

	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	// Forward the notification
	err = module.forwardNotification(ctx, notifyID)
	if err != nil {
		t.Fatalf("forwardNotification failed: %v", err)
	}

	// Verify it was marked as sent in source database
	var isSent bool
	err = sourcePool.QueryRow(ctx, `
		SELECT is_sent FROM pgb.pgb_notify WHERE id = $1
	`, notifyID).Scan(&isSent)

	if err != nil {
		t.Fatalf("Failed to query is_sent: %v", err)
	}

	if !isSent {
		t.Error("Expected is_sent to be true")
	}

	// Verify it exists in central database
	var centralID int
	var userEmail, senderDB string
	var originalID int
	err = centralPool.QueryRow(ctx, `
		SELECT id, user_email, sender_db, original_id
		FROM public.ps_notifications
		WHERE sender_db = $1 AND original_id = $2
	`, "test_db", notifyID).Scan(&centralID, &userEmail, &senderDB, &originalID)

	if err != nil {
		t.Fatalf("Failed to query central database: %v", err)
	}

	if userEmail != "user@example.com" {
		t.Errorf("Expected user_email 'user@example.com', got '%s'", userEmail)
	}
	if senderDB != "test_db" {
		t.Errorf("Expected sender_db 'test_db', got '%s'", senderDB)
	}
	if originalID != notifyID {
		t.Errorf("Expected original_id %d, got %d", notifyID, originalID)
	}
}

func TestNotifyModule_ProcessQueue(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create multiple unsent notifications
	for i := 0; i < 3; i++ {
		_, err = sourcePool.Exec(ctx, `
			INSERT INTO pgb.pgb_notify (user_email, sender_db, message, criticality)
			VALUES ($1, $2, $3, $4)
		`, "user@example.com", "test_db", "Test notification", 1)
		if err != nil {
			t.Fatalf("Failed to insert notification: %v", err)
		}
	}

	// Process the queue
	err = module.ProcessQueue(ctx)
	if err != nil {
		t.Fatalf("ProcessQueue failed: %v", err)
	}

	// Verify all were sent
	var sentCount int
	err = sourcePool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pgb.pgb_notify WHERE is_sent = true
	`).Scan(&sentCount)

	if err != nil {
		t.Fatalf("Failed to count sent notifications: %v", err)
	}

	if sentCount != 3 {
		t.Errorf("Expected 3 sent notifications, got %d", sentCount)
	}

	// Verify all exist in central database
	var centralCount int
	err = centralPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM public.ps_notifications WHERE sender_db = 'test_db'
	`).Scan(&centralCount)

	if err != nil {
		t.Fatalf("Failed to count central notifications: %v", err)
	}

	if centralCount != 3 {
		t.Errorf("Expected 3 notifications in central database, got %d", centralCount)
	}
}

func TestNotifyModule_ProcessNotification(t *testing.T) {
	sourcePool, centralPool := getTestPools(t)
	defer sourcePool.Close()
	defer centralPool.Close()

	cleanupTables(t, sourcePool, centralPool)
	defer cleanupTables(t, sourcePool, centralPool)

	createCentralTable(t, centralPool)

	module := NewNotifyModule(sourcePool, centralPool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, sourcePool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create a notification
	var notifyID int
	err = sourcePool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_notify (user_email, sender_db, message, criticality)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "user@example.com", "test_db", "Test notification", 1).Scan(&notifyID)

	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	// Process the notification via NOTIFY simulation
	payload := fmt.Sprintf("%d", notifyID)
	err = module.ProcessNotification(ctx, payload)
	if err != nil {
		t.Fatalf("ProcessNotification failed: %v", err)
	}

	// Verify it was sent
	var isSent bool
	err = sourcePool.QueryRow(ctx, `
		SELECT is_sent FROM pgb.pgb_notify WHERE id = $1
	`, notifyID).Scan(&isSent)

	if err != nil {
		t.Fatalf("Failed to query is_sent: %v", err)
	}

	if !isSent {
		t.Error("Expected is_sent to be true after ProcessNotification")
	}
}

func TestNotifyModule_ProcessNotificationInvalidPayload(t *testing.T) {
	module := NewNotifyModule(nil, nil, "test_db", nil)
	ctx := context.Background()

	err := module.ProcessNotification(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid payload")
	}

	err = module.ProcessNotification(ctx, "not_a_number")
	if err == nil {
		t.Error("Expected error for non-numeric payload")
	}
}
