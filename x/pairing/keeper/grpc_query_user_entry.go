package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UserEntry(goCtx context.Context, req *types.QueryUserEntryRequest) (*types.QueryUserEntryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Process the query
	userAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "invalid address")
	}

	epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, req.Block)
	if err != nil {
		return nil, err
	}

	existingEntry, err := k.epochStorageKeeper.GetStakeEntryForClientEpoch(ctx, req.ChainID, userAddr, epochStart)
	if err != nil {
		return nil, err
	}

	maxCU, err := k.ClientMaxCUProviderForBlock(ctx, req.Block, existingEntry)
	if err != nil {
		return nil, err
	}
	return &types.QueryUserEntryResponse{Consumer: *existingEntry, MaxCU: maxCU}, nil
}
