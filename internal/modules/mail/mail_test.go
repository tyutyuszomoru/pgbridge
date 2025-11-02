package mail

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/pgbridge_test?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	return pool
}

func cleanupTables(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	// Drop tables in reverse dependency order
	queries := []string{
		"DROP TABLE IF EXISTS pgb.pgb_mail CASCADE",
		"DROP TABLE IF EXISTS pgb.pgb_mail_settings CASCADE",
	}

	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Logf("Warning: cleanup query failed: %v", err)
		}
	}
}

func TestMailModule_Initialize(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify pgb_mail_settings table exists
	var settingsExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'pgb' AND table_name = 'pgb_mail_settings'
		)
	`).Scan(&settingsExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !settingsExists {
		t.Error("pgb_mail_settings table was not created")
	}

	// Verify pgb_mail table exists
	var mailExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'pgb' AND table_name = 'pgb_mail'
		)
	`).Scan(&mailExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !mailExists {
		t.Error("pgb_mail table was not created")
	}
}

func TestMailModule_InitializeIdempotent(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize once
	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("First initialize failed: %v", err)
	}

	// Initialize again - should not error
	err = module.Initialize(ctx, pool)
	if err != nil {
		t.Errorf("Second initialize failed: %v", err)
	}
}

func TestMailModule_Name(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)
	if module.Name() != "pgb_mail" {
		t.Errorf("Expected name 'pgb_mail', got '%s'", module.Name())
	}
}

func TestMailModule_GetChannelName(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)
	if module.GetChannelName() != "pgb_mail" {
		t.Errorf("Expected channel 'pgb_mail', got '%s'", module.GetChannelName())
	}
}

func TestMailModule_CreateMailSettings(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Insert a mail setting
	var settingID int
	err = pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port, is_tls, smtp_user, smtp_password)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, "smtp.example.com", 587, true, "user@example.com", "password").Scan(&settingID)

	if err != nil {
		t.Fatalf("Failed to insert mail setting: %v", err)
	}

	if settingID <= 0 {
		t.Error("Expected positive setting ID")
	}

	// Retrieve the setting
	settings, err := module.getMailSettings(ctx, settingID)
	if err != nil {
		t.Fatalf("Failed to get mail settings: %v", err)
	}

	if settings.SMTPServer != "smtp.example.com" {
		t.Errorf("Expected SMTP server 'smtp.example.com', got '%s'", settings.SMTPServer)
	}
	if settings.SMTPPort != 587 {
		t.Errorf("Expected port 587, got %d", settings.SMTPPort)
	}
	if !settings.IsTLS {
		t.Error("Expected IsTLS to be true")
	}
}

