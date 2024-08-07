package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/stretchr/testify/require"
)

func TestRelayPaymentOverUseWithDowntime(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	maxcu := ts.plan.PlanPolicy.EpochCuLimit
	relaySession := ts.newRelaySession(providerAddr, 0, maxcu*2, ts.BlockHeight(), 0)
	sig, err := sigs.Sign(client1Acct.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig
	// force a downtime of factor 2
	dtParams := ts.Keepers.Downtime.GetParams(ts.Ctx)
	// this gives us a downtime with factor 2, because the downtime is equal to the
	// estimated epoch duration, which means 2 epochs should have passed because we
	// had one epoch of accumulated downtime.
	doubledCUDowntime := dtParams.EpochDuration * 1
	ts.Keepers.Downtime.RecordDowntime(ts.Ctx, doubledCUDowntime)

	// we expect relay to be paid.
	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NoError(t, err)
}
