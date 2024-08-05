package grpcproxy

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v2/protocol/chainlib/grpcproxy/testproto"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestGRPCProxy(t *testing.T) {
	proxyGRPCSrv, _, err := NewGRPCProxy(func(ctx context.Context, method string, reqBody []byte) ([]byte, metadata.MD, error) {
		// the callback function just does echo proxying
		req := new(testproto.TestRequest)
		err := req.Unmarshal(reqBody)
		require.NoError(t, err)
		respBytes, err := (&testproto.TestResponse{Response: req.Request + "-callback"}).Marshal()
		require.NoError(t, err)
		responseHeaders := make(metadata.MD)
		responseHeaders["test-headers"] = append(responseHeaders["test-headers"], "55")
		return respBytes, responseHeaders, nil
	}, "", common.ConsumerCmdFlags{HeadersFlag: "*", OriginFlag: "*", MethodsFlag: "GET,POST,OPTIONS", CDNCacheDuration: "86400"}, nil)
	require.NoError(t, err)

	client := testproto.NewTestClient(testproto.InMemoryClientConn(t, proxyGRPCSrv))
	ctx := context.Background()

	do := func() {
		req := &testproto.TestRequest{Request: "echo"}
		resp, err := client.Test(ctx, req)
		require.NoError(t, err)
		require.Equal(t, req.Request+"-callback", resp.Response)
	}

	do()
	do()
}
