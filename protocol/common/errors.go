package common

import sdkerrors "cosmossdk.io/errors"

var ContextDeadlineExceededError = sdkerrors.New("ContextDeadlineExceeded Error", 300, "context deadline exceeded")
