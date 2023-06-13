package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

// TODO: This is a temp filter until the geolocation mechanism changes (this is not optimal)

type GeolocationFilter struct {
	geolocation uint64
}

func (f *GeolocationFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	if strictestPolicy.SelectedProvidersMode == projectstypes.SELECTED_PROVIDERS_MODE_DISABLED ||
		strictestPolicy.SelectedProvidersMode == projectstypes.SELECTED_PROVIDERS_MODE_ALLOWED {
		f.geolocation = strictestPolicy.GeolocationProfile
		return true
	}
	return false
}

func (f *GeolocationFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	for i := range providers {
		if isGeolocationSupported(f.geolocation, providers[i].Geolocation) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isGeolocationSupported(policyGeolocation uint64, providerGeolocation uint64) bool {
	return policyGeolocation&providerGeolocation != 0
}
