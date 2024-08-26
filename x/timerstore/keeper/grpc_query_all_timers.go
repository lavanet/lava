package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) AllTimers(goCtx context.Context, req *types.QueryAllTimersRequest) (*types.QueryAllTimersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, store := range k.timerStoresBegin {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			return &types.QueryAllTimersResponse{
				BlockTimeTimers:   store.DumpAllTimers(ctx, types.BlockTime),
				BlockHeightTimers: store.DumpAllTimers(ctx, types.BlockHeight),
				Tick:              types.BeginBlock,
			}, nil
		}
	}

	for _, store := range k.timerStoresEnd {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			return &types.QueryAllTimersResponse{
				BlockTimeTimers:   store.DumpAllTimers(ctx, types.BlockTime),
				BlockHeightTimers: store.DumpAllTimers(ctx, types.BlockHeight),
				Tick:              types.EndBlock,
			}, nil
		}
	}

	return nil, status.Error(codes.InvalidArgument, "the provided combination of store key and prefix was not found")
}
