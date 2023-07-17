package keeper

import (
	"context"

	v1 "github.com/lavanet/lava/x/downtime/v1"
)

var _ v1.QueryServer = queryServer{}

type queryServer struct {
	k Keeper
}

func (q queryServer) QueryDowntime(ctx context.Context, request *v1.QueryDowntimeRequest) (*v1.QueryDowntimeResponse, error) {
	// TODO implement me
	panic("implement me")
}

func NewQueryServer(k Keeper) v1.QueryServer {
	return &queryServer{k: k}
}
