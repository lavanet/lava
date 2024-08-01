package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

// x/plan module sentinel errors
var (
	ErrEmptyPlans                           = sdkerrors.Register(ModuleName, 1, "plans list is empty")
	ErrInvalidPlanIndex                     = sdkerrors.Register(ModuleName, 2, "plan's index field is invalid")
	ErrInvalidPlanPrice                     = sdkerrors.Register(ModuleName, 3, "plan's price field is invalid")
	ErrInvalidPlanOveruse                   = sdkerrors.Register(ModuleName, 4, "plan's CU overuse fields are invalid")
	ErrInvalidPlanServicersToPair           = sdkerrors.Register(ModuleName, 5, "plan's servicersToPair field is invalid")
	ErrInvalidPlanName                      = sdkerrors.Register(ModuleName, 6, "plan's name field is invalid")
	ErrInvalidPlanType                      = sdkerrors.Register(ModuleName, 7, "plan's type field is invalid")
	ErrInvalidPlanDescription               = sdkerrors.Register(ModuleName, 8, "plan's description field is invalid")
	ErrInvalidPlanComputeUnits              = sdkerrors.Register(ModuleName, 9, "plan's compute units fields are invalid")
	ErrInvalidPlanAnnualDiscount            = sdkerrors.Register(ModuleName, 10, "plan's annual discount field is invalid")
	ErrInvalidPolicyCuFields                = sdkerrors.Register(ModuleName, 11, "CU per epoch field can't be larger than Total CU field")
	ErrInvalidPolicyMaxProvidersToPair      = sdkerrors.Register(ModuleName, 12, "MaxProvidersToPair cannot be less than 2")
	ErrInvalidPolicy                        = sdkerrors.Register(ModuleName, 13, "Invalid policy")
	ErrPolicyBasicValidation                = sdkerrors.Register(ModuleName, 14, "invalid policy")
	ErrPolicyInvalidSelectedProvidersConfig = sdkerrors.Register(ModuleName, 15, "plan's selected providers config is invalid")
	ErrPolicyGeolocation                    = sdkerrors.Register(ModuleName, 16, "plan's geolocation is invalid")
	ErrInvalidDenom                         = sdkerrors.Register(ModuleName, 17, commontypes.ErrInvalidDenomMsg)
	ErrInvalidPlanProjects                  = sdkerrors.Register(ModuleName, 18, "plan's projects field is invalid")
)
