package testutils

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockHandler struct {
	EnabledCalls   []slog.Level
	HandleCalls    []slog.Record
	WithAttrsCalls [][]slog.Attr
	WithGroupCalls []string
}

// NewMockHandler returns a new MockHandler.
func NewMockHandler() MockHandler {
	return MockHandler{
		EnabledCalls:   make([]slog.Level, 0),
		HandleCalls:    make([]slog.Record, 0),
		WithAttrsCalls: make([][]slog.Attr, 0),
		WithGroupCalls: make([]string, 0),
	}
}

// AssertLevels asserts that the logging levels observed match the expected amount
func (h MockHandler) AssertLevels(t *testing.T, levels map[slog.Level]uint) {
	t.Helper()

	if levels == nil {
		assert.Empty(t, h.HandleCalls)
		return
	}

	have := make(map[slog.Level]uint)
	for _, r := range h.HandleCalls {
		have[r.Level] += 1
	}

	assert.Equal(t, levels, have)
}

// Enabled implements Handler.Enabled.
func (h *MockHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h.EnabledCalls = append(h.EnabledCalls, level)
	return true
}

// Handle implements Handler.Handle.
func (h *MockHandler) Handle(ctx context.Context, record slog.Record) error {
	h.HandleCalls = append(h.HandleCalls, record)
	return nil
}

// WithAttrs implements Handler.WithAttrs.
func (h *MockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.WithAttrsCalls = append(h.WithAttrsCalls, attrs)
	return h
}

// WithAttrs implements Handler.WithGroup.
func (h *MockHandler) WithGroup(name string) slog.Handler {
	h.WithGroupCalls = append(h.WithGroupCalls, name)
	return h
}
