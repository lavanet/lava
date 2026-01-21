package rpcsmartrouter

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// NoOpWSSubscriptionManager is a WebSocket subscription manager that returns errors
// for all subscription operations. It is used when no WebSocket endpoints are configured
// in the smart router's static provider list.
//
// This exists because the smart router does NOT fall back to provider-based subscriptions.
// Per the implementation plan, rpcsmartrouter uses ONLY direct RPC connections.
// If WebSocket subscriptions are needed, ws:// or wss:// URLs must be configured.
type NoOpWSSubscriptionManager struct {
	chainID      string
	apiInterface string
}

// NewNoOpWSSubscriptionManager creates a new NoOp subscription manager that returns
// clear errors when subscription operations are attempted.
func NewNoOpWSSubscriptionManager(chainID, apiInterface string) *NoOpWSSubscriptionManager {
	return &NoOpWSSubscriptionManager{
		chainID:      chainID,
		apiInterface: apiInterface,
	}
}

// StartSubscription returns an error indicating WebSocket subscriptions are not configured.
func (n *NoOpWSSubscriptionManager) StartSubscription(
	ctx context.Context,
	protocolMessage chainlib.ProtocolMessage,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) (firstReply *pairingtypes.RelayReply, repliesChan <-chan *pairingtypes.RelayReply, err error) {
	utils.LavaFormatWarning("WebSocket subscription attempted but no WebSocket endpoints configured", nil,
		utils.LogAttr("chainID", n.chainID),
		utils.LogAttr("apiInterface", n.apiInterface),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("hint", "Configure ws:// or wss:// URLs in static-providers-list to enable subscriptions"),
	)
	return nil, nil, fmt.Errorf("WebSocket subscriptions not available: no ws:// or wss:// endpoints configured for chain %s. "+
		"Add WebSocket URLs to your static-providers-list configuration to enable subscriptions", n.chainID)
}

// Unsubscribe returns an error indicating WebSocket subscriptions are not configured.
func (n *NoOpWSSubscriptionManager) Unsubscribe(
	ctx context.Context,
	protocolMessage chainlib.ProtocolMessage,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) error {
	utils.LavaFormatDebug("Unsubscribe attempted but no WebSocket endpoints configured",
		utils.LogAttr("chainID", n.chainID),
		utils.LogAttr("apiInterface", n.apiInterface),
	)
	return fmt.Errorf("WebSocket subscriptions not available: no ws:// or wss:// endpoints configured for chain %s", n.chainID)
}

// UnsubscribeAll is a no-op since no subscriptions can exist without WebSocket endpoints.
func (n *NoOpWSSubscriptionManager) UnsubscribeAll(
	ctx context.Context,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) error {
	// No subscriptions exist, so this is always a no-op success
	return nil
}

// Compile-time interface compliance check
var _ chainlib.WSSubscriptionManager = (*NoOpWSSubscriptionManager)(nil)
