package rpcsmartrouter

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRESTRelay_GET_PathParameters(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/cosmos/base/tendermint/v1beta1/blocks/17", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"block":{"header":{"height":"17","chain_id":"cosmoshub-4"}}}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/base/tendermint/v1beta1/blocks/17",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Reply)

	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "cosmoshub-4")
	assert.False(t, result.IsNodeError)
}

func TestRESTRelay_GET_QueryParameters(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/cosmos/tx/v1beta1/txs", r.URL.Path)
			q := r.URL.Query()
			assert.Equal(t, "cosmos1...", q.Get("sender"))
			assert.Equal(t, "10", q.Get("limit"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"txs":[],"pagination":{"total":"0"}}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/tx/v1beta1/txs?sender=cosmos1...&limit=10",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "pagination")
}

func TestRESTRelay_POST_JSONBody(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/cosmos/tx/v1beta1/simulate", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.NotEqual(t, int64(0), r.ContentLength)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"gas_info":{"gas_used":"12345"}}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/tx/v1beta1/simulate",
		[]byte(`{"tx_bytes":"base64encodedtx"}`),
		http.MethodPost,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Contains(t, string(result.Reply.Data), "gas_used")
}

func TestRESTRelay_404_NotFound(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"block not found"}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/base/tendermint/v1beta1/blocks/999999999",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.False(t, result.IsNodeError)
	assert.Contains(t, string(result.Reply.Data), "block not found")
}

func TestRESTRelay_429_RateLimit(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/base/tendermint/v1beta1/blocks/latest",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, http.StatusTooManyRequests, result.StatusCode)
	assert.False(t, result.IsNodeError)
}

func TestRESTRelay_503_ServiceUnavailable(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"service temporarily unavailable"}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/base/tendermint/v1beta1/blocks/latest",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-cosmos-lcd",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	assert.True(t, result.IsNodeError)
}

func TestRESTRelay_ResponseHeaders(t *testing.T) {
	ctx := context.Background()
	chainParser, _, _, closeServer, endpoint, err := chainlib.CreateChainLibMocks(
		ctx,
		"LAV1",
		spectypes.APIInterfaceRest,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "test-value")
			w.Header().Set("X-Block-Height", "12345")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
		}),
		nil,
		"../../",
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, endpoint)
	defer closeServer()

	chainMessage, err := chainParser.ParseMsg(
		"/cosmos/base/tendermint/v1beta1/blocks/latest",
		nil,
		http.MethodGet,
		nil,
		extensionslib.ExtensionInfo{LatestBlock: 0},
	)
	require.NoError(t, err)

	nodeUrl := endpoint.NodeUrls[0]
	directConn, err := lavasession.NewDirectRPCConnection(ctx, nodeUrl, 5)
	require.NoError(t, err)

	sender := &DirectRPCRelaySender{
		directConnection: directConn,
		endpointName:     "test-rest-endpoint",
	}

	result, err := sender.SendDirectRelay(ctx, chainMessage, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Reply)

	found := false
	for _, md := range result.Reply.Metadata {
		if md.Name == "X-Custom-Header" && md.Value == "test-value" {
			found = true
			break
		}
	}
	assert.True(t, found, fmt.Sprintf("expected X-Custom-Header in metadata, got: %+v", result.Reply.Metadata))
}

