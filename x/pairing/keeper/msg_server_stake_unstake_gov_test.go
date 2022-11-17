package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test that if the EpochBlocks param decreases make sure the provider is staked only after the original EpochBlocks value (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestStakeGovEpochBlocksDecrease(t *testing.T) {

	// setup testnet with mock spec
	ts := setupForPaymentTest(t)
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 10
	epochBlocksTen := uint64(10)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTen, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Add a staked provider and get its address
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].address

	// Verify the provider paid for its stake request
	firstProviderCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, firstProviderCurrentFunds.Amount.Int64())

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance an epoch (10 blocks). from its stake, 11 blocks have passed so it shouldn't be staked (it staked when EpochBlocks was 20)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 50
	epochProviderShouldntBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance another epoch (10 blocks). from its stake, 21 blocks have passed so it should be staked
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 60
	epochProviderShouldBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"providerShouldntBeStaked", epochProviderShouldntBeStaked, false}, // 11 blocks have passed since the stake - not enough blocks
		{"providerShouldBeStaked", epochProviderShouldBeStaked, true},      // 21 blocks have passed - enough blocks
	}

	sessionCounter := 0
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {

			// check if the provider is staked
			_, found := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}

}

// Test that if the EpochBlocks param increases make sure the provider is staked after the original EpochBlocks value, and not the new enlarged one (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestStakeGovEpochBlocksIncrease(t *testing.T) {

	// setup testnet with mock spec
	ts := setupForPaymentTest(t)
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 50
	epochBlocksFifty := uint64(50)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksFifty, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Add a staked provider and get its address
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].address

	// Verify the provider paid for its stake request
	firstProviderCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, firstProviderCurrentFunds.Amount.Int64())

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance to blockHeight = 59. from its stake, 20 blocks have passed so it should be staked (it staked when EpochBlocks was 20)
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochProviderShouldBeStaked := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

	// Advance to blockHeight = 90 to complete the epoch. from its stake, 51 blocks have passed so it should definitely be staked
	for i := 0; i < 31; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochProviderShouldAlsoBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"providerShouldBeStaked", epochProviderShouldBeStaked, true},         // 20 blocks have passed since the stake - should be enough blocks
		{"providerShouldAlsoBeStaked", epochProviderShouldAlsoBeStaked, true}, // 51 blocks have passed - enough blocks
	}

	sessionCounter := 0
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {

			// check if the provider is staked
			_, found := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}

}

// Test that if the EpochBlocks param decreases make sure the provider is unstaked only after the original EpochBlocks value (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestUnstakeGovEpochBlocksDecrease(t *testing.T) {

	// setup testnet with mock spec
	ts := setupForPaymentTest(t)
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Add a staked provider and get its address
	err := ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].address

	// Verify the provider paid for its stake request
	firstProviderCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, firstProviderCurrentFunds.Amount.Int64())

	// Advance an epoch to apply the provider's stake and because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 10
	epochBlocksTen := uint64(10)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTen, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Unstake the provider
	_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &types.MsgUnstakeProvider{Creator: providerAddress.String(), ChainID: ts.spec.Index})
	require.Nil(t, err)

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance an epoch (10 blocks). from its unstake, 11 blocks have passed so it shouldn't be unstaked (it unstaked when EpochBlocks was 20)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 50
	epochProviderShouldntBeUnstaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance another epoch (10 blocks). from its unstake, 21 blocks have passed so it should be unstaked
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 60
	epochProviderShouldBeUnstaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"providerShouldntBeUnstaked", epochProviderShouldntBeUnstaked, true}, // 11 blocks have passed since the stake - not enough blocks
		{"providerShouldBeUnstaked", epochProviderShouldBeUnstaked, false},    // 21 blocks have passed - enough blocks
	}

	sessionCounter := 0
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {

			// check if the provider is staked
			_, found := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}
}

// Test that if the EpochBlocks param increases make sure the provider is unstaked after the original EpochBlocks value, and not the new enlarged one (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestUnstakeGovEpochBlocksIncrease(t *testing.T) {

	// setup testnet with mock spec
	ts := setupForPaymentTest(t)
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Add a staked provider and get its address
	err := ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].address

	// Verify the provider paid for its stake request
	firstProviderCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, firstProviderCurrentFunds.Amount.Int64())

	// Advance an epoch to apply the provider's stake and because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 10
	epochBlocksFifty := uint64(50)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksFifty, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Unstake the provider
	_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &types.MsgUnstakeProvider{Creator: providerAddress.String(), ChainID: ts.spec.Index})
	require.Nil(t, err)

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance to blockHeight = 59. from its unstake, 20 blocks have passed so it should be unstaked (it staked when EpochBlocks was 20)
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochProviderShouldBeUnstaked := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

	// Advance to blockHeight = 90 to complete the epoch. from its stake, 51 blocks have passed so it should definitely be staked
	for i := 0; i < 31; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochProviderShouldAlsoBeUnstaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"providerShouldntBeUnstaked", epochProviderShouldBeUnstaked, false},   // 20 blocks have passed since the stake - should be enough blocks
		{"providerShouldBeUnstaked", epochProviderShouldAlsoBeUnstaked, false}, // 51 blocks have passed - enough blocks
	}

	sessionCounter := 0
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {

			// check if the provider is staked
			_, found := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}
}
