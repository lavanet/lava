package protocolerrors

import "errors"

var (
	ConsistencyError            = errors.New("does not meet consistency requirements")
	UnhandledRelayReceiverError = errors.New("provider does not handle requested api interface and spec")
	DisabledRelayReceiverError  = errors.New("provider does not pass verification and disabled this interface and spec")
	NoResponseTimeout           = errors.New("timeout occurred while waiting for providers responses")
)
