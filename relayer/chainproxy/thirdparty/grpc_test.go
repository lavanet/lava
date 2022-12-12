package thirdparty

import (
	"context"
	"log"
	"testing"

	pb_pkg "github.com/CosmosContracts/juno/x/mint/types"
	pairing "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Test for Lava grpc client.
var lavaGRPC = "0.0.0.0:3352"
var osmosisGRPC = "0.0.0.0:3353"
var JunGRPC = "0.0.0.0:3355"

func TestLavaGRPC(t *testing.T) {
	ctx := context.Background()

	req := &pairing.QueryClientsRequest{ChainID: "LAV1"}
	resp := &pairing.QueryClientsResponse{}

	conn, err := grpc.DialContext(ctx, lavaGRPC, grpc.WithInsecure(), grpc.WithBlock())
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

func TestJunGRPC(t *testing.T) {
	ctx := context.Background()

	req := &pb_pkg.QueryParamsRequest{}
	resp := &pb_pkg.QueryParamsResponse{}

	conn, err := grpc.DialContext(ctx, JunGRPC, grpc.WithInsecure(), grpc.WithBlock())
	require.Nil(t, err)

	defer conn.Close()
	err = grpc.Invoke(ctx, "juno.mint.Query/Params", req, resp, conn)
	require.Nil(t, err)
	log.Println("response:", resp)

}

// func TestOsmosisGRPC(t *testing.T) { WIP
// 	ctx := context.Background()

// 	req := &pairing.QueryClientsRequest{}
// 	resp := &pairing.QueryClientsResponse{}

// 	conn, err := grpc.DialContext(ctx, osmosisGRPC, grpc.WithInsecure(), grpc.WithBlock())
// 	require.Nil(t, err)

// 	defer conn.Close()
// 	err = grpc.Invoke(ctx, "osmosis.gamm.v1beta1.Query/NumPools", req, resp, conn)
// 	require.Nil(t, err)
// 	log.Println("response:", resp)

// }
