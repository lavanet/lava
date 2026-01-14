package rpcsmartrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	rpcclient "github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	rpcInterfaceMessages "github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/metrics"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSubscriptionServer creates a mock WebSocket server for testing subscriptions
type mockSubscriptionServer struct {
	server          *httptest.Server
	upgrader        websocket.Upgrader
	subscriptions   map[string]chan struct{} // subscription ID -> close channel
	lock            sync.RWMutex
	onSubscribe     func(method string, params interface{}) (string, error)
	onUnsubscribe   func(subID string) error
	messageInterval time.Duration
}

func newMockSubscriptionServer() *mockSubscriptionServer {
	ms := &mockSubscriptionServer{
		upgrader:        websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		subscriptions:   make(map[string]chan struct{}),
		messageInterval: 100 * time.Millisecond,
	}

	ms.server = httptest.NewServer(http.HandlerFunc(ms.handleWS))
	return ms
}

func (ms *mockSubscriptionServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := ms.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Parse JSON-RPC request
		var req struct {
			ID     interface{} `json:"id"`
			Method string      `json:"method"`
			Params interface{} `json:"params"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		// Handle subscription requests
		if strings.HasSuffix(req.Method, "_subscribe") || strings.HasPrefix(req.Method, "eth_subscribe") {
			subID := ms.createSubscription()

			// Send subscription confirmation
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  subID,
			}
			respBytes, _ := json.Marshal(response)
			conn.WriteMessage(websocket.TextMessage, respBytes)

			// Start sending subscription messages
			go ms.sendSubscriptionMessages(conn, subID)
		} else if strings.HasSuffix(req.Method, "_unsubscribe") || strings.HasPrefix(req.Method, "eth_unsubscribe") {
			// Get subscription ID from params
			params, ok := req.Params.([]interface{})
			if ok && len(params) > 0 {
				if subID, ok := params[0].(string); ok {
					ms.closeSubscription(subID)
				}
			}

			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  true,
			}
			respBytes, _ := json.Marshal(response)
			conn.WriteMessage(websocket.TextMessage, respBytes)
		}
	}
}

func (ms *mockSubscriptionServer) createSubscription() string {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	subID := "0x" + strings.Repeat("f", 32)[:32] // Simple subscription ID
	ms.subscriptions[subID] = make(chan struct{})
	return subID
}

func (ms *mockSubscriptionServer) closeSubscription(subID string) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ch, exists := ms.subscriptions[subID]; exists {
		close(ch)
		delete(ms.subscriptions, subID)
	}
}

func (ms *mockSubscriptionServer) sendSubscriptionMessages(conn *websocket.Conn, subID string) {
	ms.lock.RLock()
	closeCh, exists := ms.subscriptions[subID]
	ms.lock.RUnlock()
	if !exists {
		return
	}

	counter := 0
	ticker := time.NewTicker(ms.messageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-closeCh:
			return
		case <-ticker.C:
			counter++
			// Send subscription notification
			notification := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "eth_subscription",
				"params": map[string]interface{}{
					"subscription": subID,
					"result": map[string]interface{}{
						"number": counter,
					},
				},
			}
			msgBytes, _ := json.Marshal(notification)
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				return
			}
		}
	}
}

func (ms *mockSubscriptionServer) Close() {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	for _, ch := range ms.subscriptions {
		close(ch)
	}
	ms.subscriptions = make(map[string]chan struct{})
	ms.server.Close()
}

func (ms *mockSubscriptionServer) URL() string {
	return "ws" + strings.TrimPrefix(ms.server.URL, "http")
}

// mockWSProtocolMessage implements chainlib.ProtocolMessage for WebSocket subscription tests
type mockWSProtocolMessage struct {
	method string
	params interface{}
}

func (m *mockWSProtocolMessage) GetApi() *spectypes.Api {
	return &spectypes.Api{Name: m.method}
}

func (m *mockWSProtocolMessage) GetApiCollection() *spectypes.ApiCollection {
	return &spectypes.ApiCollection{
		CollectionData: spectypes.CollectionData{
			ApiInterface: "jsonrpc",
		},
	}
}

func (m *mockWSProtocolMessage) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return &mockWSGenericMessage{method: m.method, params: m.params}
}

func (m *mockWSProtocolMessage) RelayPrivateData() *pairingtypes.RelayPrivateData {
	data, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  m.method,
		"params":  m.params,
	})
	return &pairingtypes.RelayPrivateData{Data: data}
}

// Implement remaining chainlib.ProtocolMessage methods (stubs)
func (m *mockWSProtocolMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}
func (m *mockWSProtocolMessage) RequestedBlock() (latest int64, earliest int64) { return 0, 0 }
func (m *mockWSProtocolMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) bool {
	return false
}
func (m *mockWSProtocolMessage) AppendHeader(metadata []pairingtypes.Metadata) {}
func (m *mockWSProtocolMessage) GetExtensions() []*spectypes.Extension         { return nil }
func (m *mockWSProtocolMessage) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
}
func (m *mockWSProtocolMessage) DisableErrorHandling()                               {}
func (m *mockWSProtocolMessage) TimeoutOverride(...time.Duration) time.Duration      { return 0 }
func (m *mockWSProtocolMessage) GetForceCacheRefresh() bool                          { return false }
func (m *mockWSProtocolMessage) SetForceCacheRefresh(force bool) bool                { return false }
func (m *mockWSProtocolMessage) GetRawRequestHash() ([]byte, error)                  { return nil, nil }
func (m *mockWSProtocolMessage) GetRequestedBlocksHashes() []string                  { return nil }
func (m *mockWSProtocolMessage) UpdateEarliestInMessage(incomingEarliest int64) bool { return false }
func (m *mockWSProtocolMessage) SetExtension(extension *spectypes.Extension)         {}
func (m *mockWSProtocolMessage) GetUsedDefaultValue() bool                           { return false }
func (m *mockWSProtocolMessage) GetParseDirective() *spectypes.ParseDirective        { return nil }
func (m *mockWSProtocolMessage) CheckResponseError(data []byte, httpStatusCode int) (bool, string) {
	return false, ""
}
func (m *mockWSProtocolMessage) GetDirectiveHeaders() map[string]string { return nil }
func (m *mockWSProtocolMessage) HashCacheRequest(chainId string) ([]byte, func([]byte) []byte, error) {
	return nil, nil, nil
}
func (m *mockWSProtocolMessage) GetBlockedProviders() []string { return nil }
func (m *mockWSProtocolMessage) GetUserData() common.UserData  { return common.UserData{} }
func (m *mockWSProtocolMessage) IsDefaultApi() bool            { return false }
func (m *mockWSProtocolMessage) UpdateEarliestAndValidateExtensionRules(extensionParser *extensionslib.ExtensionParser, earliestBlockHashRequested int64, addon string, seenBlock int64) bool {
	return false
}
func (m *mockWSProtocolMessage) GetQuorumParameters() (common.QuorumParams, error) {
	return common.QuorumParams{}, nil
}

type mockWSGenericMessage struct {
	method string
	params interface{}
}

func (m *mockWSGenericMessage) GetHeaders() []pairingtypes.Metadata { return nil }
func (m *mockWSGenericMessage) DisableErrorHandling()               {}
func (m *mockWSGenericMessage) GetParams() interface{}              { return m.params }
func (m *mockWSGenericMessage) GetMethod() string                   { return m.method }
func (m *mockWSGenericMessage) GetResult() json.RawMessage          { return nil }
func (m *mockWSGenericMessage) GetID() json.RawMessage              { return []byte("1") }

// Package-level test metrics manager to avoid duplicate registration
var testMetricsManager *metrics.ConsumerMetricsManager
var testMetricsOnce sync.Once

func getTestMetricsManager() *metrics.ConsumerMetricsManager {
	testMetricsOnce.Do(func() {
		testMetricsManager = metrics.NewConsumerMetricsManager(metrics.ConsumerMetricsManagerOptions{})
	})
	return testMetricsManager
}

// TestNewDirectWSSubscriptionManager tests the constructor
func TestNewDirectWSSubscriptionManager(t *testing.T) {
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil, // No optimizer for basic test
		nil, // Use default WebSocket config
	)

	require.NotNil(t, manager)
	assert.NotNil(t, manager.connectedClients)
	assert.NotNil(t, manager.activeSubscriptions)
	assert.NotNil(t, manager.pendingSubscriptions)
	assert.NotNil(t, manager.upstreamPools)
	assert.NotNil(t, manager.idMapper)
	assert.Equal(t, "ETH", manager.chainID)
	assert.Len(t, manager.wsEndpoints, 1)
	assert.Equal(t, "wss://test.example.com", manager.wsEndpoints[0].Url)
}

// TestCreateWebSocketConnectionUniqueKey tests the client key generation
func TestCreateWebSocketConnectionUniqueKey(t *testing.T) {
	manager := &DirectWSSubscriptionManager{}

	key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.1", "ws-uid-123")
	assert.Equal(t, "dapp1:192.168.1.1:ws-uid-123", key)
}

// TestGetOrCreatePool tests pool creation and reuse
func TestGetOrCreatePool(t *testing.T) {
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil, // Use default WebSocket config
	)

	// First call should create a new pool
	pool1 := manager.GetOrCreatePool(nodeUrl)
	require.NotNil(t, pool1)

	// Second call should return the same pool
	pool2 := manager.GetOrCreatePool(nodeUrl)
	assert.Same(t, pool1, pool2)

	// Different URL should create different pool
	nodeUrl2 := &common.NodeUrl{Url: "wss://other.example.com"}
	pool3 := manager.GetOrCreatePool(nodeUrl2)
	assert.NotSame(t, pool1, pool3)
}

// TestDirectWSSubscriptionManager_ImplementsInterface verifies interface compliance
func TestDirectWSSubscriptionManager_ImplementsInterface(t *testing.T) {
	// Compile-time check that DirectWSSubscriptionManager implements WSSubscriptionManager
	var _ chainlib.WSSubscriptionManager = (*DirectWSSubscriptionManager)(nil)
}

// TestDirectWSSubscriptionManager_UnsubscribeAll_NoSubscriptions tests unsubscribe all with no active subscriptions
func TestDirectWSSubscriptionManager_UnsubscribeAll_NoSubscriptions(t *testing.T) {
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil, // Use default WebSocket config
	)

	ctx := context.Background()
	err := manager.UnsubscribeAll(ctx, "dapp1", "192.168.1.1", "ws-uid-123", nil)

	// Should not error when no subscriptions exist
	assert.NoError(t, err)
}

// TestDirectWSSubscriptionManager_Unsubscribe_NoSubscriptions tests unsubscribe with no active subscriptions
func TestDirectWSSubscriptionManager_Unsubscribe_NoSubscriptions(t *testing.T) {
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil, // Use default WebSocket config
	)

	ctx := context.Background()
	protocolMessage := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{"0x123"},
	}

	err := manager.Unsubscribe(ctx, protocolMessage, "dapp1", "192.168.1.1", "ws-uid-123", nil)

	// Should return subscription not found error
	assert.Error(t, err)
	assert.Equal(t, common.SubscriptionNotFoundError, err)
}

// TestDirectWSSubscriptionManager_StartSubscription_ConnectionFailure tests handling connection failures
func TestDirectWSSubscriptionManager_StartSubscription_ConnectionFailure(t *testing.T) {
	// Use an invalid WebSocket URL that will fail to connect
	nodeUrl := &common.NodeUrl{Url: "wss://invalid.nonexistent.example.com:9999"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil, // Use default WebSocket config
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	protocolMessage := &mockWSProtocolMessage{
		method: "eth_subscribe",
		params: []interface{}{"newHeads"},
	}

	reply, repliesChan, err := manager.StartSubscription(ctx, protocolMessage, "dapp1", "192.168.1.1", "ws-uid-123", nil)

	// Should fail with connection error
	assert.Error(t, err)
	assert.Nil(t, reply)
	assert.Nil(t, repliesChan)
	assert.Contains(t, err.Error(), "failed to get WebSocket connection")
}

// TestDeduplicationMultiClientUniqueRouterIDs verifies that when multiple clients subscribe
// to the same subscription parameters, each client gets their own unique router ID.
// This is critical for proper unsubscribe behavior - when one client unsubscribes,
// the others should remain connected and continue receiving messages.
//
// This test addresses the bug where a single router ID was shared across all clients,
// causing the first client's unsubscribe to tear down the shared upstream subscription.
func TestDeduplicationMultiClientUniqueRouterIDs(t *testing.T) {
	// Test that the ID mapper correctly handles multiple router IDs for one upstream ID
	mapper := NewSubscriptionIDMapper()

	// Simulate three clients subscribing to the same feed
	client1Key := "dapp1:192.168.1.1:ws-1"
	client2Key := "dapp1:192.168.1.2:ws-2"
	client3Key := "dapp1:192.168.1.3:ws-3"

	// Each client should get a unique router ID
	routerID1 := mapper.GenerateRouterID(client1Key)
	routerID2 := mapper.GenerateRouterID(client2Key)
	routerID3 := mapper.GenerateRouterID(client3Key)

	// Verify all router IDs are unique
	assert.NotEqual(t, routerID1, routerID2, "Router IDs for different clients must be unique")
	assert.NotEqual(t, routerID2, routerID3, "Router IDs for different clients must be unique")
	assert.NotEqual(t, routerID1, routerID3, "Router IDs for different clients must be unique")

	// All three router IDs map to the same upstream ID (the shared subscription)
	upstreamID := "0xupstreamABC123"
	mapper.RegisterMapping(routerID1, upstreamID)
	mapper.RegisterMapping(routerID2, upstreamID)
	mapper.RegisterMapping(routerID3, upstreamID)

	// Verify all mappings are correct
	gotUpstream1, found1 := mapper.GetUpstreamID(routerID1)
	gotUpstream2, found2 := mapper.GetUpstreamID(routerID2)
	gotUpstream3, found3 := mapper.GetUpstreamID(routerID3)
	assert.True(t, found1 && found2 && found3)
	assert.Equal(t, upstreamID, gotUpstream1)
	assert.Equal(t, upstreamID, gotUpstream2)
	assert.Equal(t, upstreamID, gotUpstream3)

	// Verify GetRouterIDs returns all three
	routerIDs := mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 3)
	assert.Contains(t, routerIDs, routerID1)
	assert.Contains(t, routerIDs, routerID2)
	assert.Contains(t, routerIDs, routerID3)

	// Client 2 unsubscribes - should NOT report lastClient
	removedUpstream, lastClient := mapper.RemoveMapping(routerID2)
	assert.Equal(t, upstreamID, removedUpstream)
	assert.False(t, lastClient, "Client 2 should NOT be the last client")

	// Client 1 and 3 should still have valid mappings
	gotUpstream1, found1 = mapper.GetUpstreamID(routerID1)
	gotUpstream3, found3 = mapper.GetUpstreamID(routerID3)
	assert.True(t, found1, "Client 1's mapping should still exist")
	assert.True(t, found3, "Client 3's mapping should still exist")
	assert.Equal(t, upstreamID, gotUpstream1)
	assert.Equal(t, upstreamID, gotUpstream3)

	// Client 2's mapping should be gone
	_, found2 = mapper.GetUpstreamID(routerID2)
	assert.False(t, found2, "Client 2's mapping should be removed")

	// Remaining router IDs
	routerIDs = mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 2)
	assert.NotContains(t, routerIDs, routerID2)

	// Client 3 unsubscribes - still NOT the last client
	removedUpstream, lastClient = mapper.RemoveMapping(routerID3)
	assert.Equal(t, upstreamID, removedUpstream)
	assert.False(t, lastClient, "Client 3 should NOT be the last client")

	// Only client 1 remains
	routerIDs = mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 1)
	assert.Equal(t, routerID1, routerIDs[0])

	// Client 1 unsubscribes - should be the LAST client
	removedUpstream, lastClient = mapper.RemoveMapping(routerID1)
	assert.Equal(t, upstreamID, removedUpstream)
	assert.True(t, lastClient, "Client 1 SHOULD be the last client")

	// No more mappings
	routerIDs = mapper.GetRouterIDs(upstreamID)
	assert.Len(t, routerIDs, 0)
}

// TestActiveSubscriptionTracksPerClientRouterIDs verifies that directActiveSubscription
// correctly tracks per-client router IDs through the clientRouterIDs map.
func TestActiveSubscriptionTracksPerClientRouterIDs(t *testing.T) {
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil,
	)

	// Simulate creating an active subscription
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	activeSub := &directActiveSubscription{
		upstreamID:       "0xupstreamXYZ",
		clientRouterIDs:  make(map[string]string),
		connectedClients: make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		hashedParams:     "test-params-hash",
		ctx:              ctx,
		cancel:           cancel,
	}

	// Add three clients
	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.1", "ws-1")
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.2", "ws-2")
	client3Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.3", "ws-3")

	// Generate unique router IDs for each client
	routerID1 := manager.idMapper.GenerateRouterID(client1Key)
	routerID2 := manager.idMapper.GenerateRouterID(client2Key)
	routerID3 := manager.idMapper.GenerateRouterID(client3Key)

	// Store in active subscription
	activeSub.clientRouterIDs[client1Key] = routerID1
	activeSub.clientRouterIDs[client2Key] = routerID2
	activeSub.clientRouterIDs[client3Key] = routerID3

	// Verify each client has a unique router ID
	assert.Len(t, activeSub.clientRouterIDs, 3)
	assert.Equal(t, routerID1, activeSub.clientRouterIDs[client1Key])
	assert.Equal(t, routerID2, activeSub.clientRouterIDs[client2Key])
	assert.Equal(t, routerID3, activeSub.clientRouterIDs[client3Key])

	// Verify router IDs are all different
	assert.NotEqual(t, routerID1, routerID2)
	assert.NotEqual(t, routerID2, routerID3)
	assert.NotEqual(t, routerID1, routerID3)

	// Simulate client 2 disconnecting
	delete(activeSub.clientRouterIDs, client2Key)

	// Client 1 and 3 should still have their router IDs
	assert.Len(t, activeSub.clientRouterIDs, 2)
	assert.Equal(t, routerID1, activeSub.clientRouterIDs[client1Key])
	assert.Equal(t, routerID3, activeSub.clientRouterIDs[client3Key])
}

// TestUnsubscribeRateLimiting verifies that unsubscribe requests are rate limited.
// This prevents clients from spamming unsubscribe requests which could cause
// unnecessary load on the subscription manager.
func TestUnsubscribeRateLimiting(t *testing.T) {
	// Create a config with very low unsubscribe limit for testing
	config := &WebsocketConfig{
		MaxSubscriptionsPerClient:       25,
		PerClientLimitEnforcement:       "warn",
		MaxTotalSubscriptions:           5000,
		TotalLimitEnforcement:           "warn",
		SubscriptionSharingEnabled:      true,
		SubscriptionsPerMinutePerClient: 10,
		UnsubscribesPerMinutePerClient:  2, // Very low limit for testing
		MaxMessageSize:                  1048576,
		CleanupInterval:                 time.Minute,
	}

	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		config,
	)

	ctx := context.Background()

	// First two unsubscribes should succeed (within burst limit)
	for i := 0; i < 2; i++ {
		protocolMessage := &mockWSProtocolMessage{
			method: "eth_unsubscribe",
			params: []interface{}{fmt.Sprintf("0x%d", i)},
		}

		err := manager.Unsubscribe(ctx, protocolMessage, "dapp1", "192.168.1.1", "ws-uid-123", nil)
		// These will return "subscription not found" but shouldn't be rate limited
		assert.Equal(t, common.SubscriptionNotFoundError, err,
			"Unsubscribe %d should return subscription not found (not rate limited)", i+1)
	}

	// Third unsubscribe should be rate limited
	protocolMessage := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{"0x999"},
	}

	err := manager.Unsubscribe(ctx, protocolMessage, "dapp1", "192.168.1.1", "ws-uid-123", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsubscribe rate limit exceeded",
		"Third unsubscribe should be rate limited")
}

// TestUnsubscribeAllNotRateLimited verifies that UnsubscribeAll is NOT rate limited
// since it's a cleanup operation that should always succeed.
func TestUnsubscribeAllNotRateLimited(t *testing.T) {
	// Create a config with very low unsubscribe limit
	config := &WebsocketConfig{
		MaxSubscriptionsPerClient:       25,
		PerClientLimitEnforcement:       "warn",
		MaxTotalSubscriptions:           5000,
		TotalLimitEnforcement:           "warn",
		SubscriptionSharingEnabled:      true,
		SubscriptionsPerMinutePerClient: 10,
		UnsubscribesPerMinutePerClient:  1, // Extremely low limit
		MaxMessageSize:                  1048576,
		CleanupInterval:                 time.Minute,
	}

	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		config,
	)

	ctx := context.Background()

	// Multiple UnsubscribeAll calls should all succeed (not rate limited)
	for i := 0; i < 5; i++ {
		err := manager.UnsubscribeAll(ctx, "dapp1", "192.168.1.1", fmt.Sprintf("ws-uid-%d", i), nil)
		assert.NoError(t, err, "UnsubscribeAll %d should not be rate limited", i+1)
	}
}

// =============================================================================
// Multi-Client Flow Integration Tests
// =============================================================================
// These tests exercise the full multi-client subscription flows to ensure:
// 1. Multiple clients can share a subscription with unique router IDs
// 2. Clients can unsubscribe independently without affecting others
// 3. Reconnection properly updates connection bookkeeping
// =============================================================================

// TestMultiClientJoinExistingSubscription tests that when multiple clients subscribe
// to the same parameters, each gets a unique router ID and can operate independently.
// This is the core deduplication flow test.
func TestMultiClientJoinExistingSubscription(t *testing.T) {
	metricsManager := getTestMetricsManager()
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil,
	)

	ctx := context.Background()
	hashedParams := "test-subscription-params-hash"

	// Manually set up an active subscription (simulating first client)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	upstreamID := "0xupstream123"
	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.1", "ws-1")
	client1RouterID := manager.idMapper.GenerateRouterID(client1Key)
	manager.idMapper.RegisterMapping(client1RouterID, upstreamID)

	// Create reply channel for client 1
	client1ReplyChan := make(chan *pairingtypes.RelayReply, 10)
	client1Sender := common.NewSafeChannelSender(subCtx, client1ReplyChan)

	// Create first reply data
	firstReplyData, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  client1RouterID,
	})

	activeSub := &directActiveSubscription{
		upstreamID:       upstreamID,
		clientRouterIDs:  make(map[string]string),
		connectedClients: make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		hashedParams:     hashedParams,
		firstReply:       &pairingtypes.RelayReply{Data: firstReplyData},
		ctx:              subCtx,
		cancel:           cancel,
		messagesChan:     make(chan *rpcclient.JsonrpcMessage, 100),
	}
	activeSub.connectedClients[client1Key] = client1Sender
	activeSub.clientRouterIDs[client1Key] = client1RouterID

	// Store in manager
	manager.lock.Lock()
	manager.activeSubscriptions[hashedParams] = activeSub
	manager.connectedClients[client1Key] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	manager.connectedClients[client1Key][hashedParams] = client1Sender
	manager.lock.Unlock()

	// Now client 2 joins the existing subscription
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp2", "192.168.1.2", "ws-2")
	client2ReplyChan := make(chan *pairingtypes.RelayReply, 10)
	client2Sender := common.NewSafeChannelSender(subCtx, client2ReplyChan)

	// Call checkForActiveSubscriptionAndConnect for client 2
	// Use a different request ID to verify proper ID handling
	client2RequestID := json.RawMessage("2")
	reply, joined := manager.checkForActiveSubscriptionAndConnect(subCtx, hashedParams, client2Key, client2Sender, client2RequestID)

	// Verify client 2 joined successfully
	assert.True(t, joined, "Client 2 should have joined existing subscription")
	assert.NotNil(t, reply, "Client 2 should receive a reply")

	// Verify client 2 got a unique router ID
	manager.lock.RLock()
	client2RouterID := activeSub.clientRouterIDs[client2Key]
	manager.lock.RUnlock()

	assert.NotEmpty(t, client2RouterID, "Client 2 should have a router ID")
	assert.NotEqual(t, client1RouterID, client2RouterID,
		"Client 2's router ID should be different from Client 1's")

	// Verify the reply contains client 2's unique router ID and correct request ID
	var replyData map[string]interface{}
	err := json.Unmarshal(reply.Data, &replyData)
	require.NoError(t, err)
	assert.Equal(t, client2RouterID, replyData["result"],
		"Reply should contain Client 2's unique router ID")
	// Verify the response ID matches client 2's request ID (JSON-RPC compliance)
	assert.Equal(t, float64(2), replyData["id"],
		"Reply should contain Client 2's request ID (2), not a hard-coded value")

	// Verify both clients are tracked
	manager.lock.RLock()
	assert.Len(t, activeSub.connectedClients, 2, "Should have 2 connected clients")
	assert.Len(t, activeSub.clientRouterIDs, 2, "Should have 2 client router IDs")
	manager.lock.RUnlock()

	// Verify ID mapper has both mappings
	upstream1, found1 := manager.idMapper.GetUpstreamID(client1RouterID)
	upstream2, found2 := manager.idMapper.GetUpstreamID(client2RouterID)
	assert.True(t, found1, "Client 1's router ID should be in mapper")
	assert.True(t, found2, "Client 2's router ID should be in mapper")
	assert.Equal(t, upstreamID, upstream1, "Client 1's mapping should point to upstream")
	assert.Equal(t, upstreamID, upstream2, "Client 2's mapping should point to upstream")
}

// TestMultiClientIndependentUnsubscribe verifies that when one client unsubscribes
// from a shared subscription, other clients remain connected and functional.
// This is the critical test for the deduplication bug fix.
func TestMultiClientIndependentUnsubscribe(t *testing.T) {
	metricsManager := getTestMetricsManager()
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil,
	)

	ctx := context.Background()
	hashedParams := "shared-subscription-hash"
	upstreamID := "0xupstreamShared"

	// Set up 3 clients sharing the same subscription
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "10.0.0.1", "ws-1")
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp2", "10.0.0.2", "ws-2")
	client3Key := manager.CreateWebSocketConnectionUniqueKey("dapp3", "10.0.0.3", "ws-3")

	// Generate unique router IDs for each client
	client1RouterID := manager.idMapper.GenerateRouterID(client1Key)
	client2RouterID := manager.idMapper.GenerateRouterID(client2Key)
	client3RouterID := manager.idMapper.GenerateRouterID(client3Key)

	// Register all mappings to the same upstream
	manager.idMapper.RegisterMapping(client1RouterID, upstreamID)
	manager.idMapper.RegisterMapping(client2RouterID, upstreamID)
	manager.idMapper.RegisterMapping(client3RouterID, upstreamID)

	// Create channels for each client
	client1Chan := make(chan *pairingtypes.RelayReply, 10)
	client2Chan := make(chan *pairingtypes.RelayReply, 10)
	client3Chan := make(chan *pairingtypes.RelayReply, 10)

	client1Sender := common.NewSafeChannelSender(subCtx, client1Chan)
	client2Sender := common.NewSafeChannelSender(subCtx, client2Chan)
	client3Sender := common.NewSafeChannelSender(subCtx, client3Chan)

	// Create active subscription with all 3 clients
	activeSub := &directActiveSubscription{
		upstreamID:       upstreamID,
		clientRouterIDs:  make(map[string]string),
		connectedClients: make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		hashedParams:     hashedParams,
		ctx:              subCtx,
		cancel:           cancel,
		messagesChan:     make(chan *rpcclient.JsonrpcMessage, 100),
	}
	activeSub.connectedClients[client1Key] = client1Sender
	activeSub.connectedClients[client2Key] = client2Sender
	activeSub.connectedClients[client3Key] = client3Sender
	activeSub.clientRouterIDs[client1Key] = client1RouterID
	activeSub.clientRouterIDs[client2Key] = client2RouterID
	activeSub.clientRouterIDs[client3Key] = client3RouterID

	// Store in manager
	manager.lock.Lock()
	manager.activeSubscriptions[hashedParams] = activeSub
	manager.connectedClients[client1Key] = map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]{hashedParams: client1Sender}
	manager.connectedClients[client2Key] = map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]{hashedParams: client2Sender}
	manager.connectedClients[client3Key] = map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]{hashedParams: client3Sender}
	manager.lock.Unlock()

	// Verify initial state
	assert.Equal(t, 3, len(activeSub.connectedClients), "Should start with 3 clients")

	// CLIENT 2 UNSUBSCRIBES
	// This simulates: eth_unsubscribe(client2RouterID)
	protocolMessage := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{client2RouterID},
	}
	err := manager.Unsubscribe(ctx, protocolMessage, "dapp2", "10.0.0.2", "ws-2", nil)
	assert.NoError(t, err, "Client 2 unsubscribe should succeed")

	// Verify client 2 was removed but clients 1 and 3 remain
	manager.lock.RLock()
	assert.Equal(t, 2, len(activeSub.connectedClients),
		"Should have 2 clients after Client 2 unsubscribed")
	_, client1Exists := activeSub.connectedClients[client1Key]
	_, client2Exists := activeSub.connectedClients[client2Key]
	_, client3Exists := activeSub.connectedClients[client3Key]
	manager.lock.RUnlock()

	assert.True(t, client1Exists, "Client 1 should still be connected")
	assert.False(t, client2Exists, "Client 2 should be disconnected")
	assert.True(t, client3Exists, "Client 3 should still be connected")

	// Verify ID mapper state
	_, found1 := manager.idMapper.GetUpstreamID(client1RouterID)
	_, found2 := manager.idMapper.GetUpstreamID(client2RouterID)
	_, found3 := manager.idMapper.GetUpstreamID(client3RouterID)

	assert.True(t, found1, "Client 1's router ID should still be in mapper")
	assert.False(t, found2, "Client 2's router ID should be removed from mapper")
	assert.True(t, found3, "Client 3's router ID should still be in mapper")

	// Verify the subscription is still active (not torn down)
	manager.lock.RLock()
	_, subExists := manager.activeSubscriptions[hashedParams]
	manager.lock.RUnlock()
	assert.True(t, subExists, "Subscription should still be active after one client unsubscribed")

	// CLIENT 3 UNSUBSCRIBES
	protocolMessage3 := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{client3RouterID},
	}
	err = manager.Unsubscribe(ctx, protocolMessage3, "dapp3", "10.0.0.3", "ws-3", nil)
	assert.NoError(t, err, "Client 3 unsubscribe should succeed")

	// Verify only client 1 remains
	manager.lock.RLock()
	assert.Equal(t, 1, len(activeSub.connectedClients),
		"Should have 1 client after Client 3 unsubscribed")
	manager.lock.RUnlock()

	// Subscription should STILL be active (client 1 is still connected)
	manager.lock.RLock()
	_, subExists = manager.activeSubscriptions[hashedParams]
	manager.lock.RUnlock()
	assert.True(t, subExists, "Subscription should still be active with 1 client")
}

// TestRouteMessageToClientsPerClientID verifies that when routing messages to clients,
// each client receives the message with their own unique subscription ID.
func TestRouteMessageToClientsPerClientID(t *testing.T) {
	metricsManager := getTestMetricsManager()
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hashedParams := "route-test-hash"
	upstreamID := "0xupstreamRoute"

	// Set up 2 clients
	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "10.0.0.1", "ws-1")
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp2", "10.0.0.2", "ws-2")

	client1RouterID := "rs_client1_00001"
	client2RouterID := "rs_client2_00001"

	// Create buffered channels to receive routed messages
	client1Chan := make(chan *pairingtypes.RelayReply, 10)
	client2Chan := make(chan *pairingtypes.RelayReply, 10)

	client1Sender := common.NewSafeChannelSender(ctx, client1Chan)
	client2Sender := common.NewSafeChannelSender(ctx, client2Chan)

	// Create active subscription
	activeSub := &directActiveSubscription{
		upstreamID:       upstreamID,
		clientRouterIDs:  make(map[string]string),
		connectedClients: make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		hashedParams:     hashedParams,
		ctx:              ctx,
		cancel:           cancel,
	}
	activeSub.connectedClients[client1Key] = client1Sender
	activeSub.connectedClients[client2Key] = client2Sender
	activeSub.clientRouterIDs[client1Key] = client1RouterID
	activeSub.clientRouterIDs[client2Key] = client2RouterID

	// Create an upstream message (eth_subscription notification)
	upstreamMsg := &rpcclient.JsonrpcMessage{
		Method: "eth_subscription",
		Params: json.RawMessage(`{"subscription":"0xupstreamRoute","result":{"blockNumber":"0x123"}}`),
	}

	// Route the message
	manager.routeMessageToClients(hashedParams, activeSub, upstreamMsg)

	// Give a moment for async sends
	time.Sleep(50 * time.Millisecond)

	// Verify client 1 received message with their router ID
	select {
	case reply1 := <-client1Chan:
		var msg1 map[string]interface{}
		err := json.Unmarshal(reply1.Data, &msg1)
		require.NoError(t, err)
		params1 := msg1["params"].(map[string]interface{})
		assert.Equal(t, client1RouterID, params1["subscription"],
			"Client 1 should receive message with their router ID")
	default:
		t.Error("Client 1 should have received a message")
	}

	// Verify client 2 received message with their router ID
	select {
	case reply2 := <-client2Chan:
		var msg2 map[string]interface{}
		err := json.Unmarshal(reply2.Data, &msg2)
		require.NoError(t, err)
		params2 := msg2["params"].(map[string]interface{})
		assert.Equal(t, client2RouterID, params2["subscription"],
			"Client 2 should receive message with their router ID")
	default:
		t.Error("Client 2 should have received a message")
	}
}

// TestReconnectionUpdatesConnectionBookkeeping verifies that after an upstream
// reconnection, the connection bookkeeping is properly updated.
func TestReconnectionUpdatesConnectionBookkeeping(t *testing.T) {
	metricsManager := getTestMetricsManager()
	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}

	config := DefaultWebsocketConfig()
	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		config,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hashedParams := "reconnect-test-hash"
	oldUpstreamID := "0xoldUpstream"
	newUpstreamID := "0xnewUpstream"

	// Set up clients
	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "10.0.0.1", "ws-1")
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp2", "10.0.0.2", "ws-2")

	client1RouterID := manager.idMapper.GenerateRouterID(client1Key)
	client2RouterID := manager.idMapper.GenerateRouterID(client2Key)

	// Register initial mappings
	manager.idMapper.RegisterMapping(client1RouterID, oldUpstreamID)
	manager.idMapper.RegisterMapping(client2RouterID, oldUpstreamID)

	// Create mock connections
	oldConn := &UpstreamWSConnection{}
	newConn := &UpstreamWSConnection{}

	// Create channels
	client1Chan := make(chan *pairingtypes.RelayReply, 10)
	client2Chan := make(chan *pairingtypes.RelayReply, 10)

	client1Sender := common.NewSafeChannelSender(ctx, client1Chan)
	client2Sender := common.NewSafeChannelSender(ctx, client2Chan)

	// Create active subscription with old connection
	activeSub := &directActiveSubscription{
		upstreamID:         oldUpstreamID,
		upstreamConnection: oldConn,
		clientRouterIDs:    make(map[string]string),
		connectedClients:   make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		hashedParams:       hashedParams,
		ctx:                ctx,
		cancel:             cancel,
	}
	activeSub.connectedClients[client1Key] = client1Sender
	activeSub.connectedClients[client2Key] = client2Sender
	activeSub.clientRouterIDs[client1Key] = client1RouterID
	activeSub.clientRouterIDs[client2Key] = client2RouterID

	// Verify initial state
	assert.Equal(t, oldConn, activeSub.upstreamConnection, "Should start with old connection")

	// Simulate what handleUpstreamDisconnect does after getting a new connection:
	// 1. Increment new connection's subscription count
	newConn.IncrementSubscriptions()

	// 2. Update the active subscription
	manager.lock.Lock()
	oldConnection := activeSub.upstreamConnection
	activeSub.upstreamID = newUpstreamID
	activeSub.upstreamConnection = newConn
	manager.lock.Unlock()

	// 3. Update ID mappings
	manager.idMapper.RemoveAllForUpstream(oldUpstreamID)
	manager.idMapper.RegisterMapping(client1RouterID, newUpstreamID)
	manager.idMapper.RegisterMapping(client2RouterID, newUpstreamID)

	// Verify connection was updated
	assert.Equal(t, newConn, activeSub.upstreamConnection, "Should have new connection")
	assert.NotEqual(t, oldConnection, activeSub.upstreamConnection, "Connection should have changed")

	// Verify new connection has subscription count
	assert.Equal(t, int32(1), newConn.subscriptionCount.Load(),
		"New connection should have subscription count incremented")

	// Verify ID mappings were updated
	upstream1, found1 := manager.idMapper.GetUpstreamID(client1RouterID)
	upstream2, found2 := manager.idMapper.GetUpstreamID(client2RouterID)

	assert.True(t, found1, "Client 1's router ID should be in mapper")
	assert.True(t, found2, "Client 2's router ID should be in mapper")
	assert.Equal(t, newUpstreamID, upstream1, "Client 1 should map to NEW upstream")
	assert.Equal(t, newUpstreamID, upstream2, "Client 2 should map to NEW upstream")

	// Verify old upstream has no mappings
	oldRouterIDs := manager.idMapper.GetRouterIDs(oldUpstreamID)
	assert.Empty(t, oldRouterIDs, "Old upstream should have no router ID mappings")

	// Verify new upstream has both mappings
	newRouterIDs := manager.idMapper.GetRouterIDs(newUpstreamID)
	assert.Len(t, newRouterIDs, 2, "New upstream should have 2 router ID mappings")
	assert.Contains(t, newRouterIDs, client1RouterID)
	assert.Contains(t, newRouterIDs, client2RouterID)
}

// TestUnsubscribeRouterIDOwnershipValidation verifies that clients can only unsubscribe
// using their own router IDs and not IDs belonging to other clients.
// This is a critical security test to prevent one client from disrupting another's subscriptions.
func TestUnsubscribeRouterIDOwnershipValidation(t *testing.T) {
	// Create a config with high limits to avoid rate limiting
	config := &WebsocketConfig{
		MaxSubscriptionsPerClient:       25,
		PerClientLimitEnforcement:       "warn",
		MaxTotalSubscriptions:           5000,
		TotalLimitEnforcement:           "warn",
		SubscriptionSharingEnabled:      true,
		SubscriptionsPerMinutePerClient: 60, // High limit to avoid rate limiting
		UnsubscribesPerMinutePerClient:  60,
		MaxMessageSize:                  1048576,
		CleanupInterval:                 time.Minute,
	}

	nodeUrl := &common.NodeUrl{Url: "wss://test.example.com"}
	wsEndpoints := []*common.NodeUrl{nodeUrl}
	metricsManager := getTestMetricsManager()

	manager := NewDirectWSSubscriptionManager(
		metricsManager,
		"jsonrpc",
		"ETH",
		"jsonrpc",
		wsEndpoints,
		nil,
		config,
	)

	// Create two different clients
	client1Key := manager.CreateWebSocketConnectionUniqueKey("dapp1", "192.168.1.1", "ws-1")
	client2Key := manager.CreateWebSocketConnectionUniqueKey("dapp2", "192.168.1.2", "ws-2")

	// Generate router IDs for both clients
	client1RouterID := manager.idMapper.GenerateRouterID(client1Key)
	client2RouterID := manager.idMapper.GenerateRouterID(client2Key)
	upstreamID := "0x12345"

	// Register mappings for both clients to the same upstream
	manager.idMapper.RegisterMapping(client1RouterID, upstreamID)
	manager.idMapper.RegisterMapping(client2RouterID, upstreamID)

	// Create an active subscription with both clients
	subCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client1ReplyChan := make(chan *pairingtypes.RelayReply, 10)
	client2ReplyChan := make(chan *pairingtypes.RelayReply, 10)
	client1Sender := common.NewSafeChannelSender(subCtx, client1ReplyChan)
	client2Sender := common.NewSafeChannelSender(subCtx, client2ReplyChan)

	hashedParams := "test_params_hash"
	activeSub := &directActiveSubscription{
		connectedClients: map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]{
			client1Key: client1Sender,
			client2Key: client2Sender,
		},
		clientRouterIDs: map[string]string{
			client1Key: client1RouterID,
			client2Key: client2RouterID,
		},
		hashedParams: hashedParams,
	}

	manager.lock.Lock()
	manager.activeSubscriptions[hashedParams] = activeSub
	manager.lock.Unlock()

	ctx := context.Background()

	// Test 1: Client 2 tries to unsubscribe using Client 1's router ID (should fail)
	// Create a protocol message with client 1's router ID
	protocolMessage1 := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{client1RouterID},
	}

	// Client 2 attempts to unsubscribe client 1's subscription (should be rejected)
	err := manager.Unsubscribe(ctx, protocolMessage1, "dapp2", "192.168.1.2", "ws-2", nil)
	assert.Equal(t, common.SubscriptionNotFoundError, err,
		"Client 2 should NOT be able to unsubscribe using Client 1's router ID")

	// Verify client 1's subscription is still intact
	manager.lock.RLock()
	_, client1StillConnected := activeSub.connectedClients[client1Key]
	client1RouterStillExists := activeSub.clientRouterIDs[client1Key] == client1RouterID
	manager.lock.RUnlock()

	assert.True(t, client1StillConnected, "Client 1 should still be connected")
	assert.True(t, client1RouterStillExists, "Client 1's router ID should still exist")

	// Verify ID mapper still has client 1's mapping
	_, found := manager.idMapper.GetUpstreamID(client1RouterID)
	assert.True(t, found, "Client 1's ID mapping should still exist")

	// Test 2: Client 1 tries to unsubscribe using their own router ID (should succeed)
	protocolMessage2 := &mockWSProtocolMessage{
		method: "eth_unsubscribe",
		params: []interface{}{client1RouterID},
	}

	err = manager.Unsubscribe(ctx, protocolMessage2, "dapp1", "192.168.1.1", "ws-1", nil)
	assert.NoError(t, err, "Client 1 should be able to unsubscribe using their own router ID")

	// Verify client 1 is now disconnected
	manager.lock.RLock()
	_, client1StillConnected = activeSub.connectedClients[client1Key]
	_, client1RouterStillExists = activeSub.clientRouterIDs[client1Key]
	manager.lock.RUnlock()

	assert.False(t, client1StillConnected, "Client 1 should be disconnected after unsubscribe")
	assert.False(t, client1RouterStillExists, "Client 1's router ID should be removed")

	// Verify client 2 is still connected (independent unsubscribe)
	manager.lock.RLock()
	_, client2StillConnected := activeSub.connectedClients[client2Key]
	client2RouterStillExists := activeSub.clientRouterIDs[client2Key] == client2RouterID
	manager.lock.RUnlock()

	assert.True(t, client2StillConnected, "Client 2 should still be connected")
	assert.True(t, client2RouterStillExists, "Client 2's router ID should still exist")
}
