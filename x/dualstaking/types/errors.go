package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dualstaking module sentinel errors
var (
	ErrProviderNotStaked      = sdkerrors.Register(ModuleName, 1000, "provider not staked")
	ErrDelegationNotFound     = sdkerrors.Register(ModuleName, 1001, "delegation not found")
	ErrInsufficientDelegation = sdkerrors.Register(ModuleName, 1002, "invalid delegation amount")
	ErrBadDelegationAmount    = sdkerrors.Register(ModuleName, 1003, "invalid delegation amount")
	ErrUnbondingInProgress    = sdkerrors.Register(ModuleName, 1004, "unbonding already exists (same block)")
)
