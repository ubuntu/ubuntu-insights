package storage

import (
	"context"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
)

// PostgresUploader implements the Uploader interface using PostgreSQL.
type PostgresUploader struct{}

// Upload uploads the given data to PostgreSQL.
func (p *PostgresUploader) Upload(ctx context.Context, db DBExecutor, data *models.DBFileData) error {
	return UploadToPostgres(ctx, db, data)
}
