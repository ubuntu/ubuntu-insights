package storage

import (
	"context"
	"database/sql"
)

// DBExecutor is an interface that wraps the ExecContext method.
// This is useful for mocking the database operations in tests.
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
