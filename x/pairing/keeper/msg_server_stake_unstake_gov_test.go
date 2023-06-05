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

// Test that if the EpochBlocks param decreases make sure the provider/client is staked only after the original EpochBlocks value (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestStakeGovEpochBlocksDecrease(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Create a mock spec
	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20 - make sure it is 20
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 10
	epochBlocksTen := uint64(10)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTen, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Add a staked provider and client and get their address
	err = ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].Addr

	// Verify the provider/client paid for its stake request
	providerCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, providerCurrentFunds.Amount.Int64())

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance an epoch (10 blocks). from their stake, 11 blocks have passed so they shouldn't be staked (they staked when EpochBlocks was 20)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 50
	epochShouldntBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance another epoch (10 blocks). from their stake, 21 blocks have passed so they should be staked
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 60
	epochShouldBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

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
			_, foundProvider, _ := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, foundProvider)
		})
	}
}

// Test that if the EpochBlocks param increases make sure the provider/client is staked after the original EpochBlocks value, and not the new enlarged one (EpochBlocks = number of blocks in an epoch. This parameter is fixated)
func TestStakeGovEpochBlocksIncrease(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Create a mock spec
	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20 - make sure it is 20
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the EpochBlocks parameter to 50
	epochBlocksFifty := uint64(50)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksFifty, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the EpochBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Add a staked provider/client and get their address
	err = ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].Addr

	// Verify the provider/client paid for its stake request
	providerCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, providerCurrentFunds.Amount.Int64())

	// Advance to the next block so the EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance to blockHeight = 59. from their stake, 20 blocks have passed but they shouldn't be staked because the chain uses the new EpochBlocks value
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochShouldntBeStaked := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

	// Advance to blockHeight = 90 to complete the epoch. from its stake, 51 blocks have passed so it should definitely be staked
	for i := 0; i < 31; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}
	epochShouldAlsoBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests
	tests := []struct {
		name           string
		epoch          uint64
		shouldBeStaked bool
	}{
		{"ShouldBeStaked", epochShouldntBeStaked, false},      // 20 blocks have passed since the stake - not enough blocks (when EpochBlocks = 50)
		{"ShouldAlsoBeStaked", epochShouldAlsoBeStaked, true}, // 51 blocks have passed - enough blocks
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if the provider/client are staked
			_, found, _ := ts.keepers.Epochstorage.GetEpochStakeEntries(sdk.UnwrapSDKContext(ts.ctx), tt.epoch, epochstoragetypes.ProviderKey, ts.spec.GetIndex())
			require.Equal(t, tt.shouldBeStaked, found)
		})
	}
}

// Test that if the UnstakeHoldBlocks param decreases make sure the provider/client is getting their funds back only by the original UnstakeHoldBlocks value (return_funds_block = next_epoch(current_block + max(UnstakeHoldBlocks, BlocksToSave)))
func TestUnstakeGovUnstakeHoldBlocksDecrease(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Create a mock spec
	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	// Add a staked provider and client and get their address
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].Addr

	// Verify the provider/client paid for its stake request
	providerCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, providerCurrentFunds.Amount.Int64())

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20 and UnstakeHoldBlocks is 210 - make sure it is 20 and 210
	epochBlocksTwenty := uint64(20)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	unstakeHoldBlocksDefaultVal := uint64(210)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyUnstakeHoldBlocks), "\""+strconv.FormatUint(unstakeHoldBlocksDefaultVal, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply the changes. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the UnstakeHoldBlocks parameter to 60
	unstakeHoldBlocksNewVal := uint64(60)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyUnstakeHoldBlocks), "\""+strconv.FormatUint(unstakeHoldBlocksNewVal, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the UnstakeHoldBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Unstake the provider/client - they are supposed to get their funds back on block #260 (39+210 = 249 -> next epoch start in 260)
	_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &types.MsgUnstakeProvider{Creator: providerAddress.String(), ChainID: ts.spec.Index})
	require.Nil(t, err)

	// Advance a block to complete the epoch and apply UnstakeHoldBlocks change to 60
	// if the unstaking refers to 60 (wrongly, since they unstaked before the change was applied), they are supposed to get their funds back on block #240 (39+200(=BlockToSave) = 240 -> this is a start of a new epoch)
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance 10 epochs to get to block #240. At this point, they shouldn't get their funds back
	for i := 0; i < 10; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// check if the client/provider got their funds back - they shouldn't get it
	providerCurrentFunds = ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.NotEqual(t, balance, providerCurrentFunds.Amount.Int64())

	// Advance another epoch to block #260. Now they should get their funds back
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 59

	// check if the client/provider got their funds back - they should get it

	providerCurrentFunds = ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance, providerCurrentFunds.Amount.Int64())
}

// Test that if the UnstakeHoldBlocks param increases make sure the provider/client is getting their funds back only by the original UnstakeHoldBlocks value (return_funds_block = next_epoch(current_block + max(UnstakeHoldBlocks, BlocksToSave)))
func TestUnstakeGovUnstakeHoldBlocksIncrease(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Create a mock spec
	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	// Add a staked provider and client and get their address
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	providerAddress := ts.providers[len(ts.providers)-1].Addr

	// Verify the provider/client paid for its stake request
	providerCurrentFunds := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance-stake, providerCurrentFunds.Amount.Int64())

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20 and UnstakeHoldBlocks is 210 - make sure it is 20 and 210
	epochBlocksTwenty := uint64(20)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	unstakeHoldBlocksDefaultVal := uint64(210)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyUnstakeHoldBlocks), "\""+strconv.FormatUint(unstakeHoldBlocksDefaultVal, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply the changes. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// change the UnstakeHoldBlocks parameter to 280
	unstakeHoldBlocksNewVal := uint64(280)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyUnstakeHoldBlocks), "\""+strconv.FormatUint(unstakeHoldBlocksNewVal, 10)+"\"")
	require.Nil(t, err)

	// Advance to blockHeight = 39, one block before the UnstakeHoldBlocks change apply
	for i := 0; i < 19; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// Unstake the provider/client - they are supposed to get their funds back on block #260 (39+210 = 249 -> next epoch start in 260)
	_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &types.MsgUnstakeProvider{Creator: providerAddress.String(), ChainID: ts.spec.Index})
	require.Nil(t, err)

	// Advance a block to complete the epoch and apply UnstakeHoldBlocks change to 280
	// if the unstaking refers to 280 (wrongly, since they unstaked before the change was applied), they are supposed to get their funds back on block #320 (40+280 = 320 -> this is a start of a new epoch)
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance 11 epochs to get to block #260. At this point, they should get their funds back
	for i := 0; i < 11; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// check if the client/provider got their funds back - they should get it
	providerCurrentFunds = ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance, providerCurrentFunds.Amount.Int64())

	// Advance 3 more epoch to block #320. Now they definitely should get their funds back
	for i := 0; i < 3; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// check if the client/provider got their funds back - they should get it
	providerCurrentFunds = ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom)
	require.Equal(t, balance, providerCurrentFunds.Amount.Int64())
}
