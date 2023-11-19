package types

import (
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/fixationstore"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/lavanet/lava/x/timerstore"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	BurnCoins(ctx sdk.Context, name string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderPool, recipientPool string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type EpochstorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64)
	UnstakeHoldBlocksStatic(ctx sdk.Context, block uint64) (res uint64)
	GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider sdk.AccAddress, epoch uint64) (entry *epochstoragetypes.StakeEntry, err error)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64)

	// Methods imported from epochstorage should be defined here
}

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (active bool, found bool, providerType spectypes.Spec_ProvidersTypes)
	GetContributorReward(ctx sdk.Context, chainId string) (contributors []sdk.AccAddress, percentage math.LegacyDec)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool)
}

type StakingKeeper interface {
	UnbondingTime(ctx sdk.Context) time.Duration
	GetAllDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress) []stakingtypes.Delegation
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	GetValidatorDelegations(ctx sdk.Context, valAddr sdk.ValAddress) (delegations []stakingtypes.Delegation)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstore.FixationStore
}

type TimerStoreKeeper interface {
	NewTimerStoreBeginBlock(storeKey storetypes.StoreKey, prefix string) *timerstore.TimerStore
}
