package grpcproxy

import (
	"context"
	"net"
	"testing"

	"github.com/lavanet/lava/grpcproxy/testproto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newTestGRPCServer(t *testing.T, grpcSrv *grpc.Server) *grpc.ClientConn {
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		defer lis.Close()
		if err := grpcSrv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	// returns the listening port
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	return conn
}

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

	client := testproto.NewTestClient(newTestGRPCServer(t, proxyGRPCSrv))
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
