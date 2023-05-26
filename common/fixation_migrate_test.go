package common

import (
	"fmt"
	"math"
	"testing"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/stretchr/testify/require"
)

const (
	mockPrefix = "mock-fs"
)

type mockEntry1to2 struct {
	index  string
	block  uint64
	head   bool
	count  int
	before uint64
	after  uint64
}

func TestMigrate1to2(t *testing.T) {
	var err error

	ctx, cdc := initCtx(t)
	fs := NewFixationStore(mockStoreKey, cdc, mockPrefix)
	coin := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(1)}

	templates := []mockEntry1to2{
		// entry_1 has 3 valid versions
		{"entry_1", 100, false, 1, 1, 1},
		{"entry_1", 200, false, 2, 2, 2},
		{"entry_1", 300, true, 0, 0, 1},
		// entry_2 has 2 valid versions, one stale-at
		{"entry_2", 100, false, 1, 1, 1},
		{"entry_2", 200, false, 0, 0, 0},
		{"entry_2", 300, true, 0, 0, 1},
		// entry_3 has 2 valid versions, head with extra refcount
		{"entry_3", 100, false, 0, 0, 0},
		{"entry_3", 200, false, 1, 1, 1},
		{"entry_3", 300, true, 1, 1, 2},
	}

	// create entries
	for _, tt := range templates {
		ctx = ctx.WithBlockHeight(int64(tt.block))
		err = fs.AppendEntry(ctx, tt.index, tt.block, &coin)
		require.Nil(t, err)
		for i := 0; i < tt.count; i++ {
			ctx = ctx.WithBlockHeight(int64(tt.block))
			found := fs.GetEntry(ctx, tt.index, &coin)
			require.True(t, found)
		}
	}

	// verify entries before migration
	for _, tt := range templates {
		what := fmt.Sprintf("before: index: %s, block: %d, count: %d", tt.index, tt.block, tt.count)
		safeIndex, err := types.SanitizeIndex(tt.index)
		require.Nil(t, err, what)
		entry := fs.getEntry(ctx, safeIndex, tt.block)
		if tt.head {
			// adjust head entry's refcount to version 1 style
			entry.Refcount -= 1
			fs.setEntry(ctx, entry)
		}
		require.Equal(t, tt.before, entry.Refcount, what)
	}

	// mock fixation version to be 2
	realFixationVersion := fixationVersion
	fixationVersion = 2

	// perform migration version 1 to version 2
	err = fs.MigrateVersion(ctx)
	require.Nil(t, err, "migration version 1 to version 2")

	fixationVersion = realFixationVersion

	// verify entries after migration
	for _, tt := range templates {
		what := fmt.Sprintf("after: index: %s, block: %d, count: %d", tt.index, tt.block, tt.count)
		safeIndex, err := types.SanitizeIndex(tt.index)
		require.Nil(t, err, what)
		entry := fs.getEntry(ctx, safeIndex, tt.block)
		require.Equal(t, tt.after, entry.Refcount, what)
		if entry.Refcount == 0 {
			require.NotEqual(t, uint64(math.MaxUint64), entry.StaleAt, what)
		} else {
			require.Equal(t, uint64(math.MaxUint64), entry.StaleAt, what)
		}
	}
}

type mockEntry2to3 struct {
	index string
	block uint64
	head  bool
}

// V2_setEntryIndex stores an Entry index in the store
func (fs FixationStore) V2_setEntryIndex(ctx sdk.Context, safeIndex string) {
	storePrefix := types.EntryIndexPrefix + fs.prefix
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(storePrefix))
	appendedValue := []byte(safeIndex) // convert the index value to a byte array
	store.Set(types.KeyPrefix(storePrefix+safeIndex), appendedValue)
}

// V2_setEntry modifies an existing entry in the store
func (fs *FixationStore) V2_setEntry(ctx sdk.Context, entry types.Entry) {
	storePrefix := types.EntryPrefix + fs.prefix + entry.Index
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(storePrefix))
	byteKey := types.EncodeKey(entry.Block)
	marshaledEntry := fs.cdc.MustMarshal(&entry)
	store.Set(byteKey, marshaledEntry)
}

func countWithPrefix(store *prefix.Store, prefix string) int {
	iterator := sdk.KVStorePrefixIterator(store, types.KeyPrefix(prefix))
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count += 1
	}
	return count
}

func TestMigrate2to3(t *testing.T) {
	var err error

	ctx, cdc := initCtx(t)
	fs := NewFixationStore(mockStoreKey, cdc, mockPrefix)
	coin := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(1)}

	templates := []mockEntry2to3{
		// entry_1 has 3 valid versions
		{"entry_1", 100, false},
		{"entry_1", 200, false},
		{"entry_1", 300, true},
		// entry_2 has 2 valid versions, one stale-at
		{"entry_2", 100, false},
		{"entry_2", 200, false},
		{"entry_2", 300, true},
		// entry_3 has 2 valid versions, head with extra refcount
		{"entry_3", 100, false},
		{"entry_3", 200, false},
		{"entry_3", 300, true},
	}

	// set version to v2
	fs.setVersion(ctx, 2)

	// create v2 entries
	numEntries, numHeads := 0, 0
	for _, tt := range templates {
		what := fmt.Sprintf("before: index: %s, block: %d", tt.index, tt.block)
		safeIndex, err := types.SanitizeIndex(tt.index)
		require.Nil(t, err, what)
		if tt.head {
			fs.V2_setEntryIndex(ctx, safeIndex)
			numHeads += 1
		}
		entry := types.Entry{
			Index:    safeIndex,
			Block:    tt.block,
			StaleAt:  math.MaxUint64,
			Data:     fs.cdc.MustMarshal(&coin),
			Refcount: 1,
		}
		fs.V2_setEntry(ctx, entry)
		numEntries += 1
	}

	// verify entry count before migration
	store_V2 := prefix.NewStore(ctx.KVStore(fs.storeKey), []byte{})
	require.Equal(t, 1+numHeads+numEntries, countWithPrefix(&store_V2, ""))
	require.Equal(t, numHeads+numEntries, countWithPrefix(&store_V2, "Entry"))

	// mock fixation version to be 3
	realFixationVersion := fixationVersion
	fixationVersion = 3

	// perform migration version 2 to version 3
	err = fs.MigrateVersion(ctx)
	require.Nil(t, err, "migration version 2 to version 3")

	fixationVersion = realFixationVersion

	// verify version upgrade
	require.Equal(t, uint64(3), fs.getVersion(ctx))

	// verify entry count before migration
	store_V3 := prefix.NewStore(ctx.KVStore(fs.storeKey), []byte{})
	require.Equal(t, 1+numHeads+numEntries, countWithPrefix(&store_V3, ""))
	require.Equal(t, 1+numHeads+numEntries, countWithPrefix(&store_V3, mockPrefix))

	// verify entries after migration
	for _, tt := range templates {
		what := fmt.Sprintf("after: index: %s, block: %d", tt.index, tt.block)
		safeIndex, err := types.SanitizeIndex(tt.index)
		require.Nil(t, err, what)
		_, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, tt.block)
		require.True(t, found, what)
	}
}
