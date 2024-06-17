package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
)

func (k msgServer) AddSpecs(goCtx context.Context, msg *types.MsgAddSpecs) (*types.MsgAddSpecsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.HandleSpecs(ctx, msg.Specs, msg.Creator)
	return &types.MsgAddSpecsResponse{}, err
}
