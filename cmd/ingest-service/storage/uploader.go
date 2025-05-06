package storage

import (
	"context"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
)

// Uploader is an interface for uploading data to a storage backend.
type Uploader interface {
	Upload(ctx context.Context, data *models.DBFileData) error
}
