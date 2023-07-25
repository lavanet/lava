package keeper_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

// Test that a provider payment is valid if claimed within the chain's memory,
// and verify the relay payment object (RPO)
func TestRelayPaymentMemoryTransferAfterEpochChange(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	firstEpoch := ts.EpochStart()
	epochsToSave := ts.EpochsToSave()
	epochBlocks := ts.EpochBlocks()

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name            string
		epochsToAdvance uint64 // number of epochs to advance in the test
		blocksToAdvance uint64 // number of blocks to advance in the test
		valid           bool
	}{
		{"PaymentFirstEpoch", 0, 0, true},                               // first block of current epoch
		{"PaymentEndOfMemoryEpochFirstBlock", epochsToSave, 0, true},    // first block of end of memory epoch
		{"PaymentEndOfMemoryEpochLastBlock", 0, epochBlocks - 1, true},  // last block of end of memory epoch
		{"PaymentOutOfMemoryEpochFirstBlock", 0, 1, false},              // first block of out of memory epoch (end of memory+1)
		{"PaymentOutOfMemoryEpochLastBlock", 0, epochBlocks - 1, false}, // last block of out of memory epoch (end of memory+1)
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	sessionId := uint64(0)

	for _, tt := range tests {
		sessionId += 1
		t.Run(tt.name, func(t *testing.T) {
			ts.AdvanceEpochs(tt.epochsToAdvance)
			ts.AdvanceBlocks(tt.blocksToAdvance)

			// Create relay request for the first epoch. Change session ID each iteration to
			// avoid double spending error (provider asks reward for the same transaction twice)
			relaySession := ts.newRelaySession(providerAddr, sessionId, cuSum, firstEpoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			// Request payment (helper validates balances and verifies error through valid)
			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}
			ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Addr, true, tt.valid)

			// Check the RPO exists (shouldn't exist after epochsToSave+1 passes)
			ts.verifyRelayPayment(relaySession, tt.valid)
		})
	}
}

