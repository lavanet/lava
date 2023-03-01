package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetProjectPolicy(goCtx context.Context, msg *types.MsgSetProjectPolicy) (*types.MsgSetProjectPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSetProjectPolicyResponse{}, nil
}
