package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Provider(goCtx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	chains := []string{req.ChainID}
	if req.ChainID == "" {
		chains = k.specKeeper.GetAllChainIDs(ctx)
	}

	stakeEntries := []epochstoragetypes.StakeEntry{}
	for _, chain := range chains {
		stakeEntry, found := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chain, req.Address)
		if !found {
			continue
		}
		stakeEntries = append(stakeEntries, stakeEntry)
	}

	return &types.QueryProviderResponse{StakeEntries: stakeEntries}, nil
}
