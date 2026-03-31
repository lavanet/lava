package protocolerrors

import "errors"

var (
	ProviderFinalizationDataError               = errors.New("provider did not sign finalization data correctly")
	ProviderFinalizationDataAccountabilityError = errors.New("provider returned invalid finalization data, with accountability")
	HashesConsensusError                        = errors.New("identified finalized responses with conflicting hashes, from two providers")
	ConsistencyError                            = errors.New("does not meet consistency requirements")
	UnhandledRelayReceiverError                 = errors.New("provider does not handle requested api interface and spec")
	DisabledRelayReceiverError                  = errors.New("provider does not pass verification and disabled this interface and spec")
	NoResponseTimeout                           = errors.New("timeout occurred while waiting for providers responses")
)
