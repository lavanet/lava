package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
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

	// Create badQos - to see the effect of changing QosWeight, the provider need to provide bad service (here, his score is 0%)
	badQoS := &pairingtypes.QualityOfServiceReport{Latency: sdk.ZeroDec(), Availability: sdk.ZeroDec(), Sync: sdk.ZeroDec()}

	// Advance an epoch and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// Simulate QosWeight to be 0.5 - the default value in the time of this writing
	initQos := sdk.NewDecWithPrec(5, 1)
	initQosBytes, _ := initQos.MarshalJSON()
	initQosStr := string(initQosBytes[:])

	// change the QoS weight parameter to 0.5
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyQoSWeight), initQosStr)
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

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request that was done in the test's epoch. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
				QoSReport:   badQoS,
			}

			// Sign and send the payment requests for block 0 tx
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Add the relay request to the Relays array (for relayPaymentMessage())
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)

			// Get provider's and consumer's balance before payment
			providerBalance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)

			// Make the payment
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
			require.Nil(t, err)

			// Check that the consumer's balance decreased correctly
			burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(relayRequest.CuSum))
			newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
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

// Test that if the EpochBlocks param decreases make sure the provider can claim reward after the new EpochBlocks*EpochsToSave, and not the original EpochBlocks (EpochBlocks = number of blocks in an epoch)
func TestRelayPaymentGovEpochBlocksDecrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	initEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Get the current values of EpochBlocks and EpochsToSave
	epochBlocks, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)
	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochBeforeChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Decrease the epochBlocks param
	newDecreasedEpochBlocks := epochBlocks / 2
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(newDecreasedEpochBlocks, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochAfterChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// The heart of the test. Make sure that the number of blocks that we'll advance is smaller than memory limit of a provider that uses the old EpochBlocks
	require.Less(t, (epochsToSave+1)*newDecreasedEpochBlocks+epochAfterChange, epochBeforeChange+(epochsToSave*epochBlocks))

	// Advance EpochsToSave+1 epochs. This will create a situation where a provider with the old EpochBlocks can get paid, but shouldn't
	for i := 0; i < int(epochsToSave)+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// define tests - different epoch, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochBlocksChanges", epochBeforeChange, false},
		{"PaymentAfterEpochBlocksChanges", epochAfterChange, false},
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request that was done in the test's epoch. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)
			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

// TODO: Currently the test passes since PaymentBeforeEpochBlocksChangesToFifty's value is false. It should be true. After bug CNS-83 is fixed, change this test
// Test that if the EpochBlocks param increases make sure the provider can claim reward after the new EpochBlocks*EpochsToSave, and not the original EpochBlocks (EpochBlocks = number of blocks in an epoch)
func TestRelayPaymentGovEpochBlocksIncrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	initEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Get the current values of EpochBlocks and EpochsToSave
	epochBlocks, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)
	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochBeforeChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Increase the epochBlocks param
	newIncreasedEpochBlocks := epochBlocks * 2
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(newIncreasedEpochBlocks, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochAfterChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Calculate the memory limit of a provider with the new EpochBlocks (in epochs) from the current epoch
	memoryLimit := (epochBlocks * epochsToSave) + epochBeforeChange
	memoryLimitInEpochsUsingNewEpochBlocks := (memoryLimit - epochAfterChange) / newIncreasedEpochBlocks // amount of epochs (of new EpochBlocks) that we need to advance to get to the memory of a provider that uses the old EpochBlocks

	// The heart of the test. Make sure that the number of epochs that we'll advance is smaller than EpochsToSave (we're gonna advance memoryLimitInEpochsUsingNewEpochBlocks+1, and we already advanced an epoch before)
	require.Less(t, memoryLimitInEpochsUsingNewEpochBlocks+2, epochsToSave)

	// Advance enough epochs to create a situation where a provider with the old EpochBlocks can't be paid, which shouldn't happen (from old EpochBlocks perspective, too many blocks past. But, the number of epochs that past are smaller than EpochsToSave)
	for i := 0; i < int(memoryLimitInEpochsUsingNewEpochBlocks)+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochBlocksChange", epochBeforeChange, false},
		{"PaymentAfterEpochBlocksChange", epochAfterChange, true},
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request that was done in the test's epoch+block. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)
			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

// Test that if the EpochToSave param decreases make sure the provider can claim reward after the new EpochBlocks*EpochsToSave, and not the original EpochBlocks (EpochsToSave = number of epochs the chain remembers (accessible memory))
func TestRelayPaymentGovEpochToSaveDecrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // 20
	initEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Get the current values of EpochBlocks and EpochsToSave
	epochBlocks, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)
	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochBeforeChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Decrease the epochBlocks param
	newDecreasedEpochsToSave := epochsToSave / 2
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(newDecreasedEpochsToSave, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochAfterChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// The heart of the test. Make sure that the number of epochs that we'll advance from epochBeforeChange is smaller than EpochsToSave (we're gonna advance newDecreasedEpochsToSave, and we already advanced 2 epochs before)
	require.Less(t, newDecreasedEpochsToSave+2, epochsToSave)

	// Advance epochs to create a situation where a provider with old EpochsToSave from epochBeforeChange can get paid (but it shouldn't, since we advanced newDecreasedEpochsToSave+2 epochs). Also a provider from epochAfterChange with the new EpochsToSave should get paid since we're one block before newDecreasedEpochsToSave+1 (which is the end of the memory for the new EpochsToSave)
	for i := 0; i < int(newDecreasedEpochsToSave); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	for i := 0; i < int(epochBlocks)-1; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
	}

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochsToSaveChanges", epochBeforeChange, false},
		{"PaymentAfterEpochsToSaveChangesBlockInMemory", epochAfterChange, true},         // the chain is one block before the memory ends for the provider from epochAfterChange
		{"PaymentAfterEpochsToSaveChangesBlockOutsideOfMemory", epochAfterChange, false}, // the chain advances inside the loop test when it reaches this test. Here we pass the memory of the provider from epochAfterChange
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// advance to one block to reach the start of the newDecreasedEpochsToSave+1 epoch -> the provider from epochAfterChange shouldn't be able to get its payments (epoch out of memory)
			if ti == 2 {
				ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
			}

			// Create relay request that was done in the test's epoch+block. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)
			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

