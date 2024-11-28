package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
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
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 10).Unix(), // was done 10 days ago
				CreditTimestamp: 0,
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation with existing credit equal time increase",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation with existing credit equal time decrease",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation older than 30 days no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
			currentTime:             timeNow,
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 30).Unix(),
		},
		{
			name: "delegation older than 30 days with credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(7000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 50).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
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
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 30).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
		},
		{
			name: "old delegation no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 100).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
		},
		{
			name: "half month delegation no credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 15).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(500)),
		},
		{
			name: "old delegation with credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 35).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 45).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
		},
		{
			name: "new delegation new credit increased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(6000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
		},
		{
			name: "new delegation new credit decreased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(6000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
		},
		{
			name: "new delegation old credit increased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(6000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(3500)),
		},
		{
			name: "new delegation old credit decreased delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(6000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(5500)),
		},
		{
			name: "last second change",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(10000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Minute).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
		},
		{
			name: "non whole hours old credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2*720000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(720000)),
				Timestamp:       timeNow.Add(-time.Hour - time.Minute).Unix(), // results in 1 hour
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),     // results in 718 hours
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(721001)), // ((718*720 + 2*720*1) / 719) *720/720 = 721001.39
		},
		{
			name: "non whole hours new credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Hour*24*5 - time.Minute).Unix(),  // 120 hours
				CreditTimestamp: timeNow.Add(-time.Hour*24*15 - time.Minute).Unix(), // 240 hours
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(480)), // (120 * 2 * 720 + 240 * 720) / 360 = 960, and monthly is: 960*360/720 = 480
		},
		{
			name: "new delegation new credit last minute",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Minute).Unix(),
				CreditTimestamp: timeNow.Add(-time.Minute * 2).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(0)),
		},
		{
			name: "delegation credit are monthly exactly",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2*720)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(720)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 30).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 30).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(720*2)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credit := k.CalculateMonthlyCredit(ctx, tt.delegation)
			require.Equal(t, tt.expectedCredit, credit)
		})
	}
}
