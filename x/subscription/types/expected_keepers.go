package types

import (
	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
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
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address string) (epochstoragetypes.StakeEntry, bool)
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
	RewardProvidersAndDelegators(ctx sdk.Context, providerAddr string, chainID string, totalReward sdk.Coins, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributor bool) (providerReward sdk.Coins, totalRewards sdk.Coins, err error)
}

type RewardsKeeper interface {
	AggregateRewards(ctx sdk.Context, provider, chainid string, adjustment sdk.Dec, rewards math.Int)
	MaxRewardBoost(ctx sdk.Context) (res uint64)
	ContributeToValidatorsAndCommunityPool(ctx sdk.Context, reward sdk.Coin, senderModule string) (updatedReward sdk.Coin, err error)
	FundCommunityPoolFromModule(ctx sdk.Context, amount sdk.Coins, senderModule string) error
	IsIprpcSubscription(ctx sdk.Context, address string) bool
	AggregateCU(ctx sdk.Context, subscription, provider string, chainID string, cu uint64)
	CalculateValidatorsAndCommunityParticipationRewards(ctx sdk.Context, reward sdk.Coin) (validatorsCoins sdk.Coins, communityCoins sdk.Coins, err error)
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
