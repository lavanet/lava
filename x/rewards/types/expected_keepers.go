package types

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	v1 "github.com/lavanet/lava/v4/x/downtime/v1"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	timerstoretypes "github.com/lavanet/lava/v4/x/timerstore/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToModule(ctx context.Context, senderPool, recipientPool string, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type SpecKeeper interface {
	GetAllChainIDs(ctx sdk.Context) (chainIDs []string)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool)
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive, found bool, providersType spectypes.Spec_ProvidersTypes)
}

type TimerStoreKeeper interface {
	NewTimerStoreEndBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore
}

type EpochstorageKeeper interface {
	GetStakeEntryCurrent(ctx sdk.Context, chainID string, address string) (epochstoragetypes.StakeEntry, bool)
	GetAllStakeEntriesCurrentForChainId(ctx sdk.Context, chainID string) []epochstoragetypes.StakeEntry
	EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error)
}

type DowntimeKeeper interface {
	GetParams(ctx sdk.Context) (params v1.Params)
}

type StakingKeeper interface {
	BondedRatio(ctx context.Context) (math.LegacyDec, error)
	BondDenom(ctx context.Context) (string, error)
	// Methods imported from bank should be defined here
}

type DualStakingKeeper interface {
	RewardProvidersAndDelegators(ctx sdk.Context, providerAddr string, chainID string, totalReward sdk.Coins, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributor bool) (providerReward sdk.Coins, err error)
	// Methods imported from bank should be defined here
}

type DistributionKeeper interface {
	GetParams(ctx sdk.Context) (params distributiontypes.Params)
	GetFeePool(ctx sdk.Context) (feePool distributiontypes.FeePool)
	SetFeePool(ctx sdk.Context, feePool distributiontypes.FeePool)
	// Methods imported from bank should be defined here
}
