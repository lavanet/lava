package rpcsmartrouter

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpWSSubscriptionManager(t *testing.T) {
	chainID := "ETH"
	apiInterface := "jsonrpc"

	manager := NewNoOpWSSubscriptionManager(chainID, apiInterface)

	require.NotNil(t, manager)
	assert.Equal(t, chainID, manager.chainID)
	assert.Equal(t, apiInterface, manager.apiInterface)
}

func TestNoOpWSSubscriptionManager_StartSubscription(t *testing.T) {
	chainID := "ETH"
	apiInterface := "jsonrpc"
	manager := NewNoOpWSSubscriptionManager(chainID, apiInterface)

	ctx := context.Background()

	reply, repliesChan, err := manager.StartSubscription(
		ctx,
		nil,       // protocolMessage
		"dapp-1",  // dappID
		"1.2.3.4", // consumerIp
		"ws-conn-1",
		nil, // metricsData
	)

	// Verify it returns an error with appropriate message
	assert.Error(t, err)
	assert.Nil(t, reply)
	assert.Nil(t, repliesChan)

	// Verify error message contains helpful information
	assert.Contains(t, err.Error(), "WebSocket subscriptions not available")
	assert.Contains(t, err.Error(), chainID)
	assert.Contains(t, err.Error(), "static-providers-list")
}

func TestNoOpWSSubscriptionManager_Unsubscribe(t *testing.T) {
	chainID := "LAVA"
	apiInterface := "tendermint"
	manager := NewNoOpWSSubscriptionManager(chainID, apiInterface)

	ctx := context.Background()

	err := manager.Unsubscribe(
		ctx,
		nil,       // protocolMessage
		"dapp-2",  // dappID
		"5.6.7.8", // consumerIp
		"ws-conn-2",
		nil, // metricsData
	)

	// Verify it returns an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket subscriptions not available")
	assert.Contains(t, err.Error(), chainID)
}

func TestNoOpWSSubscriptionManager_UnsubscribeAll(t *testing.T) {
	chainID := "ETH"
	apiInterface := "jsonrpc"
	manager := NewNoOpWSSubscriptionManager(chainID, apiInterface)

	ctx := context.Background()

	err := manager.UnsubscribeAll(
		ctx,
		"dapp-3",
		"9.10.11.12",
		"ws-conn-3",
		nil, // metricsData
	)

	// UnsubscribeAll should return nil (success) because no subscriptions exist
	assert.NoError(t, err)
}

func TestNoOpWSSubscriptionManager_ImplementsInterface(t *testing.T) {
	// Compile-time check that NoOpWSSubscriptionManager implements WSSubscriptionManager
	var _ chainlib.WSSubscriptionManager = (*NoOpWSSubscriptionManager)(nil)
}

func TestNoOpWSSubscriptionManager_DifferentChains(t *testing.T) {
	tests := []struct {
		chainID      string
		apiInterface string
	}{
		{"ETH", "jsonrpc"},
		{"LAVA", "tendermint"},
		{"COSMOSHUB", "grpcweb"},
		{"POLYGON", "jsonrpc"},
	}

	for _, tt := range tests {
		t.Run(tt.chainID, func(t *testing.T) {
			manager := NewNoOpWSSubscriptionManager(tt.chainID, tt.apiInterface)

			_, _, err := manager.StartSubscription(
				context.Background(),
				nil, "dapp", "ip", "conn", nil,
			)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.chainID)
		})
	}
}

func TestNoOpWSSubscriptionManager_CanceledContext(t *testing.T) {
	manager := NewNoOpWSSubscriptionManager("ETH", "jsonrpc")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still return the "not configured" error, not a context error
	_, _, err := manager.StartSubscription(ctx, nil, "dapp", "ip", "conn", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WebSocket subscriptions not available")
}
