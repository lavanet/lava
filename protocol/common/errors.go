package common

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var ( // Consumer Side Errors
	ContextDeadlineExceededError = sdkerrors.New("ContextDeadlineExceeded Error", 300, "Context deadline exceeded")
)
