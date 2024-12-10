package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/utils"
	commontypes "github.com/lavanet/lava/v4/utils/common/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

func (k msgServer) Delegate(goCtx context.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.MsgDelegateResponse{}, k.Keeper.DelegateFull(ctx, msg.Creator, msg.Validator, msg.Provider, msg.Amount, false)
}

// DelegateFull uses staking module for to delegate with hooks
func (k Keeper) DelegateFull(ctx sdk.Context, delegator string, validator string, provider string, amount sdk.Coin, stake bool) error {
	valAddr, valErr := sdk.ValAddressFromBech32(validator)
	if valErr != nil {
		return valErr
	}

	validatorType, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !err {
		return stakingtypes.ErrNoValidatorFound
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return err
	}

	if _, err = sdk.AccAddressFromBech32(provider); err != nil {
		return err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return err
	}
	if err := utils.ValidateCoins(ctx, bondDenom, amount, false); err != nil {
		return err
	}

	delegation, found := k.GetDelegation(ctx, commontypes.EMPTY_PROVIDER, delegator)
	amountBefore := math.ZeroInt()
	if found {
		amountBefore = delegation.Amount.Amount
	}

	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, amount.Amount, stakingtypes.Unbonded, validatorType, true)
	if err != nil {
		return err
	}

	delegation, _ = k.GetDelegation(ctx, commontypes.EMPTY_PROVIDER, delegator)

	amount.Amount = delegation.Amount.Amount.Sub(amountBefore)

	err = k.Redelegate(
		ctx,
		delegator,
		commontypes.EMPTY_PROVIDER,
		provider,
		amount,
		stake,
	)

	if err == nil {
		logger := k.Logger(ctx)
		details := map[string]string{
			"delegator": delegator,
			"provider":  provider,
			"amount":    amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.DelegateEventName, details, "Delegate")
	}

	return err
}
