package rpcsmartrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectRPCRelaySender_SendDirectRelay(t *testing.T) {
	// Create mock JSON-RPC server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock JSON-RPC response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1234"}`))
	}))
	defer mockServer.Close()

	// Create direct RPC connection
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}

	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5, "")
	require.NoError(t, err)
	require.NotNil(t, directConn)

	// Create DirectRPCRelaySender with endpoint name
	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-endpoint",
	}

	// Create mock chain message
	chainMessage := createMockChainMessage(t, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)

	// Send relay
	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Reply)
	assert.NotNil(t, result.Reply.Data)
	assert.True(t, result.Finalized)
	assert.Equal(t, 200, result.StatusCode)
	// Provider address should be the sanitized endpoint name (not full URL with potential API keys)
	assert.Equal(t, "test-endpoint", result.ProviderInfo.ProviderAddress)

	// Verify response data
	assert.Contains(t, string(result.Reply.Data), "0x1234")
}

func TestDirectRPCRelaySender_SendDirectRelay_Timeout(t *testing.T) {
	// Create slow mock server that exceeds timeout
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Sleep longer than timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1234"}`))
	}))
	defer mockServer.Close()

	// Create direct RPC connection
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}

	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5, "")
	require.NoError(t, err)

	// Create sender
	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-timeout-endpoint",
	}

	// Create mock chain message
	chainMessage := createMockChainMessage(t, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)

	// Send relay with short timeout
	result, err := sender.SendDirectRelay(ctx, chainMessage, 100*time.Millisecond)

	// Should timeout
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timeout")
}

func TestDirectRPCRelaySender_SendDirectRelay_ServerError(t *testing.T) {
	// Create mock server that returns error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"service unavailable"}`))
	}))
	defer mockServer.Close()

	// Create direct RPC connection
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}

	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5, "")
	require.NoError(t, err)

	// Create sender
	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-error-endpoint",
	}

	// Create mock chain message
	chainMessage := createMockChainMessage(t, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)

	// Send relay
	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	// Should return error for 5xx status codes (server errors)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "service unavailable")
}

func TestRelayInner_DirectRPCBranching(t *testing.T) {
	// This test verifies that relayInner correctly branches to relayInnerDirect
	// when the session is a direct RPC session

	// Create mock JSON-RPC server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0xabc"}`))
	}))
	defer mockServer.Close()

	// Create direct RPC connection
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}

	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5, "")
	require.NoError(t, err)

	// Create parent ConsumerSessionsWithProvider with endpoint
	cswp := &lavasession.ConsumerSessionsWithProvider{
		PublicLavaAddress: "test-direct-endpoint",
		PairingEpoch:      100,
		Endpoints: []*lavasession.Endpoint{
			{
				NetworkAddress:    mockServer.URL,
				Enabled:           true,
				DirectConnections: []lavasession.DirectRPCConnection{directConn},
			},
		},
	}

	// Create DirectRPCSessionConnection (smart router session)
	session := &lavasession.SingleConsumerSession{
		Parent: cswp,
		Connection: &lavasession.DirectRPCSessionConnection{
			DirectConnection: directConn,
			EndpointAddress:  mockServer.URL,
		},
	}

	// Verify session is recognized as direct RPC
	assert.True(t, session.IsDirectRPC())

	// Verify GetDirectConnection works
	conn, ok := session.GetDirectConnection()
	assert.True(t, ok)
	assert.Equal(t, directConn, conn)

	// NOTE: Full relayInner test would require:
	// - Mock RPCSmartRouterServer with chainParser, metrics, etc.
	// - This is deferred to the end-to-end test (prod-7)
	// For now, we verify the session structure is correct for branching
}

// createMockChainMessage creates a mock ChainMessage for testing
// This is a simplified mock - real implementation would use chainlib.CreateChainLibMocks
func createMockChainMessage(t *testing.T, requestData string) chainlib.ChainMessage {
	t.Helper()

	// For now, return a basic mock that implements the minimal interface
	// In real integration tests, use chainlib.CreateChainLibMocks
	return &mockChainMessage{
		requestData: []byte(requestData),
	}
}

type mockChainMessage struct {
	requestData []byte
	api         *spectypes.Api
}

func (m *mockChainMessage) GetRPCMessage() rpcInterfaceMessages.GenericMessage {
	return &mockGenericMessage{data: m.requestData}
}

func (m *mockChainMessage) GetApi() *spectypes.Api {
	if m.api == nil {
		return &spectypes.Api{Name: "eth_blockNumber"}
	}
	return m.api
}

func (m *mockChainMessage) CheckResponseError(data []byte, httpStatusCode int) (bool, string) {
	return false, ""
}

func (m *mockChainMessage) GetApiCollection() *spectypes.ApiCollection {
	return &spectypes.ApiCollection{
		CollectionData: spectypes.CollectionData{
			ApiInterface: "jsonrpc",
		},
	}
}

// Implement remaining ChainMessage interface methods (stubs for testing)
func (m *mockChainMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string { return "" }
func (m *mockChainMessage) RequestedBlock() (latest int64, earliest int64)                 { return 0, 0 }
func (m *mockChainMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) bool {
	return false
}
func (m *mockChainMessage) AppendHeader(metadata []pairingtypes.Metadata) {}
func (m *mockChainMessage) GetExtensions() []*spectypes.Extension         { return nil }
func (m *mockChainMessage) OverrideExtensions(extensionNames []string, extensionParser *extensionslib.ExtensionParser) {
}
func (m *mockChainMessage) DisableErrorHandling()                               {}
func (m *mockChainMessage) TimeoutOverride(...time.Duration) time.Duration      { return 0 }
func (m *mockChainMessage) GetForceCacheRefresh() bool                          { return false }
func (m *mockChainMessage) SetForceCacheRefresh(force bool) bool                { return false }
func (m *mockChainMessage) GetRawRequestHash() ([]byte, error)                  { return m.requestData, nil }
func (m *mockChainMessage) GetRequestedBlocksHashes() []string                  { return nil }
func (m *mockChainMessage) UpdateEarliestInMessage(incomingEarliest int64) bool { return false }
func (m *mockChainMessage) SetExtension(extension *spectypes.Extension)         {}
func (m *mockChainMessage) GetUsedDefaultValue() bool                           { return false }
func (m *mockChainMessage) GetParseDirective() *spectypes.ParseDirective        { return nil }

type mockGenericMessage struct {
	data []byte
}

func (m *mockGenericMessage) GetHeaders() []pairingtypes.Metadata {
	return []pairingtypes.Metadata{}
}

func (m *mockGenericMessage) DisableErrorHandling() {}

func (m *mockGenericMessage) GetParams() interface{} {
	return nil
}
