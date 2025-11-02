package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"pgbridge/internal/logger"
)

// LoadConfigFromDatabase loads database configurations from the central pansoinco_suite database
// This replaces file-based configuration for dynamic management
func LoadConfigFromDatabase(centralConnStr string, log *logger.Logger) (*Config, error) {
	if log != nil {
		log.LogSystemf(logger.LevelInfo, "config", "Loading configuration from central database")
	}

	// Connect to central database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, centralConnStr)
	if err != nil {
		configErr := fmt.Errorf("failed to connect to central database: %w", err)
		if log != nil {
			log.LogConfigError(configErr)
		}
		return nil, configErr
	}
	defer pool.Close()

	// Query the configuration view
	query := `
		SELECT
			ps.short_name || ' ' || si.db_name as db_name,
			si.db_connection_string as conn_string,
			sp.pgb_services as services
		FROM sw_pgb sp
		LEFT JOIN sw_instance si ON si.id = sp.sw_instance_id
		LEFT JOIN ps_sw ps ON ps.id = si.sw_id
		WHERE sp.pgb_services IS NOT NULL
		  AND array_length(sp.pgb_services, 1) > 0
		ORDER BY ps.short_name, si.db_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		queryErr := fmt.Errorf("failed to query database configurations: %w", err)
		if log != nil {
			log.LogConfigError(queryErr)
		}
		return nil, queryErr
	}
	defer rows.Close()

	config := &Config{
		Databases: []DatabaseConfig{},
	}

	for rows.Next() {
		var dbName string
		var connString string
		var services []string

		if err := rows.Scan(&dbName, &connString, &services); err != nil {
			scanErr := fmt.Errorf("failed to scan configuration row: %w", err)
			if log != nil {
				log.LogConfigError(scanErr)
			}
			return nil, scanErr
		}

		// Validate that we have required fields
		if dbName == "" {
			continue // Skip entries without a name
		}
		if connString == "" {
			if log != nil {
				log.LogSystemf(logger.LevelWarn, "config", "Skipping database %s: no connection string", dbName)
			}
			continue
		}
		if len(services) == 0 {
			if log != nil {
				log.LogSystemf(logger.LevelWarn, "config", "Skipping database %s: no services enabled", dbName)
			}
			continue
		}

		// Convert services to the format expected by DatabaseConfig
		var activeModules []string
		for _, svc := range services {
			activeModules = append(activeModules, strings.TrimSpace(svc))
		}

		dbConfig := DatabaseConfig{
			Name:             strings.TrimSpace(dbName),
			ConnectionString: strings.TrimSpace(connString),
			ActiveModules:    activeModules,
		}

		config.Databases = append(config.Databases, dbConfig)
	}

	if err := rows.Err(); err != nil {
		rowsErr := fmt.Errorf("error iterating configuration rows: %w", err)
		if log != nil {
			log.LogConfigError(rowsErr)
		}
		return nil, rowsErr
	}

	// Validate the entire configuration
	if err := config.Validate(); err != nil {
		validationErr := fmt.Errorf("configuration validation failed: %w", err)
		if log != nil {
			log.LogConfigError(validationErr)
		}
		return nil, validationErr
	}

	// Count total modules
	totalModules := 0
	for _, db := range config.Databases {
		totalModules += len(db.ActiveModules)
	}

	if log != nil {
		log.LogConfigLoaded(len(config.Databases), totalModules)
		log.LogSystemf(logger.LevelInfo, "config", "Loaded %d database configurations from central database", len(config.Databases))
	}

	return config, nil
}
