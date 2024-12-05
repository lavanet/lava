package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

func (k msgServer) Redelegate(goCtx context.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := utils.ValidateCoins(ctx, k.stakingKeeper.BondDenom(ctx), msg.Amount, false); err != nil {
		return &types.MsgRedelegateResponse{}, err
	}

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return &types.MsgRedelegateResponse{}, err
	}

	err := k.Keeper.Redelegate(
		ctx,
		msg.Creator,
		msg.FromProvider,
		msg.ToProvider,
		msg.Amount,
		false,
	)

	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"delegator":     msg.Creator,
			"from_provider": msg.FromProvider,
			"to_provider":   msg.ToProvider,
			"amount":        msg.Amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.RedelegateEventName, details, "Redelegate")
	}

	return &types.MsgRedelegateResponse{}, err
}
