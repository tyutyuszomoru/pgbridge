package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	content := `# Test configuration file
db1, postgres://user:pass@localhost:5432/db1, [pgb_mail, pgb_notify]
db2, postgres://user:pass@localhost:5432/db2, [pgb_async]
`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(config.Databases) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(config.Databases))
	}

	// Check first database
	db1 := config.Databases[0]
	if db1.Name != "db1" {
		t.Errorf("Expected name 'db1', got '%s'", db1.Name)
	}
	if db1.ConnectionString != "postgres://user:pass@localhost:5432/db1" {
		t.Errorf("Unexpected connection string: %s", db1.ConnectionString)
	}
	if len(db1.ActiveModules) != 2 {
		t.Errorf("Expected 2 modules for db1, got %d", len(db1.ActiveModules))
	}
	if db1.ActiveModules[0] != "pgb_mail" || db1.ActiveModules[1] != "pgb_notify" {
		t.Errorf("Unexpected modules: %v", db1.ActiveModules)
	}

	// Check second database
	db2 := config.Databases[1]
	if db2.Name != "db2" {
		t.Errorf("Expected name 'db2', got '%s'", db2.Name)
	}
	if len(db2.ActiveModules) != 1 {
		t.Errorf("Expected 1 module for db2, got %d", len(db2.ActiveModules))
	}
	if db2.ActiveModules[0] != "pgb_async" {
		t.Errorf("Expected module 'pgb_async', got '%s'", db2.ActiveModules[0])
	}
}

func TestLoadConfig_EmptyLinesAndComments(t *testing.T) {
	content := `
# This is a comment
# Another comment

db1, postgres://user:pass@localhost:5432/db1, [pgb_mail]

# Comment in the middle

db2, postgres://user:pass@localhost:5432/db2, [pgb_notify]

`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(config.Databases) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(config.Databases))
	}
}

func TestLoadConfig_MultipleModules(t *testing.T) {
	content := `db1, postgres://user:pass@localhost:5432/db1, [pgb_mail, pgb_notify, pgb_async, pgb_calendar]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(config.Databases[0].ActiveModules) != 4 {
		t.Errorf("Expected 4 modules, got %d", len(config.Databases[0].ActiveModules))
	}

	expectedModules := []string{"pgb_mail", "pgb_notify", "pgb_async", "pgb_calendar"}
	for i, module := range expectedModules {
		if config.Databases[0].ActiveModules[i] != module {
			t.Errorf("Expected module '%s' at index %d, got '%s'",
				module, i, config.Databases[0].ActiveModules[i])
		}
	}
}

func TestLoadConfig_WhitespaceHandling(t *testing.T) {
	content := `  db1  ,   postgres://user:pass@localhost:5432/db1   ,   [  pgb_mail  ,  pgb_notify  ]  `
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	db := config.Databases[0]
	if db.Name != "db1" {
		t.Errorf("Expected name 'db1', got '%s'", db.Name)
	}
	if db.ConnectionString != "postgres://user:pass@localhost:5432/db1" {
		t.Errorf("Connection string not trimmed properly: '%s'", db.ConnectionString)
	}
	if len(db.ActiveModules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(db.ActiveModules))
	}
	if db.ActiveModules[0] != "pgb_mail" {
		t.Errorf("Module not trimmed properly: '%s'", db.ActiveModules[0])
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/to/config.conf", nil)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "error on loading config") {
		t.Errorf("Expected 'error on loading config' error, got: %v", err)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	content := ``
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for empty config, got nil")
	}
	if !strings.Contains(err.Error(), "no databases configured") {
		t.Errorf("Expected 'no databases configured' error, got: %v", err)
	}
}

func TestLoadConfig_OnlyComments(t *testing.T) {
	content := `# Comment 1
# Comment 2
# Comment 3
`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for config with only comments, got nil")
	}
	if !strings.Contains(err.Error(), "no databases configured") {
		t.Errorf("Expected 'no databases configured' error, got: %v", err)
	}
}

func TestLoadConfig_MissingModuleList(t *testing.T) {
	content := `db1, postgres://user:pass@localhost:5432/db1`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for missing module list, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Errorf("Expected ParseError, got %T", err)
	}
	if parseErr.Line != 1 {
		t.Errorf("Expected error on line 1, got line %d", parseErr.Line)
	}
	if !strings.Contains(err.Error(), "module list not found") {
		t.Errorf("Expected 'module list not found' error, got: %v", err)
	}
}

func TestLoadConfig_EmptyModuleList(t *testing.T) {
	content := `db1, postgres://user:pass@localhost:5432/db1, []`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for empty module list, got nil")
	}
	if !strings.Contains(err.Error(), "no modules specified") {
		t.Errorf("Expected 'no modules specified' error, got: %v", err)
	}
}

func TestLoadConfig_EmptyDatabaseName(t *testing.T) {
	content := `, postgres://user:pass@localhost:5432/db1, [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for empty database name, got nil")
	}
	if !strings.Contains(err.Error(), "database name cannot be empty") {
		t.Errorf("Expected 'database name cannot be empty' error, got: %v", err)
	}
}

func TestLoadConfig_EmptyConnectionString(t *testing.T) {
	content := `db1, , [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for empty connection string, got nil")
	}
	if !strings.Contains(err.Error(), "connection string cannot be empty") {
		t.Errorf("Expected 'connection string cannot be empty' error, got: %v", err)
	}
}

