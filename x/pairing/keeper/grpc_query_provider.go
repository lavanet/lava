package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
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

	stakeEntry, found := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, req.ChainID, req.Address)
	if !found {
		return nil, utils.LavaFormatWarning("provider not staked on chain", epochstoragetypes.ErrProviderNotStaked,
			utils.LogAttr("provider", req.Address),
			utils.LogAttr("chain_id", req.ChainID),
		)
	}

	return &types.QueryProviderResponse{StakeEntry: stakeEntry}, nil
}
