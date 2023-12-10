package keeper

import (
	"context"

	sdkerror "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Unbond(goCtx context.Context, msg *types.MsgUnbond) (*types.MsgUnbondResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.MsgUnbondResponse{}, k.Keeper.UnbondFull(ctx, msg.Creator, msg.Validator, msg.Provider, msg.ChainID, msg.Amount, false)
}

// UnbondFul uses staking module for to unbond with hooks
func (k Keeper) UnbondFull(ctx sdk.Context, delegator string, validator string, provider string, chainID string, amount sdk.Coin, unstake bool) error {
	// 1.redelegate from the provider to the empty provider
	// 2.calls staking module to unbond from the validator
	// 3.calls the hooks to than unbond from the empty provider

	err := k.Redelegate(
		ctx,
		delegator,
		provider,
		types.EMPTY_PROVIDER,
		chainID,
		types.EMPTY_PROVIDER_CHAINID,
		amount,
	)
	if err != nil {
		return err
	}

	addr, err := sdk.ValAddressFromBech32(validator)
	if err != nil {
		return err
	}
	delegatorAddress, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return err
	}
	shares, err := k.stakingKeeper.ValidateUnbondAmount(
		ctx, delegatorAddress, addr, amount.Amount,
	)
	if err != nil {
		return err
	}

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if amount.Denom != bondDenom {
		return sdkerror.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", amount.Denom, bondDenom,
		)
	}

	_, err = k.stakingKeeper.Undelegate(ctx, delegatorAddress, addr, shares)
	if err != nil {
		return err
	}

	if err == nil {
		logger := k.Logger(ctx)
		details := map[string]string{
			"delegator": delegator,
			"provider":  provider,
			"chainID":   chainID,
			"amount":    amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.UnbondingEventName, details, "Unbond")
	}

	return err
}