func TestRelayPaymentBlockHeight(t *testing.T) {
	tests := []struct {
		name      string
		blockTime int64
		valid     bool
	}{
		{"HappyFlow", 0, true},
		{"OldBlock", -1, false},
		{"NewBlock", +1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

			client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
			providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
			block := int64(ts.BlockHeight()) + tt.blockTime

			relaySession := ts.newRelaySession(providerAddr, 0, cuSum, uint64(block), 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

func TestRelayPaymentOverUse(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	maxcu := ts.plan.PlanPolicy.EpochCuLimit

	relaySession := ts.newRelaySession(providerAddr, 0, maxcu*2, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	// TODO: currently over-use causes error and doesnt reach balance zero; uncomment when fixed.
	// balance := ts.GetBalance(providerAddr)

	_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.Error(t, err)

	// TODO: currently over-use causes error and doesnt reach balance zero; uncomment when fixed.
	// balance -= ts.GetBalance(providerAddr)
	// require.Zero(t, balance)
}

// one epoch is not enough for the unstaking to happen need atleast two epochs in the past
func TestRelayPaymentNotUnstakingProviderForUnresponsivenessIfNoEpochInformation(t *testing.T) {
	ts := newTester(t)

	clientsCount := 4
	providersCount := 2

	ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
	clients := ts.Accounts(common.CONSUMER)

	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	provider2Acct, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	unresponsiveProvidersData, err := json.Marshal([]string{provider2Addr})
	require.Nil(t, err)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1Addr, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData // create the complaint
		signRelaySession(relaySession, clients[clientIndex].SK)
		relays = append(relays, relaySession)
	}

	_, err = ts.TxPairingRelayPayment(provider1Addr, relays...)
	require.Nil(t, err)

	// test that the provider was not unstaked
	_, unStakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider2Acct.Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider2Acct.Addr)
	require.True(t, stakeStorageFound)
}

func TestRelayPaymentUnstakingProviderForUnresponsivenessWithBadDataInput(t *testing.T) {
	ts := newTester(t)

	clientsCount := 4
	providersCount := 2

	ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
	clients := ts.Accounts(common.CONSUMER)

	provider1Acct, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	// move to epoch 3 so we can check enough epochs in the past
	ts.AdvanceEpochs(2)

	unresponsiveProvidersData := make([]([]byte), 4)

	// test multiple bad data types
	inputData := []interface{}{
		[]int{1, 2, 3, 4, 5},
		[]string{"bad", "data", "cosmosBadAddress"},
		"cosmosBadAddress",
		[]byte("cosmosBadAddress"),
	}
	// badData2, err := json.Marshal([]string{"bad", "data", "cosmosBadAddress"}) // test bad data
	for i := 0; i < clientsCount; i++ {
		badData, err := json.Marshal(inputData[i])
		require.Nil(t, err)
		unresponsiveProvidersData[i] = badData
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	var totalCu uint64
	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1Addr, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData[clientIndex]
		signRelaySession(relaySession, clients[clientIndex].SK)

		totalCu += relaySession.CuSum
		relays = append(relays, relaySession)
	}

	balanceProviderBeforePayment := ts.GetBalance(provider1Acct.Addr)
	_, err := ts.TxPairingRelayPayment(provider1Addr, relays...)
	require.Nil(t, err)
	balanceProviderAfterPayment := ts.GetBalance(provider1Acct.Addr)
	reward := ts.Keepers.Pairing.MintCoinsPerCU(ts.Ctx).MulInt64(int64(totalCu)).TruncateInt64()
	// reward + before == after
	require.Equal(t, balanceProviderAfterPayment, reward+balanceProviderBeforePayment)
}

// test protection from unstaking if the amount of previous serices*2 is greater than complaints
func TestRelayPaymentNotUnstakingProviderForUnresponsivenessBecauseOfServices(t *testing.T) {
	ts := newTester(t)

	clientsCount := 4
	providersCount := 2

	ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
	clients := ts.Accounts(common.CONSUMER)

	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	provider2Acct, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(provider2Addr, 0, cuSum, ts.BlockHeight(), 0)
		signRelaySession(relaySession, clients[i].SK)

		_, err := ts.TxPairingRelayPayment(provider2Addr, relaySession)
		require.Nil(t, err)

		ts.AdvanceEpoch() // after payment move one epoch
	}

	unresponsiveProvidersData, err := json.Marshal([]string{provider2Addr})
	require.Nil(t, err)

	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1Addr, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		signRelaySession(relaySession, clients[clientIndex].SK)
		relays = append(relays, relaySession)
	}

	_, err = ts.TxPairingRelayPayment(provider1Addr, relays...)
	require.Nil(t, err)

	// test that the provider was not unstaked.
	_, unStakeStoragefound, _ := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider2Acct.Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider2Acct.Addr)
	require.True(t, stakeStorageFound)
}

func TestRelayPaymentDoubleSpending(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	payment := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  slices.Slice(relaySession, relaySession),
	}

	ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, false)
}

