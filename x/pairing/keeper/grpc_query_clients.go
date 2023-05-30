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

func (k Keeper) Clients(goCtx context.Context, req *types.QueryClientsRequest) (*types.QueryClientsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	stakeStorage, found := k.epochStorageKeeper.GetStakeStorageCurrent(ctx, req.ChainID)
	if !found {
		stakeStorage = epochstoragetypes.StakeStorage{}
	}
	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, req.ChainID)
	unstakingStakeStorage, found := k.epochStorageKeeper.GetStakeStorageUnstake(ctx)
	if !found {
		unstakingStakeStorage = epochstoragetypes.StakeStorage{}
	}
	outputStr := fmt.Sprintf("Staked Clients Query Output:\nChainID: %s Enabled: %t Current Block: %d\nStaked Clients:\n%v\nUnstaking Clients:\n%v\n--------------------------------------\n", req.ChainID, foundAndActive, ctx.BlockHeight(), stakeStorage.StakeEntries, unstakingStakeStorage.StakeEntries)
	return &types.QueryClientsResponse{StakeEntry: stakeStorage.StakeEntries, Output: outputStr}, nil
}
