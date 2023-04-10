package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetSubscriptionPolicy(goCtx context.Context, msg *types.MsgSetSubscriptionPolicy) (*types.MsgSetSubscriptionPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// creator := msg.Creator
	// policy := msg.Policy
	// projects := msg.Projects

	_ = ctx

	return &types.MsgSetSubscriptionPolicyResponse{}, nil
}
