package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
)

func (k msgServer) ClaimRewards(goCtx context.Context, msg *types.MsgClaimRewards) (*types.MsgClaimRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return &types.MsgClaimRewardsResponse{}, utils.LavaFormatError("invalid creator address", err)
	}

	if msg.Provider != "" && msg.Provider != commontypes.EMPTY_PROVIDER {
		_, err = sdk.AccAddressFromBech32(msg.Provider)
		if err != nil {
			return &types.MsgClaimRewardsResponse{}, utils.LavaFormatError("invalid provider address", err)
		}
	}

	claimed, err := k.Keeper.ClaimRewards(
		ctx,
		msg.Creator,
		msg.Provider,
	)
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"delegator": msg.Creator,
			"provider":  msg.Provider,
			"claimed":   claimed.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.ClaimRewardsEventName, details, "Claim Delegation Rewards")
	}

	return &types.MsgClaimRewardsResponse{}, err
}
