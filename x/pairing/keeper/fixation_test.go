package keeper_test

import (
	"strconv"
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestEpochPaymentDeletionWithMemoryShortening(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 providers, 1 clients, default providers-to-pair

	epochsToSave := ts.EpochsToSave()

	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)
	clientAcct, _ := ts.GetAccount(common.CONSUMER, 0)

	// make payment request
	cusum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 1, cusum, ts.BlockHeight(), 0)

	sig, err := sigs.Sign(clientAcct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NoError(t, err)

	// shorten memory
	paramKey := string(epochstoragetypes.KeyEpochsToSave)
	paramVal := "\"" + strconv.FormatUint(epochsToSave/2, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	// make another request
	relaySession.SessionId++

	sig, err = sigs.Sign(clientAcct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NoError(t, err)

	// check that both payments were deleted
	ts.AdvanceEpochs(epochsToSave)

	res, err := ts.QueryProjectDeveloper(clientAcct.Addr.String())
	require.NoError(t, err)
	res2, err := ts.QueryPairingProviderEpochCu(providerAddr, res.Project.Index, ts.spec.Index)
	require.NoError(t, err)
	require.Len(t, res2.Info, 0)

	list0 := ts.Keepers.Pairing.GetAllProviderEpochComplainerCuStore(ts.Ctx)
	require.Len(t, list0, 0)

	list1 := ts.Keepers.Pairing.GetAllUniqueEpochSessionStore(ts.Ctx)
	require.Len(t, list1, 0)

	list2 := ts.Keepers.Pairing.GetAllProviderConsumerEpochCuStore(ts.Ctx)
	require.Len(t, list2, 0)
}
