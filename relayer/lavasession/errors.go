package lavasession

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	PairingListEmpty = sdkerrors.New("pairingListEmpty Error", 665, "no pairings available.") // client couldnt connect to any provider.
)
