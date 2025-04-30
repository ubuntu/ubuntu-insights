package testutils

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHandler tracks calls to logging functions and implements slog.Handler.
type MockHandler struct {
	IgnoreBelow    slog.Level
	EnabledCalls   []slog.Level
	HandleCalls    []slog.Record
	WithAttrsCalls [][]slog.Attr
	WithGroupCalls []string

	mu sync.Mutex
}

// NewMockHandler returns a new MockHandler.
// levels <= ignoreBelow will not call handle.
func NewMockHandler(ignoreBelow slog.Level) MockHandler {
	return MockHandler{
		IgnoreBelow:    ignoreBelow,
		EnabledCalls:   make([]slog.Level, 0),
		HandleCalls:    make([]slog.Record, 0),
		WithAttrsCalls: make([][]slog.Attr, 0),
		WithGroupCalls: make([]string, 0),

		mu: sync.Mutex{},
	}
}

// AssertLevels asserts that the logging levels observed match the expected amount.
func (h *MockHandler) AssertLevels(t *testing.T, levels map[slog.Level]uint) bool {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()

	if levels == nil {
		return assert.Empty(t, h.HandleCalls)
	}

	have := make(map[slog.Level]uint)
	for _, r := range h.HandleCalls {
		have[r.Level]++
	}

	return assert.Equal(t, levels, have)
}

// GetLevels returns the levels of the logged records.
func (h *MockHandler) GetLevels() map[slog.Level]uint {
	h.mu.Lock()
	defer h.mu.Unlock()
	levels := make(map[slog.Level]uint)
	for _, r := range h.HandleCalls {
		levels[r.Level]++
	}
	return levels
}

// OutputLogs outputs the logs collected by the handler in a readable format.
func (h *MockHandler) OutputLogs(t *testing.T) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()

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
	h.mu.Lock()
	defer h.mu.Unlock()
	h.EnabledCalls = append(h.EnabledCalls, level)
	return level > h.IgnoreBelow
}

// Handle implements Handler.Handle.
func (h *MockHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HandleCalls = append(h.HandleCalls, record)
	return nil
}

// WithAttrs implements Handler.WithAttrs.
func (h *MockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.WithAttrsCalls = append(h.WithAttrsCalls, attrs)
	return h
}

// WithGroup implements Handler.WithGroup.
func (h *MockHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.WithGroupCalls = append(h.WithGroupCalls, name)
	return h
}
