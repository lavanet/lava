package lavaprotocol

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ProviderFinzalizationDataError               = sdkerrors.New("ProviderFinzalizationData Error", 3365, "provider did not sign finalization data correctly")
	ProviderFinzalizationDataAccountabilityError = sdkerrors.New("ProviderFinzalizationDataAccountability Error", 3366, "provider returned invalid finalization data, with accountability")
	HashesConsunsusError                         = sdkerrors.New("HashesConsunsus Error", 3367, "identified finalized responses with conflicting hashes, from two providers")
	ConsistencyError                             = sdkerrors.New("Consistency Error", 3368, "does not meet consistency requirements")
)
