package keeper

import (
	"context"
	"fmt"

	sdkerror "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/dualstaking/types"
)

func (k msgServer) Delegate(goCtx context.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.MsgDelegateResponse{}, k.Keeper.DelegateFull(ctx, msg.Creator, msg.Validator, msg.Provider, msg.ChainID, msg.Amount)
}

// DelegateFull uses staking module for to delegate with hooks
func (k Keeper) DelegateFull(ctx sdk.Context, delegator string, validator string, provider string, chainID string, amount sdk.Coin) error {
	chainIDs := k.specKeeper.GetAllChainIDs(ctx)
	if !slices.Contains(chainIDs, chainID) {
		return utils.LavaFormatWarning("invalid chain ID", fmt.Errorf("chain ID not found"),
			utils.LogAttr("chain_id", chainID))
	}

	valAddr, valErr := sdk.ValAddressFromBech32(validator)
	if valErr != nil {
		return valErr
	}

	validatorType, found := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return stakingtypes.ErrNoValidatorFound
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return err
	}

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if amount.Denom != bondDenom {
		return sdkerror.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", amount.Denom, bondDenom,
		)
	}

	if err := validateCoins(amount); err != nil {
		return err
	} else if amount.IsZero() {
		return sdkerror.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin amount: got 0")
	}

	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, amount.Amount, stakingtypes.Unbonded, validatorType, true)
	if err != nil {
		return err
	}

	err = k.Redelegate(
		ctx,
		delegator,
		types.EMPTY_PROVIDER,
		provider,
		types.EMPTY_PROVIDER_CHAINID,
		chainID,
		amount,
	)

	if err == nil {
		logger := k.Logger(ctx)
		details := map[string]string{
			"delegator": delegator,
			"provider":  provider,
			"chainID":   chainID,
			"amount":    amount.String(),
		}
		utils.LogLavaEvent(ctx, logger, types.DelegateEventName, details, "Delegate")
	}

	return err
}
