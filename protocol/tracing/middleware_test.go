package tracing

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestFiberMiddleware_CreatesRootSpan(t *testing.T) {
	sr := setupTestTracer(t)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	app := fiber.New()
	app.Use(FiberMiddleware())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, SpanChainlibHTTP, spans[0].Name())
}

func TestFiberMiddleware_ExtractsExternalTraceParent(t *testing.T) {
	sr := setupTestTracer(t)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	app := fiber.New()
	app.Use(FiberMiddleware())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	_, err := app.Test(req)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "0af7651916cd43dd8448eb211c80319c", spans[0].SpanContext().TraceID().String())
}

// TestFiberMiddleware_RoundtripPropagatesTraceID verifies the full propagation
// path: FiberMiddleware creates the root span, a handler picks up the trace
// context from fiberCtx.UserContext(), and InjectGRPC carries the same trace
// ID into outgoing headers — the same plumbing consumer→provider relies on.
func TestFiberMiddleware_RoundtripPropagatesTraceID(t *testing.T) {
	sr := setupTestTracer(t)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	var downstreamTraceParent string
	app := fiber.New()
	app.Use(FiberMiddleware())
	app.Post("/test", func(c *fiber.Ctx) error {
		// Simulate a downstream call: handler grabs the OTel context from
		// fiberCtx.UserContext() and injects it into outgoing metadata, the
		// same way the chainproxy SendNodeMsg path does.
		headers := map[string]string{}
		InjectGRPC(c.UserContext(), headers)
		downstreamTraceParent = headers["traceparent"]
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	_, err := app.Test(req)
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	rootTraceID := spans[0].SpanContext().TraceID().String()

	require.NotEmpty(t, downstreamTraceParent, "InjectGRPC produced no traceparent — context did not propagate from middleware to handler")
	require.Contains(t, downstreamTraceParent, rootTraceID, "injected traceparent does not carry the SERVER span's trace ID")
}
