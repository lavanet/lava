package types

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationstoretypes "github.com/lavanet/lava/v2/x/fixationstore/types"
)

type EpochStorageKeeper interface {
	GetEpochStart(ctx sdk.Context) uint64
	GetNextEpoch(ctx sdk.Context, block uint64) (uint64, error)
}

type SpecKeeper interface {
	GetExpectedServicesForSpec(ctx sdk.Context, chainID string, mandatory bool) (expectedInterfaces map[epochstoragetypes.EndpointService]struct{}, err error)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstoretypes.FixationStore
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
