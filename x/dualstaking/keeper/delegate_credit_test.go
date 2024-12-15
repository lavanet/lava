package keeper_test

import (
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/testutil/common"
	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/utils"
	commontypes "github.com/lavanet/lava/v4/utils/common/types"
	"github.com/lavanet/lava/v4/x/dualstaking/keeper"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func SetDelegationMock(k keeper.Keeper, ctx sdk.Context, delegation types.Delegation) (delegationRet types.Delegation) {
	credit, creditTimestamp := k.CalculateCredit(ctx, delegation)
	delegation.Credit = credit
	delegation.CreditTimestamp = creditTimestamp
	return delegation
}

func TestCalculateCredit(t *testing.T) {
	k, ctx := keepertest.DualstakingKeeper(t)
	bondDenom := commontypes.TokenDenom
	timeNow := ctx.BlockTime()
	tests := []struct {
		name                    string
		delegation              types.Delegation
		expectedCredit          sdk.Coin
		currentTime             time.Time
		expectedCreditTimestamp int64
	}{
		{
			name: "initial delegation 10days ago with no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, math.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 10).Unix(), // was done 10 days ago
				CreditTimestamp: 0,
			},
			expectedCredit:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation with existing credit equal time increase",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, math.NewInt(1500)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation with existing credit equal time decrease",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(2000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, math.NewInt(1500)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation older than 30 days no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, math.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 30).Unix(),
		},
		{
			name: "delegation older than 30 days with credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(7000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 50).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 30).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithBlockTime(tt.currentTime)
			credit, creditTimestamp := k.CalculateCredit(ctx, tt.delegation)
			require.Equal(t, tt.expectedCredit, credit)
			require.Equal(t, tt.expectedCreditTimestamp, creditTimestamp)
		})
	}
}

func TestCalculateMonthlyCredit(t *testing.T) {
	k, ctx := keepertest.DualstakingKeeper(t)
	bondDenom := commontypes.TokenDenom
	timeNow := ctx.BlockTime()
	tests := []struct {
		name           string
		delegation     types.Delegation
		expectedCredit sdk.Coin
	}{
		{
			name: "monthly delegation no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, math.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 30).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(1000)),
		},
		{
			name: "old delegation no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, math.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 100).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(1000)),
		},
		{
			name: "half month delegation no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, math.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 15).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(500)),
		},
		{
			name: "old delegation with credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 35).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 45).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(2000)),
		},
		{
			name: "new delegation new credit increased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(6000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(1500)),
		},
		{
			name: "new delegation new credit decreased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(6000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(1500)),
		},
		{
			name: "new delegation old credit increased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(6000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(3500)),
		},
		{
			name: "new delegation old credit decreased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(6000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(5500)),
		},
		{
			name: "last second change",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(10000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Minute).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(1000)),
		},
		{
			name: "non whole hours old credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2*720000)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(720000)),
				Timestamp:       timeNow.Add(-time.Hour - time.Minute).Unix(), // results in 1 hour
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),     // results in 718 hours
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(721001)), // ((718*720 + 2*720*1) / 719) *720/720 = 721001.39
		},
		{
			name: "non whole hours new credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Hour*24*5 - time.Minute).Unix(),  // 120 hours
				CreditTimestamp: timeNow.Add(-time.Hour*24*15 - time.Minute).Unix(), // 240 hours
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(480)), // (120 * 2 * 720 + 240 * 720) / 360 = 960, and monthly is: 960*360/720 = 480
		},
		{
			name: "new delegation new credit last minute",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Minute).Unix(),
				CreditTimestamp: timeNow.Add(-time.Minute * 2).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(0)),
		},
		{
			name: "delegation credit are monthly exactly",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, math.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, math.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 30).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 30).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, math.NewInt(720*2)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credit := k.CalculateMonthlyCredit(ctx, tt.delegation)
			require.Equal(t, tt.expectedCredit, credit)
		})
	}
}

func TestDelegationSet(t *testing.T) {
	ts := newTester(t)
	// 1 delegator, 1 provider staked, 0 provider unstaked, 0 provider unstaking
	ts.setupForDelegation(1, 1, 0, 0)
	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	k := ts.Keepers.Dualstaking
	bondDenom := commontypes.TokenDenom
	tests := []struct {
		amount                sdk.Coin
		expectedMonthlyCredit sdk.Coin
		timeWait              time.Duration
		remove                bool
	}{
		{ // 0
			timeWait:              0,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(0)),
		},
		{ // 1
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720/2)),
		},
		{ // 2
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
		},
		{ // 3
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
		},
		{ // 4
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
		},
		{ // 5
			timeWait:              0,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
		},
		{ // 6
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(1000*720/2+500*720/2)), // 540000
		},
		{ // 7
			timeWait:              15 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt((1000*720/2+500*720/2)/2+500*720/2)),
		},
		{ // 8
			timeWait:              30 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500*720)),
		},
		{ // 9
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(0)),
			remove:                true, // remove existing entry first
		},
		{ // 10
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500)),
		},
		{ // 11
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500+500)),
		},
		{ // 12
			timeWait:              30 * time.Hour * 24,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500*720)),
		},
		{ // 13
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(500*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(0)),
			remove:                true, // remove existing entry first
		},
		{ // 14
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500)),
		},
		{ // 15
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(2000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500+1000)),
		},
		{ // 16
			timeWait:              1 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500+1000+2000)),
		},
		{ // 17
			timeWait:              2 * time.Hour,
			amount:                sdk.NewCoin(bondDenom, math.NewInt(1000*720)),
			expectedMonthlyCredit: sdk.NewCoin(bondDenom, math.NewInt(500+1000+2000+1000*2)),
		},
	}

	for iteration := 0; iteration < len(tests); iteration++ {
		t.Run("delegation set tests "+strconv.Itoa(iteration), func(t *testing.T) {
			delegation := types.Delegation{
				Delegator: client1Addr,
				Provider:  provider1Addr,
				Amount:    tests[iteration].amount,
			}
			if tests[iteration].remove {
				k.RemoveDelegation(ts.Ctx, delegation)
			}
			utils.LavaFormatDebug("block times for credit", utils.LogAttr("block time", ts.Ctx.BlockTime()), utils.LogAttr("time wait", tests[iteration].timeWait))
			ts.Ctx = ts.Ctx.WithBlockTime(ts.Ctx.BlockTime().Add(tests[iteration].timeWait))

			err := k.SetDelegation(ts.Ctx, delegation)
			require.NoError(t, err)
			delegationGot, found := k.GetDelegation(ts.Ctx, delegation.Provider, delegation.Delegator)
			require.True(t, found)
			monthlyCredit := k.CalculateMonthlyCredit(ts.Ctx, delegationGot)
			require.Equal(t, tests[iteration].expectedMonthlyCredit, monthlyCredit)
		})
	}
}
