package tracing

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/lavanet/lava/v5/utils"
)

// TraceBodyFlag is the only Lava-specific tracing CLI flag.
// Everything else comes from the standard OTel SDK environment variables, which
// are auto-read by the OTel exporter packages and the autoexport contrib package.
//
// Tracing is enabled when:
//   - OTEL_SDK_DISABLED is not "true" (per OTel spec), AND
//   - OTEL_TRACES_EXPORTER is not "none"
//
// When no OTLP endpoint is explicitly configured, the SDK uses the spec
// defaults: http://localhost:4317 (gRPC) or http://localhost:4318 (HTTP).
// Operators who do NOT run a local collector should explicitly disable
// tracing via OTEL_SDK_DISABLED=true or OTEL_TRACES_EXPORTER=none, otherwise
// the exporter will log connection errors against the default endpoint.
//
// Standard OTel env vars honoured by the SDK / autoexport / exporter packages:
//
//	General SDK:
//	  OTEL_SDK_DISABLED                       — disable the SDK entirely
//	  OTEL_SERVICE_NAME                       — service.name resource attribute
//	  OTEL_RESOURCE_ATTRIBUTES                — additional resource attributes
//	  OTEL_TRACES_SAMPLER                     — sampler strategy (default: parentbased_always_on)
//	  OTEL_TRACES_SAMPLER_ARG                 — sampler argument (e.g. ratio)
//
//	Exporter selection (autoexport):
//	  OTEL_TRACES_EXPORTER                    — otlp (default) | none | console
//	  OTEL_EXPORTER_OTLP_PROTOCOL             — grpc | http/protobuf (default per spec)
//
//	OTLP exporter (auto-read by otlptracegrpc / otlptracehttp packages):
//	  OTEL_EXPORTER_OTLP_ENDPOINT             — collector URL (general)
//	  OTEL_EXPORTER_OTLP_TRACES_ENDPOINT      — traces-specific endpoint override
//	  OTEL_EXPORTER_OTLP_INSECURE             — disable TLS
//	  OTEL_EXPORTER_OTLP_HEADERS              — auth headers (k=v,k=v)
//	  OTEL_EXPORTER_OTLP_COMPRESSION          — gzip | none
//	  OTEL_EXPORTER_OTLP_TIMEOUT              — export timeout (ms)
//	  OTEL_EXPORTER_OTLP_CERTIFICATE          — TLS server cert
//	  OTEL_EXPORTER_OTLP_CLIENT_KEY           — mTLS client key
//	  OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE   — mTLS client cert
//	  (each has a TRACES_-prefixed per-signal variant)
const TraceBodyFlag = "otel-trace-body"

// defaultServiceName is used when OTEL_SERVICE_NAME is not set.
const defaultServiceName = "lava-smartrouter"

const shutdownTimeout = 5 * time.Second

// TraceConfig holds Lava-specific tracing configuration.
// Standard OTel knobs are read from environment variables — see TraceBodyFlag doc.
type TraceConfig struct {
	// TraceBody enables recording of request/response bodies as span attributes.
	// Body size truncation is delegated to the SDK via SpanLimits, which
	// reads OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT (default: unlimited).
	TraceBody bool
}

// traceBodyEnabled is a package-level flag checked by span instrumentation sites.
// Set once during New() and read concurrently — safe because it's only written at startup.
var traceBodyEnabled bool

// IsTraceBodyEnabled returns true when --otel-trace-body is active.
// Instrumentation sites call this to decide whether to record request/response bodies.
func IsTraceBodyEnabled() bool {
	return traceBodyEnabled
}

// TraceManager owns the OTel SDK lifecycle.
type TraceManager struct {
	provider trace.TracerProvider
	shutdown func(context.Context) error
}

// New initialises the OTel SDK based on standard environment variables.
// See TraceBodyFlag for the full env var contract.
func New(ctx context.Context, cfg TraceConfig) (*TraceManager, error) {
	if isSDKDisabled() {
		warnIfBodySetButTracingOff(cfg)
		return newNoop(), nil
	}

	// autoexport reads OTEL_TRACES_EXPORTER and OTEL_EXPORTER_OTLP_PROTOCOL,
	// then constructs the right exporter (otlptracegrpc, otlptracehttp,
	// stdouttrace, or noop). Each exporter package auto-reads its own
	// OTEL_EXPORTER_OTLP_* env vars for endpoint, headers, TLS, etc.
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("failed to create OTLP span exporter", err)
	}

	// Operator picked OTEL_TRACES_EXPORTER=none — install noop and warn the
	// body flag won't take effect.
	if autoexport.IsNoneSpanExporter(exporter) {
		warnIfBodySetButTracingOff(cfg)
		return newNoop(), nil
	}

	traceBodyEnabled = cfg.TraceBody

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(defaultServiceName)),
		resource.WithFromEnv(), // OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES
	)
	if err != nil {
		return nil, utils.LavaFormatError("failed to create OTel resource", err)
	}

	sampler := buildSamplerFromEnv()
	propagator := buildPropagatorFromEnv()

	// NewTracerProvider() uses NewSpanLimits() by default, which reads
	// OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT and truncates string attributes
	// at the SetAttributes call site — so body recording size limits are the
	// SDK's responsibility, not ours.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)

	// Surface OTel SDK internal errors (export failures, batch drops, etc.) via
	// the Lava logger. By default the SDK swallows these silently, which makes
	// "spans aren't reaching my collector" a frustrating black box to debug.
	// We deliberately install this AFTER SetTracerProvider so the handler is
	// not invoked during early SDK setup.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		utils.LavaFormatWarning("OTel SDK reported an error", err)
	}))

	utils.LavaFormatInfo("OTel tracing enabled",
		utils.LogAttr("sampler", sampler.Description()),
		utils.LogAttr("propagator", propagatorDescription(propagator)),
		utils.LogAttr("trace_body", cfg.TraceBody),
	)

	return &TraceManager{
		provider: tp,
		shutdown: tp.Shutdown,
	}, nil
}

