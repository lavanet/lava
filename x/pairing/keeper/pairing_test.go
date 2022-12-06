package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)
	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *epochstoragetypes.DefaultGenesis().EpochDetails)

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

	currentEpoch := keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ctx))
	providers1, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr, currentEpoch)
	require.Nil(t, err)

	providers2, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer2.Addr, currentEpoch)
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

// Test that get-pairing does get pairings according to the its input block
func TestGetPairingPairingFromDifferentBlocks(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*account, 0),
		clients:   make([]*account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Add a staked provider and client and get their address
	err := ts.addClient(1)
	require.Nil(t, err)
	clientAddress := ts.clients[len(ts.clients)-1].address.String()
	err = ts.addProvider(1)
	require.Nil(t, err)

	// Setup EpochBlocks = 20, BlocksOverlap = 5
	epochBlocksDefault := uint64(20)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksDefault, 10)+"\"")
	require.Nil(t, err)
	epochBlocksOverlapDefault := uint64(5)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyEpochBlocksOverlap), "\""+strconv.FormatUint(epochBlocksOverlapDefault, 10)+"\"")
	require.Nil(t, err)

	// Keep the current epoch (pairing not applied yet)
	epochBeforePairing := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance epoch so the client and provider will be paired and param setup will be applied (keep the epoch)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20
	epochAfterPairing := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - valid tells if there should be pairing
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"EpochBeforePairingApplied", epochBeforePairing, false}, // pairing shouldn't be applied
		{"EpochAfterPairingApplied", epochAfterPairing, true},    // pairing should be applied
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Execute get-pairing - check if there are providers by epoch
			req := pairingtypes.QueryGetPairingRequest{ChainID: ts.spec.GetIndex(), Client: clientAddress, BlockHeight: int64(tt.epoch)}
			_, err := ts.keepers.Pairing.GetPairing(ts.ctx, &req)
			if tt.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

		})
	}

}
