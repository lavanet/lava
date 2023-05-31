package grpc

import (
	"context"
	_ "github.com/cosmos/cosmos-sdk/types/query"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/lavanet/lava/utils"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"time"
)

const DefaultRequestTimeout = 1 * time.Second

type GRPCFetcher struct {
	GrpcConn *grpc.ClientConn
}

func NewGRPCFetcher(grpcAddr string) (*GRPCFetcher, error) {
	ctx, _ := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	grpcConn, err := grpc.DialContext(
		ctx,
		grpcAddr,            // your gRPC server address.
		grpc.WithInsecure(), // the SDK doesn't support any transport security mechanism.
		grpc.WithBlock(),
	)

	if err != nil {
		return nil, err
	}

	return &GRPCFetcher{
		GrpcConn: grpcConn,
	}, nil
}

func (fetcher *GRPCFetcher) FetchPairings(chainId string, userId string) (*[]epochtypes.StakeEntry, uint64, error) {
	utils.LavaFormatInfo("Fetching pairings for chain",
		utils.Attribute{Key: "chainId", Value: chainId},
		utils.Attribute{Key: "userId", Value: userId})
	grpcClient := pairingtypes.NewQueryClient(fetcher.GrpcConn)
	var header metadata.MD
	ctx, _ := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	ctx = metadata.AppendToOutgoingContext(ctx)

	consumerResponse, err := grpcClient.GetPairing(
		ctx,
		&pairingtypes.QueryGetPairingRequest{
			ChainID: chainId,
			Client:  userId,
		},
		grpc.Header(&header),
		retry.WithCodes(codes.DeadlineExceeded),
		retry.WithBackoff(retry.BackoffLinear(100*time.Millisecond)),
		retry.WithMax(3))
	if err != nil {
		utils.LavaFormatError("Fetching providers for chain failed.",
			err,
			utils.Attribute{Key: "chainId", Value: chainId},
			utils.Attribute{Key: "userId", Value: userId})
		return nil, 0, err
	}

	return &consumerResponse.Providers, consumerResponse.CurrentEpoch, nil
}
