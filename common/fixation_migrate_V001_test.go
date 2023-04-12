package common

import (
	"fmt"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const (
	mockPrefix = "mock-fs"
)

type mockEntry struct {
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

	templates := []mockEntry{
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
		safeIndex, err := sanitizeIndex(tt.index)
		require.Nil(t, err, what)
		entry := fs.getEntry(ctx, safeIndex, tt.block)
		if tt.head {
			// adjust head entry's refcount to version 1 style
			entry.Refcount -= 1
			fs.setEntry(ctx, entry)
		}
		require.Equal(t, tt.before, entry.Refcount, what)
	}

	// perform migration version 1 to version 2
	err = fs.MigrateVersion(ctx)
	require.Nil(t, err, "migration version 1 to version 2")

	// verify entries after migration
	for _, tt := range templates {
		what := fmt.Sprintf("after: index: %s, block: %d, count: %d", tt.index, tt.block, tt.count)
		safeIndex, err := sanitizeIndex(tt.index)
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
