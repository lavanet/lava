package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test that if the QosWeight param changes before the provider collected its reward, the provider's payment is according to the last QosWeight value (QosWeight is not fixated)
// Provider reward formula: reward = reward*(QOSScore*QOSWeight + (1-QOSWeight))
func TestRelayPaymentGovQosWeightChange(t *testing.T) {

	// setup testnet with mock spec, a staked client and a staked provider
	ts := setupForPaymentTest(t)
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)

	// Create badQos - to see the effect of changing QosWeight, the provider need to provide bad service (here, his score is 0%)
	badQoS := &pairingtypes.QualityOfServiceReport{Latency: sdk.ZeroDec(), Availability: sdk.ZeroDec(), Sync: sdk.ZeroDec()}

	// Advance an epoch and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// Create new QosWeight value (=0.5) for SimulateParamChange() because current QosWeight value is 0
	initQos := sdk.NewDecWithPrec(5, 1)
	initQosBytes, _ := initQos.MarshalJSON()
	initQosStr := string(initQosBytes[:])

	// change the QoS weight parameter to 0.5
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyQoSWeight), initQosStr)
	require.Nil(t, err)

	// Advance an epoch (only then the parameter change will be applied) and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochQosWeightFiftyPercent := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Create new QosWeight value (=0.7) for SimulateParamChange() for testing
	newQos := sdk.NewDecWithPrec(7, 1)
	newQosBytes, _ := newQos.MarshalJSON()
	newQosStr := string(newQosBytes[:])

	// change the QoS weight parameter to 0.7
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyQoSWeight), newQosStr)
	require.Nil(t, err)

	// Advance an epoch (only then the parameter change will be applied) and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochQosWeightSeventyPercent := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - epoch before/after change, valid tells if the payment request should work
	tests := []struct {
		name      string
		epoch     uint64
		qosWeight sdk.Dec
		valid     bool
	}{
		{"PaymentSeventyPercentQosEpoch", epochQosWeightSeventyPercent, sdk.NewDecWithPrec(7, 1), true}, // payment collected for an epoch with QosWeight = 0.7
		{"PaymentFiftyPercentQosEpoch", epochQosWeightFiftyPercent, sdk.NewDecWithPrec(5, 1), false},    // payment collected for an epoch with QosWeight = 0.5, still provider should be effected by QosWeight = 0.7
	}

	sessionCounter := 0
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {

			// Create relay request that was done in the test's epoch. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelayRequest{
				Provider:        ts.providers[0].address.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(sessionCounter),
				ChainID:         ts.spec.Name,
				CuSum:           ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight:     int64(tt.epoch),
				RelayNum:        0,
				RequestBlock:    -1,
				QoSReport:       badQoS,
				DataReliability: nil,
			}

			// Sign and send the payment requests for block 0 tx
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Add the relay request to the Relays array (for relayPaymentMessage())
			var Relays []*pairingtypes.RelayRequest
			Relays = append(Relays, relayRequest)

			// Get provider's and consumer's balance before payment
			providerBalance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)

			// Make the payment
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
			require.Nil(t, err)

			// Check that the consumer's balance decreased correctly
			burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(relayRequest.CuSum))
			newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
			require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			// Compute the relay request's QoS score
			score, err := relayRequest.QoSReport.ComputeQoS()
			require.Nil(t, err)

			// Calculate how much the provider wants to get paid for its service
			mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
			want := mint.MulInt64(int64(relayRequest.CuSum))
			want = want.Mul(score.Mul(tt.qosWeight).Add(sdk.OneDec().Sub(tt.qosWeight)))

			// if valid, what the provider wants and what it got should be equal
			if tt.valid == true {
				require.Equal(t, providerBalance+want.TruncateInt64(), ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())
			} else {
				require.NotEqual(t, providerBalance+want.TruncateInt64(), ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())
			}
		})
	}
}
