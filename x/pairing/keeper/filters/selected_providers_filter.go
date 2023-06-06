package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type SelectedProvidersFilter struct {
	selectedProviders []epochstoragetypes.StakeEntry
}

func (*SelectedProvidersFilter) Filter(ctx sdk.Context, stakeEntry []epochstoragetypes.StakeEntry) []bool {
	return nil
}

func (*SelectedProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	return false
}
