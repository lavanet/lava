package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Providers(goCtx context.Context, req *types.QueryProvidersRequest) (*types.QueryProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	stakeStorage, found := k.epochStorageKeeper.GetStakeStorageCurrent(ctx, req.ChainID)
	if !found {
		stakeStorage = epochstoragetypes.StakeStorage{}
	}

	stakeEntries := stakeStorage.GetStakeEntries()
	for i := range stakeEntries {
		stakeEntries[i].Moniker = stakeEntries[i].Description.Moniker
	}

	if !req.ShowFrozen {
		stakeEntriesNoFrozen := []epochstoragetypes.StakeEntry{}
		for _, stakeEntry := range stakeEntries {
			// show providers with valid stakeAppliedBlock (frozen providers have stakeAppliedBlock = MaxUint64)
			if stakeEntry.GetStakeAppliedBlock() <= uint64(ctx.BlockHeight()) {
				stakeEntriesNoFrozen = append(stakeEntriesNoFrozen, stakeEntry)
			}
		}
		stakeEntries = stakeEntriesNoFrozen
	}

	return &types.QueryProvidersResponse{StakeEntry: stakeEntries}, nil
}
