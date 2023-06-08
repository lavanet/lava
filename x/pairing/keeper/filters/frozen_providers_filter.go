package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type FrozenProvidersFilter struct{}

func (f *FrozenProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	return true
}

func (f *FrozenProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry) []bool {
	filterResult := make([]bool, len(providers))
	for i := range providers {
		if !isProviderFrozen(ctx, providers[i]) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isProviderFrozen(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry) bool {
	return stakeEntry.StakeAppliedBlock > uint64(ctx.BlockHeight())
}
