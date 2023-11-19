package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Unbond(goCtx context.Context, msg *types.MsgUnbond) (*types.MsgUnbondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.Redelegate(
		ctx,
		msg.Creator,
		msg.Provider,
		EMPTY_PROVIDER,
		msg.ChainID,
		EMPTY_PROVIDER_CHAINID,
		msg.Amount,
	)
	if err != nil {
		return nil, err
	}

	addr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, err
	}
	delegatorAddress, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}
	shares, err := k.stakingKeeper.ValidateUnbondAmount(
		ctx, delegatorAddress, addr, msg.Amount.Amount,
	)
	if err != nil {
		return nil, err
	}

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Amount.Denom != bondDenom {
		return nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", msg.Amount.Denom, bondDenom,
		)
	}

	_, err = k.stakingKeeper.Undelegate(ctx, delegatorAddress, addr, shares)
	if err != nil {
		return nil, err
	}

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
