package database

import (
	"context"
	"os"
	"testing"
	"time"
)

// getTestConnectionString returns a connection string for testing
// Set the TEST_DATABASE_URL environment variable to run integration tests
func getTestConnectionString() (string, bool) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		// Default to a local PostgreSQL instance
		connStr = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	return connStr, true
}

func TestConnectionManager_Connect(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:              "test_db",
		ConnectionString:  connStr,
		MaxConnections:    5,
		MinConnections:    1,
		HealthCheckPeriod: 30 * time.Second,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !cm.IsConnected() {
		t.Error("Expected IsConnected to return true")
	}

	pool := cm.GetPool()
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}
}

func TestConnectionManager_Ping(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cm.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestConnectionManager_Disconnect(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !cm.IsConnected() {
		t.Error("Expected IsConnected to return true before disconnect")
	}

	cm.Disconnect()

	if cm.IsConnected() {
		t.Error("Expected IsConnected to return false after disconnect")
	}
}

func TestConnectionManager_GetStats(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
		MaxConnections:   10,
		MinConnections:   2,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	// Before connecting, stats should be empty
	stats := cm.GetStats()
	if stats.MaxConns != 0 {
		t.Errorf("Expected MaxConns to be 0 before connect, got %d", stats.MaxConns)
	}

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// After connecting, we should see stats
	stats = cm.GetStats()
	if stats.MaxConns != config.MaxConnections {
		t.Errorf("Expected MaxConns to be %d, got %d", config.MaxConnections, stats.MaxConns)
	}

	if stats.IdleConns < 1 {
		t.Errorf("Expected at least 1 idle connection, got %d", stats.IdleConns)
	}
}

func TestConnectionManager_InvalidConnectionString(t *testing.T) {
	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: "invalid://connection/string",
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err == nil {
		t.Error("Expected error with invalid connection string, got nil")
	}

	if cm.IsConnected() {
		t.Error("Expected IsConnected to return false after failed connect")
	}
}

func TestConnectionManager_DoubleConnect(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("First connect failed: %v", err)
	}

	// Second connect should be a no-op and not error
	err = cm.Connect()
	if err != nil {
		t.Errorf("Second connect failed: %v", err)
	}

	if !cm.IsConnected() {
		t.Error("Expected IsConnected to return true after double connect")
	}
}

func TestConnectionManager_HealthCheck(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:              "test_db",
		ConnectionString:  connStr,
		HealthCheckPeriod: 100 * time.Millisecond, // Fast health checks for testing
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start health checks
	cm.StartHealthCheck()

	// Wait a bit for health checks to run
	time.Sleep(300 * time.Millisecond)

	// Connection should still be healthy
	if !cm.IsConnected() {
		t.Error("Expected connection to remain healthy")
	}
}

func TestConnectionManager_Shutdown(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	cm.StartHealthCheck()

	// Shutdown should clean up everything
	cm.Shutdown()

	if cm.IsConnected() {
		t.Error("Expected IsConnected to return false after shutdown")
	}

	// Additional shutdowns should not panic
	cm.Shutdown()
}

func TestConnectionManager_ExecuteQuery(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	pool := cm.GetPool()
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}

	// Execute a simple query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	connStr, ok := getTestConnectionString()
	if !ok {
		t.Skip("Skipping integration test: no database connection string provided")
	}

	config := ConnectionConfig{
		Name:             "test_db",
		ConnectionString: connStr,
		MaxConnections:   5,
	}

	cm := NewConnectionManager(config, nil)
	defer cm.Shutdown()

	err := cm.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Run multiple queries concurrently
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var result int
			err := cm.GetPool().QueryRow(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent query failed: %v", err)
	}
}
