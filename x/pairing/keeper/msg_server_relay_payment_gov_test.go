package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/slices"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test that if the QosWeight param changes before the provider collected its reward, then
// the provider's payment is according to the last QosWeight value (QosWeight is not fixated)
// Provider reward formula: reward = reward*(QOSScore*QOSWeight + (1-QOSWeight))
func TestRelayPaymentGovQosWeightChange(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// Create badQos: to see the effect of changing QosWeight, the provider need to
	// provide bad service (here, his score is 0%)
	badQoS := &pairingtypes.QualityOfServiceReport{
		Latency:      sdk.ZeroDec(),
		Availability: sdk.ZeroDec(),
		Sync:         sdk.ZeroDec(),
	}

	// Simulate QosWeight to be 0.5 - the default value at the time of this writing
	initQos := sdk.NewDecWithPrec(5, 1)
	initQosBytes, _ := initQos.MarshalJSON()
	initQosStr := string(initQosBytes)

	// change the QoS weight parameter to 0.5
	paramKey := string(pairingtypes.KeyQoSWeight)
	paramVal := initQosStr
	err := ts.TxProposalChangeParam(pairingtypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch (only then the parameter change will be applied) and get current epoch
	ts.AdvanceEpoch()
	epochQosWeightFiftyPercent := ts.EpochStart()

	// Create new QosWeight value (=0.7) for SimulateParamChange() for testing
	newQos := sdk.NewDecWithPrec(7, 1)
	newQosBytes, _ := newQos.MarshalJSON()
	newQosStr := string(newQosBytes)

	// change the QoS weight parameter to 0.7
	paramKey = string(pairingtypes.KeyQoSWeight)
	paramVal = newQosStr
	err = ts.TxProposalChangeParam(pairingtypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch (for the parameter change to take effect)
	ts.AdvanceEpoch()
	epochQosWeightSeventyPercent := ts.EpochStart()

	tests := []struct {
		name      string
		epoch     uint64
		qosWeight sdk.Dec
		valid     bool
	}{
		// payment collected for an epoch with QosWeight = 0.7
		{"PaymentSeventyPercentQosEpoch", epochQosWeightSeventyPercent, sdk.NewDecWithPrec(7, 1), true},
		// payment collected for an epoch with QosWeight = 0.5, provider judged by QosWeight = 0.7
		{"PaymentFiftyPercentQosEpoch", epochQosWeightFiftyPercent, sdk.NewDecWithPrec(5, 1), false},
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request dated to the test's epoch. Change session ID each iteration
			// to avoid double spending error (provider claims twice for same transaction)
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, tt.epoch, 0)
			relaySession.QosReport = badQoS
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, true)
		})
	}
}

