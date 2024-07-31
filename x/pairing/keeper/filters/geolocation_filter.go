package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

// TODO: This filter is disabled (InitFilter always returns false)
// will be used in the future to exclude geolocations (thus keeping this code)

type GeolocationFilter struct {
	geolocation int32
}

func (f *GeolocationFilter) IsMix() bool {
	return false
}

func (f *GeolocationFilter) InitFilter(strictestPolicy planstypes.Policy) (bool, []Filter) {
	/*
		if strictestPolicy.SelectedProvidersMode == planstypes.SELECTED_PROVIDERS_MODE_DISABLED ||
			strictestPolicy.SelectedProvidersMode == planstypes.SELECTED_PROVIDERS_MODE_ALLOWED {
			f.geolocation = strictestPolicy.GeolocationProfile
			return true
		}
	*/
	return false, nil
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

func isGeolocationSupported(policyGeolocation, providerGeolocation int32) bool {
	return policyGeolocation&providerGeolocation != 0
}
