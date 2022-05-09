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

func (k Keeper) UserMaxCu(goCtx context.Context, req *types.QueryUserMaxCuRequest) (*types.QueryUserMaxCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	//Process the query
	logger := k.Logger(ctx)
	userAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		details := map[string]string{"addr": req.Address, "chainid": req.ChainID, "error": err.Error()}
		return nil, utils.LavaError(ctx, logger, "stake addr", details, "invalid address "+req.Address)
	}

	existingEntry, entryExists, _ := k.epochStorageKeeper.StakeEntryByAddress(ctx, epochstoragetypes.ClientKey, req.ChainID, userAddr)

	if !entryExists {
		details := map[string]string{"addr": req.Address, "chainid": req.ChainID, "error": err.Error()}
		return nil, utils.LavaError(ctx, logger, "stake not found", details, "stake for addr "+req.Address+" and chainid: "+req.ChainID+"was not found")
	}

	return &types.QueryUserMaxCuResponse{MaxCu: k.ClientMaxCU(ctx, &existingEntry)}, nil
}
