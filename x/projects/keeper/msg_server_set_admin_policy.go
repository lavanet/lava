package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetPolicy(goCtx context.Context, msg *types.MsgSetPolicy) (*types.MsgSetPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	policy := msg.GetPolicy()
	err := k.SetProjectPolicy(ctx, []string{msg.GetProject()}, &policy, msg.GetCreator(), types.SET_ADMIN_POLICY)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetPolicyResponse{}, nil
}
