package metrics

// UsageEventSink is a non-blocking sink for raw relay-usage events. The
// relay-serving path calls Emit once per relay; implementations MUST never
// block the caller — full buffers should drop the event and account for it
// in DroppedCount via Stats. Telemetry latency must never become relay
// latency.
//
// Implementations are responsible for their own buffering, batching, retry,
// and shutdown semantics. The hot path is fire-and-forget.
type UsageEventSink interface {
	Emit(event RelayUsageEvent)
	Stats() SinkStats
	Close()
}

// SinkStats is a snapshot of producer-side counters. Each value is monotonic
// across the sink's lifetime.
type SinkStats struct {
	Sent    uint64
	Failed  uint64
	Dropped uint64
}

// NoopUsageSink is the zero-cost default. When usage emission is disabled
// (e.g., no OTel collector configured), the sink fans every Emit call into
// a no-op so the relay path pays nothing beyond a single virtual call.
type NoopUsageSink struct{}

func (NoopUsageSink) Emit(RelayUsageEvent) {}
func (NoopUsageSink) Stats() SinkStats     { return SinkStats{} }
func (NoopUsageSink) Close()               {}
