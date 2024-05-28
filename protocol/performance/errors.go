package performance

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	NotConnectedError   = sdkerrors.New("Not Connected Error", 700, "No Connection To grpc server")
	NotInitializedError = sdkerrors.New("Not Initialised Error", 701, "to use cache run initCache")
)
