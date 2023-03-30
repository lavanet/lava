package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetProjectPolicy(goCtx context.Context, msg *types.MsgSetProjectPolicy) (*types.MsgSetProjectPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	projectID := msg.Project
	adminKey := msg.Creator
	policy := msg.Policy

	err := k.SetPolicy(ctx, projectID, *policy, adminKey)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetProjectPolicyResponse{}, nil
}
