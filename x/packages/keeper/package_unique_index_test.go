package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/packages/keeper"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/stretchr/testify/require"
)

func createNPackageUniqueIndex(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PackageUniqueIndex {
	items := make([]types.PackageUniqueIndex, n)
	for i := range items {
		items[i].Id = keeper.AppendPackageUniqueIndex(ctx, items[i])
	}
	return items
}

func TestPackageUniqueIndexGet(t *testing.T) {
	keeper, ctx := keepertest.PackagesKeeper(t)
	items := createNPackageUniqueIndex(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetPackageUniqueIndex(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestPackageUniqueIndexRemove(t *testing.T) {
	keeper, ctx := keepertest.PackagesKeeper(t)
	items := createNPackageUniqueIndex(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePackageUniqueIndex(ctx, item.Id)
		_, found := keeper.GetPackageUniqueIndex(ctx, item.Id)
		require.False(t, found)
	}
}

func TestPackageUniqueIndexGetAll(t *testing.T) {
	keeper, ctx := keepertest.PackagesKeeper(t)
	items := createNPackageUniqueIndex(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPackageUniqueIndex(ctx)),
	)
}

func TestPackageUniqueIndexCount(t *testing.T) {
	keeper, ctx := keepertest.PackagesKeeper(t)
	items := createNPackageUniqueIndex(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetPackageUniqueIndexCount(ctx))
}
