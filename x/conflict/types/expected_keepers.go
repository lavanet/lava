package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type PairingKeeper interface {
	UnstakeEntry(ctx sdk.Context, validator, chainID, creator, unstakeDescription string) error
	FreezeProvider(ctx sdk.Context, provider string, chainIDs []string, reason string) error
	CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin) (bool, error)
	VerifyPairingData(ctx sdk.Context, chainID string, block uint64) (epoch uint64, providersType spectypes.Spec_ProvidersTypes, errorRet error)
	JailEntry(ctx sdk.Context, address string, chainID string, jailStartBlock, jailBlocks uint64, bail sdk.Coin) error
	BailEntry(ctx sdk.Context, address string, chainID string, bail sdk.Coin) error
	SlashEntry(ctx sdk.Context, address string, chainID string, percentage sdk.Dec) (sdk.Coin, error)
	GetProjectData(ctx sdk.Context, developerKey sdk.AccAddress, chainID string, blockHeight uint64) (proj projectstypes.Project, errRet error)
}

type EpochstorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetEpochStart(ctx sdk.Context) uint64
	EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error)
	BlocksToSave(ctx sdk.Context, block uint64) (res uint64, err error)
	GetEarliestEpochStart(ctx sdk.Context) uint64
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider string, epoch uint64) (entry epochstoragetypes.StakeEntry, found bool)
	GetStakeEntryForAllProvidersEpoch(ctx sdk.Context, chainID string, epoch uint64) (entrys *[]epochstoragetypes.StakeEntry, err error)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address string) (value epochstoragetypes.StakeEntry, found bool)
	PushFixatedParams(ctx sdk.Context, block, limit uint64)
}

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive, found bool, providersType spectypes.Spec_ProvidersTypes)
	IsFinalizedBlock(ctx sdk.Context, chainID string, requestedBlock, latestBlock int64) bool
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

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
