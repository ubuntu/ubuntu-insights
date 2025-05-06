package storage_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)

func TestUploadToPostgres_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	mock.ExpectExec("INSERT INTO \"wsl\"").
		WithArgs(data.Generated, data.SchemaVersion).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = storage.UploadToPostgres(ctx, db, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestUploadToPostgres_NotInitialized(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}
	err := storage.UploadToPostgres(ctx, nil, data)
	if err == nil || !strings.Contains(err.Error(), "database not initialized") {
		t.Errorf("expected database not initialized error, got: %v", err)
	}
}

func TestUploadToPostgres_CanceledContext(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data := &models.DBFileData{
		AppID:         "wsl",
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	mock.ExpectExec("INSERT INTO wsl").
		WithArgs(data.Generated, data.SchemaVersion).
		WillReturnError(context.Canceled)

	err = storage.UploadToPostgres(ctx, db, data)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled error, got: %v", err)
	}
}

func TestUploadToPostgres_InvalidInput(t *testing.T) {
	t.Parallel()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	if err := storage.UploadToPostgres(ctx, db, nil); err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Errorf("expected error for nil input, got: %v", err)
	}

	// Empty AppID
	err = storage.UploadToPostgres(ctx, db, &models.DBFileData{})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Errorf("expected error for missing AppID, got: %v", err)
	}
}

func TestUploadToPostgres_SQLInjectionSafety(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	malicious := &models.DBFileData{
		AppID:         "\"wsl\"; DROP TABLE users;", // should not match allowed AppID
		Generated:     time.Now(),
		SchemaVersion: "1.0",
	}

	err = storage.UploadToPostgres(ctx, db, malicious)

	// Ensure no exec attempt was made
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB interaction: %v", err)
	}
}
