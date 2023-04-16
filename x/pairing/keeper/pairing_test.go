package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer1, spec, stake, false)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer2, spec, stake, false)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that 2 different clients get different pairings
	providers1, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	providers2, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer2.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers2))

	different := false

	for _, provider := range providers1 {
		found := false
		for _, provider2 := range providers2 {
			if provider.Address == provider2.Address {
				found = true
			}
		}
		if !found {
			different = true
		}
	}

	require.True(t, different)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that in different epoch we get different pairings for consumer1
	providers11, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers11))
	different = false
	for i := range providers1 {
		if providers1[i].Address != providers11[i].Address {
			different = true
			break
		}
	}
	require.True(t, different)

	// test that get pairing gives the same results for the whole epoch
	epochBlocks := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	foundIndexMap := map[string]int{}
	for i := uint64(0); i < epochBlocks-1; i++ {
		ctx = testkeeper.AdvanceBlock(ctx, keepers)

		providers111, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
		require.Nil(t, err)

		for i := range providers1 {
			require.Equal(t, providers11[i].Address, providers111[i].Address)

			providerAddr, err := sdk.AccAddressFromBech32(providers11[i].Address)
			require.Nil(t, err)
			valid, _, foundIndex, _, _, _, _ := keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr, providerAddr, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.True(t, valid)
			if _, ok := foundIndexMap[providers11[i].Address]; !ok {
				foundIndexMap[providers11[i].Address] = foundIndex
			} else {
				require.Equal(t, foundIndexMap[providers11[i].Address], foundIndex)
			}
		}
	}
}

func TestValidatePairingDeterminism(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer1, spec, stake, false)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer2, spec, stake, false)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that 2 different clients get different pairings
	pairedProviders, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)
	verifyPairingOncurrentBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	testAllProviders := func() {
		for idx, provider := range pairedProviders {
			providerAddress, err := sdk.AccAddressFromBech32(provider.Address)
			require.Nil(t, err)
			valid, _, foundIndex, _, _, _, errPairing := keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr, providerAddress, verifyPairingOncurrentBlock)
			require.Nil(t, errPairing)
			require.Equal(t, idx, foundIndex, "Failed ValidatePairingForClient", provider, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.True(t, valid)
		}
	}
	startBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	for i := startBlock; i < startBlock+(func() uint64 {
		blockToSave, err := keepers.Epochstorage.BlocksToSave(sdk.UnwrapSDKContext(ctx), i)
		require.Nil(t, err)
		return blockToSave
	})(); i++ {
		ctx = testkeeper.AdvanceBlock(ctx, keepers)
		testAllProviders()
	}
}

// Test that verifies that new get-pairing return values (CurrentEpoch, TimeLeftToNextPairing, SpecLastUpdatedBlock) is working properly
func TestGetPairing(t *testing.T) {
	// BLOCK_TIME = 30sec (testutil/keeper/keepers_init.go)
	constBlockTime := testkeeper.BLOCK_TIME

	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)
	// get epochBlocks (number of blocks in an epoch)
	epochBlocks := ts.keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - different epoch, valid tells if the payment request should work
	tests := []struct {
		name                string
		validPairingExists  bool
		isEpochTimesChanged bool
	}{
		{"zeroEpoch", false, false},
		{"firstEpoch", true, false},
		{"commonEpoch", true, false},
		{"epochTimesChanged", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Advance an epoch according to the test
			switch tt.name {
			case "zeroEpoch":
				// do nothing
			case "firstEpoch":
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			case "commonEpoch":
				for i := 0; i < 5; i++ {
					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
			case "epochTimesChanged":
				for i := 0; i < 5; i++ {
					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
				smallerBlockTime := constBlockTime / 2
				ts.ctx = testkeeper.AdvanceBlocks(ts.ctx, ts.keepers, int(epochBlocks)/2, smallerBlockTime)
				ts.ctx = testkeeper.AdvanceBlocks(ts.ctx, ts.keepers, int(epochBlocks)/2)
			}

			// construct get-pairing request
			pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: ts.clients[0].Addr.String()}

			// get pairing for client (for epoch zero there is no pairing -> expect to fail)
			pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
			if !tt.validPairingExists {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)

				// verify the expected provider
				require.Equal(t, ts.providers[0].Addr.String(), pairing.Providers[0].Address)

				// verify the current epoch
				currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
				require.Equal(t, currentEpoch, pairing.CurrentEpoch)

				// verify the SpecLastUpdatedBlock
				specLastUpdatedBlock := ts.spec.BlockLastUpdated
				require.Equal(t, specLastUpdatedBlock, pairing.SpecLastUpdatedBlock)

				// get timestamps from previous epoch
				prevEpoch, err := ts.keepers.Epochstorage.GetPreviousEpochStartForBlock(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
				require.Nil(t, err)

				// if prevEpoch == 0 -> averageBlockTime = 0, else calculate the time (like the actual get-pairing function)
				averageBlockTime := uint64(0)
				if prevEpoch != 0 {
					// get timestamps
					timestampList := []time.Time{}
					for block := prevEpoch; block <= currentEpoch; block++ {
						blockCore := ts.keepers.BlockStore.LoadBlock(int64(block))
						timestampList = append(timestampList, blockCore.Time)
					}

					// calculate average block time
					totalTime := uint64(0)
					for i := 1; i < len(timestampList); i++ {
						totalTime += uint64(timestampList[i].Sub(timestampList[i-1]).Seconds())
					}
					averageBlockTime = totalTime / epochBlocks
				}

				// Get the next epoch
				nextEpochStart, err := ts.keepers.Epochstorage.GetNextEpoch(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
				require.Nil(t, err)

				// Get epochBlocksOverlap
				overlapBlocks := ts.keepers.Pairing.EpochBlocksOverlap(sdk.UnwrapSDKContext(ts.ctx))

				// calculate the block in which the next pairing will happen (+overlap)
				nextPairingBlock := nextEpochStart + overlapBlocks

				// Get number of blocks from the current block to the next epoch
				blocksUntilNewEpoch := nextPairingBlock - uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

				// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
				timeLeftToNextPairing := blocksUntilNewEpoch * averageBlockTime

				// verify the TimeLeftToNextPairing
				if !tt.isEpochTimesChanged {
					require.Equal(t, timeLeftToNextPairing, pairing.TimeLeftToNextPairing)
				} else {
					// averageBlockTime in get-pairing query -> minimal average across sampled epoch
					// averageBlockTime in this test -> normal average across epoch
					// we've used a smaller blocktime some of the time -> averageBlockTime from get-pairing is smaller than the averageBlockTime calculated in this test
					require.Less(t, pairing.TimeLeftToNextPairing, timeLeftToNextPairing)
				}

				// verify nextPairingBlock
				require.Equal(t, nextPairingBlock, pairing.BlockOfNextPairing)
			}
		})
	}
}

func TestPairingStatic(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	spec.ProvidersTypes = spectypes.Spec_static
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	servicersToPair := keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ctx))

	consumer := common.CreateNewAccount(ctx, *keepers, balance)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer, spec, stake, false)

	for i := uint64(0); i < servicersToPair*2; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake+int64(i), true)
	}

	// we expect to get all the providers in static spec

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	providers, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr)
	require.Nil(t, err)

	for i, provider := range providers {
		require.Equal(t, provider.Stake.Amount.Int64(), stake+int64(i))
	}
}
