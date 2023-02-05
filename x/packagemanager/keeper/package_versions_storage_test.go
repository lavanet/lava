package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/packagemanager/keeper"
	"github.com/lavanet/lava/x/packagemanager/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNPackageVersionsStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PackageVersionsStorage {
	items := make([]types.PackageVersionsStorage, n)
	for i := range items {
		items[i].PackageIndex = strconv.Itoa(i)

		keeper.SetPackageVersionsStorage(ctx, items[i])
	}
	return items
}

func TestPackageVersionsStorageGet(t *testing.T) {
	keeper, ctx := keepertest.PackagemanagerKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetPackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestPackageVersionsStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.PackagemanagerKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		_, found := keeper.GetPackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		require.False(t, found)
	}
}

func TestPackageVersionsStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.PackagemanagerKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPackageVersionsStorage(ctx)),
	)
}
