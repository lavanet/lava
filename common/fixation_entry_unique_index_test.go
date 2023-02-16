package common_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/common/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	packagestypes "github.com/lavanet/lava/x/packages/types"
	"github.com/stretchr/testify/require"
)

func createNUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, entryKeyPrefix string, n int) []types.UniqueIndex {
	items := make([]types.UniqueIndex, n)
	for i := range items {
		items[i].Id = common.AppendFixationEntryUniqueIndex(ctx, storeKey, cdc, entryKeyPrefix, items[i])
	}
	return items
}

func TestUniqueIndexGet(t *testing.T) {
	_, keepers, ctx := keepertest.InitAllKeepers(t)
	items := createNUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), 10)
	for _, item := range items {
		got, found := common.GetFixationEntryUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestPackageUniqueIndexRemove(t *testing.T) {
	_, keepers, ctx := keepertest.InitAllKeepers(t)
	items := createNUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), 10)
	for _, item := range items {
		common.RemoveFixationEntryUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), packagestypes.UniqueIndexKeyPrefix(), item.Id)
		_, found := common.GetFixationEntryUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), item.Id)
		require.False(t, found)
	}
}

func TestPackageUniqueIndexGetAll(t *testing.T) {
	_, keepers, ctx := keepertest.InitAllKeepers(t)
	items := createNUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(common.GetAllFixationEntryUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix())),
	)
}

func TestPackageUniqueIndexCount(t *testing.T) {
	_, keepers, ctx := keepertest.InitAllKeepers(t)
	items := createNUniqueIndex(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), keepers.Packages.GetCdc(), packagestypes.UniqueIndexKeyPrefix(), 10)
	count := uint64(len(items))
	require.Equal(t, count, common.GetFixationEntryUniqueIndexCount(sdk.UnwrapSDKContext(ctx), keepers.Packages.GetStoreKey(), packagestypes.UniqueIndexKeyPrefix()))
}
