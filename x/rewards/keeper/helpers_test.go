package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	rewardsTypes "github.com/lavanet/lava/x/rewards/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	testStake             int64 = 100000
	testBalance           int64 = 100000000
	allocationPoolBalance int64 = 30000000000000
	feeCollectorName            = authtypes.FeeCollectorName
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

func newTester(t *testing.T, addValidator bool) *tester {
	ts := &tester{Tester: *common.NewTesterRaw(t)}

	if addValidator {
		ts.addValidators(1)
		val, _ := ts.GetAccount(common.VALIDATOR, 0)
		ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	}

	ts.plan = common.CreateMockPlan()
	coins := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, rewardsTypes.ProviderRewardsDistributionPool)
	_, monthlyProvidersPool := coins.Find(ts.BondDenom())
	ts.plan.Price.Amount = monthlyProvidersPool.Amount.QuoRaw(5).AddRaw(5)
	ts.plan.PlanPolicy.EpochCuLimit = monthlyProvidersPool.Amount.Uint64() * 5
	ts.plan.PlanPolicy.TotalCuLimit = monthlyProvidersPool.Amount.Uint64() * 5
	ts.plan.PlanPolicy.MaxProvidersToPair = 5
	ts.AddPlan(ts.plan.Index, ts.plan)
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	return ts
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.VALIDATOR, start+i, testBalance)
	}
}

func (ts *tester) feeCollector() sdk.AccAddress {
	return testkeeper.GetModuleAddress(feeCollectorName)
}

func (ts *tester) getPoolBalance(pool rewardsTypes.Pool, denom string) math.Int {
	coins := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, pool)
	return coins.AmountOf(denom)
}

func (ts *tester) iprpcAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// deductParticipationFees calculates the validators and community participation
// fees and returns the providers reward after deducting them
func (ts *tester) DeductParticipationFees(reward math.Int) (updatedReward math.Int, valParticipation math.Int, communityParticipation math.Int) {
	valPerc, communityPerc, err := ts.Keepers.Rewards.CalculateContributionPercentages(ts.Ctx, reward)
	require.Nil(ts.T, err)
	valParticipation = valPerc.MulInt(reward).TruncateInt()
	communityParticipation = communityPerc.MulInt(reward).TruncateInt()
	return reward.Sub(valParticipation).Sub(communityParticipation), valParticipation, communityParticipation
}

// makeBondedRatioNonZero makes BondedRatio() to be 0.25
// assumptions:
//  1. validators was created using addValidators(1) and TxCreateValidator
//  2. TxCreateValidator was used with init funds of 30000000000000/3
func (ts *tester) makeBondedRatioNonZero() {
	bondedRatio := ts.Keepers.StakingKeeper.BondedRatio(ts.Ctx)
	if bondedRatio.Equal(sdk.NewDecWithPrec(25, 2)) {
		return
	}

	// transfer the bonded pool tokens to staking module's bonded pool tokens (which is used to calculate BondedRatio)
	// in our testing env, the bonded pool account's address is sdk.AccAddress("bonded_tokens_pool")
	// in staking module's actual bonded pool, the AccAddress is different, so we manually transfer funds there
	stakingBondedPool := ts.Keepers.StakingKeeper.GetBondedPool(ts.Ctx)
	bondedPoolBalance := ts.Keepers.BankKeeper.GetBalance(ts.Ctx, testkeeper.GetModuleAddress(stakingtypes.BondedPoolName), ts.TokenDenom())
	require.False(ts.T, bondedPoolBalance.IsZero())
	err := ts.Keepers.BankKeeper.SendCoinsFromModuleToAccount(ts.Ctx, stakingtypes.BondedPoolName, stakingBondedPool.GetAddress(), sdk.NewCoins(bondedPoolBalance))
	require.Nil(ts.T, err)
	stakingBondedPoolBalance := ts.Keepers.BankKeeper.GetBalance(ts.Ctx, stakingBondedPool.GetAddress(), ts.TokenDenom())
	require.False(ts.T, stakingBondedPoolBalance.IsZero())

	bondedRatio = ts.Keepers.StakingKeeper.BondedRatio(ts.Ctx)
	require.True(ts.T, bondedRatio.Equal(sdk.NewDecWithPrec(25, 2))) // according to "valInitBalance", bondedRatio should be 0.25
}
