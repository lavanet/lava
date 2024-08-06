package common

import sdkerrors "cosmossdk.io/errors"

var (
	ContextDeadlineExceededError = sdkerrors.New("ContextDeadlineExceeded Error", 300, "context deadline exceeded")
	StatusCodeError504           = sdkerrors.New("Disallowed StatusCode Error", 504, "Disallowed status code error")
	StatusCodeError429           = sdkerrors.New("Disallowed StatusCode Error", 429, "Disallowed status code error")
	StatusCodeErrorStrict        = sdkerrors.New("Disallowed StatusCode Error", 800, "Disallowed status code error")
	APINotSupportedError         = sdkerrors.New("APINotSupported Error", 900, "api not supported")
	SubscriptionNotFoundError    = sdkerrors.New("SubscriptionNotFoundError Error", 901, "subscription not found")
)
