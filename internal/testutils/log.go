package testutils

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExpectedRecord struct {
	Level   slog.Level
	Message string
}

func (want ExpectedRecord) Compare(t *testing.T, have slog.Record) {
	t.Helper()

	assert.Equal(t, want.Level, have.Level, "Expected Level did not match real Level")

	if want.Message == "" {
		return
	}
	assert.Contains(t, have.Message, want.Message, "Real Message does not contain Expected")
}

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