func TestRelayPaymentDataModification(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       int64
	}{
		{"ModifiedProvider", client1Addr, ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10, 1},
		{"ModifiedCU", providerAddr, ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 9, 1},
		{"ModifiedID", providerAddr, ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession.Provider = tt.provider
			relaySession.CuSum = tt.cu
			relaySession.SessionId = uint64(tt.id)

			// do not re-sign relay session request - to trigger error

			_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentDelayedDoubleSpending(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.Nil(t, err)

	tests := []struct {
		name    string
		advance uint64
	}{
		{"Epoch", 1},
		{"Memory-Epoch", ts.EpochsToSave() - 1},
		{"Memory", 1},       // epochsToSave
		{"Memory+Epoch", 1}, // epochsToSave + 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.AdvanceEpochs(tt.advance)
			_, err := ts.TxPairingRelayPayment(providerAddr, relaySession)
			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentOldEpochs(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	epochsToSave := ts.EpochsToSave()
	epochBlocks := ts.EpochBlocks()

	ts.AdvanceEpochs(epochsToSave + 1)

	tests := []struct {
		name  string
		sid   uint64
		epoch uint64
		valid bool
	}{
		{"Epoch", 1, 1, true},                        // current -1*epoch
		{"Memory-Epoch", 2, epochsToSave - 1, true},  // current -epoch to save + 1
		{"Memory", 3, epochsToSave, true},            // current - epochToSave
		{"Memory+Epoch", 4, epochsToSave + 1, false}, // current - epochToSave - 1
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	block := ts.BlockHeight()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession := ts.newRelaySession(providerAddr, tt.sid, cuSum, block-epochBlocks*tt.epoch, 0)
			signRelaySession(relaySession, client1Acct.SK)

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

func TestRelayPaymentQoS(t *testing.T) {
	tests := []struct {
		name         string
		availability sdk.Dec
		latency      sdk.Dec
		sync         sdk.Dec
		valid        bool
	}{
		{"InvalidLatency", sdk.NewDecWithPrec(2, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), false},
		{"InvalidAvailability", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(2, 0), sdk.NewDecWithPrec(1, 0), false},
		{"Invalidsync", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(2, 0), false},
		{"PerfectScore", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), true},
		{"MediumScore", sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), true},
		{"ZeroScore", sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t)
			ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

			client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
			providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
			qos := &types.QualityOfServiceReport{
				Latency:      tt.latency,
				Availability: tt.availability,
				Sync:         tt.sync,
			}

			relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
			relaySession.QosReport = qos
			signRelaySession(relaySession, client1Acct.SK)

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, tt.valid)
		})
	}
}

func TestEpochPaymentDeletion(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	signRelaySession(relaySession, client1Acct.SK)

	payment := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  slices.Slice(relaySession),
	}

	ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Addr, true, true)

	ts.AdvanceEpochs(ts.EpochsToSave() + 1)

	res, err := ts.QueryPairingListEpochPayments()
	require.Nil(t, err)
	require.Equal(t, 0, len(res.EpochPayments))
}

// Test that after the consumer uses some CU it's updated in its project and subscription
func TestCuUsageInProjectsAndSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 2)    // 0 sub, 0 adm, 2 dev
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	dev1Acct, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	projectData := projecttypes.ProjectData{
		Name:    "proj1",
		Enabled: true,
		ProjectKeys: []projecttypes.ProjectKey{
			projecttypes.ProjectDeveloperKey(dev1Addr),
		},
		Policy: &planstypes.Policy{
			GeolocationProfile: uint64(1),
			MaxProvidersToPair: 3,
			TotalCuLimit:       1000,
			EpochCuLimit:       100,
		},
	}
	err := ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Nil(t, err)

	projectData.Name = "proj2"
	projectData.ProjectKeys = []projecttypes.ProjectKey{
		projecttypes.ProjectDeveloperKey(dev2Addr),
	}
	err = ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	qos := types.QualityOfServiceReport{
		Latency:      sdk.OneDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}

	relaySession := ts.newRelaySession(providerAddr, 0, 1, ts.BlockHeight(), 0)
	relaySession.QosReport = &qos
	signRelaySession(relaySession, dev1Acct.SK)

	// Request payment (helper validates the balances and verifies expected errors through valid)
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  slices.Slice(relaySession),
	}

	ts.payAndVerifyBalance(relayPaymentMessage, dev1Acct.Addr, providerAcct.Addr, true, true)

	sub, err := ts.QuerySubscriptionCurrent(client1Addr)
	require.Nil(t, err)
	require.True(t, sub.Sub != nil)

	proj1, err := ts.QueryProjectDeveloper(dev1Addr)
	require.Nil(t, err)
	require.Equal(t, sub.Sub.MonthCuTotal-sub.Sub.MonthCuLeft, proj1.Project.UsedCu)

	proj2, err := ts.QueryProjectDeveloper(dev2Addr)
	require.Nil(t, err)
	require.NotEqual(t, sub.Sub.MonthCuTotal-sub.Sub.MonthCuLeft, proj2.Project.UsedCu)
}

