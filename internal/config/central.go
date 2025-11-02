package config

import (
	"fmt"
	"os"
	"strings"

	"pgbridge/internal/logger"
)

// CentralDatabaseConfig holds configuration for the central notification database
type CentralDatabaseConfig struct {
	ConnectionString string
	Database         string // Database name for logging purposes
}

// LoadCentralConfig loads the central database configuration from a file
// Expected format: single line with connection string
// Example: postgres://user:pass@host:5432/pansoinco_suite?sslmode=disable
func LoadCentralConfig(filepath string, log *logger.Logger) (*CentralDatabaseConfig, error) {
	if log != nil {
		log.LogSystemf(logger.LevelInfo, "config", "Loading central database configuration from: %s", filepath)
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		configErr := fmt.Errorf("failed to read central config file: %w", err)
		if log != nil {
			log.LogConfigError(configErr)
		}
		return nil, configErr
	}

	connStr := strings.TrimSpace(string(data))
	if connStr == "" {
		configErr := fmt.Errorf("central database connection string is empty")
		if log != nil {
			log.LogConfigError(configErr)
		}
		return nil, configErr
	}

	// Skip comment lines
	lines := strings.Split(connStr, "\n")
	found := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Use the first non-comment line
		connStr = line
		found = true
		break
	}

	if !found {
		configErr := fmt.Errorf("no valid connection string found in config file")
		if log != nil {
			log.LogConfigError(configErr)
		}
		return nil, configErr
	}

	// Extract database name from connection string for logging
	dbName := extractDatabaseName(connStr)

	if log != nil {
		log.LogSystemf(logger.LevelInfo, "config", "Central database configuration loaded: %s", dbName)
	}

	return &CentralDatabaseConfig{
		ConnectionString: connStr,
		Database:         dbName,
	}, nil
}

// extractDatabaseName extracts the database name from a connection string
func extractDatabaseName(connStr string) string {
	// Try to extract database name from postgres:// URL
	// Format: postgres://user:pass@host:port/database?params
	parts := strings.Split(connStr, "/")
	if len(parts) >= 4 {
		dbPart := parts[3]
		// Remove query parameters
		if idx := strings.Index(dbPart, "?"); idx != -1 {
			dbPart = dbPart[:idx]
		}
		if dbPart != "" {
			return dbPart
		}
	}

	return "central_db"
}

// Validate checks if the central database configuration is valid
func (c *CentralDatabaseConfig) Validate() error {
	if c.ConnectionString == "" {
		return fmt.Errorf("central database connection string is empty")
	}
	return nil
}
