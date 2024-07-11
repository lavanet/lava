package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Providers(goCtx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	stakeEntries := k.epochStorageKeeper.GetAllStakeEntriesCurrentForChainId(ctx, req.ChainID)

	stakeEntriesNoFrozen := []epochstoragetypes.StakeEntry{}
	for i := range stakeEntries {
		stakeEntries[i].Moniker = stakeEntries[i].Description.Moniker

		// show providers with valid stakeAppliedBlock (frozen providers have stakeAppliedBlock = MaxUint64)
		if !req.ShowFrozen && stakeEntries[i].GetStakeAppliedBlock() <= uint64(ctx.BlockHeight()) {
			stakeEntriesNoFrozen = append(stakeEntriesNoFrozen, stakeEntries[i])
		}
	}

	if len(stakeEntriesNoFrozen) != 0 {
		stakeEntries = stakeEntriesNoFrozen
	}

	return &types.QueryProvidersResponse{StakeEntry: stakeEntries}, nil
}
