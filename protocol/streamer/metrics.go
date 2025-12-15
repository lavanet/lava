package streamer

import (
	"sync/atomic"
)

// StreamerMetrics implementation with atomic operations
type StreamerMetricsImpl struct {
	blocksProcessed      atomic.Int64
	eventsEmitted        atomic.Int64
	activeSubscriptions  atomic.Int64
	websocketConnections atomic.Int64
	webhooksDelivered    atomic.Int64
	webhooksFailed       atomic.Int64
	lastBlockProcessed   atomic.Int64
	eventsPerSecond      atomic.Int64
}

// NewStreamerMetrics creates a new metrics instance
func NewStreamerMetrics() *StreamerMetrics {
	return &StreamerMetrics{}
}

// GetStats returns all metrics as a map
func (m *StreamerMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"blocks_processed":      m.BlocksProcessed,
		"events_emitted":        m.EventsEmitted,
		"active_subscriptions":  m.ActiveSubscriptions,
		"websocket_connections": m.WebSocketConnections,
		"webhooks_delivered":    m.WebhooksDelivered,
		"webhooks_failed":       m.WebhooksFailed,
		"last_block_processed":  m.LastBlockProcessed,
		"events_per_second":     m.EventsPerSecond,
	}
}

