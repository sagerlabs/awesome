package trace

import (
	"context"

	"github.com/google/uuid"
)

type traceIDKey struct{}

// NewTraceID 生成新的trace ID
func NewTraceID() string {
	return uuid.NewString()
}

// WithTraceID 将trace ID存入context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDFromContext 从context获取trace ID
func TraceIDFromContext(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey{}).(string)
	return traceID, ok
}

// TraceIDOrNew 从context获取trace ID，如果没有则生成新的
func TraceIDOrNew(ctx context.Context) (context.Context, string) {
	if traceID, ok := TraceIDFromContext(ctx); ok {
		return ctx, traceID
	}
	traceID := NewTraceID()
	return WithTraceID(ctx, traceID), traceID
}
