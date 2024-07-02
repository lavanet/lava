package types

import (
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	fixationstoretypes "github.com/lavanet/lava/x/fixationstore/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

type SpecKeeper interface {
	// Methods imported from spec should be defined here
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive, found bool, providersType spectypes.Spec_ProvidersTypes)
	GetExpandedSpec(ctx sdk.Context, index string) (val spectypes.Spec, err error)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool) // this spec is unexpanded don;t use for collections work
	GetExpectedServicesForExpandedSpec(expandedSpec spectypes.Spec, mandatory bool) map[epochstoragetypes.EndpointService]struct{}
	GetAllChainIDs(ctx sdk.Context) (chainIDs []string)
	GetMinStake(ctx sdk.Context, chainID string) sdk.Coin
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
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64, erro error)
	PopUnstakeEntries(ctx sdk.Context, block uint64) (value []epochstoragetypes.StakeEntry)
	AppendUnstakeEntry(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry, unstakeHoldBlocks uint64)
	ModifyUnstakeEntry(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry)
	GetStakeStorageUnstake(ctx sdk.Context) (epochstoragetypes.StakeStorage, bool)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	AppendStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, address string) error
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address string) (epochstoragetypes.StakeEntry, bool)
	UnstakeEntryByAddress(ctx sdk.Context, address string) (value epochstoragetypes.StakeEntry, found bool)
	GetStakeStorageCurrent(ctx sdk.Context, chainID string) (value epochstoragetypes.StakeStorage, found bool)
	GetEpochStakeEntries(ctx sdk.Context, block uint64, chainID string) (entries []epochstoragetypes.StakeEntry, found bool, epochHash []byte)
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64)
	AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any)
	GetDeletedEpochs(ctx sdk.Context) []uint64
	EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error)
	EpochBlocksRaw(ctx sdk.Context) (res uint64)
	GetUnstakeHoldBlocks(ctx sdk.Context, chainID string) uint64
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
	ChargeComputeUnitsToProject(ctx sdk.Context, project projectstypes.Project, block, cu uint64) (err error)
	GetProjectForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (proj projectstypes.Project, errRet error)
	GetProjectForBlock(ctx sdk.Context, projectID string, block uint64) (projectstypes.Project, error)
}

type SubscriptionKeeper interface {
	GetPlanFromSubscription(ctx sdk.Context, consumer string, block uint64) (planstypes.Plan, error)
	ChargeComputeUnitsToSubscription(ctx sdk.Context, subscriptionOwner string, block, cuAmount uint64) (subscriptiontypes.Subscription, error)
	GetSubscription(ctx sdk.Context, consumer string) (val subscriptiontypes.Subscription, found bool)
	GetAllSubTrackedCuIndices(ctx sdk.Context, sub string) []string
	GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, block uint64) (cu uint64, found bool, key string)
	CalcTotalMonthlyReward(ctx sdk.Context, totalAmount math.Int, trackedCu uint64, totalCuUsedBySub uint64) math.Int
	AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cu uint64, block uint64, userSpec bool) error
	GetAllSubscriptionsIndices(ctx sdk.Context) []string
	AppendAdjustment(ctx sdk.Context, consumer string, provider string, totalConsumerUsage uint64, usageWithThisProvider uint64)
	CalculateParticipationFees(ctx sdk.Context, reward sdk.Coin) (sdk.Coins, sdk.Coins, error)
	GetAllClusters(ctx sdk.Context) []string
}

type PlanKeeper interface {
	GetAllPlanIndices(ctx sdk.Context) (val []string)
	FindPlan(ctx sdk.Context, index string, block uint64) (val planstypes.Plan, found bool)
}

type DowntimeKeeper interface {
	GetDowntimeFactor(ctx sdk.Context, epochStartBlock uint64) uint64
	GetParams(ctx sdk.Context) (params v1.Params)
}

type DualstakingKeeper interface {
	RewardProvidersAndDelegators(ctx sdk.Context, providerAddr string, chainID string, totalReward sdk.Coins, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributer bool) (providerReward sdk.Coins, totalRewards sdk.Coins, err error)
	DelegateFull(ctx sdk.Context, delegator string, validator string, provider string, chainID string, amount sdk.Coin) error
	UnbondFull(ctx sdk.Context, delegator string, validator string, provider string, chainID string, amount sdk.Coin, unstake bool) error
	GetProviderDelegators(ctx sdk.Context, provider string, epoch uint64) ([]dualstakingtypes.Delegation, error)
	MinSelfDelegation(ctx sdk.Context) sdk.Coin
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstoretypes.FixationStore
}

type TimerStoreKeeper interface {
	NewTimerStoreBeginBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore
}

type StakingKeeper interface {
	GetAllDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress) []stakingtypes.Delegation
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	BondDenom(ctx sdk.Context) string
	GetUnbondingDelegations(ctx sdk.Context, delegator sdk.AccAddress, maxRetrieve uint16) (unbondingDelegations []stakingtypes.UnbondingDelegation)
	SlashUnbondingDelegation(ctx sdk.Context, unbondingDelegation stakingtypes.UnbondingDelegation, infractionHeight int64, slashFactor sdk.Dec) (totalSlashAmount math.Int)
	Undelegate(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdk.Dec) (time.Time, error)
}
