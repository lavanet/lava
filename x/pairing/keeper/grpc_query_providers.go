package keeper

import (
	"context"
	"fmt"

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

	stakeStorage, found := k.epochStorageKeeper.GetStakeStorageCurrent(ctx, req.ChainID)
	if !found {
		stakeStorage = epochstoragetypes.StakeStorage{}
	}

	stakeEntries := stakeStorage.GetStakeEntries()

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

	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, req.ChainID)
	unstakingStakeStorage, found := k.epochStorageKeeper.GetStakeStorageUnstake(ctx)
	if !found {
		unstakingStakeStorage = epochstoragetypes.StakeStorage{}
	}
	outputStr := fmt.Sprintf("Staked Providers Query Output:\nChainID: %s Enabled: %t Current Block: %d\nStaked Providers:\n%v\nUnstaking Providers:\n%v\n--------------------------------------\n", req.ChainID, foundAndActive, ctx.BlockHeight(), stakeEntries, unstakingStakeStorage.StakeEntries)

	return &types.QueryProvidersResponse{StakeEntry: stakeEntries, Output: outputStr}, nil
}
