package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UserMaxCu(goCtx context.Context, req *types.QueryUserMaxCuRequest) (*types.QueryUserMaxCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	//Process the query
	userAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "invalid address")
	}

	existingEntry, entryExists, _ := k.epochStorageKeeper.StakeEntryByAddress(ctx, epochstoragetypes.ClientKey, req.ChainID, userAddr)

	if !entryExists {
		return nil, status.Error(codes.Unavailable, "stake not found for addr: "+req.Address+", chainID: "+req.ChainID)
	}

	return &types.QueryUserMaxCuResponse{MaxCu: k.ClientMaxCUProvider(ctx, &existingEntry)}, nil
}
