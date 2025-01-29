package testutils

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHandler tracks calls to logging functions and implements slog.Handler.
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

// AssertLevels asserts that the logging levels observed match the expected amount.
func (h MockHandler) AssertLevels(t *testing.T, levels map[slog.Level]uint) bool {
	t.Helper()

	if levels == nil {
		return assert.Empty(t, h.HandleCalls)
	}

	have := make(map[slog.Level]uint)
	for _, r := range h.HandleCalls {
		have[r.Level]++
	}

	return assert.Equal(t, levels, have)
}

// OutputLogs outputs the logs collected by the handler in a readable format.
func (h MockHandler) OutputLogs(t *testing.T) {
	t.Helper()

	for _, call := range h.HandleCalls {
		t.Logf("Logged %v %s:", call.Level, call.Message)
		call.Attrs(func(attr slog.Attr) bool {
			t.Log(attr.String())
			return true
		})
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

// WithGroup implements Handler.WithGroup.
func (h *MockHandler) WithGroup(name string) slog.Handler {
	h.WithGroupCalls = append(h.WithGroupCalls, name)
	return h
}
