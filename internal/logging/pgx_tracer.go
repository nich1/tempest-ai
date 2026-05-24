package logging

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// SlowQueryThreshold is the latency above which a query is logged at WARN
// regardless of LOG_LEVEL.
const SlowQueryThreshold = 50 * time.Millisecond

// PgxTracer implements pgx.QueryTracer so every database query gets a
// structured log line at DEBUG (or WARN if slow).
//
// Args longer than 100 chars are redacted to keep prompts/file blobs out
// of the log stream.
type PgxTracer struct{}

type tracerKey struct{}

type traceData struct {
	start time.Time
	sql   string
	args  []any
}

// TraceQueryStart captures the query start time.
func (PgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, tracerKey{}, traceData{
		start: time.Now(),
		sql:   data.SQL,
		args:  data.Args,
	})
}

// TraceQueryEnd emits the structured log line.
func (PgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	td, ok := ctx.Value(tracerKey{}).(traceData)
	if !ok {
		return
	}
	dur := time.Since(td.start)
	logger := FromContext(ctx)

	level := slog.LevelDebug
	if dur >= SlowQueryThreshold {
		level = slog.LevelWarn
	}
	if data.Err != nil {
		level = slog.LevelError
	}

	logger.LogAttrs(ctx, level, "db.query",
		slog.String("sql", squash(td.sql)),
		slog.Any("args", redactArgs(td.args)),
		slog.Int64("rows", data.CommandTag.RowsAffected()),
		slog.Duration("latency", dur),
		slog.Any("error", data.Err),
	)
}

// squash collapses whitespace so multi-line SQL renders on one line.
func squash(s string) string {
	out := make([]byte, 0, len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' || c == '\t' || c == '\r' {
			c = ' '
		}
		if c == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		out = append(out, c)
	}
	return string(out)
}

func redactArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case string:
			if len(v) > 100 {
				out[i] = "<redacted>"
				continue
			}
			out[i] = v
		case []byte:
			if len(v) > 100 {
				out[i] = "<redacted>"
				continue
			}
			out[i] = string(v)
		default:
			out[i] = a
		}
	}
	return out
}
