package tracing

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// FiberMiddleware returns a Fiber middleware that creates the root SERVER span
// for every inbound HTTP request, extracting any external W3C trace context
// from the request headers. Designed to be installed once via app.Use(...)
// near the top of every chainlib listener.
//
// The created span uses tracing.SpanChainlibHTTP as its name and carries
// http.method and http.route attributes. Span status is set to Error when
// the handler returns a non-nil error or a 5xx status.
func FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract trace context from inbound headers.
		carrier := propagation.MapCarrier{}
		c.Request().Header.VisitAll(func(k, v []byte) {
			carrier[strings.ToLower(string(k))] = string(v)
		})
		ctx := otel.GetTextMapPropagator().Extract(c.UserContext(), carrier)

		// http.route is the route TEMPLATE (e.g. "/*", "/users/:id") per OTel
		// semconv — using the raw request path here would explode the cardinality
		// of any metrics-from-traces pipeline (Tempo metrics-generator, etc.)
		// since REST/Tendermint listeners typically register wildcard handlers
		// and the raw path contains addresses, block hashes, etc. The actual
		// request path goes under `url.path` (current OTel HTTP semconv).
		route := "/"
		if r := c.Route(); r != nil {
			route = r.Path
		}
		ctx, span := otel.Tracer(tracerName).Start(ctx, SpanChainlibHTTP,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.route", route),
				attribute.String("url.path", c.Path()),
			),
		)
		defer span.End()

		c.SetUserContext(ctx)

		err := c.Next()
		status := c.Response().StatusCode()
		span.SetAttributes(attribute.Int("http.status_code", status))
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else if status >= 500 {
			span.SetStatus(otelcodes.Error, "server error")
		}
		return err
	}
}
