package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type FrozenProvidersFilter struct{}

func (f *FrozenProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	return true
}

func (f *FrozenProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry) []bool {
	filterResult := make([]bool, len(providers))
	for i := range providers {
		if !isProviderFrozen(providers[i]) {
			filterResult[i] = true
		}
	}

	return filterResult
}

func isProviderFrozen(stakeEntry epochstoragetypes.StakeEntry) bool {
	return stakeEntry.StakeAppliedBlock == types.FROZEN_BLOCK
}
