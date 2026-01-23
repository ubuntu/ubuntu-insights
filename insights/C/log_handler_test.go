package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestCLogHandler_Handle(t *testing.T) {
	tests := map[string]struct {
		recordLevel slog.Level
		msg         string
		attrs       []slog.Attr
		withAttrs   []slog.Attr
		expectedLvl slog.Level
	}{
		"info message simple": {
			recordLevel: slog.LevelInfo,
			msg:         "hello world",
			expectedLvl: slog.LevelInfo,
		},
		"error message with attrs": {
			recordLevel: slog.LevelError,
			msg:         "failed",
			attrs:       []slog.Attr{slog.String("foo", "bar"), slog.Int("count", 42)},
			expectedLvl: slog.LevelError,
		},
		"debug message with Pre-attrs": {
			recordLevel: slog.LevelDebug,
			msg:         "debugging",
			withAttrs:   []slog.Attr{slog.String("component", "test")},
			attrs:       []slog.Attr{slog.Bool("valid", true)},
			expectedLvl: slog.LevelDebug,
		},
		"nested groups": {
			recordLevel: slog.LevelInfo,
			msg:         "grouped",
			attrs: []slog.Attr{
				slog.Group("g1", slog.String("k1", "v1"), slog.Group("g2", slog.Int("k2", 2))),
			},
			expectedLvl: slog.LevelInfo,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedLevel slog.Level
			var capturedMsg string
			called := false

			callback := func(l slog.Level, m string) {
				capturedLevel = l
				capturedMsg = m
				called = true
			}

			handler := NewCLogHandler(slog.HandlerOptions{Level: slog.LevelDebug}, callback)

			logger := slog.New(handler)
			if len(tt.withAttrs) > 0 {
				logger = logger.With(argsToAny(tt.withAttrs)...)
			}

			r := slog.NewRecord(time.Now(), tt.recordLevel, tt.msg, 0)
			for _, a := range tt.attrs {
				r.AddAttrs(a)
			}

			// We bypass logger.Log to avoid 'time' and 'source' attributes being added by default or runtime overhead
			// Using Handle directly mimics what the logger eventually calls
			h := logger.Handler()
			err := h.Handle(context.Background(), r)
			require.NoError(t, err)

			assert.True(t, called, "Callback should have been called")
			assert.Equal(t, tt.expectedLvl, capturedLevel)

			want := testutils.LoadWithUpdateFromGolden(t, capturedMsg)
			assert.Equal(t, want, capturedMsg)
		})
	}
}

func TestCLogHandler_Groups(t *testing.T) {
	tests := map[string]struct {
		setup   func(*slog.Logger) *slog.Logger
		message string
		args    []any
	}{
		"connected": {
			setup: func(l *slog.Logger) *slog.Logger {
				return l.WithGroup("server").With("id", 123)
			},
			message: "connected",
		},
		"handling": {
			setup: func(l *slog.Logger) *slog.Logger {
				return l.WithGroup("server").With("id", 123).WithGroup("req").With("path", "/api")
			},
			message: "handling",
		},
		"done": {
			setup: func(l *slog.Logger) *slog.Logger {
				return l.WithGroup("server").With("id", 123).WithGroup("req").With("path", "/api")
			},
			message: "done",
			args:    []any{slog.Group("meta", slog.Int("status", 200))},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedMsg string
			callback := func(_ slog.Level, m string) {
				capturedMsg = m
			}
			handler := NewCLogHandler(slog.HandlerOptions{Level: slog.LevelInfo}, callback)

			logger := slog.New(handler)
			if tt.setup != nil {
				logger = tt.setup(logger)
			}

			logger.Info(tt.message, tt.args...)

			want := testutils.LoadWithUpdateFromGolden(t, capturedMsg)
			assert.Equal(t, want, capturedMsg)
		})
	}
}

func argsToAny(attrs []slog.Attr) []any {
	var args []any
	for _, a := range attrs {
		args = append(args, a)
	}
	return args
}
