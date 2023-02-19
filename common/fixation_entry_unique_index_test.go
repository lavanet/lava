package common_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/nullify"
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
	mockKeeper, ctx := initMockKeeper(t)

	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)
	for _, item := range items {
		got, found := common.GetFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestUniqueIndexRemove(t *testing.T) {
	mockKeeper, ctx := initMockKeeper(t)
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)
	for _, item := range items {
		common.RemoveFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix, item.Id)
		_, found := common.GetFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, item.Id)
		require.False(t, found)
	}
}

func TestUniqueIndexGetAll(t *testing.T) {
	mockKeeper, ctx := initMockKeeper(t)
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(common.GetAllFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix)),
	)
}

func TestUniqueIndexCount(t *testing.T) {
	mockKeeper, ctx := initMockKeeper(t)
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)
	count := uint64(len(items))
	require.Equal(t, count, common.GetFixationEntryUniqueIndexCount(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix))
}
