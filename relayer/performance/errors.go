package performance

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	NotConnectedError   = sdkerrors.New("Not Connected Error", 700, "No Connection To grpc server")
	NotInitialisedError = sdkerrors.New("Not Initialised Error", 701, "to use cache run initCache")
)
