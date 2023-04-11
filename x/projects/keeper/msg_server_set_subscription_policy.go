package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetSubscriptionPolicy(goCtx context.Context, msg *types.MsgSetSubscriptionPolicy) (*types.MsgSetSubscriptionPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.SetPolicy(ctx, msg.GetProjects(), msg.GetPolicy(), msg.GetCreator(), false)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetSubscriptionPolicyResponse{}, nil
}
