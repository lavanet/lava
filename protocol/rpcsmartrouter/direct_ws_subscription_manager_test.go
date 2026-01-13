package rpcsmartrouter

import (
	"context"
	"encoding/json"
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
