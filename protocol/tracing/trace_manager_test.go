package tracing

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_Disabled(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "OTEL_SDK_DISABLED=true",
			env: map[string]string{
				"OTEL_SDK_DISABLED":           "true",
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
		},
		{
			name: "OTEL_SDK_DISABLED=TRUE (case-insensitive)",
			env: map[string]string{
				"OTEL_SDK_DISABLED":           "TRUE",
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
		},
		{
			name: "OTEL_TRACES_EXPORTER=none disables tracing",
			env: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_TRACES_EXPORTER":        "none",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			tm, err := New(context.Background(), TraceConfig{TraceBody: true})
			require.NoError(t, err)
			require.NotNil(t, tm)

			// Tracer returns a noop tracer — Start produces non-recording spans.
			tracer := tm.Tracer("test")
			_, span := tracer.Start(context.Background(), "test-span")
			require.False(t, span.IsRecording())
			span.End()

			tm.Shutdown()
		})
	}
}

func TestBuildSamplerFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		samplerEnv     string
		argEnv         string
		expectContains string
		expectParent   bool
	}{
		{
			name:           "default (unset) → parentbased_always_on",
			samplerEnv:     "",
			expectContains: "AlwaysOnSampler",
			expectParent:   true,
		},
		{
			name:           "always_on (raw, no parent wrapping)",
			samplerEnv:     "always_on",
			expectContains: "AlwaysOnSampler",
			expectParent:   false,
		},
		{
			name:           "always_off (raw)",
			samplerEnv:     "always_off",
			expectContains: "AlwaysOffSampler",
			expectParent:   false,
		},
		{
			name:           "traceidratio with arg",
			samplerEnv:     "traceidratio",
			argEnv:         "0.25",
			expectContains: "TraceIDRatioBased",
			expectParent:   false,
		},
		{
			// SDK optimizes TraceIDRatioBased(1.0) → AlwaysOnSampler.
			name:           "traceidratio with no arg defaults to 1.0 (SDK optimizes to AlwaysOn)",
			samplerEnv:     "traceidratio",
			expectContains: "AlwaysOnSampler",
			expectParent:   false,
		},
		{
			name:           "parentbased_always_on (explicit)",
			samplerEnv:     "parentbased_always_on",
			expectContains: "AlwaysOnSampler",
			expectParent:   true,
		},
		{
			name:           "parentbased_always_off",
			samplerEnv:     "parentbased_always_off",
			expectContains: "AlwaysOffSampler",
			expectParent:   true,
		},
		{
			name:           "parentbased_traceidratio with arg",
			samplerEnv:     "parentbased_traceidratio",
			argEnv:         "0.5",
			expectContains: "TraceIDRatioBased",
			expectParent:   true,
		},
		{
			name:           "unknown value falls back to parentbased_always_on",
			samplerEnv:     "totally_made_up",
			expectContains: "AlwaysOnSampler",
			expectParent:   true,
		},
		{
			// Falls back to ratio 1.0, which the SDK optimizes to AlwaysOnSampler.
			name:           "invalid ratio falls back to default",
			samplerEnv:     "traceidratio",
			argEnv:         "not-a-number",
			expectContains: "AlwaysOnSampler",
			expectParent:   false,
		},
		{
			// Falls back to ratio 1.0, which the SDK optimizes to AlwaysOnSampler.
			name:           "ratio above 1.0 falls back to default",
			samplerEnv:     "traceidratio",
			argEnv:         "1.5",
			expectContains: "AlwaysOnSampler",
			expectParent:   false,
		},
		{
			name:           "case insensitive sampler name",
			samplerEnv:     "ALWAYS_ON",
			expectContains: "AlwaysOnSampler",
			expectParent:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			if tc.samplerEnv != "" {
				t.Setenv("OTEL_TRACES_SAMPLER", tc.samplerEnv)
			}
			if tc.argEnv != "" {
				t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.argEnv)
			}

			s := buildSamplerFromEnv()
			require.NotNil(t, s)
			desc := s.Description()
			require.Contains(t, desc, tc.expectContains)
			if tc.expectParent {
				require.Contains(t, desc, "ParentBased", "expected ParentBased wrapping")
			} else {
				require.NotContains(t, desc, "ParentBased", "expected raw sampler (no ParentBased wrapping)")
			}
		})
	}
}

