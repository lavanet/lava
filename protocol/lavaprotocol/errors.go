package lavaprotocol

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ProviderFinzalizationDataError               = sdkerrors.New("ProviderFinzalizationData Error", 3365, "provider did not sign finalization data correctly")
	ProviderFinzalizationDataAccountabilityError = sdkerrors.New("ProviderFinzalizationDataAccountability Error", 3366, "provider returned invalid finalization data, with accountability")
	HashesConsunsusError                         = sdkerrors.New("HashesConsunsus Error", 3367, "identified finalized responses with conflicting hashes, from two providers")
)
