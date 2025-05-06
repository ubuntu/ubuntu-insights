package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)

var testCfg = config.DBConfig{
	Host:     "localhost",
	Port:     5432,
	User:     "testuser",
	Password: "testpass",
	DBName:   "testdb",
	SSLMode:  "disable",
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS wsl (
			id SERIAL PRIMARY KEY,
			generated TIMESTAMP NOT NULL,
			schema_version TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	return db
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`TRUNCATE TABLE wsl;`)
	if err != nil {
		t.Errorf("failed to truncate table: %v", err)
	}
	db.Close()
}

func TestInitializeAndGet(t *testing.T) {
	t.Parallel()
	err := storage.Initialize(testCfg)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	db := storage.Get()
	if db == nil {
		t.Fatal("expected initialized DB, got nil")
	}
}

func TestUploadToPostgres_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	setup := setupTestDB(t)
	defer teardownTestDB(t, setup)

	err := storage.Initialize(testCfg)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	err = storage.UploadToPostgres(ctx, data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUploadToPostgres_NotInitialized(t *testing.T) {
	// Bypass Initialize on purpose
	ctx := context.Background()

	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	err := storage.UploadToPostgres(ctx, data)
	if err == nil || !strings.Contains(err.Error(), "database not initialized") {
		t.Errorf("expected database not initialized error, got: %v", err)
	}
}

func TestUploadToPostgres_CanceledContext(t *testing.T) {
	t.Parallel()
	setup := setupTestDB(t)
	defer teardownTestDB(t, setup)

	err := storage.Initialize(testCfg)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	err = storage.UploadToPostgres(ctx, data)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled error, got: %v", err)
	}
}

func TestUploadToPostgres_InvalidInput(t *testing.T) {
	t.Parallel()
	setup := setupTestDB(t)
	defer teardownTestDB(t, setup)

	err := storage.Initialize(testCfg)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	ctx := context.Background()
	err = storage.UploadToPostgres(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Errorf("expected error for nil input, got: %v", err)
	}

	err = storage.UploadToPostgres(ctx, &models.DBFileData{})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Errorf("expected error for missing AppID, got: %v", err)
	}
}

func TestUploadToPostgres_SQLInjectionSafety(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	setup := setupTestDB(t)
	defer teardownTestDB(t, setup)

	data := &models.DBFileData{
		AppID:         "wsl; DROP TABLE users;", // malicious input
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	err := storage.UploadToPostgres(ctx, data)
	if err == nil {
		t.Error("Expected error due to SQL injection attempt, got nil")
	}
}
