package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// LogOutput defines the function signature for the callback.
type LogOutput func(level slog.Level, msg string)

type cLogHandler struct {
	opts        slog.HandlerOptions
	attrs       []slog.Attr
	groupPrefix string
	output      LogOutput
}

// NewCLogHandler creates a new cLogHandler with the given options and output function.
func NewCLogHandler(opts slog.HandlerOptions, output LogOutput) *cLogHandler {
	return &cLogHandler{
		opts:   opts,
		output: output,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *cLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle processes a log record.
func (h *cLogHandler) Handle(ctx context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.WriteString(r.Message)

	// Helper to write formatted attribute
	writeAttr := func(a slog.Attr) {
		fmt.Fprintf(&sb, " %s=%v", a.Key, a.Value.Any())
	}

	// 1. Write pre-formatted handler attributes
	for _, a := range h.attrs {
		writeAttr(a)
	}

	// 2. Write record attributes (flattened with current group prefix)
	r.Attrs(func(a slog.Attr) bool {
		flattened := flattenAttr(a, h.groupPrefix)
		for _, fa := range flattened {
			writeAttr(fa)
		}
		return true
	})

	h.output(r.Level, sb.String())
	return nil
}

// WithAttrs returns a new handler with the given attributes appended.
func (h *cLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a copy of the handler attributes to avoid race conditions/mutation
	newAttrs := make([]slog.Attr, len(h.attrs))
	copy(newAttrs, h.attrs)

	// Flatten and append new attributes using the current group prefix
	for _, a := range attrs {
		newAttrs = append(newAttrs, flattenAttr(a, h.groupPrefix)...)
	}

	h2 := *h
	h2.attrs = newAttrs
	return &h2
}

// WithGroup returns a new handler with the given group name appended to the prefix.
func (h *cLogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.groupPrefix += name + "."
	return &h2
}

// flattenAttr recursively flattens groups and applies prefix to keys.
func flattenAttr(a slog.Attr, prefix string) []slog.Attr {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return nil
		}
		var ret []slog.Attr
		newPrefix := prefix
		if a.Key != "" {
			newPrefix += a.Key + "."
		}
		for _, child := range attrs {
			ret = append(ret, flattenAttr(child, newPrefix)...)
		}
		return ret
	}

	if a.Key == "" {
		return nil
	}

	a.Key = prefix + a.Key
	return []slog.Attr{a}
}
