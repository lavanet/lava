package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Provider(goCtx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	metadata, err := k.epochStorageKeeper.GetMetadata(ctx, req.Address)
	if err != nil {
		return &types.QueryProviderResponse{}, nil
	}

	chains := []string{req.ChainID}
	if req.ChainID == "" {
		chains = metadata.Chains
	}

	stakeEntries := []epochstoragetypes.StakeEntry{}
	for _, chain := range chains {
		stakeEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chain, req.Address)
		if !found {
			continue
		}
		stakeEntry.Description = metadata.Description
		stakeEntry.Moniker = metadata.Description.Moniker
		stakeEntry.DelegateCommission = metadata.DelegateCommission
		stakeEntry.Vault = metadata.Vault
		stakeEntries = append(stakeEntries, stakeEntry)
	}

	return &types.QueryProviderResponse{StakeEntries: stakeEntries}, nil
}
