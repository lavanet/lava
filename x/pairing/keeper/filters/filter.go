package filters

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingscores "github.com/lavanet/lava/v2/x/pairing/keeper/scores"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

// The Filter interface allows creating filters that filter out providers in the pairing process.
// There are 3 functions defined in the interface:
//   - Filter: gets a list of providers and outputs a boolean array that indicates which providers passed
//             the filter (by index) and the number of passed providers.
//   - InitFilter: initializes the filter paramenters using the consumer's project's effective policy.
//                 The output is whether the filter is active and should be used when filtering providers and
//                 generates a list of sub-filters that are used only in mixed mode (see note below).
//   - IsMix: returns whether the filter is in mix mode (should be considered when the policy defines the MIX selected
//            providers mode).

// Note, the only filter that is divided to additional sub filters is the addon filter. When the policy defines the MIX
// selected providers mode, we look at the total requirements of the policy for addon\extensions divide them.
// Let's assume the policy defines the pairing should have providers that support the "archive" addon and the "debug"
// and "trace" extensions combined. So the addon filter will filter providers that are not supporting the whole combination,
// but if the selected providers mode is MIX, we create additional mix filters that filter providers by the defined addons and
// extensions separately. In this example, the new filters will be a filter with "archive" only, filter with "debug" only
// and filter with "trace" only.
type Filter interface {
	Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool
	InitFilter(strictestPolicy planstypes.Policy) (active bool, subMixFilters []Filter) // return if filter is usable (by the policy) and mix filters
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

func initFilters(filters []Filter, strictestPolicy planstypes.Policy) (activeFilters []Filter) {
	activeFilters = []Filter{}

	for _, filter := range filters {
		active, subMixFilters := filter.InitFilter(strictestPolicy)
		if active {
			activeFilters = append(activeFilters, filter)
			if subMixFilters != nil {
				activeFilters = append(activeFilters, subMixFilters...)
			}
		}
	}

	return activeFilters
}

func SetupScores(ctx sdk.Context, filters []Filter, providers []epochstoragetypes.StakeEntry, strictestPolicy *planstypes.Policy, currentEpoch uint64, slotCount int, cluster string, qg pairingscores.QosGetter) ([]*pairingscores.PairingScore, error) {
	filters = initFilters(filters, *strictestPolicy)

	var filtersResult [][]bool
	mixFilters := []Filter{}

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
		// check provider for each filter
		for i := 0; i < len(filters); i++ {
			// if filter result was negative meaning provider didn't pass
			if !filtersResult[i][j] {
				// check if filter is mandatory
				if filters[i].IsMix() {
					// filter is a mix filter, that didn't pass
					for _, slotIndex := range mixFilterIndexes[filters[i]] {
						slotFiltering[slotIndex] = struct{}{} // this provider won't be selected at these slot numbers
					}
				} else {
					// filter is a mandatory filter that didn't pass so we skip this provider
					result = false
					break
				}
			}
		}

		if result {
			// TODO: uncomment this code once the providerQosFS's update is implemented (it's currently always empty)
			// qos, err := qg.GetQos(ctx, providers[j].Chain, cluster, providers[j].Address)
			// if err != nil {
			// 	// only printing error and skipping provider so pairing won't fail
			// 	utils.LavaFormatError("could not construct provider qos", err)
			// 	continue
			// }
			providerScore := pairingscores.NewPairingScore(&providers[j], types.QualityOfServiceReport{})
			providerScore.SlotFiltering = slotFiltering
			providerScores = append(providerScores, providerScore)
		}
	}

	return providerScores, nil
}

// this function calculates which slot indexes should be filtered by what filters when mix filtering
// it divides the mix filters to batches where one batch is always empty and the rest contain one or more mix filters
// in case there are too many mix filters the batches grow in the amount of filters, in case there are less mix filters than slots, each batch will contain only one filter
func CalculateMixFilterSlots(mixFilters []Filter, slotCount int) (mixFiltersIndexes map[Filter][]int) {
	mixFiltersIndexes = map[Filter][]int{}
	mixFiltersCount := len(mixFilters)
	if slotCount <= 1 || len(mixFilters) == 0 {
		return mixFiltersIndexes
	}
	// filtersInBatch needs to supply this condition:
	// (mixFiltersCount/filtersInBatch)+1 <= slotCount
	filtersInBatch := (mixFiltersCount + slotCount - 2) / (slotCount - 1)
	getRelevantFilters := func(index int) []Filter {
		providersInBatch := slotCount / ((mixFiltersCount / filtersInBatch) + 1)
		if index < providersInBatch {
			return nil // no filters in first batch
		}
		batchNumber := (index / providersInBatch) - 1
		startIndex := batchNumber * filtersInBatch
		endIndex := startIndex + filtersInBatch
		if startIndex+filtersInBatch > len(mixFilters) {
			// when the numbers don't evenly divide we just disable mix filter slots of the remainder
			return nil
		}
		return mixFilters[startIndex:endIndex]
	}

	for i := 0; i < slotCount; i++ {
		for _, filter := range getRelevantFilters(i) {
			mixFiltersIndexes[filter] = append(mixFiltersIndexes[filter], i)
		}
	}
	return mixFiltersIndexes
}
