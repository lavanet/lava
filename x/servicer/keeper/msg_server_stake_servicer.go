package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) StakeServicer(goCtx context.Context, msg *types.MsgStakeServicer) (*types.MsgStakeServicerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeServicerResponse{}, nil
}
