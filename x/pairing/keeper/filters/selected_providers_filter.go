package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

type SelectedProvidersFilter struct {
	selectedProviders []string
	mix               bool
}

func (f *SelectedProvidersFilter) IsMix() bool {
	return f.mix
}

func (f *SelectedProvidersFilter) InitFilter(strictestPolicy planstypes.Policy) (bool, []Filter) {
	switch strictestPolicy.SelectedProvidersMode {
	case planstypes.SELECTED_PROVIDERS_MODE_EXCLUSIVE:
		f.selectedProviders = strictestPolicy.SelectedProviders
		return true, nil
	case planstypes.SELECTED_PROVIDERS_MODE_MIXED:
		f.mix = true // set mix as true
		f.selectedProviders = strictestPolicy.SelectedProviders
		return true, nil
	}
	return false, nil
}

func (f *SelectedProvidersFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	filterResult := make([]bool, len(providers))
	if len(f.selectedProviders) == 0 {
		return filterResult
	}

	selectedProvidersMap := map[string]struct{}{}
	for _, selectedProviderAddr := range f.selectedProviders {
		selectedProvidersMap[selectedProviderAddr] = struct{}{}
	}

	for i := range providers {
		_, filterResult[i] = selectedProvidersMap[providers[i].Address]
	}

	return filterResult
}
