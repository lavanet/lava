package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type PairingKeeper interface {
	UnstakeEntry(ctx sdk.Context, chainID string, creator string, unstakeDescription string) error
	CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin) (bool, error)
	VerifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (epoch uint64, providersType spectypes.Spec_ProvidersTypes, errorRet error)
	VerifyClientStake(ctx sdk.Context, chainID string, clientAddress sdk.Address, block uint64, epoch uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error)
	JailEntry(ctx sdk.Context, account sdk.AccAddress, chainID string, jailStartBlock uint64, jailBlocks uint64, bail sdk.Coin) error
	BailEntry(ctx sdk.Context, account sdk.AccAddress, chainID string, bail sdk.Coin) error
	SlashEntry(ctx sdk.Context, account sdk.AccAddress, chainID string, percentage sdk.Dec) (sdk.Coin, error)
	GetProjectData(ctx sdk.Context, developerKey sdk.AccAddress, chainID string, blockHeight uint64) (proj projectstypes.Project, errRet error)
}

type EpochstorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetEpochStart(ctx sdk.Context) uint64
	EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error)
	BlocksToSave(ctx sdk.Context, block uint64) (res uint64, err error)
	GetEarliestEpochStart(ctx sdk.Context) uint64
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64, err error)
	GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider sdk.AccAddress, epoch uint64) (entry *epochstoragetypes.StakeEntry, err error)
	GetStakeEntryForAllProvidersEpoch(ctx sdk.Context, chainID string, epoch uint64) (entrys *[]epochstoragetypes.StakeEntry, err error)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	PushFixatedParams(ctx sdk.Context, block uint64, limit uint64)
}

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive bool, found bool, providersType spectypes.Spec_ProvidersTypes)
	IsFinalizedBlock(ctx sdk.Context, chainID string, requestedBlock int64, latestBlock int64) bool
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}
