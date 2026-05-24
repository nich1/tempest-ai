// Package logging provides slog setup, context-carried loggers, and helpers
// for attaching correlation IDs (request_id, job_id, user_id, task_id).
//
// All log records share a consistent shape so a single grep on request_id
// returns the entire lifecycle of a request across the API and consumers.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/nich1/tempest-ai/internal/config"
)

// Options for the root logger.
type Options struct {
	Service string // "api" | "consumers" - emitted on every record
	Level   string // debug | info | warn | error
	Format  string // "json" or "text"
}

// New builds a slog.Logger with the configured handler and standard fields.
func New(opts Options) *slog.Logger {
	return NewWith(os.Stdout, opts)
}

// NewWith allows redirecting the writer (handy for tests).
func NewWith(w io.Writer, opts Options) *slog.Logger {
	level := parseLevel(opts.Level)
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var handler slog.Handler
	if strings.EqualFold(opts.Format, "json") {
		handler = slog.NewJSONHandler(w, handlerOpts)
	} else {
		handler = slog.NewTextHandler(w, handlerOpts)
	}

	logger := slog.New(handler)
	if opts.Service != "" {
		logger = logger.With(slog.String("service", opts.Service))
	}
	return logger
}

// FromConfig builds a logger from a generic logging config block.
func FromConfig(service string, c config.Logging) *slog.Logger {
	return New(Options{Service: service, Level: c.Level, Format: c.Format})
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "err":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
