package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type SpecKeeper interface {
	// Methods imported from spec should be defined here
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive bool, found bool)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool)
	GeolocationCount(ctx sdk.Context) uint64
	GetExpectedInterfacesForSpec(ctx sdk.Context, chainID string) map[string]bool
}

type EpochstorageKeeper interface {
	// Methods imported from epochStorage should be defined here
	// Methods imported from bank should be defined here
	GetParamForBlock(ctx sdk.Context, fixationKey string, block uint64, param any) error
	GetEpochStart(ctx sdk.Context) uint64
	GetEarliestEpochStart(ctx sdk.Context) uint64
	UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64)
	IsEpochStart(ctx sdk.Context) (res bool)
	BlocksToSave(ctx sdk.Context, block uint64) (res uint64, erro error)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64, err error)
	GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64, erro error)
	PopUnstakeEntries(ctx sdk.Context, storageType string, block uint64) (value []epochstoragetypes.StakeEntry)
	AppendUnstakeEntry(ctx sdk.Context, storageType string, stakeEntry epochstoragetypes.StakeEntry) error
	ModifyUnstakeEntry(ctx sdk.Context, storageType string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	GetStakeStorageUnstake(ctx sdk.Context, storageType string) (epochstoragetypes.StakeStorage, bool)
	ModifyStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	AppendStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	RemoveStakeEntryCurrent(ctx sdk.Context, storageType string, chainID string, idx uint64)
	GetStakeEntryByAddressCurrent(ctx sdk.Context, storageType string, chainID string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	UnstakeEntryByAddress(ctx sdk.Context, storageType string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	GetStakeStorageCurrent(ctx sdk.Context, storageType string, chainID string) (epochstoragetypes.StakeStorage, bool)
	GetEpochStakeEntries(ctx sdk.Context, block uint64, storageType string, chainID string) (entries []epochstoragetypes.StakeEntry, found bool)
	GetStakeEntryByAddressFromStorage(ctx sdk.Context, stakeStorage epochstoragetypes.StakeStorage, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetStakeEntryForClientEpoch(ctx sdk.Context, chainID string, selectedClient sdk.AccAddress, epoch uint64) (entry *epochstoragetypes.StakeEntry, err error)
	BypassCurrentAndAppendNewEpochStakeEntry(ctx sdk.Context, storageType string, chainID string, stakeEntry epochstoragetypes.StakeEntry) (added bool, err error)
	AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any)
	GetDeletedEpochs(ctx sdk.Context) []uint64
	GetAverageBlockTime(ctx sdk.Context) uint64
}

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error
	// Methods imported from bank should be defined here
}
