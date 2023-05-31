package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/plan module sentinel errors
var (
	ErrEmptyPlans                 = sdkerrors.Register(ModuleName, 1, "plans list is empty")
	ErrInvalidPlanPrice           = sdkerrors.Register(ModuleName, 3, "plan's price field is invalid")
	ErrInvalidPlanOveruse         = sdkerrors.Register(ModuleName, 4, "plan's CU overuse fields are invalid")
	ErrInvalidPlanServicersToPair = sdkerrors.Register(ModuleName, 5, "plan's servicersToPair field is invalid")
	ErrInvalidPlanName            = sdkerrors.Register(ModuleName, 6, "plan's name field is invalid")
	ErrInvalidPlanType            = sdkerrors.Register(ModuleName, 7, "plan's type field is invalid")
	ErrInvalidPlanDescription     = sdkerrors.Register(ModuleName, 8, "plan's description field is invalid")
	ErrInvalidPlanComputeUnits    = sdkerrors.Register(ModuleName, 9, "plan's compute units fields are invalid")
	ErrInvalidPlanAnnualDiscount  = sdkerrors.Register(ModuleName, 10, "plan's annual discount field is invalid")
)
