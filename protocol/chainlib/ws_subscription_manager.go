package chainlib

import (
	"context"

	"github.com/lavanet/lava/v5/protocol/metrics"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// WSSubscriptionManager defines the interface for WebSocket subscription management.
// This interface is implemented by both:
//   - ConsumerWSSubscriptionManager: routes subscriptions through Lava providers
//   - DirectWSSubscriptionManager: connects directly to RPC endpoints (for smart router)
//
// The interface enables the chain listener to work with either implementation,
// supporting both provider-based and direct RPC subscription models.
type WSSubscriptionManager interface {
	// StartSubscription starts a new WebSocket subscription or joins an existing one.
	// If a subscription with the same parameters already exists, the client joins it
	// (subscription deduplication).
	//
	// Returns:
	//   - firstReply: The initial subscription confirmation reply
	//   - repliesChan: Channel for receiving subscription messages (nil if joining existing)
	//   - error: Any error that occurred
	StartSubscription(
		ctx context.Context,
		protocolMessage ProtocolMessage,
		dappID string,
		consumerIp string,
		webSocketConnectionUniqueId string,
		metricsData *metrics.RelayMetrics,
	) (firstReply *pairingtypes.RelayReply, repliesChan <-chan *pairingtypes.RelayReply, err error)

	// Unsubscribe handles an explicit unsubscribe request from a client.
	// The subscription ID is extracted from the protocolMessage.
	Unsubscribe(
		ctx context.Context,
		protocolMessage ProtocolMessage,
		dappID string,
		consumerIp string,
		webSocketConnectionUniqueId string,
		metricsData *metrics.RelayMetrics,
	) error

	// UnsubscribeAll removes all subscriptions for a specific client connection.
	// Called when a WebSocket connection is closed.
	UnsubscribeAll(
		ctx context.Context,
		dappID string,
		consumerIp string,
		webSocketConnectionUniqueId string,
		metricsData *metrics.RelayMetrics,
	) error
}

// Compile-time interface compliance check
var _ WSSubscriptionManager = (*ConsumerWSSubscriptionManager)(nil)
