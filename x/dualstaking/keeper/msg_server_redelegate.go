package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Redelegate(goCtx context.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.Redelegate(
		ctx,
		msg.Creator,
		msg.FromProvider,
		msg.ToProvider,
		msg.FromChainID,
		msg.ToChainID,
		msg.Amount,
	)

	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"delegator":     msg.Creator,
			"from_provider": msg.FromProvider,
			"to_provider":   msg.ToProvider,
			"from_chainID":  msg.FromChainID,
			"to_chainID":    msg.ToChainID,
			"amount":        msg.Amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.RedelegateEventName, details, "Redelegate")
	}

	return &types.MsgRedelegateResponse{}, err
}