// TODO: Currently the test passes since PaymentBeforeEpochsToSaveChangesToTwenty's value is false. It should be true. After bug CNS-83 is fixed, change this test
// Test that if the EpochToSave param increases make sure the provider can claim reward after the new EpochBlocks*EpochsToSave, and not the original EpochBlocks (EpochsToSave = number of epochs the chain remembers (accessible memory))
func TestRelayPaymentGovEpochToSaveIncrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20, and EpochsToSave is 10 - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	epochsToSaveTen := uint64(10)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(epochsToSaveTen, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)                                             // blockHeight = 20
	epochBeforeChangeToTwenty := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx)) // blockHeight = 20

	// change the EpochToSave parameter to 20
	epochsToSaveTwenty := uint64(20)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(epochsToSaveTwenty, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch so the change applies
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 40
	epochAfterChangeToTwenty := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Advance epochs to reach blockHeight of 260
	// This will create a situation where a provider with request from epochBeforeChangeToTwenty shouldn't get payment, and from epochAfterChangeToTwenty should
	for i := 0; i < 11; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochsToSaveChangesToTwenty", epochBeforeChangeToTwenty, false}, // first block of current epoch
		{"PaymentAfterEpochsToSaveChangesToTwenty", epochAfterChangeToTwenty, true},    // first block of previous epoch
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request that was done in the test's epoch+block. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)
			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

// Test that if the StakeToMaxCU.MaxCU param decreases make sure the client can send queries according to the original StakeToMaxCUList in the current epoch (This parameter is fixated)
func TestRelayPaymentGovStakeToMaxCUListMaxCUDecrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20,and the default StakeToMaxCU list below - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	DefaultStakeToMaxCUList := pairingtypes.StakeToMaxCUList{List: []pairingtypes.StakeToMaxCU{
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(1)}, MaxComputeUnits: 5000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(500)}, MaxComputeUnits: 15000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(2000)}, MaxComputeUnits: 50000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(5000)}, MaxComputeUnits: 250000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(100000)}, MaxComputeUnits: 500000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(9999900000)}, MaxComputeUnits: 9999999999},
	}}
	stakeToMaxCUListBytes, _ := DefaultStakeToMaxCUList.MarshalJSON()
	stakeToMaxCUListStr := string(stakeToMaxCUListBytes[:])
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20
	epochBeforeChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Find the stakeToMaxEntry that is compatible to our client
	stakeToMaxCUList, _ := ts.keepers.Pairing.StakeToMaxCUList(sdk.UnwrapSDKContext(ts.ctx), 0)
	stakeToMaxCUEntryIndex := -1
	for index, stakeToMaxCUEntry := range stakeToMaxCUList.GetList() {
		if stakeToMaxCUEntry.MaxComputeUnits == uint64(500000) {
			stakeToMaxCUEntryIndex = index
			break
		}
	}
	require.NotEqual(t, stakeToMaxCUEntryIndex, -1)

	// Create new stakeToMaxCUEntry with the same stake threshold but higher MaxComuteUnits and put it in stakeToMaxCUList. For maxCU of 600000, the client will be able to use 300000CU (because maxCU is divided by servicersToPairCount)
	newStakeToMaxCUEntry := pairingtypes.StakeToMaxCU{StakeThreshold: stakeToMaxCUList.List[stakeToMaxCUEntryIndex].StakeThreshold, MaxComputeUnits: uint64(600000)}
	stakeToMaxCUList.List[stakeToMaxCUEntryIndex] = newStakeToMaxCUEntry

	// change the stakeToMaxCUList parameter
	stakeToMaxCUListBytes, _ = stakeToMaxCUList.MarshalJSON()
	stakeToMaxCUListStr = string(stakeToMaxCUListBytes[:])
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)
	require.Nil(t, err)

	// Advance an epoch (only then the parameter change will be applied) and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochAfterChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - different epochs, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeStakeToMaxCUListChange", epochBeforeChange, false}, // maxCU for this epoch is 250000, so it should fail
		{"PaymentAfterStakeToMaxCUListChange", epochAfterChange, true},    // maxCU for this epoch is 300000, so it should succeed
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       uint64(250001), // the relayRequest costs 250001 (more than the previous limit, and less than in the new limit). This should influence the validity of the request
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests for block 20 (=epochBeforeChange)
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Add the relay request to the Relays array (for relayPaymentMessage())
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)

			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

