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
		expectedCreditTimestamp int64
	}{
		{
			name: "initial delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 10).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
		},
		{
			name: "delegation with existing credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 5).Unix(),
		},
		{
			name: "delegation older than 30 days",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 35).Unix(),
			},
			expectedCredit:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
			expectedCreditTimestamp: timeNow.Add(-time.Hour * 24 * 40).Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			name: "initial delegation",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.ZeroInt()),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 10).Unix(),
				CreditTimestamp: 0,
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
		},
		{
			name: "delegation with existing credit",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(2000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1000)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 5).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 10).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
		},
		{
			name: "delegation older than 30 days",
			delegation: types.Delegation{
				Amount:          sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
				Credit:          sdk.NewCoin(bondDenom, sdk.NewInt(1500)),
				Timestamp:       timeNow.Add(-time.Hour * 24 * 40).Unix(),
				CreditTimestamp: timeNow.Add(-time.Hour * 24 * 35).Unix(),
			},
			expectedCredit: sdk.NewCoin(bondDenom, sdk.NewInt(3000)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credit := k.CalculateMonthlyCredit(ctx, tt.delegation)
			require.Equal(t, tt.expectedCredit, credit)
		})
	}
}
