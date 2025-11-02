package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

// ConnectionManager manages a PostgreSQL connection pool and handles
// connection health monitoring and automatic reconnection
type ConnectionManager struct {
	config       ConnectionConfig
	pool         *pgxpool.Pool
	isConnected  bool
	isShutdown   bool
	mu           sync.RWMutex
	shutdownOnce sync.Once
	shutdown     chan struct{}
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *logger.Logger
}

// ConnectionConfig holds the configuration for a database connection
type ConnectionConfig struct {
	Name             string
	ConnectionString string
	MaxConnections   int32
	MinConnections   int32
	HealthCheckPeriod time.Duration
}

// ConnectionStats provides information about the connection pool
type ConnectionStats struct {
	AcquireCount         int64
	AcquiredConns        int32
	CanceledAcquireCount int64
	ConstructingConns    int32
	EmptyAcquireCount    int64
	IdleConns            int32
	MaxConns             int32
	TotalConns           int32
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config ConnectionConfig, log *logger.Logger) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults if not provided
	if config.MaxConnections == 0 {
		config.MaxConnections = 10
	}
	if config.MinConnections == 0 {
		config.MinConnections = 2
	}
	if config.HealthCheckPeriod == 0 {
		config.HealthCheckPeriod = 30 * time.Second
	}

	return &ConnectionManager{
		config:      config,
		isConnected: false,
		shutdown:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		logger:      log,
	}
}

// Connect establishes a connection to the database
func (cm *ConnectionManager) Connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isConnected {
		return nil
	}

	if cm.logger != nil {
		cm.logger.LogSystemf(logger.LevelInfo, "database", "Connecting to database: %s", cm.config.Name)
	}

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(cm.config.ConnectionString)
	if err != nil {
		connErr := fmt.Errorf("failed to parse connection string: %w", err)
		if cm.logger != nil {
			cm.logger.LogDBConnectError(cm.config.Name, connErr)
		}
		return connErr
	}

	poolConfig.MaxConns = cm.config.MaxConnections
	poolConfig.MinConns = cm.config.MinConnections
	poolConfig.HealthCheckPeriod = cm.config.HealthCheckPeriod

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(cm.ctx, poolConfig)
	if err != nil {
		connErr := fmt.Errorf("failed to create connection pool: %w", err)
		if cm.logger != nil {
			cm.logger.LogDBConnectError(cm.config.Name, connErr)
		}
		return connErr
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(cm.ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		connErr := fmt.Errorf("failed to ping database: %w", err)
		if cm.logger != nil {
			cm.logger.LogDBConnectError(cm.config.Name, connErr)
		}
		return connErr
	}

	cm.pool = pool
	cm.isConnected = true

	if cm.logger != nil {
		cm.logger.LogDBConnect(cm.config.Name)
	}

	return nil
}

// Disconnect closes the connection pool
func (cm *ConnectionManager) Disconnect() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.pool != nil {
		if cm.logger != nil {
			cm.logger.LogDBDisconnect(cm.config.Name)
		}
		cm.pool.Close()
		cm.pool = nil
	}
	cm.isConnected = false
}

// IsConnected returns whether the connection is active
func (cm *ConnectionManager) IsConnected() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isConnected
}

// GetPool returns the connection pool (for executing queries)
func (cm *ConnectionManager) GetPool() *pgxpool.Pool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.pool
}

// Ping tests the database connection
func (cm *ConnectionManager) Ping(ctx context.Context) error {
	cm.mu.RLock()
	pool := cm.pool
	cm.mu.RUnlock()

	if pool == nil {
		return fmt.Errorf("connection pool is nil")
	}

	return pool.Ping(ctx)
}

// GetStats returns current connection pool statistics
func (cm *ConnectionManager) GetStats() *ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.pool == nil {
		return &ConnectionStats{}
	}

	stats := cm.pool.Stat()
	return &ConnectionStats{
		AcquireCount:         stats.AcquireCount(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}
}

// StartHealthCheck starts a goroutine that monitors connection health
func (cm *ConnectionManager) StartHealthCheck() {
	cm.wg.Add(1)
	go cm.healthCheckLoop()
}

// healthCheckLoop runs periodic health checks
func (cm *ConnectionManager) healthCheckLoop() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-cm.shutdown:
			return
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(cm.ctx, 5*time.Second)
			err := cm.Ping(ctx)
			cancel()

			if err != nil {
				// Connection lost, attempt reconnection
				if cm.logger != nil {
					cm.logger.LogHealthCheckFail(cm.config.Name, err)
				}
				cm.handleConnectionLoss()
			} else {
				if cm.logger != nil {
					cm.logger.LogHealthCheck(cm.config.Name)
				}
			}
		}
	}
}

// handleConnectionLoss handles reconnection when connection is lost
func (cm *ConnectionManager) handleConnectionLoss() {
	cm.mu.Lock()
	wasConnected := cm.isConnected
	cm.isConnected = false
	cm.mu.Unlock()

	if wasConnected {
		// Connection was lost, trigger reconnection
		go cm.Reconnect()
	}
}

// Reconnect attempts to reconnect with exponential backoff
func (cm *ConnectionManager) Reconnect() error {
	initialDelay := 1 * time.Second
	maxDelay := 60 * time.Second
	currentDelay := initialDelay
	attempt := 0

	for {
		select {
		case <-cm.shutdown:
			return fmt.Errorf("shutdown requested")
		case <-cm.ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
			attempt++

			if cm.logger != nil {
				cm.logger.LogDBReconnect(cm.config.Name, attempt, currentDelay)
			}

			// First close the existing connection
			cm.Disconnect()

			// Attempt to reconnect
			err := cm.Connect()
			if err == nil {
				// Successfully reconnected
				if cm.logger != nil {
					cm.logger.LogSystemf(logger.LevelInfo, "database", "Successfully reconnected to %s after %d attempts", cm.config.Name, attempt)
				}
				return nil
			}

			// Wait before next attempt with exponential backoff
			select {
			case <-cm.shutdown:
				return fmt.Errorf("shutdown requested")
			case <-cm.ctx.Done():
				return fmt.Errorf("context cancelled")
			case <-time.After(currentDelay):
				// Calculate next delay
				currentDelay *= 2
				if currentDelay > maxDelay {
					currentDelay = maxDelay
				}
			}
		}
	}
}

// Shutdown gracefully shuts down the connection manager
func (cm *ConnectionManager) Shutdown() {
	cm.shutdownOnce.Do(func() {
		cm.mu.Lock()
		cm.isShutdown = true
		cm.mu.Unlock()

		close(cm.shutdown)
		cm.cancel()
		cm.wg.Wait()
		cm.Disconnect()
	})
}
