package types

import (
	context "context"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	dualstakingtypes "github.com/lavanet/lava/v3/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	fixationtypes "github.com/lavanet/lava/v3/x/fixationstore/types"
	planstypes "github.com/lavanet/lava/v3/x/plans/types"
	projectstypes "github.com/lavanet/lava/v3/x/projects/types"
	rewardstypes "github.com/lavanet/lava/v3/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/v3/x/timerstore/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderPool, recipientPool string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type EpochstorageKeeper interface {
	BlocksToSave(ctx sdk.Context, block uint64) (uint64, error)
	GetEpochStart(ctx sdk.Context) uint64
	IsEpochStart(ctx sdk.Context) bool
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64)
	GetStakeEntryCurrent(ctx sdk.Context, chainID string, address string) (epochstoragetypes.StakeEntry, bool)
	GetAllStakeEntriesCurrentForChainId(ctx sdk.Context, chainID string) []epochstoragetypes.StakeEntry
	GetMetadata(ctx sdk.Context, provider string) (epochstoragetypes.ProviderMetadata, error)
	// Methods imported from epochstorage should be defined here
}

type ProjectsKeeper interface {
	CreateAdminProject(ctx sdk.Context, subscriptionAddress string, plan planstypes.Plan) error
	CreateProject(ctx sdk.Context, subscriptionAddress string, projectData projectstypes.ProjectData, plan planstypes.Plan) error
	DeleteProject(ctx sdk.Context, creator, index string) error
	SnapshotSubscriptionProjects(ctx sdk.Context, subscriptionAddr string, block uint64)
	GetAllProjectsForSubscription(ctx sdk.Context, subscription string) []string
	// Methods imported from projectskeeper should be defined here
}

type PlansKeeper interface {
	GetPlan(ctx sdk.Context, index string) (planstypes.Plan, bool)
	DelPlan(ctx sdk.Context, index string) error
	FindPlan(ctx sdk.Context, index string, block uint64) (val planstypes.Plan, found bool)
	PutPlan(ctx sdk.Context, index string, block uint64)
	GetAllPlanIndices(ctx sdk.Context) []string
	// Methods imported from planskeeper should be defined here
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationtypes.FixationStore
}

type TimerStoreKeeper interface {
	NewTimerStoreBeginBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore
	NewTimerStoreEndBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore
}

type DualStakingKeeper interface {
	RewardProvidersAndDelegators(ctx sdk.Context, providerAddr string, chainID string, totalReward sdk.Coins, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributor bool) (providerReward sdk.Coins, err error)
	GetDelegation(ctx sdk.Context, delegator, provider string) (dualstakingtypes.Delegation, bool)
	DelegateFull(ctx sdk.Context, delegator string, validator string, provider string, amount sdk.Coin, stake bool) error
	DelegatorRewards(goCtx context.Context, req *dualstakingtypes.QueryDelegatorRewardsRequest) (res *dualstakingtypes.QueryDelegatorRewardsResponse, err error)
	Delegate(ctx sdk.Context, delegator, provider string, amount sdk.Coin, stake bool) error
}

type RewardsKeeper interface {
	AggregateRewards(ctx sdk.Context, provider, chainid string, adjustment sdk.Dec, rewards math.Int)
	MaxRewardBoost(ctx sdk.Context) (res uint64)
	ContributeToValidatorsAndCommunityPool(ctx sdk.Context, reward sdk.Coin, senderModule string) (updatedReward sdk.Coin, err error)
	FundCommunityPoolFromModule(ctx sdk.Context, amount sdk.Coins, senderModule string) error
	IsIprpcSubscription(ctx sdk.Context, address string) bool
	AggregateCU(ctx sdk.Context, subscription, provider string, chainID string, cu uint64)
	CalculateValidatorsAndCommunityParticipationRewards(ctx sdk.Context, reward sdk.Coin) (validatorsCoins sdk.Coins, communityCoins sdk.Coins, err error)
	TotalPoolTokens(ctx sdk.Context, pool rewardstypes.Pool) sdk.Coins
	SpecEmissionParts(ctx sdk.Context) (emissions []rewardstypes.SpecEmissionPart)
	SpecTotalPayout(ctx sdk.Context, totalMonthlyPayout math.Int, totalProvidersBaseRewards sdk.Dec, spec rewardstypes.SpecEmissionPart) math.LegacyDec
	GetIprpcRewardsCurrentId(ctx sdk.Context) uint64
	GetIprpcReward(ctx sdk.Context, id uint64) (val rewardstypes.IprpcReward, found bool)
	AllocationPoolMonthsLeft(ctx sdk.Context) int64
	GetCommunityTax(ctx sdk.Context) math.LegacyDec
	DistributeMonthlyBonusRewards(ctx sdk.Context)
	GetLastRewardsBlock(ctx sdk.Context) (rewardsDistributionBlock uint64, after24HoursBlock uint64, err error)
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	GetDelegation(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, found bool)
	GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator
}

type SpecKeeper interface {
	GetContributorReward(ctx sdk.Context, chainId string) (contributors []sdk.AccAddress, percentage math.LegacyDec)
}
