package ingest

import (
	"context"

	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
)

type (
	DBManager      = dbManager
	DConfigManager = dConfigManager
)

// WithDBConnect sets the database connection function for the ingest service.
func WithDBConnect(dbConnect func(ctx context.Context, cfg database.Config) (DBManager, error)) Options {
	return func(o *options) {
		o.dbConnect = dbConnect
	}
}
