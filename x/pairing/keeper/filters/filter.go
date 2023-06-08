package filters

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type Filter interface {
	Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry) []bool
	InitFilter(strictestPolicy projectstypes.Policy) bool // return if filter is usable (by the policy)
}

func GetAllFilters() []Filter {
	var selectedProvidersFilter SelectedProvidersFilter
	var frozenProvidersFilter FrozenProvidersFilter
	var geolocationFilter GeolocationFilter

	filters := []Filter{&selectedProvidersFilter, &frozenProvidersFilter, &geolocationFilter}
	return filters
}

func initFilters(filters []Filter, strictestPolicy projectstypes.Policy) []Filter {
	activeFilters := []Filter{}

	for _, filter := range filters {
		active := filter.InitFilter(strictestPolicy)
		if active {
			activeFilters = append(activeFilters, filter)
		}
	}

	return activeFilters
}

func FilterProviders(ctx sdk.Context, filters []Filter, providers []epochstoragetypes.StakeEntry, strictestPolicy projectstypes.Policy) []epochstoragetypes.StakeEntry {
	filters = initFilters(filters, strictestPolicy)

	filtersResult := make([]bool, len(providers))
	for i := range filtersResult {
		filtersResult[i] = true
	}

	for _, filter := range filters {
		res := filter.Filter(ctx, providers)
		if len(res) != len(providers) {
			utils.LavaFormatError("filter result length is not equal to providers list length", fmt.Errorf("filter failed"),
				utils.Attribute{},
			)
			return []epochstoragetypes.StakeEntry{}
		}

		for i := range res {
			filtersResult[i] = filtersResult[i] && res[i]
		}
	}

	filteredProviders := []epochstoragetypes.StakeEntry{}
	for i := range filtersResult {
		if filtersResult[i] {
			filteredProviders = append(filteredProviders, providers[i])
		}
	}

	return filteredProviders
}