func TestIsSDKDisabled(t *testing.T) {
	// Per OTel spec: only case-insensitive "true" disables. Empty/unset and any
	// other value mean "not disabled". Non-{true,false,empty} values warn but
	// still resolve to false.
	tests := []struct {
		name   string
		value  string
		set    bool
		expect bool
	}{
		{name: "unset", set: false, expect: false},
		{name: "empty string treated as unset", value: "", set: true, expect: false},
		{name: "true", value: "true", set: true, expect: true},
		{name: "TRUE case insensitive", value: "TRUE", set: true, expect: true},
		{name: "True case insensitive", value: "True", set: true, expect: true},
		{name: "whitespace tolerated", value: " true ", set: true, expect: true},
		{name: "false", value: "false", set: true, expect: false},
		{name: "FALSE case insensitive", value: "FALSE", set: true, expect: false},
		// These warn but resolve to false per spec (was previously accepted as true).
		{name: "1 is NOT true per spec", value: "1", set: true, expect: false},
		{name: "0", value: "0", set: true, expect: false},
		{name: "yes", value: "yes", set: true, expect: false},
		{name: "on", value: "on", set: true, expect: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			if tc.set {
				t.Setenv("OTEL_SDK_DISABLED", tc.value)
			}
			require.Equal(t, tc.expect, isSDKDisabled())
		})
	}
}

// clearTracingEnv unsets all OTel env vars this package reads, so each test
// starts from a clean slate. We use t.Setenv first to register the cleanup
// callback (which restores the original value after the test), then Unsetenv
// to actually remove the variable for the test body. This way we exercise the
// "truly unset" path, not the "set to empty" path.
func clearTracingEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"OTEL_SDK_DISABLED",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_SERVICE_NAME",
		"OTEL_TRACES_SAMPLER",
		"OTEL_TRACES_SAMPLER_ARG",
		"OTEL_TRACES_EXPORTER",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_PROPAGATORS",
		"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
	} {
		t.Setenv(k, "") // register cleanup
		os.Unsetenv(k)  // then actually unset for the test body
	}
}

func TestBuildPropagatorFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		expectField []string // expected propagator carrier keys (order-insensitive)
	}{
		{
			name:        "unset → tracecontext + baggage default",
			env:         "",
			expectField: []string{"traceparent", "tracestate", "baggage"},
		},
		{
			name:        "tracecontext only",
			env:         "tracecontext",
			expectField: []string{"traceparent", "tracestate"},
		},
		{
			name:        "baggage only",
			env:         "baggage",
			expectField: []string{"baggage"},
		},
		{
			name:        "both, comma + whitespace tolerated",
			env:         " tracecontext , baggage ",
			expectField: []string{"traceparent", "tracestate", "baggage"},
		},
		{
			name:        "case insensitive",
			env:         "TraceContext,BAGGAGE",
			expectField: []string{"traceparent", "tracestate", "baggage"},
		},
		{
			name:        "none disables all propagators",
			env:         "none",
			expectField: nil,
		},
		{
			name:        "all unsupported → fall back to default",
			env:         "made_up,nonsense",
			expectField: []string{"traceparent", "tracestate", "baggage"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearTracingEnv(t)
			if tc.env != "" {
				t.Setenv("OTEL_PROPAGATORS", tc.env)
			}

			p := buildPropagatorFromEnv()
			// "none" returns a composite with zero propagators; don't assert
			// NotNil here because testify treats a typed empty slice wrapped
			// in an interface as nil. Field-equality is the meaningful check.
			fields := p.Fields()
			require.ElementsMatch(t, tc.expectField, fields)
		})
	}
}
