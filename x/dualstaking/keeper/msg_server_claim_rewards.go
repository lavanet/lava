package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) ClaimRewards(goCtx context.Context, msg *types.MsgClaimRewards) (*types.MsgClaimRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.ClaimRewards(
		ctx,
		msg.Creator,
		msg.Provider,
	)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"delegator": msg.Creator,
			"provider":  msg.Provider,
		}
		utils.LogLavaEvent(ctx, logger, types.DelegateEventName, details, "Claim Delegation Rewards")
	}

	return &types.MsgClaimRewardsResponse{}, err
}
