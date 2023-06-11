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

func FilterProviders(ctx sdk.Context, filters []Filter, providers []epochstoragetypes.StakeEntry, strictestPolicy projectstypes.Policy) ([]epochstoragetypes.StakeEntry, error) {
	filters = initFilters(filters, strictestPolicy)

	var filtersResult [][]bool

	for _, filter := range filters {
		res := filter.Filter(ctx, providers)
		if len(res) != len(providers) {
			return []epochstoragetypes.StakeEntry{}, utils.LavaFormatError("filter result length is not equal to providers list length", fmt.Errorf("filter failed"),
				utils.Attribute{Key: "filter result length", Value: len(res)},
				utils.Attribute{Key: "providers length", Value: len(providers)},
			)
		}
		filtersResult = append(filtersResult, res)
	}

	filteredProviders := []epochstoragetypes.StakeEntry{}
	for j := 0; j < len(providers); j++ {
		result := true
		for i := 0; i < len(filters); i++ {
			result = result && filtersResult[i][j]
		}
		if result {
			filteredProviders = append(filteredProviders, providers[j])
		}
	}

	return filteredProviders, nil
}
