package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	rewardstypes "github.com/lavanet/lava/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// TestFundIprpcTX tests the FundIprpc TX functionality in funding the IPRPC pool
// Scenarios:
//  1. fund IPRPC with different periods (1m,3m,12m) and different denominations (also combinations)
//     -> pool balance and iprpc reward should be as expected
func TestFundIprpcTX(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	err := ts.Keepers.BankKeeper.AddToBalance(consumerAcc.Addr, iprpcFunds)
	require.NoError(ts.T, err)

	type fundIprpcData struct {
		spec     string
		duration uint64
		fund     sdk.Coins
	}

	// we fund as follows (to all we add the min IPRPC price. the description below is the funds that go to the pool):
	// - 10ulava,            1 month,   mockspec
	// - 50uibc,             1 month,   mockspec
	// - 90ulava + 30uibc,   3 months,  mockspec2
	// - 130uibc,            3 months,  mockspec
	// - 10ulava + 120uibc, 12 months,  mockspec2
	fundIprpcTXsData := []fundIprpcData{
		{spec: ts.specs[0].Index, duration: 1, fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), math.NewInt(10+minIprpcCost.Amount.Int64())),
		)},
		{spec: ts.specs[0].Index, duration: 1, fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), math.NewInt(minIprpcCost.Amount.Int64())),
			sdk.NewCoin(ibcDenom, math.NewInt(50)),
		)},
		{spec: ts.specs[1].Index, duration: 3, fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), math.NewInt(90+minIprpcCost.Amount.Int64()*3)),
			sdk.NewCoin(ibcDenom, math.NewInt(30)),
		)},
		{spec: ts.specs[0].Index, duration: 3, fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), math.NewInt(minIprpcCost.Amount.Int64()*3)),
			sdk.NewCoin(ibcDenom, math.NewInt(130)),
		)},
		{spec: ts.specs[1].Index, duration: 12, fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), math.NewInt(10+minIprpcCost.Amount.Int64()*12)),
			sdk.NewCoin(ibcDenom, math.NewInt(120)),
		)},
	}

	for _, txData := range fundIprpcTXsData {
		_, err = ts.TxRewardsFundIprpc(consumer, txData.spec, txData.duration, txData.fund)
		require.NoError(t, err)
	}

	// Expected total IPRPC pool balance: 110ulava (=10+90+10) and 330uibc
	iprpcTotalBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, rewardstypes.IprpcPoolName)
	expectedIprpcTotalBalance := sdk.NewCoins(
		sdk.NewCoin(ts.BondDenom(), math.NewInt(110)),
		sdk.NewCoin(ibcDenom, math.NewInt(330)),
	)
	require.True(t, expectedIprpcTotalBalance.IsEqual(iprpcTotalBalance))

	// Expected IPRPC rewards (by months, first month is skipped):
	//   1. mockspec: 10ulava + 180uibc(=50+130), mockspec2: 100ulava(=10+90) + 150uibc(=30+120)
	//   2. mockspec: 130uibc, mockspec2: 100ulava(=10+90) + 150uibc(=30+120)
	//   3. mockspec: 130uibc, mockspec2: 100ulava(=10+90) + 150uibc(=30+120)
	//   4-12. mockspec: nothing, mockspec2: 10ulava + 120uibc
	iprpcRewards := ts.Keepers.Rewards.GetAllIprpcReward(ts.Ctx)
	require.Len(t, iprpcRewards, 12)
	for i := range iprpcRewards {
		var expectedSpecFunds []rewardstypes.Specfund
		switch i {
		case 0:
			// first month
			expectedSpecFunds = []rewardstypes.Specfund{
				{
					Spec: ts.specs[0].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ts.BondDenom(), math.NewInt(10)),
						sdk.NewCoin(ibcDenom, math.NewInt(180)),
					),
				},
				{
					Spec: ts.specs[1].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ts.BondDenom(), math.NewInt(100)),
						sdk.NewCoin(ibcDenom, math.NewInt(150)),
					),
				},
			}
		case 1:
			// second month
			expectedSpecFunds = []rewardstypes.Specfund{
				{
					Spec: ts.specs[0].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ibcDenom, math.NewInt(130)),
					),
				},
				{
					Spec: ts.specs[1].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ts.BondDenom(), math.NewInt(100)),
						sdk.NewCoin(ibcDenom, math.NewInt(150)),
					),
				},
			}
		case 2:
			// 3rd month
			expectedSpecFunds = []rewardstypes.Specfund{
				{
					Spec: ts.specs[0].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ibcDenom, math.NewInt(130)),
					),
				},
				{
					Spec: ts.specs[1].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ts.BondDenom(), math.NewInt(100)),
						sdk.NewCoin(ibcDenom, math.NewInt(150)),
					),
				},
			}
		default:
			// rest of months (until 12)
			expectedSpecFunds = []rewardstypes.Specfund{
				{
					Spec: ts.specs[1].Index, Fund: sdk.NewCoins(
						sdk.NewCoin(ts.BondDenom(), math.NewInt(10)),
						sdk.NewCoin(ibcDenom, math.NewInt(120)),
					),
				},
			}
		}
		require.Equal(t, i+1, int(iprpcRewards[i].Id))
		require.ElementsMatch(t, expectedSpecFunds, iprpcRewards[i].SpecFunds)
	}
}

