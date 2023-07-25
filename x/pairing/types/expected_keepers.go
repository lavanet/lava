package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
)

type SpecKeeper interface {
	// Methods imported from spec should be defined here
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive bool, found bool, providersType spectypes.Spec_ProvidersTypes)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool)
	GeolocationCount(ctx sdk.Context) uint64
	GetExpectedInterfacesForSpec(ctx sdk.Context, chainID string, mandatory bool) (expectedInterfaces map[epochstoragetypes.EndpointService]struct{}, err error)
	GetAllChainIDs(ctx sdk.Context) (chainIDs []string)
}

type EpochstorageKeeper interface {
	// Methods imported from epochStorage should be defined here
	// Methods imported from bank should be defined here
	GetParamForBlock(ctx sdk.Context, fixationKey string, block uint64, param any) error
	GetEpochStart(ctx sdk.Context) uint64
	GetEarliestEpochStart(ctx sdk.Context) uint64
	UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64)
	UnstakeHoldBlocksStatic(ctx sdk.Context, block uint64) (res uint64)
	IsEpochStart(ctx sdk.Context) (res bool)
	BlocksToSave(ctx sdk.Context, block uint64) (res uint64, erro error)
	BlocksToSaveRaw(ctx sdk.Context) (res uint64)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64, err error)
	GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64, erro error)
	PopUnstakeEntries(ctx sdk.Context, block uint64) (value []epochstoragetypes.StakeEntry)
	AppendUnstakeEntry(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry, unstakeHoldBlocks uint64) error
	ModifyUnstakeEntry(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	GetStakeStorageUnstake(ctx sdk.Context) (epochstoragetypes.StakeStorage, bool)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry, removeIndex uint64)
	AppendStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, idx uint64) error
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	UnstakeEntryByAddress(ctx sdk.Context, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	GetStakeStorageCurrent(ctx sdk.Context, chainID string) (epochstoragetypes.StakeStorage, bool)
	GetEpochStakeEntries(ctx sdk.Context, block uint64, chainID string) (entries []epochstoragetypes.StakeEntry, found bool, epochHash []byte)
	GetStakeEntryByAddressFromStorage(ctx sdk.Context, stakeStorage epochstoragetypes.StakeStorage, address sdk.AccAddress) (value epochstoragetypes.StakeEntry, found bool, index uint64)
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any)
	GetDeletedEpochs(ctx sdk.Context) []uint64
	EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error)
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

type ProjectsKeeper interface {
	ChargeComputeUnitsToProject(ctx sdk.Context, project projectstypes.Project, block uint64, cu uint64) (err error)
	GetProjectForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (proj projectstypes.Project, errRet error)
	GetProjectForBlock(ctx sdk.Context, projectID string, block uint64) (projectstypes.Project, error)
}

type SubscriptionKeeper interface {
	GetPlanFromSubscription(ctx sdk.Context, consumer string) (planstypes.Plan, error)
	ChargeComputeUnitsToSubscription(ctx sdk.Context, subscriptionOwner string, block uint64, cuAmount uint64) error
	GetSubscription(ctx sdk.Context, consumer string) (val subscriptiontypes.Subscription, found bool)
}

type PlanKeeper interface {
	GetAllPlanIndices(ctx sdk.Context) (val []string)
	FindPlan(ctx sdk.Context, index string, block uint64) (val planstypes.Plan, found bool)
}
