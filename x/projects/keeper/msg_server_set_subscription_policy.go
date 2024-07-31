package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/projects/types"
)

func (k msgServer) SetSubscriptionPolicy(goCtx context.Context, msg *types.MsgSetSubscriptionPolicy) (*types.MsgSetSubscriptionPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatError("Invalid creator address", err,
			utils.LogAttr("creator", msg.Creator),
		)
	}

	policy := msg.GetPolicy()

	if policy != nil {
		err := policy.ValidateBasicPolicy(false)
		if err != nil {
			return nil, err
		}
	}

	err := k.SetProjectPolicy(ctx, msg.GetProjects(), policy, msg.GetCreator(), types.SET_SUBSCRIPTION_POLICY)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetSubscriptionPolicyResponse{}, nil
}
