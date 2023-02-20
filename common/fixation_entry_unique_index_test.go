package common_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/stretchr/testify/require"
)

// Helper function that creates N fixationEntryUniqueIndex objects
func createNUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, entryKeyPrefix string, n int) []types.UniqueIndex {
	// create an empty array of uniqueIndex objects
	items := make([]types.UniqueIndex, n)
	for i := range items {
		// create a uniqueIndex object with id=i, uniqueIndex=entryKeyPrefix+"i"
		items[i].UniqueIndex = entryKeyPrefix + strconv.FormatInt(int64(i), 10)
		items[i].Id = uint64(i)

		// append the uniqueIndex object to the saved list in the store
		common.AppendFixationEntryUniqueIndex(ctx, storeKey, cdc, entryKeyPrefix, items[i])
	}

	return items
}

// Test that creates entries and gets them to verify the getter works properly
func TestUniqueIndexGet(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)

	// iterate over the uniqueIndex objects
	for _, item := range items {
		// get the uniqueIndex object from the store
		got, found := common.GetFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, item.Id)

		// verify it's found and it's equal to the one we created earlier
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

// Test that removes entries and verifies they're not in the storage anymore
func TestUniqueIndexRemove(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)

	// iterate over the uniqueIndex objects
	for i, item := range items {
		// remove the uniqueIndex object
		common.RemoveFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix, item.Id)

		// make sure the removed object cannot be found using the getter (gets from the store)
		_, found := common.GetFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, item.Id)
		require.False(t, found)

		// make sure that the uniqueIndex objects count went down by 1
		count := common.GetFixationEntryUniqueIndexCount(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix)
		require.Equal(t, uint64(10-(i+1)), count)
	}
}

// Test that creates N fixationEntryUniqueIndex objects and gets them all from storage to verify they're the same elements
func TestUniqueIndexGetAll(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)

	// get all the uniqueIndex objects from the store and verify they're the same as the ones we created earlier
	itemsFromStorage := common.GetAllFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(itemsFromStorage),
	)
}

// Test that fixationEntryUniqueIndex count works properly
func TestUniqueIndexCount(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)

	// get the amount of uniqueIndex objects
	uniqueIndexAmount := uint64(len(items))

	// verify the count saved in the store is the same as the amount of uniqueIndex objects
	require.Equal(t, uniqueIndexAmount, common.GetFixationEntryUniqueIndexCount(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix))
}

// Test that creates N fixationEntryUniqueIndex objects, removes one of them, and gets all the entries from the storage to verify that it doesn't exist
func TestRemoveAndGetAll(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)

	// save the uniqueIndex object we're going to remove
	itemToRemove := items[1]

	// remove the uniqueIndex object
	common.RemoveFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix, 1)

	// get all the uniqueIndex objects from the store and verify the number of objects decreased by one
	itemsAfterRemoval := common.GetAllFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix)
	require.Equal(t, len(items)-1, len(itemsAfterRemoval))

	// verify that the removed item is no longer in the itemsAfterRemoval list
	for _, item := range itemsAfterRemoval {
		require.NotEqual(t, item, itemToRemove)
	}

	// verify the uniqueIndex objects count saved in the store also decreased by one
	count := common.GetFixationEntryUniqueIndexCount(ctx, mockKeeper.StoreKey, mockKeeper.entryKeyPrefix)
	require.Equal(t, uint64(len(items)-1), count)
}

// Test that creates fixations with different prefixes and verifies they are mutually exclusive
func TestDifferentFixations(t *testing.T) {
	// init a mock keeper and context
	mockKeeper, ctx := initMockKeeper(t)

	// create 10 uniqueIndex objects and another 9 with a different entryKeyPrefix
	items := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix, 10)
	itemsWithDifferentPrefix := createNUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix+"1", 9)

	// get all the uniqueIndex objects of the first entryKeyPrefix from the store and verify it's equal to the amount of the uniqueIndex objects we created earlier (with the first entryKeyPrefix)
	itemsFromStorage := common.GetAllFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix)
	require.Equal(t, len(items), len(itemsFromStorage))

	// get all the uniqueIndex objects of the second entryKeyPrefix from the store and verify it's equal to the amount of the uniqueIndex objects we created earlier (with the second entryKeyPrefix)
	itemsWithDifferentPrefixFromStorage := common.GetAllFixationEntryUniqueIndex(ctx, mockKeeper.StoreKey, mockKeeper.Cdc, mockKeeper.entryKeyPrefix+"1")
	require.Equal(t, len(itemsWithDifferentPrefix), len(itemsWithDifferentPrefixFromStorage))

	// go over all the objects of the two lists and verify none is equal (the value of the created uniqueIndex is dependent on the entryKeyPrefix, so they must be different)
	for _, item := range itemsFromStorage {
		for _, itemDifferentPrefix := range itemsWithDifferentPrefix {
			require.NotEqual(t, item, itemDifferentPrefix)
		}
	}
}
