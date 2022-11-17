package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

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

	// Advance to the next block so he EpochBlocks change apply
	ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers) // blockHeight = 40

	// Advance an epoch (10 blocks). from its stake, 11 blocks have passed so it shouldn't be staked (it staked when EpochBlocks was 20)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochProviderShouldntBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance another epoch (10 blocks). from its stake, 21 blocks have passed so it should be staked
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochProviderShouldBeStaked := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
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
			require.Equal(t, tt.valid, found)
		})
	}

}
