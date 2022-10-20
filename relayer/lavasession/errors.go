package lavasession

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	PairingListEmpty                = sdkerrors.New("pairingListEmpty Error", 665, "no pairings available.")                       // client couldnt connect to any provider.
	AllProviderEndpointsDisabled    = sdkerrors.New("AllProviderEndpointsDisabled Error", 667, "all endpoints are not available.") // a provider is completly unresponsive all endpoints are not available
	MaximumNumberOfSessionsExceeded = sdkerrors.New("MaximumNumberOfSessionsExceeded Error", 668, "provider reached maximum number of active sessions.")
)
