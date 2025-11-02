package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EventType constants for different log events
const (
	EventServiceStart       = "SERVICE_START"
	EventServiceStop        = "SERVICE_STOP"
	EventConfigLoaded       = "CONFIG_LOADED"
	EventConfigError        = "CONFIG_ERROR"
	EventDBConnectSuccess   = "DB_CONNECT_SUCCESS"
	EventDBConnectFail      = "DB_CONNECT_FAIL"
	EventDBDisconnect       = "DB_DISCONNECT"
	EventDBReconnect        = "DB_RECONNECT"
	EventListenerStarted    = "LISTENER_STARTED"
	EventListenerStopped    = "LISTENER_STOPPED"
	EventListenerError      = "LISTENER_ERROR"
	EventModuleInit         = "MODULE_INIT"
	EventModuleStart        = "MODULE_START"
	EventModuleStop         = "MODULE_STOP"
	EventModuleError        = "MODULE_ERROR"
	EventNotificationRecv   = "NOTIFICATION_RECV"
	EventNotificationProc   = "NOTIFICATION_PROC"
	EventNotificationError  = "NOTIFICATION_ERROR"
	EventQueueProcess       = "QUEUE_PROCESS"
	EventMailSent           = "MAIL_SENT"
	EventMailFailed         = "MAIL_FAILED"
	EventNotifyForwarded    = "NOTIFY_FORWARDED"
	EventNotifyFailed       = "NOTIFY_FAILED"
	EventHealthCheck        = "HEALTH_CHECK"
	EventHealthCheckFail    = "HEALTH_CHECK_FAIL"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry for database logging
type LogEntry struct {
	EventType    string
	DatabaseName string
	ModuleName   string
	Message      string
	Details      map[string]interface{}
}

// Logger handles both system (stdout) and database logging
type Logger struct {
	serviceName  string
	systemLogger *log.Logger
	dbPool       *pgxpool.Pool
	logChan      chan *LogEntry
	wg           sync.WaitGroup
	shutdown     chan struct{}
	shutdownOnce sync.Once
	mu           sync.RWMutex
}

// NewLogger creates a new logger instance
func NewLogger(serviceName string, dbPool *pgxpool.Pool) *Logger {
	return &Logger{
		serviceName:  serviceName,
		systemLogger: log.New(os.Stdout, fmt.Sprintf("[%s] ", serviceName), log.LstdFlags|log.Lmsgprefix),
		dbPool:       dbPool,
		logChan:      make(chan *LogEntry, 1000), // Buffered channel for async writes
		shutdown:     make(chan struct{}),
	}
}

// Start begins the async database logging goroutine
func (l *Logger) Start(ctx context.Context) {
	l.wg.Add(1)
	go l.databaseLogWriter(ctx)
}

// Shutdown gracefully stops the logger, flushing all pending logs
func (l *Logger) Shutdown() {
	l.shutdownOnce.Do(func() {
		close(l.shutdown)
		l.wg.Wait()
	})
}

// LogSystem logs a message to system output (stdout/journald)
func (l *Logger) LogSystem(level LogLevel, component, message string) {
	l.systemLogger.Printf("[%s] [%s] %s", level, component, message)
}

// LogSystemf logs a formatted message to system output
func (l *Logger) LogSystemf(level LogLevel, component, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.LogSystem(level, component, message)
}

// LogDatabase sends a log entry to be written to the database asynchronously
func (l *Logger) LogDatabase(entry *LogEntry) {
	select {
	case l.logChan <- entry:
		// Successfully queued
	case <-l.shutdown:
		// Logger is shutting down, log to system instead
		l.LogSystem(LevelWarn, "logger", "Cannot log to database during shutdown, logging to system")
	default:
		// Channel full, log to system as fallback
		l.LogSystem(LevelWarn, "logger", "Database log channel full, dropping log entry")
	}
}

// LogDatabaseWithDetails is a convenience method for logging with details
func (l *Logger) LogDatabaseWithDetails(eventType, dbName, moduleName, message string, details map[string]interface{}) {
	l.LogDatabase(&LogEntry{
		EventType:    eventType,
		DatabaseName: dbName,
		ModuleName:   moduleName,
		Message:      message,
		Details:      details,
	})
}

// Log is a convenience method that logs to both system and database
func (l *Logger) Log(level LogLevel, component string, entry *LogEntry) {
	// Log to system
	detailsStr := ""
	if entry.Details != nil && len(entry.Details) > 0 {
		if jsonBytes, err := json.Marshal(entry.Details); err == nil {
			detailsStr = fmt.Sprintf(" | details: %s", string(jsonBytes))
		}
	}

	systemMsg := fmt.Sprintf("[%s] [%s] %s%s",
		entry.EventType,
		component,
		entry.Message,
		detailsStr)

	l.LogSystem(level, component, systemMsg)

	// Log to database
	l.LogDatabase(entry)
}

// databaseLogWriter runs in a goroutine and writes log entries to the database
func (l *Logger) databaseLogWriter(ctx context.Context) {
	defer l.wg.Done()

	for {
		select {
		case <-l.shutdown:
			// Flush remaining logs before exiting
			l.flushLogs(ctx)
			return
		case <-ctx.Done():
			l.flushLogs(ctx)
			return
		case entry := <-l.logChan:
			if err := l.writeLogEntry(ctx, entry); err != nil {
				// If database write fails, log to system as fallback
				l.LogSystem(LevelError, "logger", fmt.Sprintf("Failed to write log to database: %v", err))
			}
		}
	}
}

// flushLogs writes all remaining log entries in the channel
func (l *Logger) flushLogs(ctx context.Context) {
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case entry := <-l.logChan:
			if err := l.writeLogEntry(flushCtx, entry); err != nil {
				l.LogSystem(LevelError, "logger", fmt.Sprintf("Failed to flush log entry: %v", err))
			}
		case <-flushCtx.Done():
			// Timeout while flushing
			remaining := len(l.logChan)
			if remaining > 0 {
				l.LogSystem(LevelWarn, "logger", fmt.Sprintf("Flush timeout: %d log entries remaining", remaining))
			}
			return
		default:
			// Channel empty
			return
		}
	}
}

