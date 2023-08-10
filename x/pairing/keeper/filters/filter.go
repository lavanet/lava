package filters

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingscores "github.com/lavanet/lava/x/pairing/keeper/scores"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

type Filter interface {
	Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool
	InitFilter(strictestPolicy planstypes.Policy) bool // return if filter is usable (by the policy)
	IsMix() bool
}

func GetAllFilters() []Filter {
	var selectedProvidersFilter SelectedProvidersFilter
	var frozenProvidersFilter FrozenProvidersFilter
	var geolocationFilter GeolocationFilter
	var addonFilter AddonFilter

	filters := []Filter{&selectedProvidersFilter, &frozenProvidersFilter, &geolocationFilter, &addonFilter}
	return filters
}

func initFilters(filters []Filter, strictestPolicy planstypes.Policy) []Filter {
	activeFilters := []Filter{}

	for _, filter := range filters {
		active := filter.InitFilter(strictestPolicy)
		if active {
			activeFilters = append(activeFilters, filter)
		}
	}

	return activeFilters
}

func FilterProviders(ctx sdk.Context, filters []Filter, providers []epochstoragetypes.StakeEntry, strictestPolicy *planstypes.Policy, currentEpoch uint64, slotCount int) ([]*pairingscores.PairingScore, error) {
	filters = initFilters(filters, *strictestPolicy)

	var filtersResult [][]bool
	mixFilters := []Filter{} // mix filters

	for _, filter := range filters {
		res := filter.Filter(ctx, providers, currentEpoch)
		if len(res) != len(providers) {
			return nil, utils.LavaFormatError("filter result length is not equal to providers list length", fmt.Errorf("filter failed"),
				utils.Attribute{Key: "filter result length", Value: len(res)},
				utils.Attribute{Key: "providers length", Value: len(providers)},
			)
		}
		filtersResult = append(filtersResult, res)
		if filter.IsMix() {
			mixFilters = append(mixFilters, filter)
		}
	}
	mixFilterIndexes := CalculateMixFilterSlots(mixFilters, slotCount)

	// create providerScore array with all possible providers
	providerScores := []*pairingscores.PairingScore{}
	for j := 0; j < len(providers); j++ {
		result := true
		slotFiltering := map[int]struct{}{} // for mix filters
		for i := 0; i < len(filters); i++ {
			// if filter is mandatory
			if !filtersResult[i][j] {
				if !filters[i].IsMix() {
					result = false
					break
				} else {
					// filter is a mix filter, that didn't pass
					for _, index := range mixFilterIndexes[filters[i]] {
						slotFiltering[index] = struct{}{} // this provider won't be selected at these slot numbers
					}
				}
			}
		}

		if result {
			providerScore := pairingscores.NewPairingScore(&providers[j])
			providerScores = append(providerScores, providerScore)
		}
	}

	return providerScores, nil
}

// this function calculates which slot indexes should be filtered by what filters when mix filtering
// it divides the mix filters to abtches where one batch is always empty and the rest contain one or more mix filters
// in case there are too many mix filters the batches grow in the amount of filters, in case there are less mix filters than slots, each batch will contain only one filter
func CalculateMixFilterSlots(mixFilters []Filter, slotCount int) (mixFiltersIndexes map[Filter][]int) {
	mixFiltersIndexes = map[Filter][]int{}
	mixFiltersCount := len(mixFilters)
	if slotCount <= 1 || len(mixFilters) == 0 {
		return mixFiltersIndexes
	}
	filtersInBatch := 1

	//
	// +1 for no mix filters
	for (mixFiltersCount/filtersInBatch)+1 > slotCount {
		filtersInBatch++
	}

	getRelevantFilters := func(index int) []Filter {
		providersInBatch := slotCount/(mixFiltersCount/filtersInBatch) + 1
		if index < providersInBatch {
			return nil // no filters in first batch
		}
		batchNumber := (index / providersInBatch) - 1
		return mixFilters[batchNumber : batchNumber+filtersInBatch]
	}

	for i := 0; i < slotCount; i++ {
		for _, filter := range getRelevantFilters(i) {
			if _, ok := mixFiltersIndexes[filter]; !ok {
				mixFiltersIndexes[filter] = []int{i}
			} else {
				mixFiltersIndexes[filter] = append(mixFiltersIndexes[filter], i)
			}
		}
	}
	return mixFiltersIndexes
}
