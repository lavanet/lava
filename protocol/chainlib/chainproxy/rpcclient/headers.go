package rpcclient

import (
	"context"
	"net/http"
)

type headersCtxKey struct{}

// WithHeaders returns a new context carrying per-request HTTP headers that
// will be merged with the client's default headers on the next CallContext /
// BatchCallContext call. Use this for per-request data (e.g. W3C trace
// context) that must not leak across concurrent calls on the shared client
// — SetHeader mutates connection-wide state and races under concurrency.
//
// Headers set here take precedence over equally-named defaults set via
// (*Client).SetHeader.
func WithHeaders(ctx context.Context, headers http.Header) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	return context.WithValue(ctx, headersCtxKey{}, headers)
}

// headersFromContext returns the per-request headers attached via WithHeaders.
// Returns nil when none are set. Internal to the package.
func headersFromContext(ctx context.Context) http.Header {
	if ctx == nil {
		return nil
	}
	h, _ := ctx.Value(headersCtxKey{}).(http.Header)
	return h
}
