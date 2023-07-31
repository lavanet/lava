package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/stretchr/testify/require"
)

func TestRelayPaymentOverUseWithDowntime(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	maxcu := ts.plan.PlanPolicy.EpochCuLimit
	relaySession := ts.newRelaySession(providerAddr, 0, maxcu*2, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	// force a downtime of factor 2
	dtParams := ts.Keepers.Downtime.GetParams(ts.Ctx)
	doubledCUDowntime := dtParams.EpochDuration * 2 // this gives us a downtime with factor 2
	ts.Keepers.Downtime.RecordDowntime(ts.Ctx, doubledCUDowntime)

	// we expect relay to be paid.
	_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NoError(t, err)
}