func TestBadgeValidation(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	epochStart := ts.EpochStart()
	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits

	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, epochStart, 0)
	relaySession.QosReport = qos
	signRelaySession(relaySession, client1Acct.SK)

	// make the context's lava chain ID a non-empty string (as usually defined in most tests)
	lavaChainID := "lavanet"
	ts.SetChainID(lavaChainID)

	// times 2 to have enough CU for the two tests that succeed
	badgeCuAllocation := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 2

	tests := []struct {
		name         string
		badgeAddress sdk.AccAddress // badge user (in happy flow)
		relaySigner  common.Account // badge user (in happy flow)
		badgeSigner  common.Account // badge granter, i.e. project developer (in happy flow)
		epoch        uint64
		lavaChainID  string
		valid        bool
	}{
		{"happy flow", badgeAcct.Addr, badgeAcct, client1Acct, epochStart, lavaChainID, true},
		{"badge address != badge user", client1Acct.Addr, badgeAcct, client1Acct, epochStart, lavaChainID, false},
		{"relay signer != badge user", badgeAcct.Addr, client1Acct, client1Acct, epochStart, lavaChainID, false},
		{"badge signer != project developer", badgeAcct.Addr, badgeAcct, badgeAcct, epochStart, lavaChainID, false},
		{"badge epoch != relay epoch", badgeAcct.Addr, badgeAcct, client1Acct, epochStart - 1, lavaChainID, false},
		{"badge lava chain id != relay lava chain id", badgeAcct.Addr, badgeAcct, client1Acct, epochStart, "dummy-lavanet", false},
		{"badge epoch != relay epoch (epoch passed)", badgeAcct.Addr, badgeAcct, client1Acct, epochStart, lavaChainID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// this test ensures that we only compare the relay's and badge's epoch fields
			// (unrelated to the advancement of blocks). So this test should have no errors
			if tt.name == "badge epoch != relay epoch (epoch passed)" {
				ts.AdvanceEpoch()
				ts.AdvanceBlock()

				// remove past payments to avoid double spending (first  error payment succeeded)
				ts.Keepers.Pairing.RemoveAllEpochPaymentsForBlock(ts.Ctx, tt.epoch)
			}

			badge := types.CreateBadge(badgeCuAllocation, tt.epoch, tt.badgeAddress, tt.lavaChainID, []byte{})
			signBadge(badge, tt.badgeSigner.SK)

			relaySession.Badge = badge
			relaySession.LavaChainId = tt.lavaChainID
			signRelaySession(relaySession, tt.relaySigner.SK)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			validConsumer := true
			if !tt.badgeSigner.Addr.Equals(client1Acct.Addr) {
				validConsumer = false
			}

			ts.payAndVerifyBalance(relayPaymentMessage, tt.badgeSigner.Addr, providerAcct.Addr, validConsumer, tt.valid)
		})
	}
}

func TestAddressEpochBadgeMap(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	badge := types.CreateBadge(10, epochStart, badgeAcct.Addr, "", []byte{})
	signBadge(badge, client1Acct.SK)

	// create 5 identical relays. Assign the badge only to the first one
	var relays []*types.RelaySession
	for i := 0; i < 5; i++ {
		qos := &types.QualityOfServiceReport{
			Latency:      sdk.NewDecWithPrec(1, 0),
			Availability: sdk.NewDecWithPrec(1, 0),
			Sync:         sdk.NewDecWithPrec(1, 0),
		}

		// vary session ID to avoid double spending
		relaySession := ts.newRelaySession(providerAddr, uint64(i), 1, ts.BlockHeight(), 0)
		relaySession.QosReport = qos
		if i == 0 {
			relaySession.Badge = badge
		}
		signRelaySession(relaySession, badgeAcct.SK)

		relays = append(relays, relaySession)
	}

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  relays,
	}

	ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Addr, true, true)
}