// Test that if the StakeToMaxCU.StakeThreshold param increases make sure the client can send queries according to the original StakeToMaxCUList in the current epoch (This parameter is fixated)
func TestRelayPaymentGovStakeToMaxCUListStakeThresholdIncrease(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20,and the default StakeToMaxCU list below - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	DefaultStakeToMaxCUList := pairingtypes.StakeToMaxCUList{List: []pairingtypes.StakeToMaxCU{
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(1)}, MaxComputeUnits: 5000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(500)}, MaxComputeUnits: 15000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(2000)}, MaxComputeUnits: 50000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(5000)}, MaxComputeUnits: 250000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(100000)}, MaxComputeUnits: 500000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(9999900000)}, MaxComputeUnits: 9999999999},
	}}
	stakeToMaxCUListBytes, _ := DefaultStakeToMaxCUList.MarshalJSON()
	stakeToMaxCUListStr := string(stakeToMaxCUListBytes[:])
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20
	epochBeforeChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Find the stakeToMaxEntry that is compatible to our client
	stakeToMaxCUList, _ := ts.keepers.Pairing.StakeToMaxCUList(sdk.UnwrapSDKContext(ts.ctx), 0)
	stakeToMaxCUEntryIndex := -1
	for index, stakeToMaxCUEntry := range stakeToMaxCUList.GetList() {
		if stakeToMaxCUEntry.MaxComputeUnits == uint64(500000) {
			stakeToMaxCUEntryIndex = index
			break
		}
	}
	require.NotEqual(t, stakeToMaxCUEntryIndex, -1)

	// Create new stakeToMaxCUEntry with the same MaxCU but higher StakeThreshold (=110000) and put it in stakeToMaxCUList. The client is staked with 100000ulava, so if it will downgrade to lower MaxCU, it'll get MaxCU = 250000 (per provider: 125000)
	newStakeToMaxCUEntry := pairingtypes.StakeToMaxCU{StakeThreshold: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(110000)), MaxComputeUnits: stakeToMaxCUList.List[stakeToMaxCUEntryIndex].MaxComputeUnits}
	stakeToMaxCUList.List[stakeToMaxCUEntryIndex] = newStakeToMaxCUEntry

	// change the stakeToMaxCUList parameter
	stakeToMaxCUListBytes, _ = stakeToMaxCUList.MarshalJSON()
	stakeToMaxCUListStr = string(stakeToMaxCUListBytes[:])
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)
	require.Nil(t, err)

	// Advance an epoch (only then the parameter change will be applied) and get current epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	epochAfterChange := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - different epochs, valid tells if the payment request should work
	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeStakeToMaxCUListChange", epochBeforeChange, true}, // StakeThreshold for this epoch allows MaxCU = 250000, so it should work
		{"PaymentAfterStakeToMaxCUListChange", epochAfterChange, false},  // StakeThreshold for this epoch allows MaxCU = 125000, so it shouldn't work
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       uint64(200000), // the relayRequest costs 200000 (less than the previous limit, and more than in the new limit). This should influence the validity of the request
				BlockHeight: int64(tt.epoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests for block 20 (=epochBeforeChange)
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Add the relay request to the Relays array (for relayPaymentMessage())
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)

			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

