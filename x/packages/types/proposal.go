package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// Function to verify a package object fields
func checkPackagesProposal(packageToCheck Package) error {
	// check that the package's duration is non-zero
	if packageToCheck.GetDuration() == 0 {
		return sdkerrors.Wrap(ErrInvalidPackageDuration, "package's duration can't be zero")
	}

	// check that the package's price is non-zero
	if packageToCheck.GetPrice() == sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidPackagePrice, "package's price can't be zero")
	}

	// check that if overuse is allowed then the overuse rate is non-zero
	if packageToCheck.GetAllowOveruse() && packageToCheck.GetOveruseRate() == 0 {
		return sdkerrors.Wrap(ErrInvalidPackageOveruse, "package can't allow CU overuse and have overuse rate of zero")
	}

	// check compute units fields
	if packageToCheck.GetComputeUnits() == 0 || packageToCheck.GetComputeUnitsPerEpoch() == 0 {
		return sdkerrors.Wrap(ErrInvalidPackageComputeUnits, "package's compute units fields can't be zero")
	}

	// check that the package's servicersToPair is larger than 1
	if packageToCheck.GetServicersToPair() <= 1 {
		return sdkerrors.Wrap(ErrInvalidPackageServicersToPair, "package's servicersToPair field can't be zero")
	}

	// check that the subscriptions field is zero (this field counts the number of users subscribed to this package. If it's a new package, it must be zero)
	if packageToCheck.GetSubscriptions() != 0 {
		return sdkerrors.Wrap(ErrInvalidPackageSubscriptions, "package's subscriptions field can't be non-zero")
	}

	// check that the package's name length is below the max length
	if len(packageToCheck.GetName()) > MAX_LEN_PACKAGE_NAME {
		return sdkerrors.Wrap(ErrInvalidPackageName, "package's name is too long")
	}

	return nil
}

func stringPackage(packageToString *Package, b strings.Builder) strings.Builder {
	b.WriteString(packageToString.String())
	return b
}
