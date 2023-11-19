package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Delegate(goCtx context.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	valAddr, valErr := sdk.ValAddressFromBech32(msg.Validator)
	if valErr != nil {
		return nil, valErr
	}

	validator, found := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return nil, stakingtypes.ErrNoValidatorFound
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Amount.Denom != bondDenom {
		return nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", msg.Amount.Denom, bondDenom,
		)
	}

	if err := validateCoins(msg.Amount); err != nil {
		return nil, err
	} else if msg.Amount.IsZero() {
		return nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin amount: got 0")
	}

	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Amount.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.Redelegate(
		ctx,
		msg.Creator,
		EMPTY_PROVIDER,
		msg.Provider,
		EMPTY_PROVIDER_CHAINID,
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
		utils.LogLavaEvent(ctx, logger, types.DelegateEventName, details, "Delegate")
	}

	return &types.MsgDelegateResponse{}, err
}