func TestRelayPaymentGovEpochBlocksMultipleChanges(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20, and EpochsToSave is 10 - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	epochsToSaveTen := uint64(10)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(epochsToSaveTen, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// struct that holds the new values for EpochBlocks and the block the chain will advance to
	epochTests := []struct {
		epochBlocksNewValues uint64 // EpochBlocks new value
		epochNum             uint64 // The number of epochs the chain will advance (after EpochBlocks changed)
		blockNum             uint64 // The number of blocks the chain will advance (after EpochBlocks changed)
	}{
		{4, 3, 5},    // Test #0 - latest epoch start: 52
		{9, 0, 7},    // Test #1 - latest epoch start: 56
		{24, 37, 42}, // Test #2 - latest epoch start: 953
		{41, 45, 30}, // Test #3 - latest epoch start: 2781
		{36, 15, 12}, // Test #4 - latest epoch start: 3326
		{25, 40, 7},  // Test #5 - latest epoch start: 4337
		{5, 45, 22},  // Test #6 - latest epoch start: 4602
		{45, 37, 18}, // Test #7 - latest epoch start: 6227
	}

	// define tests - for each test, the paymentEpoch will be +-1 of the latest epoch start of the test
	tests := []struct {
		name         string // Test name
		paymentEpoch uint64 // The epoch inside the relay request (the payment is requested according to this epoch)
		valid        bool   // Is the test supposed to succeed?
	}{
		{"Test #1", 51, true},
		{"Test #2", 57, true},
		{"Test #3", 952, true},
		{"Test #4", 2782, true},
		{"Test #5", 3325, true},
		{"Test #6", 4338, true},
		{"Test #7", 4601, true},
		{"Test #8", 6228, true},
	}

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// change the EpochBlocks parameter according to the epoch test values
			epochBlocksNew := uint64(epochTests[ti].epochBlocksNewValues)
			err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksNew, 10)+"\"")
			require.Nil(t, err)

			// Advance epochs according to the epoch test values
			for i := 0; i < int(epochTests[ti].epochNum); i++ {
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			}

			// Advance blocks according to the epoch test values
			for i := 0; i < int(epochTests[ti].blockNum); i++ {
				ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
			}

			// Create relay request that was done in the test's epoch+block. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relayRequest := &pairingtypes.RelaySession{
				Provider:    ts.providers[0].address.String(),
				ContentHash: []byte(ts.spec.Apis[0].Name),
				SessionId:   uint64(ti),
				ChainID:     ts.spec.Name,
				CuSum:       ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight: int64(tt.paymentEpoch),
				RelayNum:    0,
			}

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*pairingtypes.RelaySession
			Relays = append(Relays, relayRequest)
			relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].address, ts.providers[0].address)
		})
	}
}

