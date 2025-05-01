package ingest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		cm       ingest.DConfigManager
		dbConfig database.Config
		options  []ingest.Options

		wantErr bool
	}{
		"Successful creation": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			options:  nil,
			wantErr:  false,
		},
		"Config load failure": {
			cm:       &mockConfigManager{loadErr: errors.New("load error")},
			dbConfig: database.Config{},
			options:  nil,
			wantErr:  true,
		},
		"Database connection failure": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			options: []ingest.Options{
				ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
					return nil, errors.New("db connect error")
				}),
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := ingest.New(tc.cm, tc.dbConfig, tc.options...)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, s, "Returned service should not be nil")
		})
	}
}

type mockConfigManager struct {
	loadErr  error
	watchErr error
}

func (m *mockConfigManager) Load() error {
	return m.loadErr
}

func (m *mockConfigManager) Watch(ctx context.Context) (<-chan struct{}, <-chan error, error) {
	if m.watchErr != nil {
		return nil, nil, m.watchErr
	}
	reloadCh := make(chan struct{})
	errCh := make(chan error)
	return reloadCh, errCh, nil
}

func (m *mockConfigManager) AllowList() []string {
	return []string{"app1", "app2"}
}

func (m *mockConfigManager) BaseDir() string {
	return "/mock/base/dir"
}

type mockDBManager struct {
	closeErr error
}

func (m *mockDBManager) Upload(ctx context.Context, app string, data *models.FileData) error {
	return nil
}

func (m *mockDBManager) Close() error {
	return m.closeErr
}
