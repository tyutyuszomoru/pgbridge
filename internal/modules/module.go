package modules

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module defines the interface that all pgbridge modules must implement
type Module interface {
	// Name returns the unique identifier for this module
	Name() string

	// Initialize sets up the module (creates tables, indexes, etc.)
	// Called once when the module is first loaded
	Initialize(ctx context.Context, pool *pgxpool.Pool) error

	// Start begins the module's operation
	// Called after initialization to start listening/processing
	Start(ctx context.Context) error

	// Stop gracefully stops the module
	Stop() error

	// GetChannelName returns the PostgreSQL NOTIFY channel to listen on
	GetChannelName() string

	// ProcessNotification handles an incoming NOTIFY message
	// The payload is the content sent via NOTIFY
	ProcessNotification(ctx context.Context, payload string) error

	// ProcessQueue processes any pending items that accumulated while offline
	// Called during initialization to handle messages that arrived when service was down
	ProcessQueue(ctx context.Context) error
}

// ModuleConfig holds common configuration for all modules
type ModuleConfig struct {
	DatabaseName string
	Pool         *pgxpool.Pool
}