func TestRelayPaymentGovStakeToMaxCUListStakeThresholdMultipleChanges(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	// Also, the client and provider will be paired (new pairing is determined every epoch)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20,and the default StakeToMaxCU list below - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	DefaultStakeToMaxCUList := pairingtypes.StakeToMaxCUList{List: []pairingtypes.StakeToMaxCU{
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(1)}, MaxComputeUnits: 5000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(500)}, MaxComputeUnits: 15000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(2000)}, MaxComputeUnits: 50000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(5000)}, MaxComputeUnits: 250000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(100000)}, MaxComputeUnits: 500000},
		{StakeThreshold: sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(9999900000)}, MaxComputeUnits: 9999999999},
	}}
	stakeToMaxCUListBytes, _ := DefaultStakeToMaxCUList.MarshalJSON()
	stakeToMaxCUListStr := string(stakeToMaxCUListBytes[:])
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	// Get the StakeToMaxCU list
	stakeToMaxCUList, _ := ts.keepers.Pairing.StakeToMaxCUList(sdk.UnwrapSDKContext(ts.ctx), 0)

	// struct that holds the new values for EpochBlocks and the block the chain will advance to
	stakeToMaxCUThresholdTests := []struct {
		newStakeThreshold int64  // newStakeThreshold new value
		newMaxCU          uint64 // MaxCU new value
		stakeToMaxCUIndex int    // stakeToMaxCU entry index that will change (see types/params.go for the default values)
	}{
		{10, 20000, 0},   // Test #0
		{400, 16000, 1},  // Test #1
		{2001, 14000, 2}, // Test #2
		{1, 0, 0},        // Test #3
	}

	// define tests - for each test, the paymentEpoch will be +-1 of the latest epoch start of the test
	tests := []struct {
		name  string // Test name
		valid bool   // Is the change of StakeToMaxCUList entry valid?
	}{
		{"Test #0", false},
		{"Test #1", true},
		{"Test #2", false},
		{"Test #3", true},
	}

	for ti, tt := range tests {

		// Get current StakeToMaxCU list
		stakeToMaxCUList = ts.keepers.Pairing.StakeToMaxCUListRaw(sdk.UnwrapSDKContext(ts.ctx))

		// Create new stakeToMaxCUEntry with the same stake threshold but higher MaxComuteUnits and put it in stakeToMaxCUList. I picked the stake entry with: StakeThreshold = 100000ulava, MaxCU = 500000
		newStakeToMaxCUEntry := pairingtypes.StakeToMaxCU{StakeThreshold: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stakeToMaxCUThresholdTests[ti].newStakeThreshold)), MaxComputeUnits: stakeToMaxCUThresholdTests[ti].newMaxCU}
		stakeToMaxCUList.List[stakeToMaxCUThresholdTests[ti].stakeToMaxCUIndex] = newStakeToMaxCUEntry

		// change the stakeToMaxCUList parameter
		stakeToMaxCUListBytes, _ := stakeToMaxCUList.MarshalJSON()
		stakeToMaxCUListStr := string(stakeToMaxCUListBytes[:])
		err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, pairingtypes.ModuleName, string(pairingtypes.KeyStakeToMaxCUList), stakeToMaxCUListStr)

		// Advance an epoch (only then the parameter change will be applied) and get current epoch
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

		if tt.valid {
			require.Nil(t, err)
		} else {
			require.NotNil(t, err)
		}

	}
}