// TestIprpcProviderRewardQuery tests the IprpcProviderReward query functionality
// Scenarios:
// 1. two providers provide different CU for two consumers, which only one is IPRPC eligible -> query should return expected reward
// 2. advance a month, fund the pool and check the query's output again (without sending relays -> provider rewards should be empty)
func TestIprpcProviderRewardQuery(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(true) // setup funds IPRPC for mock2 spec

	// get consumers and providers (note, only c1 is IPRPC eligible)
	c1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
	c2Acc, _ := ts.GetAccount(common.CONSUMER, 1)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)

	// send relays from both consumers to both providers
	type relayInfo struct {
		consumer sigs.Account
		provider string
		cu       uint64
	}
	relaysInfo := []relayInfo{
		{consumer: c1Acc, provider: p1, cu: 100},
		{consumer: c2Acc, provider: p1, cu: 150},
		{consumer: c1Acc, provider: p2, cu: 400},
		{consumer: c2Acc, provider: p2, cu: 450},
	}
	for _, info := range relaysInfo {
		msg := ts.SendRelay(info.provider, info.consumer, []string{ts.specs[1].Index}, info.cu)
		_, err := ts.Servers.PairingServer.RelayPayment(ts.GoCtx, &msg)
		require.NoError(t, err)
	}

	// check the IprpcProviderReward query
	// p1 should get 1/5 of the reward and p2 4/5 of the reward (p1 relative serviced CU is 100/500)
	// note: setupForIprpcTests() funds the IPRPC pool with 1000ulava and 500uibc
	type providerRewards struct {
		provider string
		fund     sdk.Coins
	}
	expectedProviderRewards := []providerRewards{
		{provider: p1, fund: iprpcFunds.Sub(minIprpcCost).QuoInt(sdk.NewInt(5))},
		{provider: p2, fund: iprpcFunds.Sub(minIprpcCost).MulInt(sdk.NewInt(4)).QuoInt(sdk.NewInt(5))},
	}
	for _, expectedProviderReward := range expectedProviderRewards {
		res, err := ts.QueryRewardsIprpcProviderReward(expectedProviderReward.provider)
		require.NoError(t, err)
		require.ElementsMatch(t, expectedProviderReward.fund, res.SpecFunds[0].Fund) // taking 0 index because there's a single spec
	}

	// advance month to distribute monthly rewards
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	// check that rewards were distributed as expected
	for _, expectedProviderReward := range expectedProviderRewards {
		res2, err := ts.QueryDualstakingDelegatorRewards(expectedProviderReward.provider, expectedProviderReward.provider, ts.specs[1].Index)
		require.NoError(t, err)
		require.True(t, res2.Rewards[0].Amount.IsEqual(expectedProviderReward.fund)) // taking 0 index because there are no delegators
	}
}

