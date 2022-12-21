package keeper_test

import (
	"math"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer1, spec, stake, false)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer2, spec, stake, false)

	providers := []common.Account{}
	for i := 1; i <= 1000; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake, true)
		providers = append(providers, provider)
	}

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	providers1, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	providers2, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer2.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers2))

	diffrent := false

	for _, provider := range providers1 {
		found := false
		for _, provider2 := range providers2 {
			if provider.Address == provider2.Address {
				found = true
			}
		}
		if !found {
			diffrent = true
		}
	}

	require.True(t, diffrent)

}

// Test that verifies that new get-pairing return values (CurrentEpoch, TimeLeftToNextPairing, SpecLastUpdatedBlock) is working properly
func TestGetPairing(t *testing.T) {

	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch so the pairing between the staked client and provider will be applied, and another one
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// construct get-pairing request
	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: ts.clients[0].address.String()}

	// get pairing for client
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	// verify the expected provider
	require.Equal(t, ts.providers[0].address.String(), pairing.Providers[0].Address)

	// verify the current epoch
	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	require.Equal(t, currentEpoch, pairing.CurrentEpoch)

	// verify the SpecLastUpdatedBlock
	specLastUpdatedBlock := ts.spec.BlockLastUpdated
	require.Equal(t, specLastUpdatedBlock, pairing.SpecLastUpdatedBlock)

	// init timestamps list
	epochBlockDivider := ts.keepers.Pairing.GetEpochBlockDivider()
	timestampList := make([]time.Time, epochBlockDivider)

	// get timestamps from previous epoch
	prevEpoch, err := ts.keepers.Epochstorage.GetPreviousEpochStartForBlock(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
	require.Nil(t, err)

	// calculate sample step according to EpochBlockDivider
	epochBlocks := ts.keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ts.ctx))
	sampleStep := epochBlocks / epochBlockDivider

	// get timestamps
	loopCounter := 0
	for block := prevEpoch; block < currentEpoch; block = block + sampleStep {
		blockCore := ts.keepers.BlockStore.LoadBlock(int64(block))
		timestampList[loopCounter] = blockCore.Time
		loopCounter++
	}

	// calculate average block time
	averageBlockTime := math.MaxFloat64
	for i := 1; i < len(timestampList); i++ {
		currAverageBlockTime := timestampList[i].Sub(timestampList[i-1]).Seconds() / float64(sampleStep)
		if currAverageBlockTime < averageBlockTime {
			averageBlockTime = currAverageBlockTime
		}
	}

	// Get the next epoch
	nextEpochStart, err := ts.keepers.Epochstorage.GetNextEpoch(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
	require.Nil(t, err)

	// Get epochBlocksOverlap
	overlapBlocks := ts.keepers.Pairing.EpochBlocksOverlap(sdk.UnwrapSDKContext(ts.ctx))

	// Get number of blocks from the current block to the next epoch
	blocksUntilNewEpoch := nextEpochStart + overlapBlocks - uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

	// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
	timeLeftToNextEpoch := blocksUntilNewEpoch * uint64(averageBlockTime)

	// verify the TimeLeftToNextPairing
	require.Equal(t, timeLeftToNextEpoch, pairing.TimeLeftToNextPairing)
}
