package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	commontypes "github.com/lavanet/lava/v4/utils/common/types"
)

func (k Keeper) BalanceDelegator(ctx sdk.Context, delegator sdk.AccAddress) (int, error) {
	diff, providers, err := k.VerifyDelegatorBalance(ctx, delegator)
	if err != nil {
		return providers, err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return providers, err
	}
	// if diff is zero, do nothing, this is a redelegate
	if diff.IsZero() {
		return providers, nil
	} else if diff.IsPositive() {
		// less provider delegations,a delegation operation was done, delegate to empty provider
		err = k.Delegate(ctx, delegator.String(), commontypes.EMPTY_PROVIDER,
			sdk.NewCoin(bondDenom, diff), false)
		if err != nil {
			return providers, err
		}
	} else if diff.IsNegative() {
		// more provider delegation, unbond operation was done, unbond from providers
		err = k.UnbondUniformProviders(ctx, delegator.String(), sdk.NewCoin(bondDenom, diff.Neg()))
		if err != nil {
			return providers, err
		}
	}

	diff, _, err = k.VerifyDelegatorBalance(ctx, delegator)
	if err != nil {
		return providers, err
	}
	// now it needs to be zero
	if !diff.IsZero() {
		return providers, utils.LavaFormatError("validator and provider balances are not balanced", nil,
			utils.Attribute{Key: "delegator", Value: delegator.String()},
			utils.Attribute{Key: "diff", Value: diff.String()},
		)
	}

	return providers, nil
}
