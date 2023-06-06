package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type Filter interface {
	Filter(ctx sdk.Context, stakeEntry []epochstoragetypes.StakeEntry) []bool
	InitFilter(strictestPolicy projectstypes.Policy) bool // return if filter is usable (by the policy)
}

func GetAllFilters() []Filter {
	var selectedProvidersFilter SelectedProvidersFilter
	var frozenProvidersFilter FrozenProvidersFilter

	filters := []Filter{&selectedProvidersFilter, &frozenProvidersFilter}
	return filters
}
