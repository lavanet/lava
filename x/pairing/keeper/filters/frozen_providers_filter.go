package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

type FrozenProvidersFilter struct{}

func (f *FrozenProvidersFilter) IsMix() bool {
	return false
}

func (f *FrozenProvidersFilter) InitFilter(strictestPolicy planstypes.Policy) (bool, []Filter) {
	// frozen providers (or providers that their stake is not applied yet) can't be part of the pairing - this filter is always active
	return true, nil
}

func (f *FrozenProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	for i := range providers {
		if !isProviderFrozen(providers[i], currentEpoch) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isProviderFrozen(stakeEntry epochstoragetypes.StakeEntry, currentEpoch uint64) bool {
	return stakeEntry.StakeAppliedBlock > currentEpoch
}
