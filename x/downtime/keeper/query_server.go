package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ v1.QueryServer = queryServer{}

type queryServer struct {
	k Keeper
}

func (q queryServer) QueryParams(ctx context.Context, request *v1.QueryParamsRequest) (*v1.QueryParamsResponse, error) {
	params := q.k.GetParams(sdk.UnwrapSDKContext(ctx))
	return &v1.QueryParamsResponse{Params: &params}, nil
}

func (q queryServer) QueryDowntime(ctx context.Context, request *v1.QueryDowntimeRequest) (*v1.QueryDowntimeResponse, error) {
	resp := new(v1.QueryDowntimeResponse)
	resp.CumulativeDowntimeDuration = 0 * time.Second

	if request.StartBlock > request.EndBlock {
		return nil, status.Error(codes.InvalidArgument, "start block must be less than or equal to end block")
	}

	q.k.IterateDowntimes(sdk.UnwrapSDKContext(ctx), request.StartBlock, request.EndBlock, func(height uint64, downtime time.Duration) (stop bool) {
		resp.Downtimes = append(resp.Downtimes, &v1.Downtime{
			Block:    height,
			Duration: downtime,
		})
		resp.CumulativeDowntimeDuration += downtime
		return false
	})

	return resp, nil
}

func NewQueryServer(k Keeper) v1.QueryServer {
	return &queryServer{k: k}
}
