package reliabilitymanager

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var NoVoteDeadline = sdkerrors.New("Not Connected Error", 800, "No Connection To grpc server")
