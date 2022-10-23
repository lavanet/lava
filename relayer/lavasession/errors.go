package lavasession

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	PairingListEmpty                = sdkerrors.New("pairingListEmpty Error", 665, "No pairings available.") // client couldnt connect to any provider.
	UnreachableCodeError            = sdkerrors.New("UnreachableCode Error", 666, "Should not get here.")
	AllProviderEndpointsDisabled    = sdkerrors.New("AllProviderEndpointsDisabled Error", 667, "All endpoints are not available.") // a provider is completly unresponsive all endpoints are not available
	MaximumNumberOfSessionsExceeded = sdkerrors.New("MaximumNumberOfSessionsExceeded Error", 668, "Provider reached maximum number of active sessions.")
	MaxComputeUnitsExceeded         = sdkerrors.New("MaxComputeUnitsExceeded Error", 669, "Consumer is trying to exceed the maximum number of compute uints available.")
)