// this test checks what happens if a single provider stake, get payment, and then unstake and gets its money.
func TestStakePaymentUnstake(t *testing.T) {
	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)

	// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = initEpochBlocks

	// The test assumes that EpochBlocks default value is 20, and EpochsToSave is 10, and unstakeHoldBlocks is 210 - make sure it is
	epochBlocksTwenty := uint64(20)
	err := testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(epochBlocksTwenty, 10)+"\"")
	require.Nil(t, err)
	epochsToSaveTen := uint64(10)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochsToSave), "\""+strconv.FormatUint(epochsToSaveTen, 10)+"\"")
	require.Nil(t, err)
	unstakeHoldBlocksDefaultVal := uint64(210)
	err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyUnstakeHoldBlocks), "\""+strconv.FormatUint(unstakeHoldBlocksDefaultVal, 10)+"\"")
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change. From here, the documented blockHeight is with offset of initEpochBlocks
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // blockHeight = 20

	relayRequest := &pairingtypes.RelaySession{
		Provider:    ts.providers[0].address.String(),
		ContentHash: []byte(ts.spec.Apis[0].Name),
		SessionId:   uint64(1),
		ChainID:     ts.spec.Name,
		CuSum:       uint64(10000),
		BlockHeight: int64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()),
		RelayNum:    0,
	}

	// Sign and send the payment requests for block 20 (=epochBeforeChange)
	sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	// Add the relay request to the Relays array (for relayPaymentMessage())
	var Relays []*pairingtypes.RelaySession
	Relays = append(Relays, relayRequest)

	// get payment
	relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
	payAndVerifyBalance(t, ts, relayPaymentMessage, true, ts.clients[0].address, ts.providers[0].address)

	// advance another epoch and unstake the provider
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &pairingtypes.MsgUnstakeProvider{Creator: ts.providers[0].address.String(), ChainID: ts.spec.Index})
	require.Nil(t, err)

	// advance enough epochs to make the provider get its money back, this will panic if there's something wrong in the unstake process
	for i := 0; i < 11; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
}

