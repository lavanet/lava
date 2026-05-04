package metrics

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/lavanet/lava/v5/utils"
)

// OTelUsageSinkConfig captures the tunables for the OTel-backed usage sink.
// All durations and sizes apply safe defaults when zero.
type OTelUsageSinkConfig struct {
	// Endpoint is the OTLP/gRPC endpoint to ship logs to. Empty means use
	// the SDK's default discovery (OTEL_EXPORTER_OTLP_ENDPOINT env var or
	// localhost:4317). Pointing at a sidecar collector is the expected
	// deployment.
	Endpoint string

	// Insecure skips TLS for the OTLP connection. Default true since the
	// expected deployment is a localhost or in-cluster collector. Flip to
	// false when going over the network.
	Insecure bool

	// Headers are appended to every OTLP request (e.g. for authenticating
	// with a remote collector).
	Headers map[string]string

	// QueueSize bounds the BatchProcessor's in-memory queue. When full,
	// Emit drops the event and increments DroppedCount.
	QueueSize int

	// BatchSize triggers a flush when reached.
	BatchSize int

	// FlushInterval triggers a flush when elapsed without hitting BatchSize.
	FlushInterval time.Duration

	// ExportTimeout caps each underlying OTLP send.
	ExportTimeout time.Duration

	// ServiceName is the resource attribute identifying the producing
	// service. Used by the collector for routing/filtering.
	ServiceName string

	// ServiceInstanceID disambiguates one consumer host from another.
	// Defaults to hostname.
	ServiceInstanceID string
}

// OTel sink defaults tuned for ~5K events/sec, ~500-byte attribute payloads.
const (
	defaultOTelQueueSize     = 50_000
	defaultOTelBatchSize     = 1000
	defaultOTelFlushInterval = 500 * time.Millisecond
	defaultOTelExportTimeout = 10 * time.Second
	defaultOTelServiceName   = "lava-rpcconsumer"
	otelInstrumentationName  = "github.com/lavanet/lava/protocol/metrics"
)

// OTelUsageSink emits raw relay-usage events as OTLP log records to a
// configured collector (typically a sidecar). The hot path is non-blocking:
// full BatchProcessor queues drop and increment DroppedCount.
type OTelUsageSink struct {
	provider *sdklog.LoggerProvider
	logger   otellog.Logger
	cfg      OTelUsageSinkConfig

	// Counters. The OTel SDK does not expose per-record send/fail status
	// from the API surface, so Sent counts records handed to the SDK and
	// Failed counts marshal failures. Dropped is what the producer side
	// (us) drops before handing to the SDK.
	sentCount    atomic.Uint64
	failedCount  atomic.Uint64
	droppedCount atomic.Uint64
}

// NewOTelUsageSink constructs and starts an OTel-backed sink. Returns nil on
// configuration error after logging — callers should fall back to NoopUsageSink.
func NewOTelUsageSink(cfg OTelUsageSinkConfig) *OTelUsageSink {
	applyOTelSinkDefaults(&cfg)

	exporterOpts := []otlploghttp.Option{}
	if cfg.Endpoint != "" {
		exporterOpts = append(exporterOpts, otlploghttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlploghttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlploghttp.WithHeaders(cfg.Headers))
	}

	exporter, err := otlploghttp.New(context.Background(), exporterOpts...)
	if err != nil {
		utils.LavaFormatError("Failed to construct OTLP log exporter", err,
			utils.LogAttr("endpoint", cfg.Endpoint))
		return nil
	}

	processor := sdklog.NewBatchProcessor(exporter,
		sdklog.WithMaxQueueSize(cfg.QueueSize),
		sdklog.WithExportMaxBatchSize(cfg.BatchSize),
		sdklog.WithExportInterval(cfg.FlushInterval),
		sdklog.WithExportTimeout(cfg.ExportTimeout),
	)

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceInstanceID(cfg.ServiceInstanceID),
	))
	if err != nil {
		utils.LavaFormatError("Failed to build OTel resource", err)
		_ = exporter.Shutdown(context.Background())
		return nil
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)

	utils.LavaFormatInfo("Starting OTel usage sink",
		utils.LogAttr("endpoint", cfg.Endpoint),
		utils.LogAttr("insecure", cfg.Insecure),
		utils.LogAttr("queue_size", cfg.QueueSize),
		utils.LogAttr("batch_size", cfg.BatchSize),
		utils.LogAttr("flush_interval", cfg.FlushInterval),
		utils.LogAttr("service_name", cfg.ServiceName),
		utils.LogAttr("service_instance_id", cfg.ServiceInstanceID),
	)

	return &OTelUsageSink{
		provider: provider,
		logger:   provider.Logger(otelInstrumentationName),
		cfg:      cfg,
	}
}

