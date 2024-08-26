package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/testutil/common"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/utils/sigs"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	testStake             int64 = 100000
	testBalance           int64 = 100000000
	allocationPoolBalance int64 = 30000000000000
	feeCollectorName            = authtypes.FeeCollectorName
)

var (
	ibcDenom     string    = "uibc"
	minIprpcCost sdk.Coin  = sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(100))
	iprpcFunds   sdk.Coins = sdk.NewCoins(
		sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(1100)),
		sdk.NewCoin(ibcDenom, sdk.NewInt(500)),
	)
	mockSpec2 string = "mock2"
)

type tester struct {
	common.Tester
	plan  planstypes.Plan
	specs []spectypes.Spec
}

func newTester(t *testing.T, addValidator bool) *tester {
	ts := &tester{Tester: *common.NewTesterRaw(t)}

	if addValidator {
		ts.addValidators(1)
		val, _ := ts.GetAccount(common.VALIDATOR, 0)
		ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))
	}

	ts.plan = common.CreateMockPlan()
	coins := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, rewardstypes.ProviderRewardsDistributionPool)
	_, monthlyProvidersPool := coins.Find(ts.BondDenom())
	ts.plan.Price.Amount = monthlyProvidersPool.Amount.QuoRaw(5).AddRaw(5)
	ts.plan.PlanPolicy.EpochCuLimit = monthlyProvidersPool.Amount.Uint64() * 5
	ts.plan.PlanPolicy.TotalCuLimit = monthlyProvidersPool.Amount.Uint64() * 5
	ts.plan.PlanPolicy.MaxProvidersToPair = 5
	ts.AddPlan(ts.plan.Index, ts.plan)
	ts.specs = []spectypes.Spec{ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")}

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

func (ts *tester) getPoolBalance(pool rewardstypes.Pool, denom string) math.Int {
	coins := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, pool)
	return coins.AmountOf(denom)
}

func (ts *tester) iprpcAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// setupForIprpcTests performs the following to set a proper env for iprpc tests:
// 0. it assumes that ts.newTester(t) was already executed
// 1. setting IPRPC data
// 2. fund the iprpc pool (optional, can specify if to fund from the next month)
func (ts *tester) setupForIprpcTests(fundIprpcPool bool) {
	// add two consumers and buy subscriptions
	consumerAcc, consumer := ts.AddAccount(common.CONSUMER, 0, testBalance*10000)
	_, consumer2 := ts.AddAccount(common.CONSUMER, 1, testBalance*10000)
	_, err := ts.TxSubscriptionBuy(consumer, consumer, ts.plan.Index, 5, true, false)
	require.NoError(ts.T, err)
	_, err = ts.TxSubscriptionBuy(consumer2, consumer2, ts.plan.Index, 5, true, false)
	require.NoError(ts.T, err)

	// set iprpc data (only consumer is IPRPC eligible)
	_, err = ts.TxRewardsSetIprpcDataProposal(ts.iprpcAuthority(), minIprpcCost, []string{consumer})
	require.NoError(ts.T, err)

	// create a new spec
	spec2 := common.CreateMockSpec()
	spec2.Index = mockSpec2
	spec2.Name = mockSpec2
	ts.specs = append(ts.specs, ts.AddSpec(mockSpec2, spec2).Spec(mockSpec2))

	// add two providers and stake them both on the two specs
	pAcc, provider := ts.AddAccount(common.PROVIDER, 0, testBalance)
	p2Acc, provider2 := ts.AddAccount(common.PROVIDER, 1, testBalance)
	err = ts.StakeProvider(pAcc.GetVaultAddr(), provider, ts.specs[0], testStake)
	require.NoError(ts.T, err)
	err = ts.StakeProvider(pAcc.GetVaultAddr(), provider, ts.specs[1], testStake)
	require.NoError(ts.T, err)
	err = ts.StakeProvider(p2Acc.GetVaultAddr(), provider2, ts.specs[0], testStake)
	require.NoError(ts.T, err)
	err = ts.StakeProvider(p2Acc.GetVaultAddr(), provider2, ts.specs[1], testStake)
	require.NoError(ts.T, err)

	ts.AdvanceEpoch() // apply pairing

	// reset time to the start of the month
	startOfMonth := time.Date(ts.Ctx.BlockTime().Year(), ts.Ctx.BlockTime().Month(), 1, 0, 0, 0, 0, ts.Ctx.BlockTime().Location())
	ts.Ctx = ts.Ctx.WithBlockTime(startOfMonth)
	ts.GoCtx = sdk.WrapSDKContext(ts.Ctx)

	if fundIprpcPool {
		duration := uint64(1)
		err = ts.Keepers.BankKeeper.AddToBalance(consumerAcc.Addr, iprpcFunds.MulInt(sdk.NewIntFromUint64(duration)))
		require.NoError(ts.T, err)
		balanceBeforeFund := ts.GetBalances(consumerAcc.Addr)
		_, err = ts.TxRewardsFundIprpc(consumer, mockSpec2, duration, iprpcFunds)
		require.NoError(ts.T, err)
		expectedBalanceAfterFund := balanceBeforeFund.Sub(iprpcFunds.MulInt(math.NewIntFromUint64(duration))...)
		require.True(ts.T, ts.GetBalances(consumerAcc.Addr).IsEqual(expectedBalanceAfterFund))
		ts.AdvanceMonths(1).AdvanceEpoch() // fund only fund for next month, so advance a month
	}
}

