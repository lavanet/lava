package metrics

// UsageEventSink is a non-blocking sink for telemetry events emitted by the
// consumer / smart-router. Two event shapes are currently routed through it:
//
//   - RelayUsageEvent: one per relay, fired from the chainlib transports
//     (rest, jsonRPC, tendermintRPC, grpc, websocket).
//   - OptimizerQoSReportToSend: periodic per-(chain, provider) score
//     snapshots, fired from ConsumerOptimizerQoSClient on its sampling tick.
//
// Both call paths MUST be non-blocking: implementations drop events on full
// buffers and account for them in DroppedCount via Stats. Telemetry latency
// must never become hot-path latency.
//
// Implementations own their buffering, batching, retry, and shutdown
// semantics. The producer side is fire-and-forget.
type UsageEventSink interface {
	Emit(event RelayUsageEvent)
	EmitOptimizerQoS(report OptimizerQoSReportToSend)
	Stats() SinkStats
	Close()
}

// SinkStats is a snapshot of producer-side counters. Sent is monotonic
// across the sink's lifetime and counts records handed to the underlying
// SDK; whether they actually reach the collector is the SDK's concern and
// surfaces through its own diagnostics, not this struct.
type SinkStats struct {
	Sent uint64
}

// NoopUsageSink is the zero-cost default. When usage emission is disabled
// (e.g., no OTel collector configured), the sink fans every Emit call into
// a no-op so the relay path pays nothing beyond a single virtual call.
type NoopUsageSink struct{}

func (NoopUsageSink) Emit(RelayUsageEvent)                      {}
func (NoopUsageSink) EmitOptimizerQoS(OptimizerQoSReportToSend) {}
func (NoopUsageSink) Stats() SinkStats                          { return SinkStats{} }
func (NoopUsageSink) Close()                                    {}
