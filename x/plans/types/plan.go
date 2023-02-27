package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// Function to validate a package object fields
func (p Plan) ValidatePlan() error {
	// check that the package's duration is non-zero
	if p.GetDuration() == 0 {
		return sdkerrors.Wrap(ErrInvalidPlanDuration, "package's duration can't be zero")
	}

	// check that the package's price is non-zero
	if p.GetPrice().IsEqual(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())) {
		return sdkerrors.Wrap(ErrInvalidPlanPrice, "package's price can't be zero")
	}

	// check that if overuse is allowed then the overuse rate is non-zero
	if p.GetAllowOveruse() && p.GetOveruseRate() == 0 {
		return sdkerrors.Wrap(ErrInvalidPlanOveruse, "package can't allow CU overuse and have overuse rate of zero")
	}

	// check the compute units field are non-zero
	if p.GetComputeUnits() == 0 || p.GetComputeUnitsPerEpoch() == 0 {
		return sdkerrors.Wrap(ErrInvalidPlanComputeUnits, "package's compute units fields can't be zero")
	}

	// check that the package's servicersToPair is larger than 1
	if p.GetServicersToPair() <= 1 {
		return sdkerrors.Wrap(ErrInvalidPlanServicersToPair, "package's servicersToPair field can't be one or lower")
	}

	// check that the package's name length is below the max length
	if len(p.GetName()) > MAX_LEN_PACKAGE_NAME {
		return sdkerrors.Wrap(ErrInvalidPlanName, "package's name is too long")
	}

	// check that the package's description length is below the max length
	if len(p.GetDescription()) > MAX_LEN_PACKAGE_DESCRIPTION {
		return sdkerrors.Wrap(ErrInvalidPlanDescription, "package's description is too long")
	}

	// check that the package's type length is below the max length
	if len(p.GetType()) > MAX_LEN_PACKAGE_TYPE {
		return sdkerrors.Wrap(ErrInvalidPlanType, "package's type is too long")
	}

	// check that the package's annual discount is valid
	if p.GetAnnualDiscountPercentage() < uint64(0) || p.GetAnnualDiscountPercentage() > uint64(100) {
		return sdkerrors.Wrap(ErrInvalidPlanAnnualDiscount, "package's annual discount is invalid (not between 0-100)")
	}

	return nil
}
