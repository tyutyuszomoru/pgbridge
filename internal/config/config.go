package config

import (
	"fmt"
)

type DatabaseConfig struct {
	Name             string
	ConnectionString string
	ActiveModules    []string
}

type Config struct {
	Databases []DatabaseConfig
}

func (dc *DatabaseConfig) Validate() error {
	if dc.Name == "" {
		return fmt.Errorf("missing database name in configuration")
	}

	if dc.ConnectionString == "" {
		return fmt.Errorf("missing connection string in configuration for database %s", dc.Name)
	}

	if len(dc.ActiveModules) == 0 {

		return fmt.Errorf("at least one module must be enabled in configuration for database %s", dc.Name)
	}

	return nil
}

func (c *Config) Validate() error {
	if len(c.Databases) == 0 {
		return fmt.Errorf("no databases configured")
	}

	// Check for duplicate database names
	names := make(map[string]bool)
	for _, db := range c.Databases {
		if names[db.Name] {
			return fmt.Errorf("duplicate database name: '%s'", db.Name)
		}
		names[db.Name] = true

		// Validate each database config
		if err := db.Validate(); err != nil {
			return err
		}
	}

	return nil
}
