package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	"github.com/lavanet/lava/v4/x/pairing/types"
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
		metadata, err := k.epochStorageKeeper.GetMetadata(ctx, stakeEntries[i].Address)
		if err != nil {
			return nil, err
		}
		stakeEntries[i].DelegateCommission = metadata.DelegateCommission
		stakeEntries[i].Description = metadata.Description
		stakeEntries[i].Moniker = metadata.Description.Moniker
		stakeEntries[i].Vault = metadata.Vault

		// show providers with valid stakeAppliedBlock (frozen providers have stakeAppliedBlock = MaxUint64)
		if !req.ShowFrozen && stakeEntries[i].GetStakeAppliedBlock() <= uint64(ctx.BlockHeight()) {
			stakeEntriesNoFrozen = append(stakeEntriesNoFrozen, stakeEntries[i])
		}
	}

	if !req.ShowFrozen {
		stakeEntries = stakeEntriesNoFrozen
	}

	return &types.QueryProvidersResponse{StakeEntry: stakeEntries}, nil
}
