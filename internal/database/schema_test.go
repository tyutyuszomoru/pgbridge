package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSchemaInitializer_Initialize(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_schema_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Initialize schema
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Verify schema exists
	pool := cm.GetPool()
	var schemaExists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'pgb')").
		Scan(&schemaExists)
	if err != nil {
		t.Fatalf("Failed to check schema existence: %v", err)
	}

	if !schemaExists {
		t.Error("Expected pgb schema to exist")
	}

	// Verify pgb_log table exists
	var tableExists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'pgb' AND table_name = 'pgb_log')").
		Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	if !tableExists {
		t.Error("Expected pgb_log table to exist")
	}
}

func TestSchemaInitializer_LogTableStructure(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_structure_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Verify table structure by checking key columns
	pool := cm.GetPool()
	expectedColumns := []string{"id", "timestamp", "service_name", "event_type", "database_name", "module_name", "message", "details"}

	for _, colName := range expectedColumns {
		var exists bool
		err = pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = 'pgb' AND table_name = 'pgb_log' AND column_name = $1)",
			colName).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", colName, err)
		}

		if !exists {
			t.Errorf("Expected column %s to exist in pgb_log table", colName)
		}
	}
}

func TestSchemaInitializer_Indexes(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_indexes_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Verify indexes exist
	pool := cm.GetPool()
	expectedIndexes := []string{"idx_pgb_log_timestamp", "idx_pgb_log_event_type"}

	for _, indexName := range expectedIndexes {
		var exists bool
		err = pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE schemaname = 'pgb' AND tablename = 'pgb_log' AND indexname = $1)",
			indexName).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check index %s: %v", indexName, err)
		}

		if !exists {
			t.Errorf("Expected index %s to exist", indexName)
		}
	}
}

func TestSchemaInitializer_StartupLog(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_startup_log_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Verify startup log was created
	pool := cm.GetPool()
	var logCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM pgb.pgb_log WHERE event_type = 'SERVICE_START' AND database_name = $1",
		config.Name).Scan(&logCount)
	if err != nil {
		t.Fatalf("Failed to query log entries: %v", err)
	}

	if logCount < 1 {
		t.Error("Expected at least one SERVICE_START log entry")
	}
}

func TestSchemaInitializer_Idempotent(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_idempotent_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize once
	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}

	// Initialize again - should not error
	err = InitializeWithConnectionManager(ctx, cm)
	if err != nil {
		t.Errorf("Second initialization failed: %v", err)
	}

	// Verify we didn't create duplicate structures
	pool := cm.GetPool()
	var schemaCount int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = 'pgb'").
		Scan(&schemaCount)
	if err != nil {
		t.Fatalf("Failed to count schemas: %v", err)
	}

	if schemaCount != 1 {
		t.Errorf("Expected exactly 1 pgb schema, got %d", schemaCount)
	}
}

func TestNewSchemaInitializer(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	config := ConnectionConfig{
		Name:             "test_new_initializer",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	pool := cm.GetPool()
	initializer := NewSchemaInitializer(pool, "test_db", nil)

	if initializer == nil {
		t.Error("Expected non-nil initializer")
	}

	if initializer.databaseName != "test_db" {
		t.Errorf("Expected database name 'test_db', got '%s'", initializer.databaseName)
	}

	if initializer.pool == nil {
		t.Error("Expected non-nil pool in initializer")
	}
}