// newNoop installs a noop TracerProvider as the global and returns a TraceManager
// whose Shutdown is a no-op.
func newNoop() *TraceManager {
	np := noop.NewTracerProvider()
	otel.SetTracerProvider(np)
	return &TraceManager{
		provider: np,
		shutdown: func(context.Context) error { return nil },
	}
}

// Shutdown flushes all pending spans. Blocks up to 5 seconds.
func (tm *TraceManager) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := tm.shutdown(ctx); err != nil {
		utils.LavaFormatWarning("OTel TracerProvider shutdown did not complete cleanly", err)
	}
}

// Tracer returns a named tracer from the managed provider.
func (tm *TraceManager) Tracer(name string) trace.Tracer {
	return tm.provider.Tracer(name)
}

// warnIfBodySetButTracingOff logs a warning when the body flag is on but tracing
// will be a noop, so operators don't wonder why they aren't seeing bodies.
func warnIfBodySetButTracingOff(cfg TraceConfig) {
	if cfg.TraceBody {
		utils.LavaFormatWarning(
			"--"+TraceBodyFlag+" is set but tracing is not enabled "+
				"(OTEL_SDK_DISABLED=true or OTEL_TRACES_EXPORTER=none); ignored",
			nil,
		)
	}
}

// isSDKDisabled returns true when OTEL_SDK_DISABLED is set to a truthy value
// per the OTel spec (case-insensitive "true", nothing else). Per the spec:
//
//	"Any value not explicitly defined here as a true value, including unset and
//	 empty values, MUST be interpreted as false. If any value other than a true
//	 value, case-insensitive string 'false', empty, or unset is used, a warning
//	 SHOULD be logged."
func isSDKDisabled() bool {
	raw, ok := os.LookupEnv("OTEL_SDK_DISABLED")
	if !ok {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "true":
		return true
	case "", "false":
		return false
	default:
		utils.LavaFormatWarning(
			"unrecognized OTEL_SDK_DISABLED value; treating as false (per OTel spec, only 'true' enables disable)",
			nil,
			utils.LogAttr("value", raw),
		)
		return false
	}
}

// buildSamplerFromEnv reads OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG
// per the OTel spec. Default is parentbased_always_on.
//
// Supported values:
//
//	always_on               — sample every span (no parent check)
//	always_off              — never sample
//	traceidratio            — sample at OTEL_TRACES_SAMPLER_ARG ratio
//	parentbased_always_on   — honor parent, fall back to always_on for root spans (default)
//	parentbased_always_off  — honor parent, never sample new root spans
//	parentbased_traceidratio — honor parent, sample new root spans at the configured ratio
//
// Unrecognized values fall back to the default with a warning, per spec.
func buildSamplerFromEnv() sdktrace.Sampler {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER")))
	arg := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))

	parseRatio := func(defaultValue float64) float64 {
		if arg == "" {
			return defaultValue
		}
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil || v < 0 || v > 1 {
			utils.LavaFormatWarning("invalid OTEL_TRACES_SAMPLER_ARG; using default", err,
				utils.LogAttr("value", arg),
				utils.LogAttr("default", defaultValue),
			)
			return defaultValue
		}
		return v
	}

	switch name {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(parseRatio(1.0))
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(parseRatio(1.0)))
	case "", "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		utils.LavaFormatWarning("unknown OTEL_TRACES_SAMPLER value; using parentbased_always_on", nil,
			utils.LogAttr("value", name),
		)
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

// buildPropagatorFromEnv reads OTEL_PROPAGATORS and returns a composite
// TextMapPropagator per the OTel spec. Default is `tracecontext,baggage`.
//
// Supported values:
//
//	tracecontext   — W3C TraceContext (traceparent, tracestate)
//	baggage        — W3C Baggage
//	none           — disable propagation entirely
//
// Unrecognised values warn and are skipped. An empty resulting list (e.g.
// every value was unsupported) falls back to the default.
func buildPropagatorFromEnv() propagation.TextMapPropagator {
	raw := strings.TrimSpace(os.Getenv("OTEL_PROPAGATORS"))
	if raw == "" {
		return propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}
	var props []propagation.TextMapPropagator
	for _, name := range strings.Split(raw, ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		switch name {
		case "tracecontext":
			props = append(props, propagation.TraceContext{})
		case "baggage":
			props = append(props, propagation.Baggage{})
		case "none":
			// Per spec, "none" disables propagation entirely.
			return propagation.NewCompositeTextMapPropagator()
		default:
			utils.LavaFormatWarning("unsupported OTEL_PROPAGATORS value; ignoring", nil,
				utils.LogAttr("value", name),
			)
		}
	}
	if len(props) == 0 {
		return propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}
	return propagation.NewCompositeTextMapPropagator(props...)
}

// propagatorDescription returns a short, log-friendly description of which
// propagators a TextMapPropagator carries. The composite propagator's Fields()
// method returns the carrier keys it injects/extracts, which is exactly the
// signal an operator wants when debugging "is my baggage being propagated?".
func propagatorDescription(p propagation.TextMapPropagator) string {
	fields := p.Fields()
	if len(fields) == 0 {
		return "none"
	}
	return strings.Join(fields, ",")
}
