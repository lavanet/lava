package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// Function to validate a plan object fields
func (p Plan) ValidatePlan() error {
	// check that the plan's price is non-zero
	if p.GetPrice().IsEqual(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())) {
		return sdkerrors.Wrap(ErrInvalidPlanPrice, "plan's price can't be zero")
	}

	// check that if overuse is allowed then the overuse rate is non-zero
	if p.GetAllowOveruse() && p.GetOveruseRate() == 0 {
		return sdkerrors.Wrap(ErrInvalidPlanOveruse, "plan can't allow CU overuse and have overuse rate of zero")
	}

	// check that if overuse is not allowed then the overuse rate is zero
	if !p.GetAllowOveruse() && p.GetOveruseRate() != 0 {
		return sdkerrors.Wrap(ErrInvalidPlanOveruse, "plan can't forbid CU overuse and have a non-zero overuse rate")
	}

	// check that the plan's description length is below the max length
	if len(p.GetDescription()) > MAX_LEN_PLAN_DESCRIPTION {
		return sdkerrors.Wrap(ErrInvalidPlanDescription, "plan's description is too long")
	}

	// check that the plan's type length is below the max length
	if len(p.GetType()) > MAX_LEN_PLAN_TYPE {
		return sdkerrors.Wrap(ErrInvalidPlanType, "plan's type is too long")
	}

	// check that the plan's annual discount is valid
	if p.GetAnnualDiscountPercentage() > uint64(100) {
		return sdkerrors.Wrap(ErrInvalidPlanAnnualDiscount, "plan's annual discount is invalid (not between 0-100 percent)")
	}

	err := p.PlanPolicy.ValidateBasicPolicy(true)
	if err != nil {
		return err
	}

	return nil
}
