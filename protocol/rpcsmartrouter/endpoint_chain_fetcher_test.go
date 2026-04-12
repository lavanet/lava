package rpcsmartrouter

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/stretchr/testify/require"
)

// TestEndpointChainFetcher_CustomMessage_POSTDelegatesToConnection verifies the
// Solana path: SVMChainTracker calls CustomMessage with the getLatestBlockhash
// JSON-RPC body. The previous implementation returned a hard error, so on every
// Solana-family chain the per-endpoint ChainTracker silently failed to start —
// no OnNewBlock callback, no per-endpoint metrics, backup rows stuck at N/A.
// This test asserts that CustomMessage now delegates to the direct RPC connection
// and returns the real response payload.
func TestEndpointChainFetcher_CustomMessage_POSTDelegatesToConnection(t *testing.T) {
	const (
		url        = "https://solana.lava.build:443/"
		svmRequest = `{"jsonrpc":"2.0","id":1,"method":"getLatestBlockhash","params":[{"commitment":"finalized"}]}`
		svmResp    = `{"jsonrpc":"2.0","id":1,"result":{"context":{"slot":100},"value":{"blockhash":"abc","lastValidBlockHeight":42}}}`
	)

	conn := &mockDirectRPCConnection{
		url:     url,
		healthy: true,
		responses: map[string][]byte{
			svmRequest: []byte(svmResp),
		},
	}
	fetcher := NewEndpointChainFetcher(
		&lavasession.Endpoint{NetworkAddress: url, Enabled: true},
		conn,
		nil, // chainParser unused by the POST path
		"SOLANA",
		"jsonrpc",
	)

	got, err := fetcher.CustomMessage(context.Background(), "", []byte(svmRequest), "POST", "getLatestBlockhash")
	require.NoError(t, err,
		"CustomMessage must not return a stub error — SVMChainTracker depends on it for getLatestBlockhash")
	require.Equal(t, svmResp, string(got),
		"CustomMessage must return the actual upstream response body")
}

// TestEndpointChainFetcher_CustomMessage_PropagatesUnhealthyConnection guards against
// silently swallowing upstream failures: when the direct connection is unhealthy,
// CustomMessage should surface an error so the SVM tracker's retry logic kicks in
// instead of treating an empty body as a successful fetch.
func TestEndpointChainFetcher_CustomMessage_PropagatesUnhealthyConnection(t *testing.T) {
	conn := &mockDirectRPCConnection{
		url:     "https://solana.lava.build:443/",
		healthy: false, // the whole point of the check
	}
	fetcher := NewEndpointChainFetcher(
		&lavasession.Endpoint{NetworkAddress: conn.url, Enabled: true},
		conn,
		nil,
		"SOLANA",
		"jsonrpc",
	)

	_, err := fetcher.CustomMessage(context.Background(), "", []byte(`{}`), "POST", "getLatestBlockhash")
	require.Error(t, err, "CustomMessage must fail when the direct connection is not healthy")
}
