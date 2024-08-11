package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/cosmos/cosmos-sdk/types/query"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const DefaultRequestTimeout = 1 * time.Second

type GRPCFetcher struct {
	GrpcConn   *grpc.ClientConn
	CancelFunc context.CancelFunc
}

func NewGRPCFetcher(grpcAddr string) (*GRPCFetcher, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	grpcConn, err := grpc.DialContext(
		ctx,
		grpcAddr, // your gRPC server address.
		grpc.WithTransportCredentials(insecure.NewCredentials()), // the SDK doesn't support any transport security mechanism.
		grpc.WithBlock(),
	)
	if err != nil {
		cancelFunc()
		return nil, err
	}

	return &GRPCFetcher{
		GrpcConn:   grpcConn,
		CancelFunc: cancelFunc,
	}, nil
}

func (fetcher *GRPCFetcher) FetchPairings(chainId, userId string) (*pairingtypes.QueryGetPairingResponse, error) {
	defer fetcher.CancelFunc()
	utils.LavaFormatInfo("Fetching pairings for chain",
		utils.Attribute{Key: "chainId", Value: chainId},
		utils.Attribute{Key: "userId", Value: userId})
	grpcClient := pairingtypes.NewQueryClient(fetcher.GrpcConn)
	var header metadata.MD
	ctx, cancelFunc := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancelFunc()
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
		return nil, err
	}

	return consumerResponse, nil
}