// TestIprpcSpecRewardQuery tests the IprpcSpecReward query functionality
// Scenarios:
// 0. assume IPRPC pool is funded with two denoms over different periods of vesting with two specs
// 1. query with no args should return all
// 2. query with arg should return the IPRPC rewards for the specific spec
// 3. advance a month, this month reward should transfer to next month -> query should return updated iprpc pool balance
// 4. make a provider provide service, advance a month to get his reward -> query should return updated iprpc pool balance
func TestIprpcSpecRewardQuery(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(true) // setup funds IPRPC for mock2 spec for 1 month and advances a month

	_, consumer := ts.GetAccount(common.CONSUMER, 0)

	// do another funding for mockspec and mock2 for 3 months
	// Expected funds:
	// first month: mock2 - 500uibc + 3000ulava, mockspec - 100000ulava
	// second + third month: mock2 - 2000ulava, mockspec - 100000ulava
	duration := int64(3)
	minIprpcCostForFund := minIprpcCost.Amount.MulRaw(duration)
	_, err := ts.TxRewardsFundIprpc(consumer, ts.specs[0].Index, uint64(duration),
		sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000).Add(minIprpcCostForFund))))
	require.NoError(ts.T, err)

	_, err = ts.TxRewardsFundIprpc(consumer, ts.specs[1].Index, uint64(duration),
		sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(2000).Add(minIprpcCostForFund))))
	require.NoError(ts.T, err)

	expectedResults := []rewardstypes.IprpcReward{
		{
			Id: 1, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[1].Index, Fund: sdk.NewCoins(sdk.NewCoin(ibcDenom, sdk.NewInt(500)),
					sdk.NewCoin(ts.BondDenom(), sdk.NewInt(1000)))},
			},
		},
		{
			Id: 2, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
				{Spec: ts.specs[1].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(2000)))},
			},
		},
		{
			Id: 3, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
				{Spec: ts.specs[1].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(2000)))},
			},
		},
		{
			Id: 4, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
				{Spec: ts.specs[1].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(2000)))},
			},
		},
	}

	// query with no args
	res, err := ts.QueryRewardsIprpcSpecReward("")
	require.NoError(t, err)
	require.ElementsMatch(t, expectedResults, res.IprpcRewards)

	// query with arg = mockspec
	mockspecExpectedResults := []rewardstypes.IprpcReward{
		{
			Id: 2, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
			},
		},
		{
			Id: 3, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
			},
		},
		{
			Id: 4, SpecFunds: []rewardstypes.Specfund{
				{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
			},
		},
	}
	res, err = ts.QueryRewardsIprpcSpecReward(ts.specs[0].Index)
	require.NoError(t, err)
	require.ElementsMatch(t, mockspecExpectedResults, res.IprpcRewards)

	// advance a month with no providers getting rewarded this month's reward should transfer to the next month
	// 2nd month expected funds: mockspec - 100000ulava, mock2 - 3000ulava(=2000+1000) and 500uibc
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	afterMonthExpectedResults := expectedResults[1:]
	afterMonthExpectedResults[0].SpecFunds = []rewardstypes.Specfund{
		{Spec: ts.specs[0].Index, Fund: sdk.NewCoins(sdk.NewCoin(ts.BondDenom(), sdk.NewInt(100000)))},
		{Spec: ts.specs[1].Index, Fund: sdk.NewCoins(
			sdk.NewCoin(ts.BondDenom(), sdk.NewInt(3000)),
			sdk.NewCoin(ibcDenom, sdk.NewInt(500)),
		)},
	}
	res, err = ts.QueryRewardsIprpcSpecReward("")
	require.NoError(t, err)
	require.Len(t, res.IprpcRewards, len(afterMonthExpectedResults))
	for i := range res.IprpcRewards {
		require.Equal(t, afterMonthExpectedResults[i].Id, res.IprpcRewards[i].Id)
		require.ElementsMatch(t, afterMonthExpectedResults[i].SpecFunds, res.IprpcRewards[i].SpecFunds)
	}

	// make a provider provide some service to an IPRPC eligible subscription
	c1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	relay := ts.SendRelay(p1, c1Acc, []string{ts.specs[1].Index}, 100)
	_, err = ts.Servers.PairingServer.RelayPayment(ts.GoCtx, &relay)
	require.NoError(t, err)

	// advance month to distribute monthly rewards
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	// check that the latest iprpc object has been deleted
	afterProviderServiceExpectedResults := afterMonthExpectedResults[1:]
	res, err = ts.QueryRewardsIprpcSpecReward("")
	require.NoError(t, err)
	require.ElementsMatch(t, afterProviderServiceExpectedResults, res.IprpcRewards)
}

