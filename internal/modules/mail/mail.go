package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

const (
	moduleName   = "pgb_mail"
	channelName  = "pgb_mail"
	maxRetries   = 3
	retryDelay   = 5 * time.Second
)

// MailModule handles asynchronous email sending from PostgreSQL
type MailModule struct {
	pool     *pgxpool.Pool
	dbName   string
	logger   *logger.Logger
	mu       sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// MailSettings represents SMTP configuration
type MailSettings struct {
	ID           int
	SMTPServer   string
	SMTPPort     int
	IsTLS        bool
	IsSSL        bool
	SMTPUser     string
	SMTPPassword string
	SMTPToken    string
}

// MailMessage represents an email to be sent
type MailMessage struct {
	ID            int
	MailSettingID int
	HeaderFrom    string
	HeaderTo      string
	HeaderCC      string
	HeaderBCC     string
	Subject       string
	BodyText      string
	IsSent        bool
	SentTS        *time.Time
	ErrorMessage  string
}

// NewMailModule creates a new mail module instance
func NewMailModule(pool *pgxpool.Pool, dbName string, log *logger.Logger) *MailModule {
	return &MailModule{
		pool:     pool,
		dbName:   dbName,
		logger:   log,
		shutdown: make(chan struct{}),
	}
}

// Name returns the module name
func (m *MailModule) Name() string {
	return moduleName
}

// GetChannelName returns the NOTIFY channel name
func (m *MailModule) GetChannelName() string {
	return channelName
}

// Initialize creates the required tables and indexes
func (m *MailModule) Initialize(ctx context.Context, pool *pgxpool.Pool) error {
	if m.logger != nil {
		m.logger.LogModuleInit(m.dbName, moduleName)
	}

	// Create pgb_mail_settings table
	if err := m.createMailSettingsTable(ctx); err != nil {
		initErr := fmt.Errorf("failed to create mail_settings table: %w", err)
		if m.logger != nil {
			m.logger.LogModuleError(m.dbName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	// Create pgb_mail table
	if err := m.createMailTable(ctx); err != nil {
		initErr := fmt.Errorf("failed to create mail table: %w", err)
		if m.logger != nil {
			m.logger.LogModuleError(m.dbName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	// Create indexes
	if err := m.createIndexes(ctx); err != nil {
		initErr := fmt.Errorf("failed to create indexes: %w", err)
		if m.logger != nil {
			m.logger.LogModuleError(m.dbName, moduleName, "initialize", initErr)
		}
		return initErr
	}

	if m.logger != nil {
		m.logger.LogSystemf(logger.LevelInfo, "mail", "Mail module initialized successfully for database: %s", m.dbName)
	}

	return nil
}

// createMailSettingsTable creates the pgb.pgb_mail_settings table
func (m *MailModule) createMailSettingsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pgb.pgb_mail_settings (
			id SERIAL PRIMARY KEY,
			smtp_server VARCHAR(255) NOT NULL,
			smtp_port INTEGER NOT NULL DEFAULT 587,
			is_tls BOOLEAN DEFAULT true,
			is_ssl BOOLEAN DEFAULT false,
			smtp_user VARCHAR(255),
			smtp_password VARCHAR(255),
			smtp_token TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT valid_port CHECK (smtp_port > 0 AND smtp_port <= 65535)
		);

		COMMENT ON TABLE pgb.pgb_mail_settings IS 'SMTP server configuration for pgbridge mail module';
		COMMENT ON COLUMN pgb.pgb_mail_settings.is_tls IS 'Use STARTTLS encryption';
		COMMENT ON COLUMN pgb.pgb_mail_settings.is_ssl IS 'Use SSL/TLS from the start';
		COMMENT ON COLUMN pgb.pgb_mail_settings.smtp_token IS 'API token for token-based authentication';
	`

	_, err := m.pool.Exec(ctx, query)
	return err
}

// createMailTable creates the pgb.pgb_mail table
func (m *MailModule) createMailTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pgb.pgb_mail (
			id SERIAL PRIMARY KEY,
			mail_setting_id INTEGER NOT NULL,
			header_from VARCHAR(255) NOT NULL,
			header_to TEXT NOT NULL,
			header_cc TEXT,
			header_bcc TEXT,
			subject VARCHAR(998) NOT NULL,
			body_text TEXT NOT NULL,
			is_sent BOOLEAN DEFAULT false,
			sent_ts TIMESTAMP,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_mail_setting
				FOREIGN KEY (mail_setting_id)
				REFERENCES pgb.pgb_mail_settings(id)
				ON DELETE RESTRICT,
			CONSTRAINT valid_email_to CHECK (header_to <> ''),
			CONSTRAINT valid_subject CHECK (subject <> '')
		);

		COMMENT ON TABLE pgb.pgb_mail IS 'Email queue for pgbridge mail module';
		COMMENT ON COLUMN pgb.pgb_mail.header_to IS 'Comma-separated list of recipient email addresses';
		COMMENT ON COLUMN pgb.pgb_mail.header_cc IS 'Comma-separated list of CC email addresses';
		COMMENT ON COLUMN pgb.pgb_mail.header_bcc IS 'Comma-separated list of BCC email addresses';
		COMMENT ON COLUMN pgb.pgb_mail.retry_count IS 'Number of send attempts';
	`

	_, err := m.pool.Exec(ctx, query)
	return err
}

// createIndexes creates indexes for performance
func (m *MailModule) createIndexes(ctx context.Context) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_pgb_mail_is_sent
			ON pgb.pgb_mail(is_sent) WHERE is_sent = false;`,
		`CREATE INDEX IF NOT EXISTS idx_pgb_mail_created_at
			ON pgb.pgb_mail(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_pgb_mail_setting_id
			ON pgb.pgb_mail(mail_setting_id);`,
	}

	for _, index := range indexes {
		if _, err := m.pool.Exec(ctx, index); err != nil {
			return err
		}
	}

	return nil
}

// Start begins processing (currently a no-op, processing happens on notifications)
func (m *MailModule) Start(ctx context.Context) error {
	return nil
}

// Stop gracefully stops the module
func (m *MailModule) Stop() error {
	close(m.shutdown)
	m.wg.Wait()
	return nil
}

// ProcessNotification handles incoming NOTIFY messages with mail IDs
func (m *MailModule) ProcessNotification(ctx context.Context, payload string) error {
	// Parse the payload as a mail ID
	var mailID int
	if _, err := fmt.Sscanf(payload, "%d", &mailID); err != nil {
		return fmt.Errorf("invalid mail ID in payload '%s': %w", payload, err)
	}

	// Send the mail
	return m.sendMail(ctx, mailID)
}

// ProcessQueue processes unsent emails from the queue (called at startup)
func (m *MailModule) ProcessQueue(ctx context.Context) error {
	// Find all unsent emails
	query := `
		SELECT id FROM pgb.pgb_mail
		WHERE is_sent = false
		AND retry_count < $1
		ORDER BY created_at ASC
	`

	rows, err := m.pool.Query(ctx, query, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to query unsent mails: %w", err)
	}
	defer rows.Close()

	var mailIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan mail ID: %w", err)
		}
		mailIDs = append(mailIDs, id)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating mail rows: %w", err)
	}

	// Process each unsent mail
	for _, mailID := range mailIDs {
		if err := m.sendMail(ctx, mailID); err != nil {
			// Log error but continue processing other mails
			// The error is already recorded in the database by sendMail
			continue
		}
	}

	return nil
}

// sendMail retrieves a mail message and sends it
func (m *MailModule) sendMail(ctx context.Context, mailID int) error {
	// Retrieve mail message
	mail, err := m.getMailMessage(ctx, mailID)
	if err != nil {
		mailErr := fmt.Errorf("failed to retrieve mail %d: %w", mailID, err)
		if m.logger != nil {
			m.logger.LogMailFailed(m.dbName, mailID, mailErr)
		}
		return mailErr
	}

	// Retrieve mail settings
	settings, err := m.getMailSettings(ctx, mail.MailSettingID)
	if err != nil {
		settingsErr := fmt.Errorf("failed to retrieve mail settings: %w", err)
		m.recordError(ctx, mailID, fmt.Sprintf("Failed to retrieve mail settings: %v", err))
		if m.logger != nil {
			m.logger.LogMailFailed(m.dbName, mailID, settingsErr)
		}
		return settingsErr
	}

	// Send the email with retry logic
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			if m.logger != nil {
				m.logger.LogSystemf(logger.LevelInfo, "mail", "Retrying email send for mail_id=%d (attempt %d/%d)", mailID, attempt+1, maxRetries)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		if err := m.sendSMTP(ctx, mail, settings); err != nil {
			lastErr = err
			m.incrementRetryCount(ctx, mailID)
			continue
		}

		// Success!
		if m.logger != nil {
			m.logger.LogMailSent(m.dbName, mailID, mail.HeaderTo)
		}
		return m.markAsSent(ctx, mailID)
	}

	// All retries failed
	errMsg := fmt.Sprintf("Failed after %d attempts: %v", maxRetries, lastErr)
	m.recordError(ctx, mailID, errMsg)
	finalErr := fmt.Errorf("failed to send mail %d: %w", mailID, lastErr)
	if m.logger != nil {
		m.logger.LogMailFailed(m.dbName, mailID, finalErr)
	}
	return finalErr
}

// getMailMessage retrieves a mail message from the database
func (m *MailModule) getMailMessage(ctx context.Context, mailID int) (*MailMessage, error) {
	query := `
		SELECT id, mail_setting_id, header_from, header_to,
		       COALESCE(header_cc, ''), COALESCE(header_bcc, ''),
		       subject, body_text, is_sent, sent_ts, COALESCE(error_message, '')
		FROM pgb.pgb_mail
		WHERE id = $1
	`

	mail := &MailMessage{}
	err := m.pool.QueryRow(ctx, query, mailID).Scan(
		&mail.ID,
		&mail.MailSettingID,
		&mail.HeaderFrom,
		&mail.HeaderTo,
		&mail.HeaderCC,
		&mail.HeaderBCC,
		&mail.Subject,
		&mail.BodyText,
		&mail.IsSent,
		&mail.SentTS,
		&mail.ErrorMessage,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query mail: %w", err)
	}

	return mail, nil
}

// getMailSettings retrieves SMTP settings from the database
func (m *MailModule) getMailSettings(ctx context.Context, settingID int) (*MailSettings, error) {
	query := `
		SELECT id, smtp_server, smtp_port, is_tls, is_ssl,
		       COALESCE(smtp_user, ''), COALESCE(smtp_password, ''), COALESCE(smtp_token, '')
		FROM pgb.pgb_mail_settings
		WHERE id = $1
	`

	settings := &MailSettings{}
	err := m.pool.QueryRow(ctx, query, settingID).Scan(
		&settings.ID,
		&settings.SMTPServer,
		&settings.SMTPPort,
		&settings.IsTLS,
		&settings.IsSSL,
		&settings.SMTPUser,
		&settings.SMTPPassword,
		&settings.SMTPToken,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query mail settings: %w", err)
	}

	return settings, nil
}

// sendSMTP sends an email via SMTP
func (m *MailModule) sendSMTP(ctx context.Context, mail *MailMessage, settings *MailSettings) error {
	// Build the email message
	message := m.buildMessage(mail)

	// Prepare SMTP address
	addr := fmt.Sprintf("%s:%d", settings.SMTPServer, settings.SMTPPort)

	// Collect all recipients
	recipients := m.collectRecipients(mail)
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Setup authentication
	var auth smtp.Auth
	if settings.SMTPUser != "" && settings.SMTPPassword != "" {
		auth = smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPassword, settings.SMTPServer)
	}

	// Create a channel to receive the result
	errChan := make(chan error, 1)

	// Run SMTP send in a goroutine with context timeout
	go func() {
		var err error
		// Send email based on TLS/SSL configuration
		if settings.IsSSL {
			// SSL/TLS from the start
			err = m.sendWithTLS(addr, auth, mail.HeaderFrom, recipients, message)
		} else if settings.IsTLS {
			// STARTTLS
			err = smtp.SendMail(addr, auth, mail.HeaderFrom, recipients, []byte(message))
		} else {
			// Plain connection (not recommended)
			err = smtp.SendMail(addr, auth, mail.HeaderFrom, recipients, []byte(message))
		}
		errChan <- err
	}()

	// Wait for result or context timeout
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("SMTP send timeout: %w", ctx.Err())
	}
}

// sendWithTLS sends email with TLS/SSL from the start
func (m *MailModule) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg string) error {
	// Parse host from addr
	host := strings.Split(addr, ":")[0]

	// TLS config
	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}

	// Connect to SMTP server with TLS
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to initialize data transfer: %w", err)
	}

	_, err = writer.Write([]byte(msg))
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

// buildMessage constructs the email message with headers
func (m *MailModule) buildMessage(mail *MailMessage) string {
	var builder strings.Builder

	// From header
	builder.WriteString(fmt.Sprintf("From: %s\r\n", mail.HeaderFrom))

	// To header
	builder.WriteString(fmt.Sprintf("To: %s\r\n", mail.HeaderTo))

	// CC header
	if mail.HeaderCC != "" {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", mail.HeaderCC))
	}

	// Subject
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", mail.Subject))

	// MIME headers
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: 8bit\r\n")

	// Blank line separating headers from body
	builder.WriteString("\r\n")

	// Body
	builder.WriteString(mail.BodyText)

	return builder.String()
}

// collectRecipients gathers all recipients (To, CC, BCC)
func (m *MailModule) collectRecipients(mail *MailMessage) []string {
	var recipients []string

	// Add To recipients
	if mail.HeaderTo != "" {
		recipients = append(recipients, m.parseEmailList(mail.HeaderTo)...)
	}

	// Add CC recipients
	if mail.HeaderCC != "" {
		recipients = append(recipients, m.parseEmailList(mail.HeaderCC)...)
	}

	// Add BCC recipients
	if mail.HeaderBCC != "" {
		recipients = append(recipients, m.parseEmailList(mail.HeaderBCC)...)
	}

	return recipients
}

// parseEmailList parses a comma-separated list of email addresses
func (m *MailModule) parseEmailList(emailList string) []string {
	var emails []string
	parts := strings.Split(emailList, ",")
	for _, part := range parts {
		email := strings.TrimSpace(part)
		if email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

// markAsSent updates the mail record as successfully sent
func (m *MailModule) markAsSent(ctx context.Context, mailID int) error {
	query := `
		UPDATE pgb.pgb_mail
		SET is_sent = true,
		    sent_ts = CURRENT_TIMESTAMP,
		    error_message = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := m.pool.Exec(ctx, query, mailID)
	if err != nil {
		return fmt.Errorf("failed to mark mail as sent: %w", err)
	}

	return nil
}

// recordError records an error message for a mail
func (m *MailModule) recordError(ctx context.Context, mailID int, errorMsg string) error {
	query := `
		UPDATE pgb.pgb_mail
		SET error_message = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := m.pool.Exec(ctx, query, mailID, errorMsg)
	return err
}

// incrementRetryCount increments the retry counter
func (m *MailModule) incrementRetryCount(ctx context.Context, mailID int) error {
	query := `
		UPDATE pgb.pgb_mail
		SET retry_count = retry_count + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := m.pool.Exec(ctx, query, mailID)
	return err
}