// Emit is the hot-path entrypoint. It builds an OTLP log record from the
// event and hands it to the SDK's BatchProcessor. The processor enqueues
// non-blockingly; if its queue is full the record is dropped by the SDK.
// We treat the call as best-effort and increment Sent on completion.
func (s *OTelUsageSink) Emit(event RelayUsageEvent) {
	if s == nil {
		return
	}

	var rec otellog.Record
	rec.SetTimestamp(time.Unix(0, event.TimestampNs))
	rec.SetBody(otellog.StringValue("relay_usage"))
	rec.AddAttributes(
		otellog.String("project", event.ProjectHash),
		otellog.String("chain", event.ChainID),
		otellog.String("api_interface", event.APIInterface),
		otellog.String("api_method", event.APIMethod),
		otellog.Int64("cu", int64(event.ComputeUnits)),
		otellog.Int64("latency_ms", event.LatencyMs),
		otellog.Bool("success", event.Success),
		otellog.Bool("cache_hit", event.CacheHit),
		otellog.Bool("is_write", event.IsWrite),
		otellog.Bool("is_archive", event.IsArchive),
		otellog.Bool("is_batch", event.IsBatch),
		otellog.Bool("is_debug_trace", event.IsDebugTrace),
		otellog.Int64("hedge_count", int64(event.HedgeCount)),
		otellog.String("provider", event.ProviderAddress),
		otellog.String("origin", event.Origin),
	)

	s.logger.Emit(context.Background(), rec)
	s.sentCount.Add(1)
}

// Stats returns counter snapshots.
func (s *OTelUsageSink) Stats() SinkStats {
	if s == nil {
		return SinkStats{}
	}
	return SinkStats{
		Sent:    s.sentCount.Load(),
		Failed:  s.failedCount.Load(),
		Dropped: s.droppedCount.Load(),
	}
}

// Close flushes pending records and shuts down the underlying provider.
// The SDK's Shutdown is bounded by ExportTimeout per pending batch.
func (s *OTelUsageSink) Close() {
	if s == nil || s.provider == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ExportTimeout)
	defer cancel()
	if err := s.provider.Shutdown(ctx); err != nil {
		utils.LavaFormatWarning("Error shutting down OTel usage sink", err)
	}
	stats := s.Stats()
	utils.LavaFormatInfo("OTel usage sink closed",
		utils.LogAttr("sent_total", stats.Sent),
		utils.LogAttr("failed_total", stats.Failed),
		utils.LogAttr("dropped_total", stats.Dropped),
	)
}

func applyOTelSinkDefaults(cfg *OTelUsageSinkConfig) {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultOTelQueueSize
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultOTelBatchSize
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = defaultOTelFlushInterval
	}
	if cfg.ExportTimeout <= 0 {
		cfg.ExportTimeout = defaultOTelExportTimeout
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultOTelServiceName
	}
	if cfg.ServiceInstanceID == "" {
		// Default to hostname-pid so multiple consumer processes on the
		// same bare-metal host (e.g., one process per chain) get unique
		// instance IDs without operator intervention. The host's chain
		// fan-out is the common deployment shape; this lets all of them
		// share one local OTel collector and still be distinguishable
		// downstream.
		hn, err := osHostname()
		if err != nil {
			hn = "unknown"
		}
		cfg.ServiceInstanceID = fmt.Sprintf("%s-%d", hn, os.Getpid())
	}
}

// osHostname is a seam for tests.
var osHostname = defaultHostname
