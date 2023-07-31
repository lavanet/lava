package common

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var ContextDeadlineExceededError = sdkerrors.New("ContextDeadlineExceeded Error", 300, "context deadline exceeded")
