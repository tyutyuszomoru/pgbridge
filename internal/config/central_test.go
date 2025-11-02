package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCentralConfig_ValidConfig(t *testing.T) {
	content := `postgres://user:pass@localhost:5432/pansoinco_suite?sslmode=disable`
	configFile := createTempCentralConfig(t, content)
	defer os.Remove(configFile)

	config, err := LoadCentralConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.ConnectionString != content {
		t.Errorf("Expected connection string '%s', got '%s'", content, config.ConnectionString)
	}

	if config.Database != "pansoinco_suite" {
		t.Errorf("Expected database name 'pansoinco_suite', got '%s'", config.Database)
	}
}

func TestLoadCentralConfig_WithComments(t *testing.T) {
	content := `# Central database configuration
# This is the pansoinco_suite database
postgres://user:pass@localhost:5432/pansoinco_suite?sslmode=disable
`
	configFile := createTempCentralConfig(t, content)
	defer os.Remove(configFile)

	config, err := LoadCentralConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "postgres://user:pass@localhost:5432/pansoinco_suite?sslmode=disable"
	if config.ConnectionString != expected {
		t.Errorf("Expected connection string '%s', got '%s'", expected, config.ConnectionString)
	}
}

func TestLoadCentralConfig_EmptyFile(t *testing.T) {
	content := ``
	configFile := createTempCentralConfig(t, content)
	defer os.Remove(configFile)

	_, err := LoadCentralConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected 'empty' error, got: %v", err)
	}
}

func TestLoadCentralConfig_OnlyComments(t *testing.T) {
	content := `# Comment 1
# Comment 2
`
	configFile := createTempCentralConfig(t, content)
	defer os.Remove(configFile)

	_, err := LoadCentralConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for file with only comments, got nil")
	}
}

func TestLoadCentralConfig_FileNotFound(t *testing.T) {
	_, err := LoadCentralConfig("/nonexistent/path/central.conf", nil)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read") {
		t.Errorf("Expected 'failed to read' error, got: %v", err)
	}
}

func TestExtractDatabaseName(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected string
	}{
		{
			name:     "standard postgres URL",
			connStr:  "postgres://user:pass@localhost:5432/mydb",
			expected: "mydb",
		},
		{
			name:     "with query parameters",
			connStr:  "postgres://user:pass@localhost:5432/pansoinco_suite?sslmode=disable",
			expected: "pansoinco_suite",
		},
		{
			name:     "with multiple query parameters",
			connStr:  "postgres://user:pass@localhost:5432/testdb?sslmode=require&connect_timeout=10",
			expected: "testdb",
		},
		{
			name:     "invalid format",
			connStr:  "invalid connection string",
			expected: "central_db",
		},
		{
			name:     "missing database",
			connStr:  "postgres://user:pass@localhost:5432/",
			expected: "central_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDatabaseName(tt.connStr)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCentralDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CentralDatabaseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CentralDatabaseConfig{
				ConnectionString: "postgres://localhost/db",
				Database:         "db",
			},
			wantErr: false,
		},
		{
			name: "empty connection string",
			config: CentralDatabaseConfig{
				ConnectionString: "",
				Database:         "db",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// Helper function to create temporary central config files for testing
func createTempCentralConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "central.conf")

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	return configFile
}
