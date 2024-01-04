package keeper

import (
	"context"

	sdkerror "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Redelegate(goCtx context.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Amount.Denom != bondDenom {
		return &types.MsgRedelegateResponse{}, sdkerror.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", msg.Amount.Denom, bondDenom,
		)
	}

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
