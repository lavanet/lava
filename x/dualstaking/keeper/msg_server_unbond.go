package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Unbond(goCtx context.Context, msg *types.MsgUnbond) (*types.MsgUnbondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.Unbond(
		ctx,
		msg.Creator,
		msg.Provider,
		msg.ChainID,
		msg.Amount,
	)

	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"delegator": msg.Creator,
			"provider":  msg.Provider,
			"chainID":   msg.ChainID,
			"amount":    msg.Amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.UnbondingEventName, details, "Unbond")
	}

	return &types.MsgUnbondResponse{}, err
}