// TestIprpcRewardObjectsUpdate tests that the IPRPC reward objects' management works as expected:
// Scenarios:
// 0. fund iprpc pool for 2 months, current should be 0 and first iprpc reward should be with id=1 (fund is always for the next month)
// 1. there is no service to eligible subscriptions, month passes -> current shouldn't increment and there should be no IPRPC object
// 2. provider provides service for eligible subscription, month passes -> current should increment by 1 and a new IPRPC reward should be created with id=current
func TestIprpcRewardObjectsUpdate(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)

	// fund iprpc pool
	duration := uint64(2)
	iprpcCost := sdk.NewCoin(ts.BondDenom(), minIprpcCost.Amount.MulRaw(int64(duration)))
	fundForIprpc := iprpcFunds
	err := ts.Keepers.BankKeeper.AddToBalance(consumerAcc.Addr, fundForIprpc)
	require.NoError(ts.T, err)
	_, err = ts.TxRewardsFundIprpc(consumer, mockSpec2, duration, iprpcFunds)
	require.NoError(ts.T, err)

	// check there are 2 iprpc reward object, and the first one is with id=1
	currentIprpcRewardId := ts.Keepers.Rewards.GetIprpcRewardsCurrent(ts.Ctx)
	require.Equal(t, uint64(0), currentIprpcRewardId)
	res, err := ts.QueryRewardsIprpcSpecReward(mockSpec2)
	require.NoError(t, err)
	require.Len(t, res.IprpcRewards, 2)
	require.Equal(t, uint64(0), res.CurrentMonthId)
	for i := range res.IprpcRewards {
		require.Equal(t, uint64(i+1), res.IprpcRewards[i].Id)
		require.True(t, fundForIprpc.Sub(iprpcCost).IsEqual(res.IprpcRewards[i].SpecFunds[0].Fund))
	}

	// advance month to reach the first iprpc reward (first object is with id=1)
	// there should still be the exact two objects as before
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	currentIprpcRewardId = ts.Keepers.Rewards.GetIprpcRewardsCurrent(ts.Ctx)
	require.Equal(t, uint64(1), currentIprpcRewardId)
	res, err = ts.QueryRewardsIprpcSpecReward(mockSpec2)
	require.NoError(t, err)
	require.Len(t, res.IprpcRewards, 2)
	require.Equal(t, uint64(1), res.CurrentMonthId)
	for i := range res.IprpcRewards {
		require.Equal(t, uint64(i+1), res.IprpcRewards[i].Id)
		require.True(t, fundForIprpc.Sub(iprpcCost).IsEqual(res.IprpcRewards[i].SpecFunds[0].Fund))
	}

	// advance month without any provider service, there should be one IPRPC object with combined reward
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	currentIprpcRewardId = ts.Keepers.Rewards.GetIprpcRewardsCurrent(ts.Ctx)
	require.Equal(t, uint64(2), currentIprpcRewardId)
	res, err = ts.QueryRewardsIprpcSpecReward(mockSpec2)
	require.NoError(t, err)
	require.Len(t, res.IprpcRewards, 1)
	require.Equal(t, uint64(2), res.CurrentMonthId)
	require.True(t, fundForIprpc.Sub(iprpcCost).MulInt(sdk.NewInt(2)).IsEqual(res.IprpcRewards[0].SpecFunds[0].Fund))

	// make a provider service an IPRPC eligible consumer and advance a month
	// there should be no iprpc rewards objects
	c1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	relay := ts.SendRelay(p1, c1Acc, []string{ts.specs[1].Index}, 100)
	_, err = ts.Servers.PairingServer.RelayPayment(ts.GoCtx, &relay)
	require.NoError(t, err)
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	res, err = ts.QueryRewardsIprpcSpecReward(mockSpec2)
	require.NoError(t, err)
	require.Len(t, res.IprpcRewards, 0)
	require.Equal(t, uint64(3), res.CurrentMonthId)
}

