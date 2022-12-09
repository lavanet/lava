package thirdparty

import (
	"context"
	"log"
	"testing"

	pairing "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Test for Lava grpc client.
var lavaGRPC = "0.0.0.0:3342"

func Test_gRPC(t *testing.T) {
	ctx := context.Background()

	req := &pairing.QueryClientsRequest{ChainID: "LAV1"}
	resp := &pairing.QueryClientsResponse{}

	conn, err := grpc.DialContext(ctx, lavaGRPC, grpc.WithInsecure(), grpc.WithBlock()) // TODO, keep an open connection similar to others
	require.Nil(t, err)

	defer conn.Close()
	err = grpc.Invoke(ctx, "lavanet.lava.pairing.Query/Clients", req, resp, conn)
	require.Nil(t, err)
	log.Println("response:", resp)

	req1 := &pairing.QueryProvidersRequest{ChainID: "LAV1"}
	resp1 := &pairing.QueryProvidersResponse{}
	err = grpc.Invoke(ctx, "lavanet.lava.pairing.Query/Providers", req1, resp1, conn)
	require.Nil(t, err)
	log.Println("response:", resp1)
}
