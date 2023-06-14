package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/projects module sentinel errors
var (
	ErrInvalidPolicyCuFields           = sdkerrors.Register(ModuleName, 1099, "CU per epoch field can't be larger than Total CU field")
	ErrInvalidPolicyMaxProvidersToPair = sdkerrors.Register(ModuleName, 1101, "MaxProvidersToPair cannot be less than 2")
	ErrInvalidPolicy                   = sdkerrors.Register(ModuleName, 1102, "Invalid policy")
	ErrPolicyBasicValidation           = sdkerrors.Register(ModuleName, 1100, "invalid policy")
	ErrInvalidKeyType                  = sdkerrors.Register(ModuleName, 1103, "invalid project key type")
	ErrInvalidSelectedProvidersConfig  = sdkerrors.Register(ModuleName, 1104, "plan's selected providers config is invalid")
)
