package keeper_test

import (
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNPendingIprpcFunds(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PendingIprpcFund {
	items := make([]types.PendingIprpcFund, n)
	for i := range items {
		items[i] = types.PendingIprpcFund{
			Index:       uint64(i),
			Creator:     "dummy",
			Spec:        "mock",
			Month:       1,
			Expiry:      uint64(ctx.BlockTime().UTC().Unix()) + uint64(i),
			CostCovered: sdk.NewCoin("denom", math.OneInt()),
		}
		keeper.SetPendingIprpcFund(ctx, items[i])
	}
	return items
}

func TestPendingIprpcFundsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		require.True(t, found)
	}
}

func TestPendingIprpcFundsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePendingIprpcFund(ctx, item.Index)
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		require.False(t, found)
	}
}

func TestPendingIprpcFundsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPendingIprpcFund(ctx)),
	)
}

func TestPendingIprpcFundsRemoveExpired(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(3 * time.Second))
	keeper.RemoveExpiredPendingIprpcFund(ctx)
	for _, item := range items {
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		if item.Index <= 3 {
			require.False(t, found)
		} else {
			require.True(t, found)
		}
	}
}

func TestPendingIprpcFundGetLatest(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	require.True(t, latest.IsEmpty())
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	latest = keeper.GetLatestPendingIprpcFund(ctx)
	require.True(t, latest.IsEqual(items[len(items)-1]))
}

func TestPendingIprpcFundNew(t *testing.T) {
	ts := newTester(t, false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	spec := ts.Spec("mock")
	err := keeper.NewPendingIprpcFund(ctx, "creator", "eth", 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.Error(t, err) // error for non-existent spec
	err = keeper.NewPendingIprpcFund(ctx, "creator", "eth", 1, nil)
	require.Error(t, err) // error for invalid funds
	err = keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	err = keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	require.Equal(t, uint64(1), latest.Index) // latest is the second object with index 1
}

func TestPendingIprpcFundIsMinCostCovered(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	spec := ts.Spec("mock")
	err := keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	minCost := keeper.GetMinIprpcCost(ctx)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	latest.CostCovered = minCost
	require.True(t, keeper.IsIprpcMinCostCovered(ctx, latest))
	latest.CostCovered = minCost.SubAmount(math.OneInt())
	require.False(t, keeper.IsIprpcMinCostCovered(ctx, latest))
}
