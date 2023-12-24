package types

import (
	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
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
	// Methods imported from bank should be defined here
}

type EpochstorageKeeper interface {
	BlocksToSave(ctx sdk.Context, block uint64) (uint64, error)
	GetEpochStart(ctx sdk.Context) uint64
	IsEpochStart(ctx sdk.Context) bool
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
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
	RewardProvidersAndDelegators(ctx sdk.Context, providerAddr sdk.AccAddress, chainID string, totalReward math.Int, senderModule string, calcOnlyProvider bool, calcOnlyDelegators bool, calcOnlyContributer bool) (providerReward math.Int, totalRewards math.Int, err error)
}

type RewardsKeeper interface {
	AggregateRewards(ctx sdk.Context, provider, chainid string, adjustmentDenom uint64, rewards math.Int)
	ContributeToValidatorsAndCommunityPool(ctx sdk.Context, reward math.Int, senderModule string) (updatedReward math.Int, err error)
	FundCommunityPoolFromModule(ctx sdk.Context, amount math.Int, senderModule string) error
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
