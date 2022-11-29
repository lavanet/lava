package performance

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	NotConnectedError = sdkerrors.New("Not Connected Error", 700, "No Connection To grpc server")
)
