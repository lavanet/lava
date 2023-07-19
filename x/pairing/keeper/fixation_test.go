package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestEpochPaymentDeletionWithMemoryShortening(t *testing.T) {
	ts := setupForPaymentTest(t) // reset the keepers state before each state
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	relayRequest := &pairingtypes.RelaySession{
		Provider:    ts.providers[0].Addr.String(),
		ContentHash: []byte(ts.spec.ApiCollections[0].Apis[0].Name),
		SessionId:   uint64(1),
		SpecId:      ts.spec.Name,
		CuSum:       ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10,
		Epoch:       sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:    0,
	}

	sig, err := sigs.SignStruct(ts.clients[0].SK, *relayRequest, sigs.PrepareRelaySession)
	relayRequest.Sig = sig
	require.Nil(t, err)

	// make payment request
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &pairingtypes.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*pairingtypes.RelaySession{relayRequest}})
	require.Nil(t, err)

	// shorten memory
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(epochsToSave/2, 10)+"\"")
	require.NoError(t, err)

	// advance epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// make another request
	relayRequest.SessionId++

	sig, err = sigs.SignStruct(ts.clients[0].SK, *relayRequest, sigs.PrepareRelaySession)
	relayRequest.Sig = sig
	require.Nil(t, err)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &pairingtypes.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*pairingtypes.RelaySession{relayRequest}})
	require.Nil(t, err)

	// check that both payments were deleted
	for i := 0; i < int(epochsToSave); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// check second payment was deleted
	ans, err := ts.keepers.Pairing.EpochPaymentsAll(ts.ctx, &pairingtypes.QueryAllEpochPaymentsRequest{})
	require.Nil(t, err)
	require.Equal(t, 0, len(ans.EpochPayments))
}
