package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/config"
	"pgbridge/internal/database"
	"pgbridge/internal/logger"
	"pgbridge/internal/modules/mail"
	"pgbridge/internal/modules/notify"
	"pgbridge/internal/modules/roles"
)

const (
	serviceName = "pgbridge"
	version     = "1.0.0"
)

// DatabaseManager manages a single database connection and its modules
type DatabaseManager struct {
	name       string
	connMgr    *database.ConnectionManager
	modules    []Module
	listeners  map[string]*Listener
	logger     *logger.Logger
	shutdown   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// Module interface for pgbridge modules
type Module interface {
	Name() string
	Initialize(ctx context.Context, pool *pgxpool.Pool) error
	Start(ctx context.Context) error
	Stop() error
	GetChannelName() string
	ProcessNotification(ctx context.Context, payload string) error
	ProcessQueue(ctx context.Context) error
}

// Listener manages LISTEN connections for a channel
type Listener struct {
	dbName      string
	channel     string
	module      Module
	conn        *pgx.Conn
	logger      *logger.Logger
	shutdown    chan struct{}
	wg          sync.WaitGroup
}

func main() {
	// Print banner
	fmt.Printf("╔═══════════════════════════════════════╗\n")
	fmt.Printf("║   pgbridge - PostgreSQL Bridge        ║\n")
	fmt.Printf("║   Version: %-28s║\n", version)
	fmt.Printf("╚═══════════════════════════════════════╝\n\n")

	// Create system logger (without database logging initially)
	systemLogger := logger.NewLogger(serviceName, nil)

	// Determine configuration source
	var cfg *config.Config
	var err error

	// Check for --db-config flag (database-based configuration)
	useDBConfig := false
	centralConfigPath := "/etc/pgbridge/central.conf"

	if len(os.Args) >= 2 && os.Args[1] == "--db-config" {
		useDBConfig = true
		if len(os.Args) >= 3 {
			centralConfigPath = os.Args[2]
		}
	}

	systemLogger.LogSystemf(logger.LevelInfo, "main", "Starting %s version %s", serviceName, version)

	if useDBConfig {
		// Load configuration from central database
		systemLogger.LogSystemf(logger.LevelInfo, "main", "Using database-based configuration")

		// Load central database connection string
		centralConfig, err := config.LoadCentralConfig(centralConfigPath, systemLogger)
		if err != nil {
			systemLogger.LogConfigError(err)
			fmt.Fprintf(os.Stderr, "Failed to load central config: %v\n", err)
			os.Exit(1)
		}

		cfg, err = config.LoadConfigFromDatabase(centralConfig.ConnectionString, systemLogger)
		if err != nil {
			systemLogger.LogConfigError(err)
			fmt.Fprintf(os.Stderr, "Failed to load configuration from database: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Load configuration from file (legacy mode)
		if len(os.Args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "       %s --db-config [central-config-file]\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "\nExamples:\n")
			fmt.Fprintf(os.Stderr, "  File-based:     %s /etc/pgbridge/pgbridge.conf\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  Database-based: %s --db-config /etc/pgbridge/central.conf\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  Database-based: %s --db-config  (uses default /etc/pgbridge/central.conf)\n", os.Args[0])
			os.Exit(1)
		}

		systemLogger.LogSystemf(logger.LevelInfo, "main", "Using file-based configuration")
		configPath := os.Args[1]

		cfg, err = config.LoadConfig(configPath, systemLogger)
		if err != nil {
			systemLogger.LogConfigError(err)
			fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
			os.Exit(1)
		}
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Create database managers
	var dbManagers []*DatabaseManager
	var mainLogger *logger.Logger

	for i, dbConfig := range cfg.Databases {
		systemLogger.LogSystemf(logger.LevelInfo, "main", "Setting up database: %s (%d/%d)", dbConfig.Name, i+1, len(cfg.Databases))

		// Create connection manager
		connConfig := database.ConnectionConfig{
			Name:             dbConfig.Name,
			ConnectionString: dbConfig.ConnectionString,
		}
		connMgr := database.NewConnectionManager(connConfig, systemLogger)

		// Connect to database
		if err := connMgr.Connect(); err != nil {
			systemLogger.LogDBConnectError(dbConfig.Name, err)
			fmt.Fprintf(os.Stderr, "Failed to connect to database %s: %v\n", dbConfig.Name, err)
			cleanup(dbManagers, systemLogger)
			os.Exit(1)
		}

		// Initialize schema (creates pgb schema and pgb_log table)
		schemaInit := database.NewSchemaInitializer(connMgr.GetPool(), dbConfig.Name, systemLogger)
		if err := schemaInit.Initialize(ctx); err != nil {
			systemLogger.LogSystemf(logger.LevelError, "main", "Failed to initialize schema for %s: %v", dbConfig.Name, err)
			cleanup(dbManagers, systemLogger)
			os.Exit(1)
		}

		// For the first database, create the main logger with database logging
		if i == 0 {
			mainLogger = logger.NewLogger(serviceName, connMgr.GetPool())
			mainLogger.Start(ctx)
			systemLogger.LogSystemf(logger.LevelInfo, "main", "Database logging initialized on: %s", dbConfig.Name)
		}

		// Create database manager
		dbMgr := &DatabaseManager{
			name:      dbConfig.Name,
			connMgr:   connMgr,
			modules:   make([]Module, 0),
			listeners: make(map[string]*Listener, 0),
			logger:    mainLogger,
			shutdown:  make(chan struct{}),
		}

		// Initialize modules for this database
		for _, moduleName := range dbConfig.ActiveModules {
			var module Module

			switch moduleName {
			case "pgb_mail":
				module = mail.NewMailModule(connMgr.GetPool(), dbConfig.Name, mainLogger)
			case "pgb_notify":
				// For pgb_notify, we need to load the central database config
				centralConfigPath := "/etc/pgbridge/central.conf"
				if envPath := os.Getenv("PGBRIDGE_CENTRAL_CONFIG"); envPath != "" {
					centralConfigPath = envPath
				}

				centralConfig, err := config.LoadCentralConfig(centralConfigPath, mainLogger)
				if err != nil {
					mainLogger.LogSystemf(logger.LevelError, "main", "Failed to load central config for pgb_notify: %v", err)
					fmt.Fprintf(os.Stderr, "Failed to load central config: %v\n", err)
					cleanup(dbManagers, mainLogger)
					os.Exit(1)
				}

				// Create connection to central database
				centralPool, err := pgxpool.New(ctx, centralConfig.ConnectionString)
				if err != nil {
					mainLogger.LogSystemf(logger.LevelError, "main", "Failed to connect to central database: %v", err)
					cleanup(dbManagers, mainLogger)
					os.Exit(1)
				}

				module = notify.NewNotifyModule(connMgr.GetPool(), centralPool, dbConfig.Name, mainLogger)
			case "pgb_instance_roles":
				// pgb_instance_roles module operates on central database only
				// It listens for new instance notifications and discovers their roles
				module = roles.NewRolesModule(connMgr.GetPool(), mainLogger)
			default:
				mainLogger.LogSystemf(logger.LevelWarn, "main", "Unknown module: %s", moduleName)
				continue
			}

			// Initialize module
			if err := module.Initialize(ctx, connMgr.GetPool()); err != nil {
				mainLogger.LogModuleError(dbConfig.Name, moduleName, "initialize", err)
				fmt.Fprintf(os.Stderr, "Failed to initialize module %s for %s: %v\n", moduleName, dbConfig.Name, err)
				cleanup(dbManagers, mainLogger)
				os.Exit(1)
			}

			// Start module
			if err := module.Start(ctx); err != nil {
				mainLogger.LogModuleError(dbConfig.Name, moduleName, "start", err)
				fmt.Fprintf(os.Stderr, "Failed to start module %s for %s: %v\n", moduleName, dbConfig.Name, err)
				cleanup(dbManagers, mainLogger)
				os.Exit(1)
			}

			dbMgr.modules = append(dbMgr.modules, module)
			mainLogger.LogModuleStart(dbConfig.Name, moduleName)

			// Process queued items for this module
			if err := module.ProcessQueue(ctx); err != nil {
				mainLogger.LogSystemf(logger.LevelWarn, "main", "Queue processing for %s/%s had errors: %v", dbConfig.Name, moduleName, err)
			}

			// Setup LISTEN for this module
			listener := &Listener{
				dbName:   dbConfig.Name,
				channel:  module.GetChannelName(),
				module:   module,
				logger:   mainLogger,
				shutdown: make(chan struct{}),
			}

			if err := listener.Start(ctx, dbConfig.ConnectionString); err != nil {
				mainLogger.LogListenerError(dbConfig.Name, module.GetChannelName(), err)
				fmt.Fprintf(os.Stderr, "Failed to start listener for %s/%s: %v\n", dbConfig.Name, moduleName, err)
				cleanup(dbManagers, mainLogger)
				os.Exit(1)
			}

			dbMgr.listeners[module.GetChannelName()] = listener
			mainLogger.LogListenerStarted(dbConfig.Name, module.GetChannelName())
		}

		// Start health check
		connMgr.StartHealthCheck()

		dbManagers = append(dbManagers, dbMgr)
		systemLogger.LogSystemf(logger.LevelInfo, "main", "Database %s ready with %d modules", dbConfig.Name, len(dbMgr.modules))
	}

	fmt.Printf("\n✓ pgbridge is running with %d databases\n", len(dbManagers))
	fmt.Printf("✓ Press Ctrl+C to stop\n\n")

	// Log service start
	if mainLogger != nil {
		mainLogger.Log(logger.LevelInfo, "main", &logger.LogEntry{
			EventType: logger.EventServiceStart,
			Message:   fmt.Sprintf("pgbridge %s started successfully with %d databases", version, len(cfg.Databases)),
			Details: map[string]interface{}{
				"version":        version,
				"database_count": len(cfg.Databases),
			},
		})
	}

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n\n⏳ Shutting down gracefully...")

	if mainLogger != nil {
		mainLogger.LogSystemf(logger.LevelInfo, "main", "Received shutdown signal")
	}

	// Cleanup
	cleanup(dbManagers, mainLogger)

	if mainLogger != nil {
		mainLogger.Log(logger.LevelInfo, "main", &logger.LogEntry{
			EventType: logger.EventServiceStop,
			Message:   "pgbridge stopped",
		})
		mainLogger.Shutdown()
	}

	fmt.Println("✓ Shutdown complete")
}

// cleanup gracefully shuts down all database managers
func cleanup(managers []*DatabaseManager, log *logger.Logger) {
	for _, mgr := range managers {
		if log != nil {
			log.LogSystemf(logger.LevelInfo, "main", "Shutting down database: %s", mgr.name)
		}

		// Stop all listeners
		for channel, listener := range mgr.listeners {
			if log != nil {
				log.LogSystemf(logger.LevelInfo, "main", "Stopping listener %s/%s", mgr.name, channel)
			}
			listener.Stop()
		}

		// Stop all modules
		for _, module := range mgr.modules {
			if log != nil {
				log.LogSystemf(logger.LevelInfo, "main", "Stopping module %s/%s", mgr.name, module.Name())
			}
			if err := module.Stop(); err != nil {
				if log != nil {
					log.LogModuleError(mgr.name, module.Name(), "stop", err)
				}
			}
		}

		// Shutdown connection manager
		mgr.connMgr.Shutdown()
	}
}

// Start begins the LISTEN loop for a channel
func (l *Listener) Start(ctx context.Context, connString string) error {
	// Create a dedicated connection for LISTEN
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("failed to create LISTEN connection: %w", err)
	}

	l.conn = conn

	// Start LISTEN
	_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", l.channel))
	if err != nil {
		conn.Close(ctx)
		return fmt.Errorf("failed to LISTEN on channel %s: %w", l.channel, err)
	}

	// Start listener goroutine
	l.wg.Add(1)
	go l.listen(ctx)

	return nil
}

// listen waits for notifications on the channel
func (l *Listener) listen(ctx context.Context) {
	defer l.wg.Done()

	for {
		select {
		case <-l.shutdown:
			return
		case <-ctx.Done():
			return
		default:
			// Wait for notification with timeout
			waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			notification, err := l.conn.WaitForNotification(waitCtx)
			cancel()

			if err != nil {
				// Check if it's just a timeout (handles both direct and wrapped context errors)
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}

				// Check if context was cancelled
				if errors.Is(err, context.Canceled) || ctx.Err() != nil {
					return
				}

				// Real error
				if l.logger != nil {
					l.logger.LogListenerError(l.dbName, l.channel, err)
				}
				continue
			}

			// Process notification
			if notification != nil {
				if l.logger != nil {
					l.logger.LogSystemf(logger.LevelDebug, "listener", "Received notification on %s/%s: %s", l.dbName, l.channel, notification.Payload)
				}

				// Process in a separate goroutine to not block listening
				go func(payload string) {
					processCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					if err := l.module.ProcessNotification(processCtx, payload); err != nil {
						if l.logger != nil {
							l.logger.LogSystemf(logger.LevelError, "listener", "Failed to process notification on %s/%s: %v", l.dbName, l.channel, err)
						}
					}
				}(notification.Payload)
			}
		}
	}
}

// Stop stops the listener
func (l *Listener) Stop() {
	close(l.shutdown)
	if l.conn != nil {
		l.conn.Close(context.Background())
	}
	l.wg.Wait()
}
