package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
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
	dt, _ := q.k.GetDowntime(sdk.UnwrapSDKContext(ctx), request.EpochStartBlock)
	return &v1.QueryDowntimeResponse{CumulativeDowntimeDuration: dt}, nil
}

func NewQueryServer(k Keeper) v1.QueryServer {
	return &queryServer{k: k}
}
