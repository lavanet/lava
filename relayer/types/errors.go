package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrconsumerSession = sdkerrors.New("consumerSession Error", 352, "consumerSession out of sync") // if a client session error happens we block list the client session for one epoch.
)
