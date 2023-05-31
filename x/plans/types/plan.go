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

	// check the compute units field are non-zero
	if p.PlanPolicy.GetTotalCuLimit() == 0 || p.PlanPolicy.GetEpochCuLimit() == 0 {
		return sdkerrors.Wrap(ErrInvalidPlanComputeUnits, "plan's compute units fields can't be zero")
	}

	// check that the plan's servicersToPair is larger than 1
	if p.PlanPolicy.GetMaxProvidersToPair() <= 1 {
		return sdkerrors.Wrap(ErrInvalidPlanServicersToPair, "plan's servicersToPair field can't be one or lower")
	}

	// check that the plan's description length is below the max length
	if len(p.GetDescription()) > MAX_LEN_PACKAGE_DESCRIPTION {
		return sdkerrors.Wrap(ErrInvalidPlanDescription, "plan's description is too long")
	}

	// check that the plan's type length is below the max length
	if len(p.GetType()) > MAX_LEN_PACKAGE_TYPE {
		return sdkerrors.Wrap(ErrInvalidPlanType, "plan's type is too long")
	}

	// check that the plan's annual discount is valid
	if p.GetAnnualDiscountPercentage() > uint64(100) {
		return sdkerrors.Wrap(ErrInvalidPlanAnnualDiscount, "plan's annual discount is invalid (not between 0-100 percent)")
	}

	// check that if the selected providers mode is 0 or 3, the selected providers list should be empty
	if (p.PlanPolicy.SelectedProvidersMode == 0 || p.PlanPolicy.SelectedProvidersMode == 3) &&
		len(p.PlanPolicy.SelectedProviders) != 0 {
		return sdkerrors.Wrap(ErrInvalidSelectedProvidersConfig, `cannot configure mode = 0 (no providers restrictions) 
			or 3 (selected providers feature is disabled) and non-empty list of selected providers`)
	}

	// check that the selected providers addresses are valid and there are no duplicates
	seen := map[string]bool{}
	for _, addr := range p.PlanPolicy.SelectedProviders {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selected provider address (%s)", err)
		}

		if seen[addr] {
			return sdkerrors.Wrapf(ErrInvalidSelectedProvidersConfig, "found duplicate provider address %s", addr)
		}
		seen[addr] = true
	}

	return nil
}
