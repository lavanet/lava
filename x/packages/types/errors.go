package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/packages module sentinel errors
var (
	ErrEmptyPackages                 = sdkerrors.Register(ModuleName, 1, "packages list is empty")
	ErrInvalidPackageDuration        = sdkerrors.Register(ModuleName, 2, "package's duration field is invalid")
	ErrInvalidPackagePrice           = sdkerrors.Register(ModuleName, 3, "package's price field is invalid")
	ErrInvalidPackageOveruse         = sdkerrors.Register(ModuleName, 4, "package's CU overuse fields are invalid")
	ErrInvalidPackageServicersToPair = sdkerrors.Register(ModuleName, 5, "package's servicersToPair field is invalid")
	ErrInvalidPackageName            = sdkerrors.Register(ModuleName, 6, "package's name field is invalid")
	ErrInvalidPackageType            = sdkerrors.Register(ModuleName, 7, "package's type field is invalid")
	ErrInvalidPackageDescription     = sdkerrors.Register(ModuleName, 8, "package's description field is invalid")
	ErrInvalidPackageComputeUnits    = sdkerrors.Register(ModuleName, 9, "package's compute units fields are invalid")
	ErrInvalidPackageAnnualDiscount  = sdkerrors.Register(ModuleName, 10, "package's annual discount field is invalid")
)
