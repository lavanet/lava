package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetAdminPolicy(goCtx context.Context, msg *types.MsgSetAdminPolicy) (*types.MsgSetAdminPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.SetPolicy(ctx, []string{msg.GetProject()}, msg.GetPolicy(), msg.GetCreator(), true)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetAdminPolicyResponse{}, nil
}
