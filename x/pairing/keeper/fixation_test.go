package keeper_test

import (
	"strconv"
	"testing"

	"github.com/lavanet/lava/testutil/common"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
	signRelaySession(relaySession, clientAcct.SK) // updates relayRequest

	_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.Nil(t, err)

	// shorten memory
	paramKey := string(epochstoragetypes.KeyEpochsToSave)
	paramVal := "\"" + strconv.FormatUint(epochsToSave/2, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	// make another request
	relaySession.SessionId++
	signRelaySession(relaySession, clientAcct.SK) // updates relayRequest

	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.Nil(t, err)

	// check that both payments were deleted
	ts.AdvanceEpochs(epochsToSave)

	res, err := ts.QueryPairingListEpochPayments()
	require.Nil(t, err)
	require.Equal(t, 0, len(res.EpochPayments))
}
