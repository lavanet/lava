package keeper_test

import (
	"strconv"
	"testing"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Test that if EpochBlocks decreases then provider gets staked only after the original EpochBlocks
func TestStakeGovEpochBlocksDecrease(t *testing.T) {
	ts := newTester(t)

	// The test assumes that EpochBlocks default value is 20 - make sure it is 20
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance an epoch to apply EpochBlocks change, one block before EpochBlocks changes.
	ts.AdvanceEpoch() // blockHeight = 20

	// change the EpochBlocks parameter to 10
	paramKey = string(epochstoragetypes.KeyEpochBlocks)
	paramVal = "\"" + strconv.FormatUint(10, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	ts.AdvanceBlocks(19)

	// stake a provider
	err = ts.addProvider(1)
	require.NoError(t, err)

	// Advance to the next block so the EpochBlocks change apply
	ts.AdvanceBlock() // blockHeight = 40

	// Advance an epoch (10 blocks). Total 11 blocks passed since the stake transaction,
	// so stake should not be in effect yet (EpochBlocks was 20 at that time)
	ts.AdvanceEpoch() // blockHeight = 50
	epochShouldntBeStaked := ts.EpochStart()

	// Advance another epoch (10 blocks). Total 21 blocks have passed so stake should apply
	ts.AdvanceEpoch() // blockHeight = 60
	epochShouldBeStaked := ts.EpochStart()

	// define tests
	tests := []struct {
		name  string
		epoch uint64
	}{
		{"ShouldntBeStaked", epochShouldntBeStaked}, // 11 blocks have passed since the stake - not enough blocks
		{"ShouldBeStaked", epochShouldBeStaked},     // 21 blocks have passed - enough blocks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if the provider/client are staked
			entries := ts.Keepers.Epochstorage.GetAllStakeEntriesForEpoch(ts.Ctx, tt.epoch)
			require.Len(t, entries, 1)
		})
	}
}

// Test that if the EpochBlocks param increases make sure the provider/client is staked after
// the original EpochBlocks value, and not the new longer value
func TestStakeGovEpochBlocksIncrease(t *testing.T) {
	ts := newTester(t)

	// The test assumes that EpochBlocks default value is 20 - make sure it is 20
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance an epoch to apply EpochBlocks change
	ts.AdvanceEpoch() // blockHeight = 20

	// change the EpochBlocks parameter to 50
	paramKey = string(epochstoragetypes.KeyEpochBlocks)
	paramVal = "\"" + strconv.FormatUint(50, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	ts.AdvanceBlocks(19)
	// stake a provider
	err = ts.addProvider(1)
	require.NoError(t, err)

	// Advance to the next block so the EpochBlocks change apply
	ts.AdvanceBlock() // blockHeight = 40

	// Advance to blockHeight = 59. Total 20 blocks have passed since the stake transaction,
	// so stake should not be in effect yet (EpochBlocks was 20 at that time)
	ts.AdvanceBlocks(19)
	epochShouldntBeStaked := ts.BlockHeight()

	// Advance to blockHeight = 90 to complete the epoch. Total 51 blocks have passed so
	// stake should already apply
	ts.AdvanceBlocks(31)
	epochShouldBeStaked := ts.EpochStart()

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"ShouldntBeStaked", epochShouldntBeStaked, false}, // 20 blocks have passed since the stake - not enough blocks (when EpochBlocks = 50)
		{"ShouldBeStaked", epochShouldBeStaked, true},      // 51 blocks have passed - enough blocks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if the provider/client are staked
			entries := ts.Keepers.Epochstorage.GetAllStakeEntriesForEpoch(ts.Ctx, tt.epoch)
			if tt.shouldBeStaked {
				require.Len(t, entries, 1)
			} else {
				require.Len(t, entries, 0)
			}
		})
	}
}