// getConsumersForIprpcSubTest is a helper function specifically for the TestIprpcEligibleSubscriptions unit test
// this function returns two consumer addresses to test depending on the input mode:
//
//	SUB_OWNERS                 = 0
//	DEVELOPERS_ADMIN_PROJECT   = 1
//	DEVELOPERS_REGULAR_PROJECT = 2
//
// this function assumes that ts.setupForIprpcTests ran before it
func (ts *tester) getConsumersForIprpcSubTest(mode int) (sigs.Account, sigs.Account) {
	switch mode {
	case 0:
		sub1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
		sub2Acc, _ := ts.GetAccount(common.CONSUMER, 1)
		return sub1Acc, sub2Acc
	case 1:
		sub1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
		sub2Acc, _ := ts.GetAccount(common.CONSUMER, 1)
		adminDev1, _ := ts.AddAccount(common.CONSUMER, 2, testBalance*10000)
		adminDev2, _ := ts.AddAccount(common.CONSUMER, 3, testBalance*10000)
		res1, err := ts.QueryProjectDeveloper(sub1Acc.Addr.String())
		require.NoError(ts.T, err)
		res2, err := ts.QueryProjectDeveloper(sub2Acc.Addr.String())
		require.NoError(ts.T, err)
		err = ts.TxProjectAddKeys(res1.Project.Index, sub1Acc.Addr.String(), types.ProjectDeveloperKey(adminDev1.Addr.String()))
		require.NoError(ts.T, err)
		err = ts.TxProjectAddKeys(res2.Project.Index, sub2Acc.Addr.String(), types.ProjectDeveloperKey(adminDev2.Addr.String()))
		require.NoError(ts.T, err)
		return adminDev1, adminDev2
	case 2:
		sub1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
		sub2Acc, _ := ts.GetAccount(common.CONSUMER, 1)
		dev1, _ := ts.AddAccount(common.CONSUMER, 2, testBalance*10000)
		dev2, _ := ts.AddAccount(common.CONSUMER, 3, testBalance*10000)
		err := ts.TxSubscriptionAddProject(sub1Acc.Addr.String(), types.ProjectData{
			Name: "test1", Enabled: true, ProjectKeys: []types.ProjectKey{{Key: dev1.Addr.String(), Kinds: 2}},
		})
		require.NoError(ts.T, err)
		err = ts.TxSubscriptionAddProject(sub2Acc.Addr.String(), types.ProjectData{
			Name: "test2", Enabled: true, ProjectKeys: []types.ProjectKey{{Key: dev2.Addr.String(), Kinds: 2}},
		})
		require.NoError(ts.T, err)
		return dev1, dev2
	}

	return sigs.Account{}, sigs.Account{}
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
