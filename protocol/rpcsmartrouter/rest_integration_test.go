package rpcsmartrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRESTRelay_GET_PathParameters(t *testing.T) {
	// Mock Cosmos LCD server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method
		assert.Equal(t, "GET", r.Method)
		
		// Verify path
		assert.Equal(t, "/cosmos/base/tendermint/v1beta1/blocks/17", r.URL.Path)
		
		// Return mock Cosmos block response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"block":{"header":{"height":"17","chain_id":"cosmoshub-4"}}}`))
	}))
	defer mockServer.Close()

	// Create REST chain message for GET request
	chainMessage := createMockRESTChainMessage(t, 
		"/cosmos/base/tendermint/v1beta1/blocks/17", 
		"GET",
		nil,  // No body for GET
	)

	// Create direct RPC connection
	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	// Create DirectRPCRelaySender
	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	// Send REST relay
	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Reply)
	assert.NotNil(t, result.Reply.Data)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "cosmoshub-4")
	assert.False(t, result.IsNodeError)
}

func TestRESTRelay_GET_QueryParameters(t *testing.T) {
	// Mock server that expects query parameters
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/cosmos/tx/v1beta1/txs", r.URL.Path)
		
		// Verify query parameters preserved
		query := r.URL.Query()
		assert.Equal(t, "cosmos1...", query.Get("sender"))
		assert.Equal(t, "10", query.Get("limit"))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"txs":[],"pagination":{"total":"0"}}`))
	}))
	defer mockServer.Close()

	// Create REST message with query parameters
	chainMessage := createMockRESTChainMessage(t,
		"/cosmos/tx/v1beta1/txs?sender=cosmos1...&limit=10",
		"GET",
		nil,
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "pagination")
}

func TestRESTRelay_POST_JSONBody(t *testing.T) {
	// Mock server expecting POST with JSON body
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/cosmos/tx/v1beta1/simulate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		// Verify body is present
		assert.NotEqual(t, 0, r.ContentLength)
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"gas_info":{"gas_used":"12345"}}`))
	}))
	defer mockServer.Close()

	// Create REST message with POST body
	chainMessage := createMockRESTChainMessage(t,
		"/cosmos/tx/v1beta1/simulate",
		"POST",
		[]byte(`{"tx_bytes":"base64encodedtx"}`),
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "gas_used")
}

func TestRESTRelay_404_NotFound(t *testing.T) {
	// Mock server returning 404
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":5,"message":"block not found"}`))
	}))
	defer mockServer.Close()

	chainMessage := createMockRESTChainMessage(t,
		"/cosmos/base/tendermint/v1beta1/blocks/999999999",
		"GET",
		nil,
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	// Should NOT return error (REST returns result with status code)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.False(t, result.IsNodeError)  // 404 is client error, not node error
	assert.Contains(t, string(result.Reply.Data), "block not found")
}

func TestRESTRelay_429_RateLimit(t *testing.T) {
	// Mock server returning 429 Rate Limit
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limit exceeded"}`))
	}))
	defer mockServer.Close()

	chainMessage := createMockRESTChainMessage(t,
		"/cosmos/base/tendermint/v1beta1/blocks/latest",
		"GET",
		nil,
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, result.StatusCode)
	assert.False(t, result.IsNodeError)  // 429 is not a node error (endpoint is healthy, just busy)
	assert.Contains(t, string(result.Reply.Data), "rate limit")
}

func TestRESTRelay_503_ServiceUnavailable(t *testing.T) {
	// Mock server returning 503
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"service temporarily unavailable"}`))
	}))
	defer mockServer.Close()

	chainMessage := createMockRESTChainMessage(t,
		"/cosmos/base/tendermint/v1beta1/blocks/latest",
		"GET",
		nil,
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	assert.True(t, result.IsNodeError)  // 503 is a node/server error
	assert.Contains(t, string(result.Reply.Data), "unavailable")
}

func TestRESTRelay_ResponseHeaders(t *testing.T) {
	// Mock server with custom headers
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.Header().Set("X-Block-Height", "12345")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"success"}`))
	}))
	defer mockServer.Close()

	chainMessage := createMockRESTChainMessage(t,
		"/test/endpoint",
		"GET",
		nil,
	)

	ctx := context.Background()
	nodeUrl := common.NodeUrl{Url: mockServer.URL}
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-rest-endpoint",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	
	// Verify headers are captured
	assert.NotNil(t, result.Reply.Metadata)
	// Note: Headers are filtered by chainParser.HandleHeaders in production
}

// Helper function to create mock REST chain message
func createMockRESTChainMessage(t *testing.T, path, method string, body []byte) chainlib.ChainMessage {
	// This is a simplified mock - in production, RestChainParser creates the real message
	// For now, return a mock that implements the required interfaces
	return &mockRESTChainMessage{
		path:   path,
		method: method,
		body:   body,
	}
}

// Mock REST chain message for testing
type mockRESTChainMessage struct {
	path   string
	method string
	body   []byte
}

func (m *mockRESTChainMessage) GetRPCMessage() chainlib.RPCInput {
	return &mockRESTMessage{
		path: m.path,
		body: m.body,
	}
}

func (m *mockRESTChainMessage) GetApiCollection() *chainlib.ApiCollection {
	return &chainlib.ApiCollection{
		CollectionData: chainlib.CollectionData{
			ApiInterface: "rest",
			Type:         m.method,
		},
	}
}

func (m *mockRESTChainMessage) GetApi() *chainlib.Api {
	return &chainlib.Api{Name: m.path}
}

func (m *mockRESTChainMessage) RequestedBlock() (int64, error) {
	return -2, nil // LATEST_BLOCK
}

func (m *mockRESTChainMessage) CheckResponseError(data []byte, statusCode int) (bool, string) {
	return statusCode >= 400, ""
}

// Implement other required chainlib.ChainMessage methods (minimal for testing)
func (m *mockRESTChainMessage) GetExtensions() []*chainlib.Extension { return nil }
func (m *mockRESTChainMessage) GetRequestedBlocksHashes() []string    { return nil }
func (m *mockRESTChainMessage) GetDirectiveHeaders() map[string]string { return nil }
func (m *mockRESTChainMessage) UpdateLatestBlockInMessage(int64, bool) {}

// Mock REST RPC message
type mockRESTMessage struct {
	path string
	body []byte
}

func (m *mockRESTMessage) GetPath() string {
	return m.path
}

func (m *mockRESTMessage) GetMsg() []byte {
	return m.body
}

func (m *mockRESTMessage) GetHeaders() []chainlib.Metadata {
	return []chainlib.Metadata{}
}

func (m *mockRESTMessage) GetParams() interface{} {
	return nil
}

// Implement other required RPCInput methods
func (m *mockRESTMessage) ParseBlock(inp string) (int64, error) { return 0, nil }
