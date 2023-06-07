package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type SelectedProvidersFilter struct {
	selectedProviders []string
}

func (f *SelectedProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry) []bool {
	filterResult := make([]bool, len(providers))
	if len(f.selectedProviders) == 0 {
		return filterResult
	}

	selectedProvidersMap := map[string]string{}
	for _, selectedProviderAddr := range f.selectedProviders {
		selectedProvidersMap[selectedProviderAddr] = ""
	}

	for i := range providers {
		_, found := selectedProvidersMap[providers[i].Address]
		if found {
			filterResult[i] = true
		}
	}

	return filterResult
}

func (f *SelectedProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	switch strictestPolicy.SelectedProvidersMode {
	case projectstypes.Policy_EXCLUSIVE, projectstypes.Policy_MIXED:
		f.selectedProviders = strictestPolicy.SelectedProviders
		return true
	}
	return false
}
