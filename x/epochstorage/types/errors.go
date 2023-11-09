package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/epochstorage module sentinel errors
var (
	ErrProviderNotStaked    = sdkerrors.Register(ModuleName, 1000, "provider not staked")
	ErrStakeStorageNotFound = sdkerrors.Register(ModuleName, 1001, "stake storage not found")
)
