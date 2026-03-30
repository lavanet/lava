package rewardserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBadgerDB_BatchSaveLargeTransaction(t *testing.T) {
	// This test verifies that BatchSave handles a large number of entries
	// that would exceed BadgerDB's single transaction size limit.
	// Before the fix, this would fail with: "Txn is too big to fit into one request"
	db, ok := NewMemoryDB("specId").(*BadgerDB)
	require.True(t, ok)
	defer db.Close()

	// Each entry has ~10KB of data. With default MemTableSize of 64MB,
	// BadgerDB's max batch size is ~9.6MB (15% of MemTableSize).
	// 2000 entries * 10KB = ~20MB, which exceeds the limit.
	const (
		numEntries = 2000
		dataSize   = 10 * 1024 // 10KB per entry
	)

	entries := make([]*DBEntry, numEntries)
	for i := range numEntries {
		data := make([]byte, dataSize)
		// Fill with non-zero data to ensure realistic size
		for j := range data {
			data[j] = byte(i % 256)
		}
		entries[i] = &DBEntry{
			Key:  fmt.Sprintf("key-%d", i),
			Data: data,
			Ttl:  24 * time.Hour,
		}
	}

	err := db.BatchSave(entries)
	require.NoError(t, err)

	// Verify all entries were persisted
	all, err := db.FindAll()
	require.NoError(t, err)
	require.Equal(t, numEntries, len(all))

	for i := range numEntries {
		key := fmt.Sprintf("key-%d", i)
		val, ok := all[key]
		require.True(t, ok, "missing key: %s", key)
		require.Equal(t, dataSize, len(val))
	}
}
