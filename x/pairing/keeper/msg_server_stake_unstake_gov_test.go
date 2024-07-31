package keeper_test

import (
	"strconv"
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
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
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"ShouldntBeStaked", epochShouldntBeStaked, true}, // 11 blocks have passed since the stake - not enough blocks
		{"ShouldBeStaked", epochShouldBeStaked, true},     // 21 blocks have passed - enough blocks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if the provider/client are staked
			_, foundProvider, _ := ts.Keepers.Epochstorage.GetEpochStakeEntries(ts.Ctx, tt.epoch, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, foundProvider)
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
			_, found, _ := ts.Keepers.Epochstorage.GetEpochStakeEntries(ts.Ctx, tt.epoch, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}
}

// Test that if UnstakeHoldBlocks decreases then the provider gets the funds back only by the original
// value (return_funds_block = next_epoch(current_block + max(UnstakeHoldBlocks, BlocksToSave)))
func TestUnstakeGovUnstakeHoldBlocksDecrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 0, 0) // 1 provider, 0 client, default providers-to-pair

	providerAcct, _ := ts.GetAccount(common.PROVIDER, 0)

	// make sure EpochBlocks default value is 20 and UnstakeHoldBlocks is 210
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	paramKey = string(epochstoragetypes.KeyUnstakeHoldBlocks)
	paramVal = "\"" + strconv.FormatUint(210, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance an epoch to apply the changes
	ts.AdvanceEpoch() // blockHeight = 20

	// change the UnstakeHoldBlocks parameter to 60
	paramKey = string(epochstoragetypes.KeyUnstakeHoldBlocks)
	paramVal = "\"" + strconv.FormatUint(60, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance to blockHeight = 39, one block before the UnstakeHoldBlocks change apply
	ts.AdvanceBlocks(19)

	// Unstake the provider:
	// should get the funds on block #260 (39+210 = 249 -> next epoch start in 260)
	_, err = ts.TxPairingUnstakeProvider(providerAcct.GetVaultAddr(), ts.spec.Index)
	require.NoError(t, err)

	// Advance a block to complete the epoch and apply UnstakeHoldBlocks change to 60
	// if the unstaking refers to 60 (wrongly, since they unstaked before the change applied),
	// they are supposed to get the funds back on block #240 (39+200(=BlockToSave) = 240 ->
	// this is a start of a new epoch)
	ts.AdvanceBlock() // blockHeight = 40

	// Advance 10 epochs to get to block #240. At this point, they shouldn't get their funds back
	ts.AdvanceEpochs(10)

	// Advance another epoch to block #260. Now they should get their funds back
	ts.AdvanceEpoch() // blockHeight = 260
}

// Test that if the UnstakeHoldBlocks param increases make then the provider gets the funds back only
// by the original value (return_funds_block = next_epoch(current_block + max(UnstakeHoldBlocks, BlocksToSave)))
func TestUnstakeGovUnstakeHoldBlocksIncrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 0, 0) // 1 provider, 0 client, default providers-to-pair

	providerAcct, _ := ts.GetAccount(common.PROVIDER, 0)

	// make sure EpochBlocks default value is 20 and UnstakeHoldBlocks is 210
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	paramKey = string(epochstoragetypes.KeyUnstakeHoldBlocks)
	paramVal = "\"" + strconv.FormatUint(210, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance an epoch to apply the changes.
	ts.AdvanceEpoch() // blockHeight = 20

	// change the UnstakeHoldBlocks parameter to 280
	paramKey = string(epochstoragetypes.KeyUnstakeHoldBlocks)
	paramVal = "\"" + strconv.FormatUint(280, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	// Advance to blockHeight = 39, one block before the UnstakeHoldBlocks change apply
	ts.AdvanceBlocks(19)

	// Unstake the provider:
	// should get the funds back on block #260 (39+210 = 249 -> next epoch start in 260)
	_, err = ts.TxPairingUnstakeProvider(providerAcct.GetVaultAddr(), ts.spec.Index)
	require.NoError(t, err)

	// Advance a block to complete the epoch and apply UnstakeHoldBlocks change to 280
	// if the unstaking refers to 280 (wrongly, since they unstaked before the change applied),
	// they are supposed to get the funds back on block #320 (40+280 = 320 ->
	// this is a start of a new epoch)
	ts.AdvanceBlock() // blockHeight = 40

	// Advance 11 epochs to get to block #260. At this point, they should get their funds back
	ts.AdvanceEpochs(11)

	// Advance 3 more epoch to block #320. Now they definitely should get their funds back
	ts.AdvanceEpochs(14)
}