// TestIprpcMinCost tests that a fund TX fails if it doesn't have enough tokens to cover for the minimum IPRPC costs
// Scenarios:
// 1. fund TX with the minimum cost available -> TX success
// 2. assume min cost = 100ulava, fund TX with 50ulava and 200ibc -> TX fails (ibc "has enough funds")
// 3. fund TX without the minimum cost available -> TX fails
// 4. fund TX with the minimum cost but creator doesn't have enough balance for the funding -> TX fails
func TestIprpcMinCost(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	err := ts.Keepers.BankKeeper.AddToBalance(consumerAcc.Addr, sdk.NewCoins(sdk.NewCoin(ibcDenom, sdk.NewInt(500))))

	_, poorConsumer := ts.AddAccount(common.CONSUMER, 1, minIprpcCost.Amount.Int64()-10)

	testCases := []struct {
		name    string
		creator string
		fund    sdk.Coins
		success bool
	}{
		{
			name:    "Happy flow - creator with enough funds and above min iprpc cost",
			creator: consumer,
			fund:    sdk.NewCoins(minIprpcCost.AddAmount(sdk.NewInt(10))),
			success: true,
		},
		{
			name:    "fund without min iprpc cost",
			creator: consumer,
			fund:    sdk.NewCoins(minIprpcCost.SubAmount(sdk.NewInt(10))),
			success: false,
		},
		{
			name:    "fund with other denom above min iprpc cost",
			creator: consumer,
			fund:    sdk.NewCoins(sdk.NewCoin(ibcDenom, minIprpcCost.Amount.AddRaw(10))),
			success: false,
		},
		{
			name:    "insufficient balance for fund",
			creator: poorConsumer,
			fund:    sdk.NewCoins(minIprpcCost.AddAmount(sdk.NewInt(10))),
			success: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err = ts.TxRewardsFundIprpc(tt.creator, mockSpec2, 1, tt.fund)
			if tt.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestIprpcEligibleSubscriptions tests that IPRPC CU is counted only if serviced an eligible subscription
// Scenarios:
// 0. assume two providers: p1, p2 and two consumers: c1, c2. Only c1 is IPRPC eligible
// 1. p1 provides service for both consumers, p2 provides service for c1 -> IPRPC reward should divide equally between p1 and p2
// 2. both providers provide service for c2 -> No IPRPC rewards should be given
func TestIprpcEligibleSubscriptions(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(true) // setup creates consumers and providers and funds IPRPC pool for mock2 spec

	c1Acc, c1 := ts.GetAccount(common.CONSUMER, 0)
	c2Acc, _ := ts.GetAccount(common.CONSUMER, 1)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)

	// p1 provides service for both consumers, p2 provides service for c1
	msg := ts.SendRelay(p1, c1Acc, []string{mockSpec2}, 100)
	_, err := ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	msg = ts.SendRelay(p1, c2Acc, []string{mockSpec2}, 100)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	msg = ts.SendRelay(p2, c1Acc, []string{mockSpec2}, 100)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	// check expected reward for each provider, it should be equal (the service for c1 was equal)
	res1, err := ts.QueryRewardsIprpcProviderReward(p1)
	require.NoError(t, err)
	res2, err := ts.QueryRewardsIprpcProviderReward(p2)
	require.NoError(t, err)
	require.True(t, res1.SpecFunds[0].Fund.IsEqual(res2.SpecFunds[0].Fund))
	require.True(t, iprpcFunds.Sub(minIprpcCost).QuoInt(sdk.NewInt(2)).IsEqual(res1.SpecFunds[0].Fund))

	// fund the pool again (advance month to apply)
	_, err = ts.TxRewardsFundIprpc(c1, mockSpec2, 1, sdk.NewCoins(minIprpcCost.AddAmount(sdk.NewInt(10))))
	require.NoError(ts.T, err)
	ts.AdvanceMonths(1).AdvanceEpoch()

	// provide service only for c2
	msg = ts.SendRelay(p1, c2Acc, []string{mockSpec2}, 100)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	msg = ts.SendRelay(p2, c2Acc, []string{mockSpec2}, 100)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	// check none of the providers should get rewards
	res1, err = ts.QueryRewardsIprpcProviderReward(p1)
	require.NoError(t, err)
	res2, err = ts.QueryRewardsIprpcProviderReward(p2)
	require.NoError(t, err)
	require.Len(t, res1.SpecFunds, 0)
	require.Len(t, res2.SpecFunds, 0)
}

// TestMultipleIprpcSpec checks that rewards are distributed correctly when multiple specs are configured in the IPRPC pool
// Scenarios:
// 0. IPRPC pool is funded for two specs for different periods and different denom (some are the same)
// 1. two providers provide service for consumer on 3 specs, two of them are the IPRPC ones -> they get rewarded relative to their serviced CU on each spec
func TestMultipleIprpcSpec(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false) // creates consumers and providers staked on two stakes

	c1Acc, c1 := ts.GetAccount(common.CONSUMER, 0)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)

	// add another spec and stake the providers
	mockSpec3 := "mock3"
	spec3 := common.CreateMockSpec()
	spec3.Index = mockSpec3
	spec3.Name = mockSpec3
	ts.specs = append(ts.specs, ts.AddSpec(mockSpec3, spec3).Spec(mockSpec3))
	err := ts.StakeProvider(p1, ts.specs[2], testStake)
	require.NoError(ts.T, err)
	err = ts.StakeProvider(p2, ts.specs[2], testStake)
	require.NoError(ts.T, err)

	// fund iprpc pool for mock2 spec for 1 months
	duration := uint64(1)
	iprpcCost := sdk.NewCoin(ts.BondDenom(), minIprpcCost.Amount.MulRaw(int64(duration)))
	mock2Fund := sdk.NewCoin(ts.BondDenom(), sdk.NewInt(1700))
	_, err = ts.TxRewardsFundIprpc(c1, mockSpec2, duration, sdk.NewCoins(mock2Fund.Add(iprpcCost)))
	require.NoError(t, err)

	// fund iprpc pool for mock3 spec for 3 months
	duration = uint64(3)
	iprpcCost = sdk.NewCoin(ts.BondDenom(), minIprpcCost.Amount.MulRaw(int64(duration)))
	mock3Fund := sdk.NewCoin(ts.BondDenom(), sdk.NewInt(400))
	_, err = ts.TxRewardsFundIprpc(c1, mockSpec3, duration, sdk.NewCoins(mock3Fund.Add(iprpcCost)))
	require.NoError(t, err)

	// advance month and epoch to apply pairing and iprpc fund
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	// make both providers service the consumer on 3 specs, only 2 are funded by IPRPC
	nonIprpcSpec := ts.specs[0].Index
	type relayData struct {
		provider string
		spec     string
		cu       uint64
	}
	relaysData := []relayData{
		{provider: p1, spec: nonIprpcSpec, cu: 100},
		{provider: p1, spec: mockSpec2, cu: 200},
		{provider: p1, spec: mockSpec3, cu: 300},
		{provider: p2, spec: nonIprpcSpec, cu: 700},
		{provider: p2, spec: mockSpec2, cu: 200},
		{provider: p2, spec: mockSpec3, cu: 300},
	}
	for _, rd := range relaysData {
		msg := ts.SendRelay(rd.provider, c1Acc, []string{rd.spec}, rd.cu)
		_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
		require.NoError(t, err)
	}

	// p1 total CU: 600, p2 total CU: 1200 -> if the rewards were divided by total CU (wrong) the rewards ratio should've been 1:2
	// p1 total iprpc CU: 500, p2 total iprpc CU: 500 -> if the rewards were divided by total iprpc CU the rewards should be equal
	res1, err := ts.QueryRewardsIprpcProviderReward(p1)
	require.NoError(t, err)
	res2, err := ts.QueryRewardsIprpcProviderReward(p2)
	require.NoError(t, err)
	require.Equal(t, len(res1.SpecFunds), len(res2.SpecFunds))
	responses := []*rewardstypes.QueryIprpcProviderRewardResponse{res1, res2}
	for _, res := range responses {
		for _, sf := range res.SpecFunds {
			switch sf.Spec {
			case mockSpec2:
				expectedReward := sdk.NewCoins(mock2Fund).QuoInt(sdk.NewInt(2))
				require.True(t, expectedReward.IsEqual(sf.Fund))
			case mockSpec3:
				expectedReward := sdk.NewCoins(mock3Fund).QuoInt(sdk.NewInt(2))
				require.True(t, expectedReward.IsEqual(sf.Fund))
			}
		}
	}
}

