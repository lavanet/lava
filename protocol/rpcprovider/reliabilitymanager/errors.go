package reliabilitymanager

import (
	sdkerrors "cosmossdk.io/errors"
)

var NoVoteDeadline = sdkerrors.New("Not Connected Error", 800, "No Connection To grpc server")