// TODO: Currently the test passes since second call to verifyRelayPaymentObjects is called with true (see TODO comment right next to it). It should be false. After bug CNS-83 is fixed, change this test
// Test that the payment object is deleted in the end of the memory and can't be used to double spend all while making gov changes
func TestRelayPaymentMemoryTransferAfterEpochChangeWithGovParamChange(t *testing.T) {
	tests := []struct {
		name                string // Test name
		decreaseEpochBlocks bool   // flag to indicate if EpochBlocks is decreased or not
	}{
		{"DecreasedEpochBlocks", true},
		{"IncreasedEpochBlocks", false},
	}

	for _, tt := range tests {

		// setup testnet with mock spec, a staked client and a staked provider
		ts := setupForPaymentTest(t)

		// Advance an epoch because gov params can't change in block 0 (this is a bug. In the time of this writing, it's not fixed)
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		initEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

		// Get epochBlocks and epochsToSave
		epochBlocks, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
		require.Nil(t, err)
		epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), initEpoch)
		require.Nil(t, err)

		// Change the epochBlocks param
		newEpochBlocks := uint64(0)
		if tt.decreaseEpochBlocks {
			newEpochBlocks = epochBlocks / 2
		} else {
			newEpochBlocks = epochBlocks * 2
		}
		err = testkeeper.SimulateParamChange(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.ParamsKeeper, epochstoragetypes.ModuleName, string(epochstoragetypes.KeyEpochBlocks), "\""+strconv.FormatUint(newEpochBlocks, 10)+"\"")
		require.Nil(t, err)

		// Advance an epoch to apply EpochBlocks change
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		epochAfterEpochBlocksChanged := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

		relayRequest := &pairingtypes.RelaySession{
			Provider:    ts.providers[0].address.String(),
			ContentHash: []byte(ts.spec.Apis[0].Name),
			SessionId:   uint64(1),
			ChainID:     ts.spec.Name,
			CuSum:       uint64(10000),
			BlockHeight: int64(epochAfterEpochBlocksChanged),
			RelayNum:    0,
		}

		// Sign the payment request
		sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)

		// Add the relay request to the Relays array (for relayPaymentMessage())
		var Relays []*pairingtypes.RelayRequest
		Relays = append(Relays, relayRequest)

		// get payment
		relayPaymentMessage := pairingtypes.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays}
		payAndVerifyBalance(t, ts, relayPaymentMessage, true, ts.clients[0].address, ts.providers[0].address)

		// Advance epoch and verify the relay payment objects
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		verifyRelayPaymentObjects(t, ts, relayRequest, true)

		// try to get payment again - shouldn't work because of double spend (that's why it's called with false)
		payAndVerifyBalance(t, ts, relayPaymentMessage, false, ts.clients[0].address, ts.providers[0].address)

		// Advance enough epochs so the chain will forget the relay payment object (the chain's memory is limited). Note, we already advanced one epoch since epochAfterEpochBlocksChanged (the relay payment object's creation epoch)
		for i := 0; i < int(epochsToSave)-1; i++ {
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		}

		// Check the relay payment object is deleted
		verifyRelayPaymentObjects(t, ts, relayRequest, true) // TODO: fix bug CNS-83 and turn to false (the real expected value).

		// try to get payment again - shouldn't work (relay payment object should not exist and if it exists, the code shouldn't allow double spending)
		payAndVerifyBalance(t, ts, relayPaymentMessage, false, ts.clients[0].address, ts.providers[0].address)

	}
}

