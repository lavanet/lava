package keeper

import v1 "github.com/lavanet/lava/x/downtime/v1"

var _ v1.QueryServer = Keeper{}

type queryServer struct {
	k Keeper
}

func NewQueryServer(k Keeper) v1.QueryServer {
	return &queryServer{k: k}
}
