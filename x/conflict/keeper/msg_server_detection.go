package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) Detection(goCtx context.Context, msg *types.MsgDetection) (*types.MsgDetectionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)
	utils.LogLavaEvent(ctx, logger, "conflict_detection", map[string]string{"client": msg.Creator}, "Got a new conflict detection from consumer")
	return &types.MsgDetectionResponse{}, nil
}