// Test that if the EpochBlocks param decreases the provider can claim reward after the new
// EpochBlocks*EpochsToSave, and not the original EpochBlocks
func TestRelayPaymentGovEpochBlocksDecrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	epochBlocks := ts.EpochBlocks()
	epochsToSave := ts.EpochsToSave()

	epochBefore := ts.EpochStart()

	// Decrease the epochBlocks param
	smallerEpochBlocks := epochBlocks / 2
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(smallerEpochBlocks, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.AdvanceEpochs(2)
	epochAfter := ts.EpochStart()

	// the number of blocks to advance must be smaller than memory limit with the old EpochBlocks
	require.Less(t, (epochsToSave+1)*smallerEpochBlocks+epochAfter, epochBefore+(epochsToSave*epochBlocks))

	// Advance EpochsToSave+1 epochs so provider with the old EpochBlocks can get paid, but shouldn't
	ts.AdvanceEpochs(epochsToSave + 1)

	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochBlocksChanges", epochBefore, false},
		{"PaymentAfterEpochBlocksChanges", epochAfter, false},
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request dated to the test's epoch. Change session ID each iteration
			// to avoid double spending error (provider claims twice for same transaction)
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, tt.epoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

// TODO: Currently the test passes since PaymentBeforeEpochBlocksChangesToFifty's value is false.
// It should be true. After bug CNS-83 is fixed, change this test
// Test that if the EpochBlocks param increases make sure the provider can claim reward after the
// new EpochBlocks*EpochsToSave, and not the original EpochBlocks
func TestRelayPaymentGovEpochBlocksIncrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	epochBlocks := ts.EpochBlocks()
	epochsToSave := ts.EpochsToSave()

	ts.AdvanceEpoch()
	epochBeforeChange := ts.EpochStart()

	// Increase the epochBlocks param
	biggerEpochBlocks := epochBlocks * 2
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(biggerEpochBlocks, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.AdvanceEpochs(2)
	epochAfterChange := ts.EpochStart()

	// Calculate the memory limit of a provider with the new EpochBlocks (in epochs)
	memoryLimit := (epochBlocks * epochsToSave) + epochBeforeChange
	memoryLimitInEpochsUsingNewEpochBlocks := (memoryLimit - epochAfterChange) / biggerEpochBlocks

	// Make sure that the number of epochs that we'll advance is smaller than EpochsToSave
	// (advance memoryLimitInEpochsUsingNewEpochBlocks+1; we already advanced an epoch before)
	require.Less(t, memoryLimitInEpochsUsingNewEpochBlocks+2, epochsToSave)

	// Advance enough epochs so a provider with the old EpochBlocks can't be paid, which
	// shouldn't happen (from old EpochBlocks perspective, too many blocks passed. But, the
	// number of epochs that passed is smaller than EpochsToSave)
	ts.AdvanceEpochs(memoryLimitInEpochsUsingNewEpochBlocks + 1)

	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochBlocksChange", epochBeforeChange, false},
		{"PaymentAfterEpochBlocksChange", epochAfterChange, true},
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request dated to the test's epoch. Change session ID each iteration
			// to avoid double spending error (provider claims twice for same transaction)
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, tt.epoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

// Test that if the EpochsToSave param decreases make sure the provider can claim reward after the
// new EpochBlocks*EpochsToSave, and not the original EpochBlocks
func TestRelayPaymentGovEpochToSaveDecrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	epochBlocks := ts.EpochBlocks()
	epochsToSave := ts.EpochsToSave()

	epochBefore := ts.EpochStart()

	// Decrease the epochBlocks param
	smallerEpochsToSave := epochsToSave / 2
	paramKey := string(epochstoragetypes.KeyEpochsToSave)
	paramVal := "\"" + strconv.FormatUint(smallerEpochsToSave, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch so the change applies, and another one
	ts.AdvanceEpochs(2)
	epochAfter := ts.EpochStart()

	// Make sure that the number of epochs that we'll advance from epochBeforeChange is smaller
	// than EpochsToSave (advance smallerEpochsToSave; already advanced 2 epochs before)
	require.Less(t, smallerEpochsToSave+2, epochsToSave)

	// Advance epochs so that a provider with old EpochsToSave from epochBeforeChange can get paid
	// (but it shouldn't, since we advanced smallerEpochsToSave+2 epochs). Also a provider
	// from epochAfterChange with the new EpochsToSave should get paid since we're one block
	// before smalleEpochsToSave+1 (which is the end of the memory for the new EpochsToSave)
	ts.AdvanceEpochs(smallerEpochsToSave)
	ts.AdvanceBlocks(epochBlocks - 1)

	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochsToSaveChanges", epochBefore, false},
		{"PaymentAfterEpochsToSaveChangesBlockInMemory", epochAfter, true},         // the chain is one block before the memory ends for the provider from epochAfterChange
		{"PaymentAfterEpochsToSaveChangesBlockOutsideOfMemory", epochAfter, false}, // the chain advances inside the loop test when it reaches this test. Here we pass the memory of the provider from epochAfterChange
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// advance to one block to reach the start of the smalerEpochsToSave+1 epoch ->
			// the provider from epochAfterChange shouldn't be able to get its payments
			if ti == 2 {
				ts.AdvanceBlock()
			}

			// Create relay request dated to the test's epoch. Change session ID each iteration
			// to avoid double spending error (provider claims twice for same transaction)
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, tt.epoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

// TODO: Currently the test passes since PaymentBeforeEpochsToSaveChangesToTwenty's value is false.
// It should be true. After bug CNS-83 is fixed, change this test.
// Test that if the EpochToSave param increases make sure the provider can claim reward after the
// new EpochBlocks*EpochsToSave, and not the original EpochBlocks
func TestRelayPaymentGovEpochToSaveIncrease(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// make sure EpochBlocks default value is 20, and EpochsToSave is 10
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)
	paramKey = string(epochstoragetypes.KeyEpochsToSave)
	paramVal = "\"" + strconv.FormatUint(10, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change.
	ts.AdvanceEpoch()
	epochBefore := ts.EpochStart() // blockHeight = 20

	// change the EpochsToSave parameter to 20
	paramKey = string(epochstoragetypes.KeyEpochsToSave)
	paramVal = "\"" + strconv.FormatUint(20, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch so the change applies
	ts.AdvanceEpoch() // blockHeight = 40
	epochAfter := ts.EpochStart()

	// Advance to reach blockHeight of 260, so provider request from epochBeforeChangeToTwenty
	// shouldn't get payment, and from epochAfterChangeToTwenty should
	ts.AdvanceEpochs(11)

	tests := []struct {
		name  string
		epoch uint64
		valid bool
	}{
		{"PaymentBeforeEpochsToSaveChangesToTwenty", epochBefore, false}, // first block of current epoch
		{"PaymentAfterEpochsToSaveChangesToTwenty", epochAfter, true},    // first block of previous epoch
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create relay request dated to the test's epoch. Change session ID each iteration
			// to avoid double spending error (provider claims twice for same transaction)
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, tt.epoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

func TestRelayPaymentGovEpochBlocksMultipleChanges(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// make sure EpochBlocks default value is 20, and EpochsToSave is 10
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)
	paramKey = string(epochstoragetypes.KeyEpochsToSave)
	paramVal = "\"" + strconv.FormatUint(10, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change.
	ts.AdvanceEpoch() // blockHeight = 20

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

	tests := []struct {
		name         string // Test name
		paymentEpoch uint64 // The epoch inside the relay request (delta from test start)
		valid        bool   // Is the test supposed to succeed?
	}{
		{"Test #1", 0, true},
		{"Test #2", 6, true},
		{"Test #3", 901, true},
		{"Test #4", 2731, true},
		{"Test #5", 3274, true},
		{"Test #6", 4287, true},
		{"Test #7", 4550, true},
		{"Test #8", 6177, true},
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	startBlock := ts.BlockHeight()

	for ti, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramKey := string(epochstoragetypes.KeyEpochBlocks)
			paramVal := "\"" + strconv.FormatUint(epochTests[ti].epochBlocksNewValues, 10) + "\""
			err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
			require.Nil(t, err)

			ts.AdvanceEpochs(epochTests[ti].epochNum)
			ts.AdvanceBlocks(epochTests[ti].blockNum)

			paymentEpoch := startBlock + tt.paymentEpoch
			relaySession := ts.newRelaySession(providerAddr, uint64(ti), cuSum, paymentEpoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := pairingtypes.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

// this test checks what happens if a single provider stake, get payment, and then unstake and gets its money.
func TestStakePaymentUnstake(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	// ensure that EpochBlocks default value is 20, EpochsToSave is 10,  unstakeHoldBlocks is 210
	paramKey := string(epochstoragetypes.KeyEpochBlocks)
	paramVal := "\"" + strconv.FormatUint(20, 10) + "\""
	err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)
	paramKey = string(epochstoragetypes.KeyEpochsToSave)
	paramVal = "\"" + strconv.FormatUint(10, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)
	paramKey = string(epochstoragetypes.KeyUnstakeHoldBlocks)
	paramVal = "\"" + strconv.FormatUint(210, 10) + "\""
	err = ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
	require.Nil(t, err)

	// Advance an epoch to apply EpochBlocks change
	ts.AdvanceEpoch() // blockHeight = 20

	relaySession := ts.newRelaySession(providerAddr, 1, 10000, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	payment := pairingtypes.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  slices.Slice(relaySession),
	}

	ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, true)

	// advance another epoch and unstake the provider
	ts.AdvanceEpoch()

	_, err = ts.TxPairingUnstakeProvider(providerAddr, ts.spec.Index)
	require.Nil(t, err)

	// advance enough epochs to make the provider get its money back:
	// this will panic if there's something wrong in the unstake process
	ts.AdvanceEpochs(11)
}

// TODO: Currently the test passes since second call to verifyRelayPaymentObjects is called
// with true (see TODO comment right next to it). It should be false. After bug CNS-83 is fixed,
// change this test.
// Test that the payment object is deleted in the end of the memory and can't be used to double
// spend all while making gov changes
func TestRelayPaymentMemoryTransferAfterEpochChangeWithGovParamChange(t *testing.T) {
	tests := []struct {
		name                string // Test name
		decreaseEpochBlocks bool   // flag to indicate if EpochBlocks is decreased or not
	}{
		{"DecreasedEpochBlocks", true},
		{"IncreasedEpochBlocks", false},
	}

	for _, tt := range tests {
		ts := newTester(t)
		ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

		client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
		providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

		epochBlocks := ts.EpochBlocks()
		epochsToSave := ts.EpochsToSave()

		// Change the epochBlocks param
		var newEpochBlocks uint64
		if tt.decreaseEpochBlocks {
			newEpochBlocks = epochBlocks / 2
		} else {
			newEpochBlocks = epochBlocks * 2
		}

		paramKey := string(epochstoragetypes.KeyEpochBlocks)
		paramVal := "\"" + strconv.FormatUint(newEpochBlocks, 10) + "\""
		err := ts.TxProposalChangeParam(epochstoragetypes.ModuleName, paramKey, paramVal)
		require.Nil(t, err)

		// Advance an epoch to apply EpochBlocks change
		ts.AdvanceEpoch()

		relaySession := ts.newRelaySession(providerAddr, 1, 10000, ts.EpochStart(), 0)
		signRelaySession(relaySession, client1Acct.SK)

		payment := pairingtypes.MsgRelayPayment{
			Creator: providerAddr,
			Relays:  slices.Slice(relaySession),
		}

		ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, true)

		// Advance epoch and verify the relay payment objects
		ts.AdvanceEpoch()
		ts.verifyRelayPayment(relaySession, true)

		// try to get payment again - should fail work because of double spend
		ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, false)

		// Advance enough epochs so the chain will forget the relay payment object. Note that
		// we already advanced one epoch since epochAfterEpochBlocksChanged.
		ts.AdvanceEpochs(epochsToSave - 1)
		// Check the relay payment object is deleted
		ts.verifyRelayPayment(relaySession, true) // TODO: fix bug CNS-83 and turn to false

		// try to get payment again - should fail (relay payment object should not exist and if
		// it exists, the code shouldn't allow double spending)
		ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, false)
	}
}
