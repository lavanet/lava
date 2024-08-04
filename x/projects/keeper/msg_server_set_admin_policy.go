package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/projects/types"
)

func (k msgServer) SetPolicy(goCtx context.Context, msg *types.MsgSetPolicy) (*types.MsgSetPolicyResponse, error) {
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

	err := k.SetProjectPolicy(ctx, []string{msg.GetProject()}, policy, msg.GetCreator(), types.SET_ADMIN_POLICY)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetPolicyResponse{}, nil
}