// writeLogEntry writes a single log entry to the database
func (l *Logger) writeLogEntry(ctx context.Context, entry *LogEntry) error {
	if l.dbPool == nil {
		return fmt.Errorf("database pool is nil")
	}

	var detailsJSON []byte
	var err error
	if entry.Details != nil {
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO pgb.pgb_log (
			service_name,
			event_type,
			database_name,
			module_name,
			message,
			details
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = l.dbPool.Exec(ctx, query,
		l.serviceName,
		entry.EventType,
		nullStringIfEmpty(entry.DatabaseName),
		nullStringIfEmpty(entry.ModuleName),
		nullStringIfEmpty(entry.Message),
		detailsJSON,
	)

	return err
}

// nullStringIfEmpty returns nil if the string is empty, otherwise returns the string
func nullStringIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Helper methods for common logging patterns

// LogConfigLoaded logs successful config loading
func (l *Logger) LogConfigLoaded(dbCount, moduleCount int) {
	l.Log(LevelInfo, "config", &LogEntry{
		EventType: EventConfigLoaded,
		Message:   fmt.Sprintf("Configuration loaded successfully: %d databases, %d total modules", dbCount, moduleCount),
		Details: map[string]interface{}{
			"database_count": dbCount,
			"module_count":   moduleCount,
		},
	})
}

// LogConfigError logs config loading errors
func (l *Logger) LogConfigError(err error) {
	l.Log(LevelError, "config", &LogEntry{
		EventType: EventConfigError,
		Message:   fmt.Sprintf("Configuration error: %v", err),
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	})
}

// LogDBConnect logs successful database connection
func (l *Logger) LogDBConnect(dbName string) {
	l.Log(LevelInfo, "database", &LogEntry{
		EventType:    EventDBConnectSuccess,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Successfully connected to database: %s", dbName),
	})
}

// LogDBConnectError logs database connection failure
func (l *Logger) LogDBConnectError(dbName string, err error) {
	l.Log(LevelError, "database", &LogEntry{
		EventType:    EventDBConnectFail,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Failed to connect to database %s: %v", dbName, err),
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	})
}

// LogDBDisconnect logs database disconnection
func (l *Logger) LogDBDisconnect(dbName string) {
	l.Log(LevelInfo, "database", &LogEntry{
		EventType:    EventDBDisconnect,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Disconnected from database: %s", dbName),
	})
}

// LogDBReconnect logs database reconnection attempt
func (l *Logger) LogDBReconnect(dbName string, attempt int, delay time.Duration) {
	l.Log(LevelInfo, "database", &LogEntry{
		EventType:    EventDBReconnect,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Reconnection attempt %d for %s (delay: %v)", attempt, dbName, delay),
		Details: map[string]interface{}{
			"attempt": attempt,
			"delay":   delay.String(),
		},
	})
}

// LogModuleInit logs module initialization
func (l *Logger) LogModuleInit(dbName, moduleName string) {
	l.Log(LevelInfo, "module", &LogEntry{
		EventType:    EventModuleInit,
		DatabaseName: dbName,
		ModuleName:   moduleName,
		Message:      fmt.Sprintf("Initializing module %s for database %s", moduleName, dbName),
	})
}

// LogModuleStart logs module start
func (l *Logger) LogModuleStart(dbName, moduleName string) {
	l.Log(LevelInfo, "module", &LogEntry{
		EventType:    EventModuleStart,
		DatabaseName: dbName,
		ModuleName:   moduleName,
		Message:      fmt.Sprintf("Started module %s for database %s", moduleName, dbName),
	})
}

// LogModuleError logs module errors
func (l *Logger) LogModuleError(dbName, moduleName string, operation string, err error) {
	l.Log(LevelError, "module", &LogEntry{
		EventType:    EventModuleError,
		DatabaseName: dbName,
		ModuleName:   moduleName,
		Message:      fmt.Sprintf("Module %s error during %s: %v", moduleName, operation, err),
		Details: map[string]interface{}{
			"operation": operation,
			"error":     err.Error(),
		},
	})
}

// LogMailSent logs successful email sending
func (l *Logger) LogMailSent(dbName string, mailID int, recipient string) {
	l.Log(LevelInfo, "mail", &LogEntry{
		EventType:    EventMailSent,
		DatabaseName: dbName,
		ModuleName:   "pgb_mail",
		Message:      fmt.Sprintf("Email sent successfully: ID=%d, recipient=%s", mailID, recipient),
		Details: map[string]interface{}{
			"mail_id":   mailID,
			"recipient": recipient,
		},
	})
}

// LogMailFailed logs email sending failure
func (l *Logger) LogMailFailed(dbName string, mailID int, err error) {
	l.Log(LevelError, "mail", &LogEntry{
		EventType:    EventMailFailed,
		DatabaseName: dbName,
		ModuleName:   "pgb_mail",
		Message:      fmt.Sprintf("Failed to send email ID=%d: %v", mailID, err),
		Details: map[string]interface{}{
			"mail_id": mailID,
			"error":   err.Error(),
		},
	})
}

// LogNotifyForwarded logs successful notification forwarding
func (l *Logger) LogNotifyForwarded(dbName string, notifyID int, centralID int) {
	l.Log(LevelInfo, "notify", &LogEntry{
		EventType:    EventNotifyForwarded,
		DatabaseName: dbName,
		ModuleName:   "pgb_notify",
		Message:      fmt.Sprintf("Notification forwarded: source_id=%d, central_id=%d", notifyID, centralID),
		Details: map[string]interface{}{
			"source_id":  notifyID,
			"central_id": centralID,
		},
	})
}

// LogNotifyFailed logs notification forwarding failure
func (l *Logger) LogNotifyFailed(dbName string, notifyID int, err error) {
	l.Log(LevelError, "notify", &LogEntry{
		EventType:    EventNotifyFailed,
		DatabaseName: dbName,
		ModuleName:   "pgb_notify",
		Message:      fmt.Sprintf("Failed to forward notification ID=%d: %v", notifyID, err),
		Details: map[string]interface{}{
			"notify_id": notifyID,
			"error":     err.Error(),
		},
	})
}

// LogListenerStarted logs LISTEN connection started
func (l *Logger) LogListenerStarted(dbName, channel string) {
	l.Log(LevelInfo, "listener", &LogEntry{
		EventType:    EventListenerStarted,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Started listening on channel %s", channel),
		Details: map[string]interface{}{
			"channel": channel,
		},
	})
}

// LogListenerError logs LISTEN connection errors
func (l *Logger) LogListenerError(dbName, channel string, err error) {
	l.Log(LevelError, "listener", &LogEntry{
		EventType:    EventListenerError,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Listener error on channel %s: %v", channel, err),
		Details: map[string]interface{}{
			"channel": channel,
			"error":   err.Error(),
		},
	})
}

// LogHealthCheck logs successful health check
func (l *Logger) LogHealthCheck(dbName string) {
	l.Log(LevelDebug, "health", &LogEntry{
		EventType:    EventHealthCheck,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Health check passed for %s", dbName),
	})
}

// LogHealthCheckFail logs failed health check
func (l *Logger) LogHealthCheckFail(dbName string, err error) {
	l.Log(LevelError, "health", &LogEntry{
		EventType:    EventHealthCheckFail,
		DatabaseName: dbName,
		Message:      fmt.Sprintf("Health check failed for %s: %v", dbName, err),
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	})
}
