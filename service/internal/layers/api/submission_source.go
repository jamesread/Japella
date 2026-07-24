package api

import "context"

type submissionSourceKey struct{}

// WithSubmissionSource tags the context with mcp or ui (set by MCP handler; UI defaults to ui).
func WithSubmissionSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, submissionSourceKey{}, source)
}

// SubmissionSourceFromContext returns mcp or ui (default ui).
func SubmissionSourceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(submissionSourceKey{}).(string); ok && v != "" {
		return v
	}
	return "ui"
}
