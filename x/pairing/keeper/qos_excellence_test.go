package keeper_test

import (
	"errors"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// TODO: All tests are not implemented since providerQosFS update and Qos score are not implemented yet

// TestAggregateQosExcellence tests that aggregating QoS excellence works as expected
func TestAggregateQosExcellence(t *testing.T) {
	ts := newTester(t)
	validQos := types.QualityOfServiceReport{Latency: math.LegacyOneDec(), Sync: math.LegacyOneDec(), Availability: math.LegacyOneDec()}
	invalidQos := types.QualityOfServiceReport{Latency: math.LegacyZeroDec(), Sync: math.LegacyZeroDec(), Availability: math.LegacyZeroDec()}
	pcec := types.NewProviderConsumerEpochCu()
	err := errors.New("")
	templates := []struct {
		name           string
		qosToAdd       types.QualityOfServiceReport
		expectedSum    math.LegacyDec
		expectedAmount uint64
		success        bool
	}{
		{"valid", validQos, math.LegacyOneDec(), uint64(1), true},
		{"another valid", validQos, math.LegacyNewDec(2), uint64(2), true},
		{"invalid", invalidQos, math.LegacyNewDec(2), uint64(2), false},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			pcec, err = ts.Keepers.Pairing.AggregateQosExcellence(pcec, &tt.qosToAdd)
			if tt.success {
				require.NoError(t, err)
				require.True(t, pcec.QosSum.Equal(tt.expectedSum))
				require.Equal(t, tt.expectedAmount, pcec.QosAmount)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestAggregateQosExcellenceWithRelayPayment tests that the ProviderConsumerEpochCu objects are correctly updated
// with QoS from relay payments
func TestAggregateQosExcellenceWithRelayPayment(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0)
	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	consumerAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	fmt.Printf("consumerAcc.Addr.String(): %v\n", consumerAcc.Addr.String())

	// create perfect QoS and mediocore QoS (all metrics = 0.5)
	qos1 := types.QualityOfServiceReport{Latency: math.LegacyOneDec(), Sync: math.LegacyOneDec(), Availability: math.LegacyOneDec()}
	qos2 := types.QualityOfServiceReport{Latency: math.LegacyNewDecWithPrec(5, 1), Sync: math.LegacyNewDecWithPrec(5, 1), Availability: math.LegacyNewDecWithPrec(5, 1)}
	qosArray := []types.QualityOfServiceReport{qos1, qos2}

	// create relays and send relay payment TX
	var relays []*types.RelaySession
	for i := range qosArray {
		relaySession := ts.newRelaySession(provider, uint64(i), 100, ts.BlockHeight(), 0)
		relaySession.QosExcellenceReport = &qosArray[i]
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relays = append(relays, relaySession)
	}
	_, err := ts.TxPairingRelayPayment(provider, relays...)
	require.NoError(t, err)

	// check ProviderConsumerEpochCu
	// the QosSum should be larger than 1 since qos1 score is 1 and qos2 score is >0 (0.5^(2/3))
	pcecs := ts.Keepers.Pairing.GetAllProviderConsumerEpochCu(ts.Ctx, ts.BlockHeight())
	require.Len(t, pcecs, 1)
	require.True(t, pcecs[0].ProviderConsumerEpochCu.QosSum.GT(math.LegacyOneDec()))
	require.Equal(t, uint64(2), pcecs[0].ProviderConsumerEpochCu.QosAmount)
}

// TestProviderQosMap checks that getting a providers' Qos map for specific chainID and cluster works properly
func TestProviderQosMap(t *testing.T) {
}

// TestGetQos checks that using GetQos() returns the right Qos
func TestGetQos(t *testing.T) {
}

// TestQosReqForSlots checks that if Qos req is active, all slots are assigned with Qos req
func TestQosReqForSlots(t *testing.T) {
}

// TestQosScoreCluster that consumer pairing uses the correct cluster for QoS score calculations.
func TestQosScoreCluster(t *testing.T) {
}

// TestQosScore checks that the qos score component is as expected (score == ComputeQos(), new users (sub usage less
// than a month) are not infuenced by Qos score, invalid Qos score == 1)
func TestQosScore(t *testing.T) {
}

// TestUpdateClusteringCriteria checks that updating the clustering criteria doesn't make different version clusters to be mixed
func TestUpdateClusteringCriteria(t *testing.T) {
}
