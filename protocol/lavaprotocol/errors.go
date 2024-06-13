package lavaprotocol

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	HashesConsensusError        = sdkerrors.New("HashesConsensus Error", 3367, "identified finalized responses with conflicting hashes, from two providers")
	ConsistencyError            = sdkerrors.New("Consistency Error", 3368, "does not meet consistency requirements")
	UnhandledRelayReceiverError = sdkerrors.New("UnhandledRelayReceiver Error", 3369, "provider does not handle requested api interface and spec")
	DisabledRelayReceiverError  = sdkerrors.New("DisabledRelayReceiverError Error", 3370, "provider does not pass verification and disabled this interface and spec")
)
