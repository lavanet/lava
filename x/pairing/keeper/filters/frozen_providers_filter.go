package filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

type FrozenProvidersFilter struct{}

func (*FrozenProvidersFilter) Filter(ctx sdk.Context, stakeEntry []epochstoragetypes.StakeEntry) []bool {
	return nil
}

func (*FrozenProvidersFilter) InitFilter(strictestPolicy projectstypes.Policy) bool {
	return false
}
