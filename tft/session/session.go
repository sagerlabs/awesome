// Package session carries a per-conversation identifier through context.
//
// The feedback loop (see tft/agent) keeps "what did I tell you last turn"
// state. That state must be scoped to a single conversation, otherwise
// concurrent users would see each other's previous-turn context. The session
// ID is the scoping key; it is threaded through context the same way trace IDs
// are, so request handlers set it once and downstream code reads it without
// extra plumbing.
package session

import "context"

type sessionIDKey struct{}

// WithID stores the session ID in the context.
func WithID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// IDFromContext returns the session ID, or an empty string when none was set.
// An empty ID means "no conversation scope": callers should treat the request
// as stateless rather than falling back to a shared default.
func IDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey{}).(string)
	return id
}
