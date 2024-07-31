package keeper

import (
	"context"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/x/timerstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func createQueryNextResponse(nextBlockHeight, nextBlockTime uint64, tick string) *types.QueryNextResponse {
	blockTimeStr := ""
	if nextBlockTime != math.MaxUint64 {
		blockTimeStr = commontypes.ConvertUnixTimestampToString(nextBlockTime)
	}

	var blockHeight uint64 = 0
	if nextBlockHeight != math.MaxUint64 {
		blockHeight = nextBlockHeight
	}

	return &types.QueryNextResponse{
		NextBlockHeight: blockHeight,
		NextBlockTime:   blockTimeStr,
		Tick:            tick,
	}
}

func (k *Keeper) Next(goCtx context.Context, req *types.QueryNextRequest) (*types.QueryNextResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, store := range k.timerStoresBegin {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			return createQueryNextResponse(
				store.GetNextTimeoutBlockHeight(ctx),
				store.GetNextTimeoutBlockTime(ctx),
				types.BeginBlock,
			), nil
		}
	}

	for _, store := range k.timerStoresEnd {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			return createQueryNextResponse(
				store.GetNextTimeoutBlockHeight(ctx),
				store.GetNextTimeoutBlockTime(ctx),
				types.EndBlock,
			), nil
		}
	}

	return nil, status.Error(codes.InvalidArgument, "the provided combination of store key and prefix was not found")
}
