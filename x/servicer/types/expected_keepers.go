package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	usertypes "github.com/lavanet/lava/x/user/types"
)

type EvidenceKeeper interface {
	// Methods imported from evidence should be defined here
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
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

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, specName string) (bool, bool, uint64)
	IsSpecIDFoundAndActive(ctx sdk.Context, id uint64) (bool, bool)
	GetSpec(ctx sdk.Context, id uint64) (val spectypes.Spec, found bool)
}

type UserKeeper interface {
	GetSpecStakeStorage(sdk.Context, string) (usertypes.SpecStakeStorage, bool)
	GetCoinsPerCU(ctx sdk.Context) (res float64)
	GetUnstakingUsersForSpec(sdk.Context, usertypes.SpecName) ([]usertypes.UserStake, []int)
	BurnUserStake(sdk.Context, usertypes.SpecName, sdk.AccAddress, sdk.Coin, bool) (bool, error)
	EnforceUserCUsUsageInSession(sdk.Context, *usertypes.UserStake, uint64) error
	BlocksToSave(sdk.Context) uint64
	SessionsToSave(ctx sdk.Context) (res uint64)
	IsSessionStart(ctx sdk.Context) (res bool)
	SessionBlocks(ctx sdk.Context) (res uint64)
}

type EpochStorageKeeper interface {
	// Methods imported from bank should be defined here
	GetEpochStart(ctx sdk.Context) uint64
	GetEarliestEpochStart(ctx sdk.Context) uint64
	UnstakeHoldBlocks(ctx sdk.Context) (res uint64)
	IsEpochStart(ctx sdk.Context) (res bool)
	BlocksToSave(ctx sdk.Context) (res uint64)
	PopUnstakeEntries(ctx sdk.Context, storageType string, block uint64) (value []epochstoragetypes.StakeEntry)
	AppendUnstakeEntry(ctx sdk.Context, storageType string, stakeEntry epochstoragetypes.StakeEntry)
	GetStakeStorageUnstake(ctx sdk.Context, storageType string) (epochstoragetypes.StakeStorage, bool)
	ModifyStakeEntry(ctx sdk.Context, storageType string, chainID string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	AppendStakeEntry(ctx sdk.Context, storageType string, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	RemoveStakeEntry(ctx sdk.Context, storageType string, chainID string, idx uint64)
	StakeEntryByAddress(ctx sdk.Context, storageType string, chainID string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	GetStakeStorageCurrent(ctx sdk.Context, storageType string, chainID string) (epochstoragetypes.StakeStorage, bool)
}
