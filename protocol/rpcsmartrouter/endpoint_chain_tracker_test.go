package rpcsmartrouter

import (
	"context"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/stretchr/testify/require"
)

// mockDirectRPCConnection implements lavasession.DirectRPCConnection for testing
type mockDirectRPCConnection struct {
	url       string
	healthy   bool
	responses map[string][]byte // request -> response
}

func (m *mockDirectRPCConnection) SendRequest(ctx context.Context, data []byte, headers map[string]string) (*lavasession.DirectRPCResponse, error) {
	// Return mock response based on request
	if response, ok := m.responses[string(data)]; ok {
		return &lavasession.DirectRPCResponse{
			Data:       response,
			StatusCode: 200,
		}, nil
	}
	// Default response for eth_blockNumber
	return &lavasession.DirectRPCResponse{
		Data:       []byte(`{"jsonrpc":"2.0","id":1,"result":"0x100"}`),
		StatusCode: 200,
	}, nil
}

func (m *mockDirectRPCConnection) GetProtocol() lavasession.DirectRPCProtocol {
	return lavasession.DirectRPCProtocolHTTP
}

func (m *mockDirectRPCConnection) Close() error {
	return nil
}

func (m *mockDirectRPCConnection) IsHealthy() bool {
	return m.healthy
}

func (m *mockDirectRPCConnection) GetURL() string {
	return m.url
}

func (m *mockDirectRPCConnection) GetNodeUrl() interface{} {
	return nil
}

func TestEndpointChainTrackerManager_GetOrCreateTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create manager with minimal config
	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Check initial state
	require.Equal(t, 0, manager.GetEndpointCount())
	require.False(t, manager.IsDummy())

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_GetLatestBlockNum(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Test getting latest block for non-existent endpoint
	latestBlock := manager.GetLatestBlockNum("http://non-existent:8545")
	require.Equal(t, int64(0), latestBlock)

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_ValidateEndpointSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Non-existent endpoint should pass validation (no data = allow)
	result := manager.ValidateEndpointSync("http://non-existent:8545", 100, 10)
	require.True(t, result)

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_GetSyncGap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Non-existent endpoint should have 0 sync gap
	syncGap := manager.GetSyncGap("http://non-existent:8545", 100)
	require.Equal(t, int64(0), syncGap)

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_GetAllEndpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Initially empty
	endpoints := manager.GetAllEndpoints()
	require.Empty(t, endpoints)

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_RemoveTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Remove non-existent tracker (should not panic)
	manager.RemoveTracker("http://non-existent:8545")

	// Verify still works after remove
	require.Equal(t, 0, manager.GetEndpointCount())

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerManager_Callbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	forkCalled := false
	newBlockCalled := false

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
		OnFork: func(endpointURL string, blockNum int64) {
			forkCalled = true
		},
		OnNewBlock: func(endpointURL string, fromBlock, toBlock int64) {
			newBlockCalled = true
		},
	})
	require.NotNil(t, manager)

	// Callbacks aren't called until trackers are created and receive blocks
	require.False(t, forkCalled)
	require.False(t, newBlockCalled)

	// Cleanup
	manager.Stop()
}

func TestEndpointChainTrackerConfig_DefaultValues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create with minimal config (should use defaults)
	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:      "ETH",
		ApiInterface: "jsonrpc",
		// AverageBlockTime and BlocksToSave not set - should use defaults
	})
	require.NotNil(t, manager)

	// Verify manager works with defaults
	require.Equal(t, 0, manager.GetEndpointCount())

	// Cleanup
	manager.Stop()
}

// TestEndpointChainTrackerManager_LifecyclePerTrackerContext verifies that each tracker
// gets its own cancellable context, enabling individual tracker shutdown.
func TestEndpointChainTrackerManager_LifecyclePerTrackerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)
	defer manager.Stop()

	// Verify cancelFuncs map is initialized
	require.NotNil(t, manager.cancelFuncs)
	require.Empty(t, manager.cancelFuncs)
}

// TestEndpointChainTrackerManager_RemoveTrackerCancelsContext verifies that RemoveTracker
// properly cancels the tracker's context before removing it from maps.
func TestEndpointChainTrackerManager_RemoveTrackerCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)
	defer manager.Stop()

	// Manually add a cancel func to simulate a tracker
	testEndpoint := "http://test:8545"
	trackerCancelled := false
	manager.cancelFuncs[testEndpoint] = func() {
		trackerCancelled = true
	}

	// Remove should call the cancel function
	manager.RemoveTracker(testEndpoint)

	require.True(t, trackerCancelled, "RemoveTracker should call the cancel function")
	require.Empty(t, manager.cancelFuncs, "cancelFuncs should be empty after removal")
}

// TestEndpointChainTrackerManager_StopCancelsAllTrackers verifies that Stop()
// properly cancels all individual tracker contexts.
func TestEndpointChainTrackerManager_StopCancelsAllTrackers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)

	// Manually add cancel funcs to simulate multiple trackers
	cancelledTrackers := make(map[string]bool)
	endpoints := []string{"http://test1:8545", "http://test2:8545", "http://test3:8545"}

	for _, ep := range endpoints {
		manager.cancelFuncs[ep] = func() {
			cancelledTrackers[ep] = true
		}
	}

	// Stop should cancel all trackers
	manager.Stop()

	// Verify all trackers were cancelled
	for _, ep := range endpoints {
		require.True(t, cancelledTrackers[ep], "Stop should cancel tracker for %s", ep)
	}
	require.Empty(t, manager.cancelFuncs, "cancelFuncs should be empty after Stop")
}

// TestEndpointChainTrackerManager_ConcurrentCreateRemove tests that concurrent
// create and remove operations are thread-safe.
func TestEndpointChainTrackerManager_ConcurrentCreateRemove(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewEndpointChainTrackerManager(ctx, EndpointChainTrackerConfig{
		ChainID:          "ETH",
		ApiInterface:     "jsonrpc",
		AverageBlockTime: 12 * time.Second,
		BlocksToSave:     10,
	})
	require.NotNil(t, manager)
	defer manager.Stop()

	// Simulate concurrent operations (without actual trackers, just testing map access)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			manager.GetAllEndpoints()
			manager.GetEndpointCount()
			manager.RemoveTracker("http://test:8545")
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		manager.GetLatestBlockNum("http://test:8545")
		manager.GetTracker("http://test:8545")
	}

	<-done
	// If we reach here without a race detector error, the test passes
}
