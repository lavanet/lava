package grpcproxy

import (
	"context"
	"testing"

	"github.com/lavanet/lava/grpcproxy/testproto"
	"github.com/stretchr/testify/require"
)

func TestGRPCProxy(t *testing.T) {
	proxyGRPCSrv, _, err := NewGRPCProxy(func(ctx context.Context, method string, reqBody []byte) ([]byte, error) {
		// the callback function just does echo proxying
		req := new(testproto.TestRequest)
		err := req.Unmarshal(reqBody)
		require.NoError(t, err)
		respBytes, err := (&testproto.TestResponse{Response: req.Request + "-callback"}).Marshal()
		require.NoError(t, err)
		return respBytes, nil
	})
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
