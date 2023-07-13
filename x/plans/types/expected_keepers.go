package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

type EpochStorageKeeper interface {
	GetEpochStart(ctx sdk.Context) uint64
	GetNextEpoch(ctx sdk.Context, block uint64) (uint64, error)
}

type SpecKeeper interface {
	GetExpectedInterfacesForSpec(ctx sdk.Context, chainID string, mandatory bool) (expectedInterfaces map[epochstoragetypes.EndpointService]struct{}, found bool)
}
