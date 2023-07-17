package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
)

var _ v1.QueryServer = queryServer{}

type queryServer struct {
	k Keeper
}

func (q queryServer) QueryDowntime(ctx context.Context, request *v1.QueryDowntimeRequest) (*v1.QueryDowntimeResponse, error) {
	resp := new(v1.QueryDowntimeResponse)
	resp.CumulativeDowntimeDuration = 0 * time.Second

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
