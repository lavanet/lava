package chainproxy

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	errFailedToConvertMessage = sdkerrors.New("RPC error", 1000, "failed to convert a message")
)