// TestIprpcRewardWithZeroSubRewards checks that even if a subscription is free (providers won't get paid for their service)
// if the providers service an IPRPC eligible subscription, they get IPRPC rewards
// Scenarios:
// 0. consumer is IPRPC eligible and community tax = 100% -> provider won't get paid for its service
// 1. two providers provide service -> they get IPRPC reward relative to their serviced CU
func TestIprpcRewardWithZeroSubRewards(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(true) // create a consumer and buys subscription + funds iprpc

	c1Acc, _ := ts.GetAccount(common.CONSUMER, 0)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)

	// make community participation percentage to be 100% to make the provider not get rewarded for its service later
	distParams := distributiontypes.DefaultParams()
	distParams.CommunityTax = sdk.OneDec()
	err := ts.Keepers.Distribution.SetParams(ts.Ctx, distParams)
	require.NoError(t, err)

	// make providers service the IPRPC eligible consumer
	msg := ts.SendRelay(p1, c1Acc, []string{mockSpec2}, 100)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	msg = ts.SendRelay(p2, c1Acc, []string{mockSpec2}, 400)
	_, err = ts.TxPairingRelayPayment(msg.Creator, msg.Relays...)
	require.NoError(t, err)

	// advance month to trigger monthly iprpc reward + blocksToSave to trigger sub rewards
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// check provider rewards (should be only expected IPRPC rewards)
	p1ExpectedReward := iprpcFunds.Sub(minIprpcCost).QuoInt(sdk.NewInt(5))
	res1, err := ts.QueryDualstakingDelegatorRewards(p1, p1, mockSpec2)
	require.NoError(t, err)
	require.True(t, p1ExpectedReward.IsEqual(res1.Rewards[0].Amount))

	p2ExpectedReward := p1ExpectedReward.MulInt(sdk.NewInt(4))
	res2, err := ts.QueryDualstakingDelegatorRewards(p2, p2, mockSpec2)
	require.NoError(t, err)
	require.True(t, p2ExpectedReward.IsEqual(res2.Rewards[0].Amount))
}