// Helper function to verify the relay payment objects that are saved on-chain after getting payment from a relay request
func verifyRelayPaymentObjects(t *testing.T, ts *testStruct, relayRequest *pairingtypes.RelaySession, objectExists bool) {
	// Get EpochPayment struct from current epoch and perform basic verifications
	epochPayments, found, epochPaymentKey := ts.keepers.Pairing.GetEpochPaymentsFromBlock(sdk.UnwrapSDKContext(ts.ctx), uint64(relayRequest.GetBlockHeight()))
	if objectExists {
		require.Equal(t, true, found)
		require.Equal(t, epochPaymentKey, epochPayments.GetIndex())
	} else {
		require.Equal(t, false, found)
		return
	}

	// Get the providerPaymentStorageKey
	providerPaymentStorageKey := ts.keepers.Pairing.GetProviderPaymentStorageKey(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, uint64(relayRequest.GetBlockHeight()), ts.providers[0].address)

	// Get the providerPaymentStorage struct from epochPayments
	providerPaymentStorageFromEpochPayments := pairingtypes.ProviderPaymentStorage{}
	for _, providerPaymentStorageFromEpochPaymentsElemKey := range epochPayments.GetProviderPaymentStorageKeys() {
		if providerPaymentStorageFromEpochPaymentsElemKey == providerPaymentStorageKey {
			providerPaymentStorageFromEpochPayments, found = ts.keepers.Pairing.GetProviderPaymentStorage(sdk.UnwrapSDKContext(ts.ctx), providerPaymentStorageFromEpochPaymentsElemKey)
			require.True(t, found)
		}
	}
	require.NotEmpty(t, providerPaymentStorageFromEpochPayments.GetIndex())
	require.Equal(t, uint64(relayRequest.GetBlockHeight()), providerPaymentStorageFromEpochPayments.GetEpoch())

	// Get the UniquePaymentStorageClientProvider key
	uniquePaymentStorageClientProviderKey := ts.keepers.Pairing.EncodeUniquePaymentKey(sdk.UnwrapSDKContext(ts.ctx), ts.clients[0].address, ts.providers[0].address, strconv.FormatUint(relayRequest.SessionId, 16), ts.spec.Name)

	// Get one of the uniquePaymentStorageClientProvider struct from providerPaymentStorageFromEpochPayments (note, this is one of the unique.. structs. So usedCU was calculated above with a function that takes into account all the structs)
	uniquePaymentStorageClientProviderFromProviderPaymentStorage := pairingtypes.UniquePaymentStorageClientProvider{}
	for _, uniquePaymentStorageClientProviderFromProviderPaymentStorageElemKey := range providerPaymentStorageFromEpochPayments.GetUniquePaymentStorageClientProviderKeys() {
		if uniquePaymentStorageClientProviderFromProviderPaymentStorageElemKey == uniquePaymentStorageClientProviderKey {
			uniquePaymentStorageClientProviderFromProviderPaymentStorage, found = ts.keepers.Pairing.GetUniquePaymentStorageClientProvider(sdk.UnwrapSDKContext(ts.ctx), uniquePaymentStorageClientProviderFromProviderPaymentStorageElemKey)
			require.True(t, found)
		}
	}
	require.NotEmpty(t, uniquePaymentStorageClientProviderFromProviderPaymentStorage.GetIndex())
	require.Equal(t, uint64(relayRequest.GetBlockHeight()), uniquePaymentStorageClientProviderFromProviderPaymentStorage.GetBlock())
	require.Equal(t, relayRequest.GetCuSum(), uniquePaymentStorageClientProviderFromProviderPaymentStorage.GetUsedCU())

	// when checking CU, the client may be trying to use a relay request with more CU than his MaxCU (determined by StakeThreshold)
	clientStakeEntry, err := ts.keepers.Epochstorage.GetStakeEntryForClientEpoch(sdk.UnwrapSDKContext(ts.ctx), relayRequest.GetChainID(), ts.clients[0].address, uint64(relayRequest.GetBlockHeight()))
	require.Nil(t, err)
	clientMaxCU, err := ts.keepers.Pairing.ClientMaxCUProviderForBlock(sdk.UnwrapSDKContext(ts.ctx), uint64(relayRequest.GetBlockHeight()), clientStakeEntry)
	require.Nil(t, err)
	if clientMaxCU < relayRequest.CuSum {
		require.Equal(t, relayRequest.GetCuSum(), clientMaxCU)
	} else {
		require.Equal(t, relayRequest.GetCuSum(), uniquePaymentStorageClientProviderFromProviderPaymentStorage.GetUsedCU())
	}

	// Get the providerPaymentStorage struct directly
	providerPaymentStorage, found := ts.keepers.Pairing.GetProviderPaymentStorage(sdk.UnwrapSDKContext(ts.ctx), providerPaymentStorageKey)
	require.Equal(t, true, found)
	require.Equal(t, uint64(relayRequest.GetBlockHeight()), providerPaymentStorage.GetEpoch())

	// Get one of the UniquePaymentStorageClientProvider struct directly
	uniquePaymentStorageClientProvider, found := ts.keepers.Pairing.GetUniquePaymentStorageClientProvider(sdk.UnwrapSDKContext(ts.ctx), uniquePaymentStorageClientProviderKey)
	require.Equal(t, true, found)
	require.Equal(t, uint64(relayRequest.GetBlockHeight()), uniquePaymentStorageClientProvider.GetBlock())

	if clientMaxCU < relayRequest.CuSum {
		require.Equal(t, relayRequest.GetCuSum(), clientMaxCU)
	} else {
		require.Equal(t, relayRequest.GetCuSum(), uniquePaymentStorageClientProvider.GetUsedCU())
	}
	require.Equal(t, relayRequest.GetCuSum(), uniquePaymentStorageClientProvider.GetUsedCU())
}
