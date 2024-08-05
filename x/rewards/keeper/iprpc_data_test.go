package keeper_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/rewards/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/testutil/sample"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createTestMinIprpcCost(keeper *keeper.Keeper, ctx sdk.Context) sdk.Coin {
	item := sdk.NewCoin("denom", math.OneInt())
	keeper.SetMinIprpcCost(ctx, item)
	return item
}

func TestMinIprpcCostGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	item := createTestMinIprpcCost(keeper, ctx)
	rst := keeper.GetMinIprpcCost(ctx)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func createNIprpcSubscriptions(keeper *keeper.Keeper, ctx sdk.Context, n int) []string {
	items := make([]string, n)
	for i := range items {
		items[i] = strconv.Itoa(i)
		keeper.SetIprpcSubscription(ctx, items[i])
	}
	return items
}

func TestIprpcSubscriptionsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	for _, item := range items {
		require.True(t, keeper.IsIprpcSubscription(ctx, item))
	}
}

func TestIprpcSubscriptionsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveIprpcSubscription(ctx, item)
		require.False(t, keeper.IsIprpcSubscription(ctx, item))
	}
}

func TestIprpcSubscriptionsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllIprpcSubscription(ctx)),
	)
}

// TestIprpcDataValidation tests that IPRPC data's validation works as expected
func TestIprpcDataValidation(t *testing.T) {
	ts := newTester(t, true)
	_, val := ts.GetAccount(common.VALIDATOR, 0)

	template := []struct {
		name      string
		authority string
		cost      sdk.Coin
		subs      []string
		success   bool
	}{
		{
			name:      "valid data",
			authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			cost:      sdk.NewCoin(ts.TokenDenom(), math.OneInt()),
			subs:      []string{sample.AccAddress()},
			success:   true,
		},
		{
			name:      "invalid auth",
			authority: val,
			cost:      sdk.NewCoin(ts.TokenDenom(), math.OneInt()),
			subs:      []string{sample.AccAddress()},
			success:   false,
		},
		{
			name:      "invalid subs",
			authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			cost:      sdk.NewCoin(ts.TokenDenom(), math.OneInt()),
			subs:      []string{sample.AccAddress(), "invalid_addr"},
			success:   false,
		},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ts.TxRewardsSetIprpcDataProposal(tt.authority, tt.cost, tt.subs)
			if tt.success {
				require.NoError(t, err)
				res, err := ts.QueryRewardsShowIprpcData()
				require.NoError(t, err)
				require.True(t, tt.cost.IsEqual(res.MinCost))
				require.Equal(t, tt.subs, res.IprpcSubscriptions)
			} else {
				require.Error(t, err)
			}
		})
	}
}
