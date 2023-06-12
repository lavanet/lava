package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type FrozenProvidersFilter struct{}

func (f *FrozenProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	// frozen providers (or providers that their stake is not applied yet) can't be part of the pairing - this filter is always active
	return true
}

func (f *FrozenProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	for i := range providers {
		if !isProviderFrozen(ctx, providers[i], currentEpoch) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isProviderFrozen(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry, currentEpoch uint64) bool {
	return stakeEntry.StakeAppliedBlock > currentEpoch
}