func TestLoadConfig_DuplicateDatabaseNames(t *testing.T) {
	content := `db1, postgres://user:pass@localhost:5432/db1, [pgb_mail]
db1, postgres://user:pass@localhost:5432/db2, [pgb_notify]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for duplicate database names, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate database name") {
		t.Errorf("Expected 'duplicate database name' error, got: %v", err)
	}
}

func TestLoadConfig_MalformedLine(t *testing.T) {
	content := `db1 postgres://user:pass@localhost:5432/db1 [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for malformed line, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Errorf("Expected ParseError, got %T", err)
	}
	if parseErr.Line != 1 {
		t.Errorf("Expected error on line 1, got line %d", parseErr.Line)
	}
}

func TestLoadConfig_TooManyCommas(t *testing.T) {
	// Note: With the new parser, everything between first comma and [ is the connection string,
	// so "extra," will be part of the connection string. This is actually valid parsing.
	// To test invalid format, we need a line without a comma
	content := `db1 postgres://user:pass@localhost:5432/db1 [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for missing comma separator, got nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected 'invalid format' error, got: %v", err)
	}
}

func TestLoadConfig_SingleModule(t *testing.T) {
	content := `db1, postgres://user:pass@localhost:5432/db1, [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(config.Databases[0].ActiveModules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(config.Databases[0].ActiveModules))
	}
	if config.Databases[0].ActiveModules[0] != "pgb_mail" {
		t.Errorf("Expected 'pgb_mail', got '%s'", config.Databases[0].ActiveModules[0])
	}
}

func TestLoadConfig_ComplexConnectionString(t *testing.T) {
	connStr := "postgres://user:complex_p@ss!word@db.example.com:5433/mydb?sslmode=require&connect_timeout=10"
	content := `db1, ` + connStr + `, [pgb_mail]`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Databases[0].ConnectionString != connStr {
		t.Errorf("Connection string mismatch.\nExpected: %s\nGot: %s",
			connStr, config.Databases[0].ConnectionString)
	}
}

func TestLoadConfig_ParseErrorLineNumber(t *testing.T) {
	content := `# Comment line 1
db1, postgres://user:pass@localhost:5432/db1, [pgb_mail]

# Comment line 4
invalid line without proper format
db2, postgres://user:pass@localhost:5432/db2, [pgb_notify]
`
	configFile := createTempConfigFile(t, content)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, nil)
	if err == nil {
		t.Error("Expected error for invalid line, got nil")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected ParseError, got %T", err)
	}

	// Line 5 is the invalid line (after comments and empty lines)
	if parseErr.Line != 5 {
		t.Errorf("Expected error on line 5, got line %d", parseErr.Line)
	}

	if !strings.Contains(parseErr.Content, "invalid line") {
		t.Errorf("ParseError should contain the line content, got: %s", parseErr.Content)
	}
}

// Helper function to create temporary config files for testing
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.conf")

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	return configFile
}

// Test the Validate methods directly
func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: DatabaseConfig{
				Name:             "db1",
				ConnectionString: "postgres://localhost/db1",
				ActiveModules:    []string{"pgb_mail"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: DatabaseConfig{
				Name:             "",
				ConnectionString: "postgres://localhost/db1",
				ActiveModules:    []string{"pgb_mail"},
			},
			wantErr: true,
			errMsg:  "missing database name",
		},
		{
			name: "missing connection string",
			config: DatabaseConfig{
				Name:             "db1",
				ConnectionString: "",
				ActiveModules:    []string{"pgb_mail"},
			},
			wantErr: true,
			errMsg:  "missing connection string",
		},
		{
			name: "no modules",
			config: DatabaseConfig{
				Name:             "db1",
				ConnectionString: "postgres://localhost/db1",
				ActiveModules:    []string{},
			},
			wantErr: true,
			errMsg:  "at least one module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Databases: []DatabaseConfig{
					{
						Name:             "db1",
						ConnectionString: "postgres://localhost/db1",
						ActiveModules:    []string{"pgb_mail"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no databases",
			config: Config{
				Databases: []DatabaseConfig{},
			},
			wantErr: true,
			errMsg:  "no databases configured",
		},
		{
			name: "duplicate names",
			config: Config{
				Databases: []DatabaseConfig{
					{
						Name:             "db1",
						ConnectionString: "postgres://localhost/db1",
						ActiveModules:    []string{"pgb_mail"},
					},
					{
						Name:             "db1",
						ConnectionString: "postgres://localhost/db2",
						ActiveModules:    []string{"pgb_notify"},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate database name",
		},
		{
			name: "invalid database config",
			config: Config{
				Databases: []DatabaseConfig{
					{
						Name:             "db1",
						ConnectionString: "",
						ActiveModules:    []string{"pgb_mail"},
					},
				},
			},
			wantErr: true,
			errMsg:  "missing connection string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