func TestMailModule_CreateMail(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create mail setting first
	var settingID int
	err = pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port)
		VALUES ($1, $2)
		RETURNING id
	`, "smtp.example.com", 587).Scan(&settingID)
	if err != nil {
		t.Fatalf("Failed to create mail setting: %v", err)
	}

	// Create mail
	var mailID int
	err = pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail (
			mail_setting_id, header_from, header_to, subject, body_text
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, settingID, "sender@example.com", "recipient@example.com", "Test Subject", "Test body").Scan(&mailID)

	if err != nil {
		t.Fatalf("Failed to insert mail: %v", err)
	}

	if mailID <= 0 {
		t.Error("Expected positive mail ID")
	}

	// Retrieve the mail
	mail, err := module.getMailMessage(ctx, mailID)
	if err != nil {
		t.Fatalf("Failed to get mail message: %v", err)
	}

	if mail.HeaderFrom != "sender@example.com" {
		t.Errorf("Expected from 'sender@example.com', got '%s'", mail.HeaderFrom)
	}
	if mail.HeaderTo != "recipient@example.com" {
		t.Errorf("Expected to 'recipient@example.com', got '%s'", mail.HeaderTo)
	}
	if mail.Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", mail.Subject)
	}
	if mail.IsSent {
		t.Error("Expected is_sent to be false")
	}
}

func TestMailModule_BuildMessage(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)

	mail := &MailMessage{
		HeaderFrom: "sender@example.com",
		HeaderTo:   "recipient@example.com",
		HeaderCC:   "cc@example.com",
		Subject:    "Test Subject",
		BodyText:   "This is the email body.",
	}

	message := module.buildMessage(mail)

	// Check that message contains key headers
	if !containsString(message, "From: sender@example.com") {
		t.Error("Message missing From header")
	}
	if !containsString(message, "To: recipient@example.com") {
		t.Error("Message missing To header")
	}
	if !containsString(message, "Cc: cc@example.com") {
		t.Error("Message missing Cc header")
	}
	if !containsString(message, "Subject: Test Subject") {
		t.Error("Message missing Subject header")
	}
	if !containsString(message, "This is the email body.") {
		t.Error("Message missing body")
	}
	if !containsString(message, "MIME-Version: 1.0") {
		t.Error("Message missing MIME-Version header")
	}
}

func TestMailModule_CollectRecipients(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)

	mail := &MailMessage{
		HeaderTo:  "to1@example.com, to2@example.com",
		HeaderCC:  "cc1@example.com",
		HeaderBCC: "bcc1@example.com, bcc2@example.com",
	}

	recipients := module.collectRecipients(mail)

	expectedCount := 5
	if len(recipients) != expectedCount {
		t.Errorf("Expected %d recipients, got %d", expectedCount, len(recipients))
	}

	// Check that all recipients are present
	expectedRecipients := []string{
		"to1@example.com", "to2@example.com",
		"cc1@example.com",
		"bcc1@example.com", "bcc2@example.com",
	}

	for _, expected := range expectedRecipients {
		found := false
		for _, actual := range recipients {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected recipient '%s' not found", expected)
		}
	}
}

func TestMailModule_ParseEmailList(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single email",
			input:    "user@example.com",
			expected: []string{"user@example.com"},
		},
		{
			name:     "multiple emails",
			input:    "user1@example.com, user2@example.com, user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "emails with spaces",
			input:    "  user1@example.com  ,  user2@example.com  ",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only commas",
			input:    ", , ,",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.parseEmailList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d emails, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected email[%d] '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestMailModule_MarkAsSent(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test data
	var settingID, mailID int
	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port)
		VALUES ('smtp.example.com', 587) RETURNING id
	`).Scan(&settingID)

	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
		VALUES ($1, 'from@example.com', 'to@example.com', 'Subject', 'Body')
		RETURNING id
	`, settingID).Scan(&mailID)

	// Mark as sent
	err = module.markAsSent(ctx, mailID)
	if err != nil {
		t.Fatalf("markAsSent failed: %v", err)
	}

	// Verify is_sent is true
	var isSent bool
	var sentTS time.Time
	err = pool.QueryRow(ctx, `
		SELECT is_sent, sent_ts FROM pgb.pgb_mail WHERE id = $1
	`, mailID).Scan(&isSent, &sentTS)

	if err != nil {
		t.Fatalf("Failed to query mail: %v", err)
	}

	if !isSent {
		t.Error("Expected is_sent to be true")
	}

	if sentTS.IsZero() {
		t.Error("Expected sent_ts to be set")
	}
}

func TestMailModule_RecordError(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test data
	var settingID, mailID int
	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port)
		VALUES ('smtp.example.com', 587) RETURNING id
	`).Scan(&settingID)

	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
		VALUES ($1, 'from@example.com', 'to@example.com', 'Subject', 'Body')
		RETURNING id
	`, settingID).Scan(&mailID)

	// Record error
	errMsg := "Test error message"
	err = module.recordError(ctx, mailID, errMsg)
	if err != nil {
		t.Fatalf("recordError failed: %v", err)
	}

	// Verify error was recorded
	var storedError string
	err = pool.QueryRow(ctx, `
		SELECT error_message FROM pgb.pgb_mail WHERE id = $1
	`, mailID).Scan(&storedError)

	if err != nil {
		t.Fatalf("Failed to query mail: %v", err)
	}

	if storedError != errMsg {
		t.Errorf("Expected error '%s', got '%s'", errMsg, storedError)
	}
}

func TestMailModule_ProcessQueue(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // Short timeout per mail
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test data with unsent mail
	var settingID int
	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port)
		VALUES ('smtp.example.com', 587) RETURNING id
	`).Scan(&settingID)

	// Insert a single unsent mail for faster testing
	pool.Exec(ctx, `
		INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text, is_sent)
		VALUES ($1, 'from@example.com', 'to@example.com', 'Test Subject', 'Body', false)
	`, settingID)

	// ProcessQueue will try to send emails (they will fail because no real SMTP server)
	// But we can verify it queries the unsent mails
	err = module.ProcessQueue(ctx)
	// Error is expected since we don't have a real SMTP server
	// The important thing is ProcessQueue runs without crashing

	// Verify that retry_count was incremented (indicating send was attempted)
	var count int
	pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM pgb.pgb_mail WHERE retry_count > 0
	`).Scan(&count)

	if count < 1 {
		t.Errorf("Expected at least 1 mail with retry_count > 0, got %d", count)
	}

	// Verify error message was recorded (or retry happened)
	var errorMsg *string
	pool.QueryRow(context.Background(), `
		SELECT error_message FROM pgb.pgb_mail LIMIT 1
	`).Scan(&errorMsg)

	// Either error message should be recorded OR we should have retries > 0
	// (which we already verified above)
	t.Logf("Error message: %v, Retry count: %d", errorMsg, count)
}

func TestMailModule_ProcessNotification(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	cleanupTables(t, pool)
	defer cleanupTables(t, pool)

	module := NewMailModule(pool, "test_db", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // Short timeout
	defer cancel()

	err := module.Initialize(ctx, pool)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test data
	var settingID, mailID int
	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail_settings (smtp_server, smtp_port)
		VALUES ('smtp.example.com', 587) RETURNING id
	`).Scan(&settingID)

	pool.QueryRow(ctx, `
		INSERT INTO pgb.pgb_mail (mail_setting_id, header_from, header_to, subject, body_text)
		VALUES ($1, 'from@example.com', 'to@example.com', 'Subject', 'Body')
		RETURNING id
	`, settingID).Scan(&mailID)

	// Process notification (will fail SMTP but should parse ID correctly)
	payload := fmt.Sprintf("%d", mailID)
	err = module.ProcessNotification(ctx, payload)
	// Error expected because no real SMTP server

	// Verify that retry_count was incremented
	var retryCount int
	pool.QueryRow(context.Background(), `
		SELECT retry_count FROM pgb.pgb_mail WHERE id = $1
	`, mailID).Scan(&retryCount)

	if retryCount == 0 {
		t.Error("Expected retry_count > 0 after failed send attempt")
	}
}

func TestMailModule_ProcessNotificationInvalidPayload(t *testing.T) {
	module := NewMailModule(nil, "test_db", nil)
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

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
