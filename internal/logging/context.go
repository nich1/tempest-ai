package logging

import (
	"context"
	"log/slog"
)

type ctxKey int

const loggerKey ctxKey = 0

// WithLogger stores the given logger in ctx. FromContext returns it.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the logger attached to ctx, or slog.Default if none.
// Callers should never receive a nil logger from this function.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// WithRequestID attaches request_id to the logger in ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).With(slog.String("request_id", id)))
}

// WithJobID attaches job_id to the logger in ctx.
func WithJobID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).With(slog.String("job_id", id)))
}

// WithUserID attaches user_id to the logger in ctx.
func WithUserID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).With(slog.String("user_id", id)))
}

// WithTaskID attaches task_id (Asynq's identifier) to the logger in ctx.
func WithTaskID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).With(slog.String("task_id", id)))
}

// WithAttrs attaches arbitrary attrs to the logger in ctx.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	args := make([]any, 0, len(attrs))
	for _, a := range attrs {
		args = append(args, a)
	}
	return WithLogger(ctx, FromContext(ctx).With(args...))
}