// Test:
// 1. basic badge CU enforcement works (can't ask more CU than badge's cuAllocation)
// 2. claiming payment with different session IDs in different relay payment messages works
func TestBadgeCuAllocationEnforcement(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	plan, err := ts.GetPlanFromSubscription(client1Addr)
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	signBadge(badge, client1Acct.SK)

	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	tests := []struct {
		name  string
		cuSum uint64
		valid bool
	}{
		{"happy flow", badgeCuAllocation / 2, true},
		{"one cu over the limit", badgeCuAllocation/2 + 1, false},
		{"many cu over the limit", badgeCuAllocation, false},
		{"one cu below the limit", badgeCuAllocation/2 - 1, true},
		{"finish cu allocation", 1, true},
	}

	var usedCuSoFar uint64

	for it, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession := ts.newRelaySession(providerAddr, uint64(it), tt.cuSum, ts.BlockHeight(), 0)
			relaySession.QosReport = qos
			relaySession.Badge = badge
			signRelaySession(relaySession, badgeAcct.SK)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  slices.Slice(relaySession),
			}

			ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Addr, true, tt.valid)

			if tt.valid {
				usedCuSoFar += tt.cuSum
			}

			badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, providerAddr)
			badgeUsedCuMapEntry, found := ts.Keepers.Pairing.GetBadgeUsedCu(ts.Ctx, badgeUsedCuMapKey)
			require.True(t, found)
			require.Equal(t, usedCuSoFar, badgeUsedCuMapEntry.UsedCu)
		})
	}
}

// Test:
// 1. backward payment claims work (can claim up to epochBlocks*epochsToSave)
// 2. badgeUsedCuEntry is valid from the badge's creation block plus epochBlocks*epochsToSave. Then, it should be deleted
// 3. claiming payment for relays with different session IDs in the same relay payment message works
func TestBadgeUsedCuMapTimeout(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	plan, err := ts.GetPlanFromSubscription(client1Addr)
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	signBadge(badge, client1Acct.SK)

	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	relayNum := 5
	cuSum := badgeCuAllocation / (uint64(relayNum) * 2)

	badgeUsedCuExpiry := ts.Keepers.Pairing.BadgeUsedCuExpiry(ts.Ctx, *badge)

	tests := []struct {
		name            string
		blocksToAdvance uint64
		valid           bool
	}{
		{"valid backwards claim payment", badgeUsedCuExpiry - epochStart - 1, true},
		{"invalid backwards claim payment", 2, false}, // cuSum is valid (not passing badgeCuAllocation), but we've passed the backward payment period
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relays := []*types.RelaySession{}
			for i := 0; i < relayNum; i++ {
				relaySession := ts.newRelaySession(providerAddr, uint64(i), cuSum, epochStart, 0)
				relaySession.QosReport = qos
				signRelaySession(relaySession, badgeAcct.SK)

				if i == 0 {
					relaySession.Badge = badge
				}

				relays = append(relays, relaySession)
			}

			ts.AdvanceBlocks(tt.blocksToAdvance)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  relays,
			}
			ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Addr, true, tt.valid)

			// verify that the badgeUsedCu entry was deleted after it expired (and has the
			// right value of used cu before expiring)
			badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, providerAddr)
			badgeUsedCuMapEntry, found := ts.Keepers.Pairing.GetBadgeUsedCu(ts.Ctx, badgeUsedCuMapKey)
			if ts.BlockHeight() > badgeUsedCuExpiry {
				require.False(t, found)
			} else {
				require.True(t, found)
				require.Equal(t, cuSum*uint64(relayNum), badgeUsedCuMapEntry.UsedCu)
			}
		})
	}
}

// Test that in the same epoch a badge it is possible to exceed the total CU allocation
// badge multiple times when talking to different providers
func TestBadgeDifferentProvidersCuAllocation(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 2 provider, 1 client, default providers-to-pair
	providers := ts.Accounts(common.PROVIDER)

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	plan, err := ts.GetPlanFromSubscription(client1Addr)
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit / 2

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	signBadge(badge, client1Acct.SK)

	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	cuSum := badgeCuAllocation

	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(providers[i].Addr.String(), 0, cuSum, epochStart, 0)
		relaySession.QosReport = qos
		signRelaySession(relaySession, badgeAcct.SK)
		relaySession.Badge = badge

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: providers[i].Addr.String(),
			Relays:  slices.Slice(relaySession),
		}
		ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providers[i].Addr, true, true)

		badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, providers[i].Addr.String())
		badgeUsedCuMapEntry, found := ts.Keepers.Pairing.GetBadgeUsedCu(ts.Ctx, badgeUsedCuMapKey)
		require.True(t, found)
		require.Equal(t, cuSum, badgeUsedCuMapEntry.UsedCu)
	}
}
