package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/testutil/common"
	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	"github.com/stretchr/testify/require"
)

// Test that a provider payment is valid if claimed within the chain's memory,
// and verify the relay payment object (RPO)
func TestRelayPaymentMemoryTransferAfterEpochChange(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

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
			// Sign and send the payment requests
			sig, err := sigs.Sign(client1Acct.SK, *relaySession)
			relaySession.Sig = sig
			require.NoError(t, err)

			// Request payment (helper validates balances and verifies error through valid)
			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			ts.relayPaymentWithoutPay(relayPaymentMessage, tt.valid)
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
			_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
			block := int64(ts.BlockHeight()) + tt.blockTime

			relaySession := ts.newRelaySession(providerAddr, 0, cuSum, uint64(block), 0)
			relaySession.Epoch = block
			sig, err := sigs.Sign(client1Acct.SK, *relaySession)
			relaySession.Sig = sig
			require.NoError(t, err)

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}

			ts.relayPaymentWithoutPay(payment, tt.valid)
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
	sig, err := sigs.Sign(client1Acct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	// TODO: currently over-use causes error and doesnt reach balance zero; uncomment when fixed.
	// balance := ts.GetBalance(providerAddr)

	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NoError(t, err)

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

	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)

	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider2}}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData // create the complaint
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relays = append(relays, relaySession)
	}

	_, err := ts.TxPairingRelayPayment(provider1, relays...)
	require.NoError(t, err)

	// test that the provider was not unstaked
	_, unStakeStoragefound := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider2)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider2)
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
	unresponsiveProvidersData := make([](*types.ReportedProvider), 4)

	// test multiple bad data types
	inputData := []string{
		"123",
		"baddatacosmosBadAddress",
		"lava@cosmosBadAddress",
		string([]byte{1, 2, 3}),
	}
	// badData2, err := json.Marshal([]string{"bad", "data", "cosmosBadAddress"}) // test bad data
	for i := 0; i < clientsCount; i++ {
		unresponsiveProvidersData[i] = &types.ReportedProvider{Address: inputData[i]}
	}

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	var reward uint64
	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1Addr, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = []*types.ReportedProvider{unresponsiveProvidersData[clientIndex]}
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		reward += ts.plan.Price.Amount.Uint64()
		relays = append(relays, relaySession)
	}

	balanceProviderBeforePayment := ts.GetBalance(provider1Acct.Vault.Addr)
	_, err := ts.TxPairingRelayPayment(provider1Addr, relays...)
	require.NoError(t, err)
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// reward + before == after
	_, err = ts.TxDualstakingClaimRewards(provider1Acct.GetVaultAddr(), provider1Acct.Addr.String())
	require.Nil(ts.T, err)

	balanceProviderAfterPayment := ts.GetBalance(provider1Acct.Vault.Addr)
	require.Equal(t, balanceProviderAfterPayment, int64(reward)+balanceProviderBeforePayment)
}

// test protection from unstaking if the amount of previous serices*2 is greater than complaints
func TestRelayPaymentNotUnstakingProviderForUnresponsivenessBecauseOfServices(t *testing.T) {
	ts := newTester(t)

	clientsCount := 4
	providersCount := 2

	ts.setupForPayments(providersCount, clientsCount, providersCount) // set providers-to-pair
	clients := ts.Accounts(common.CONSUMER)

	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(provider2, 0, cuSum, ts.BlockHeight(), 0)
		sig, err := sigs.Sign(clients[i].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		_, err = ts.TxPairingRelayPayment(provider2, relaySession)
		require.NoError(t, err)

		ts.AdvanceEpoch() // after payment move one epoch
	}

	unresponsiveProvidersData := []*types.ReportedProvider{{Address: provider2}}

	var relays []*types.RelaySession
	for clientIndex := 0; clientIndex < clientsCount; clientIndex++ {
		relaySession := ts.newRelaySession(provider1, 0, cuSum, ts.BlockHeight(), 0)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData
		sig, err := sigs.Sign(clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)
		relays = append(relays, relaySession)
	}

	_, err := ts.TxPairingRelayPayment(provider1, relays...)
	require.NoError(t, err)

	// test that the provider was not unstaked.
	_, unStakeStoragefound := ts.Keepers.Epochstorage.UnstakeEntryByAddress(ts.Ctx, provider2)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryByAddressCurrent(ts.Ctx, ts.spec.Name, provider2)
	require.True(t, stakeStorageFound)
}

func TestRelayPaymentDoubleSpending(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	sig, err := sigs.Sign(client1Acct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	payment := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  lavaslices.Slice(relaySession, relaySession),
	}

	ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Vault.Addr, true, false, 100)
}

func TestRelayPaymentDataModification(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       uint64
	}{
		{"ModifiedProvider", client1Addr, relayCuSum * 10, 1},
		{"ModifiedCU", providerAddr, relayCuSum * 9, 1},
		{"ModifiedID", providerAddr, relayCuSum * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// do not re-sign relay session request - to trigger error
			msg := sendRelay(ts, providerAddr, client1Acct, []string{ts.spec.Index})
			msg.Relays[0].Provider = tt.provider
			msg.Relays[0].CuSum = tt.cu
			msg.Relays[0].SessionId = tt.id
			ts.relayPaymentWithoutPay(msg, false)
		})
	}
}

func TestRelayPaymentDelayedDoubleSpending(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	msg := sendRelay(ts, providerAddr, client1Acct, []string{ts.spec.Index})
	ts.relayPaymentWithoutPay(msg, true)

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
			ts.relayPaymentWithoutPay(msg, false)
		})
	}
}

func TestRelayPaymentOldEpochs(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

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
			sig, err := sigs.Sign(client1Acct.SK, *relaySession)
			relaySession.Sig = sig
			require.NoError(t, err)

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}
			ts.relayPaymentWithoutPay(payment, tt.valid)
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
			_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

			cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10

			qos := &types.QualityOfServiceReport{
				Latency:      tt.latency,
				Availability: tt.availability,
				Sync:         tt.sync,
			}

			relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
			relaySession.QosReport = qos
			sig, err := sigs.Sign(client1Acct.SK, *relaySession)
			require.NoError(t, err)
			relaySession.Sig = sig

			payment := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}
			ts.relayPaymentWithoutPay(payment, tt.valid)
		})
	}
}

// TestVaultProviderRelayPayment tests that relay payment is sent by the provider and not vault
// Scenarios:
// 1. only provider (not vault) should send relay payments
func TestVaultProviderRelayPayment(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	clientAcc, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcc, provider := ts.GetAccount(common.PROVIDER, 0)
	vault := providerAcc.GetVaultAddr()
	qos := &types.QualityOfServiceReport{
		Latency:      sdk.OneDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}

	tests := []struct {
		name            string
		creator         string
		providerInRelay string
		valid           bool
	}{
		{"creator=provider, providerInRelay=provider", provider, provider, true},
		{"creator=vault, providerInRelay=provider", vault, provider, false},
		{"creator=provider, providerInRelay=vault", provider, vault, false},
		{"creator=vault, providerInRelay=vault", vault, vault, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession := ts.newRelaySession(tt.providerInRelay, 0, 100, ts.BlockHeight(), 0)
			relaySession.QosReport = qos
			sig, err := sigs.Sign(clientAcc.SK, *relaySession)
			require.NoError(t, err)
			relaySession.Sig = sig

			payment := types.MsgRelayPayment{
				Creator: tt.creator,
				Relays:  lavaslices.Slice(relaySession),
			}
			ts.relayPaymentWithoutPay(payment, tt.valid)
		})
	}
}

func TestEpochPaymentDeletion(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	providerAcct, providerAddr := ts.GetAccount(common.PROVIDER, 0)
	_, unresponsiveprovider := ts.GetAccount(common.PROVIDER, 1)

	cuSum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerAddr, 0, cuSum, ts.BlockHeight(), 0)
	relaySession.UnresponsiveProviders = []*types.ReportedProvider{{Address: unresponsiveprovider, Disconnections: 5, Errors: 2}}
	sig, err := sigs.Sign(client1Acct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	payment := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  lavaslices.Slice(relaySession),
	}

	ts.payAndVerifyBalance(payment, client1Acct.Addr, providerAcct.Vault.Addr, true, true, 100)

	ts.AdvanceEpochs(ts.EpochsToSave() + 1)

	listA := ts.Keepers.Pairing.GetAllUniqueEpochSessionStore(ts.Ctx)
	require.Len(t, listA, 0)

	list0 := ts.Keepers.Pairing.GetAllProviderEpochCuStore(ts.Ctx)
	require.Len(t, list0, 0)

	list1 := ts.Keepers.Pairing.GetAllProviderEpochComplainerCuStore(ts.Ctx)
	require.Len(t, list1, 0)

	list2 := ts.Keepers.Pairing.GetAllProviderConsumerEpochCuStore(ts.Ctx)
	require.Len(t, list2, 0)
}

// Test that after the consumer uses some CU it's updated in its project and subscription
func TestCuUsageInProjectsAndSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 2)    // 0 sub, 0 adm, 2 dev
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	dev1Acct, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	projectData := projectstypes.ProjectData{
		Name:    "proj1",
		Enabled: true,
		ProjectKeys: []projectstypes.ProjectKey{
			projectstypes.ProjectDeveloperKey(dev1Addr),
		},
		Policy: &planstypes.Policy{
			GeolocationProfile: int32(1),
			MaxProvidersToPair: 3,
			TotalCuLimit:       1000,
			EpochCuLimit:       100,
		},
	}
	err := ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.NoError(t, err)

	projectData.Name = "proj2"
	projectData.ProjectKeys = []projectstypes.ProjectKey{
		projectstypes.ProjectDeveloperKey(dev2Addr),
	}
	err = ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	qos := types.QualityOfServiceReport{
		Latency:      sdk.OneDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}

	relaySession := ts.newRelaySession(providerAddr, 0, 1, ts.BlockHeight(), 0)
	relaySession.QosReport = &qos
	relaySession.Sig, err = sigs.Sign(dev1Acct.SK, *relaySession)
	require.NoError(t, err)

	// Request payment (helper validates the balances and verifies expected errors through valid)
	relayPaymentMessage := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  lavaslices.Slice(relaySession),
	}
	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	sub, err := ts.QuerySubscriptionCurrent(client1Addr)
	require.NoError(t, err)
	require.True(t, sub.Sub != nil)

	proj1, err := ts.QueryProjectDeveloper(dev1Addr)
	require.NoError(t, err)
	require.Equal(t, sub.Sub.MonthCuTotal-sub.Sub.MonthCuLeft, proj1.Project.UsedCu)

	proj2, err := ts.QueryProjectDeveloper(dev2Addr)
	require.NoError(t, err)
	require.NotEqual(t, sub.Sub.MonthCuTotal-sub.Sub.MonthCuLeft, proj2.Project.UsedCu)
}

func TestBadgeValidation(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, _ := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

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
	sig, err := sigs.Sign(client1Acct.SK, relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig

	// make the context's lava chain ID a non-empty string (as usually defined in most tests)
	lavaChainID := "lavanet"
	ts.SetChainID(lavaChainID)

	// times 2 to have enough CU for the two tests that succeed
	badgeCuAllocation := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 2

	tests := []struct {
		name         string
		badgeAddress sdk.AccAddress // badge user (in happy flow)
		relaySigner  sigs.Account   // badge user (in happy flow)
		badgeSigner  sigs.Account   // badge granter, i.e. project developer (in happy flow)
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
				ts.Keepers.Pairing.RemoveAllEpochPaymentsForBlockAppendAdjustments(ts.Ctx, tt.epoch)
			}

			badge := types.CreateBadge(badgeCuAllocation, tt.epoch, tt.badgeAddress, tt.lavaChainID, []byte{})
			sig, err := sigs.Sign(tt.badgeSigner.SK, *badge)
			require.NoError(t, err)
			badge.ProjectSig = sig

			relaySession.Badge = badge
			relaySession.LavaChainId = tt.lavaChainID
			relaySession.Sig, err = sigs.Sign(tt.relaySigner.SK, *relaySession)
			require.NoError(t, err)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}

			ts.relayPaymentWithoutPay(relayPaymentMessage, tt.valid)
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
	sig, err := sigs.Sign(client1Acct.SK, *badge)
	require.NoError(t, err)
	badge.ProjectSig = sig

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

		// change session ID to avoid double spending
		relaySession.Sig, err = sigs.Sign(badgeAcct.SK, *relaySession)
		require.NoError(t, err)
		relays = append(relays, relaySession)
	}

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: providerAddr,
		Relays:  relays,
	}

	ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Vault.Addr, true, true, 100)
}

// Test:
// 1. basic badge CU enforcement works (can't ask more CU than badge's cuAllocation)
// 2. claiming payment with different session IDs in different relay payment messages works
func TestBadgeCuAllocationEnforcement(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	plan, err := ts.GetPlanFromSubscription(client1Addr, ts.BlockHeight())
	require.NoError(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	sig, err := sigs.Sign(client1Acct.SK, *badge)
	require.NoError(t, err)
	badge.ProjectSig = sig

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
			relaySession.Sig, err = sigs.Sign(badgeAcct.SK, *relaySession)
			require.NoError(t, err)

			relayPaymentMessage := types.MsgRelayPayment{
				Creator: providerAddr,
				Relays:  lavaslices.Slice(relaySession),
			}

			ts.relayPaymentWithoutPay(relayPaymentMessage, tt.valid)

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

	_, err := ts.TxSubscriptionBuy(client1Addr, client1Addr, "free", 1, false, false) // extend by a month so the sub won't expire
	require.NoError(t, err)

	badgeAcct, _ := ts.AddAccount("badge", 0, testBalance)

	epochStart := ts.EpochStart()

	plan, err := ts.GetPlanFromSubscription(client1Addr, ts.BlockHeight())
	require.NoError(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	sig, err := sigs.Sign(client1Acct.SK, *badge)
	require.NoError(t, err)
	badge.ProjectSig = sig

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
				relaySession.Epoch = int64(epochStart) // BuildRelayRequest always takes the current blockHeight, which is not desirable in this test
				relaySession.SessionId = uint64(i)
				relaySession.Sig, err = sigs.Sign(badgeAcct.SK, *relaySession)
				require.NoError(t, err)

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
			ts.payAndVerifyBalance(relayPaymentMessage, client1Acct.Addr, providerAcct.Vault.Addr, true, tt.valid, 100)

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

	plan, err := ts.GetPlanFromSubscription(client1Addr, ts.BlockHeight())
	require.NoError(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit / 2

	badge := types.CreateBadge(badgeCuAllocation, epochStart, badgeAcct.Addr, "", []byte{})
	sig, err := sigs.Sign(client1Acct.SK, *badge)
	require.NoError(t, err)
	badge.ProjectSig = sig

	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	cuSum := badgeCuAllocation

	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(providers[i].Addr.String(), 0, cuSum, epochStart, 0)
		relaySession.QosReport = qos
		relaySession.Sig, err = sigs.Sign(badgeAcct.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Badge = badge

		relayPaymentMessage := types.MsgRelayPayment{
			Creator: providers[i].Addr.String(),
			Relays:  lavaslices.Slice(relaySession),
		}
		ts.relayPaymentWithoutPay(relayPaymentMessage, true)

		badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, providers[i].Addr.String())
		badgeUsedCuMapEntry, found := ts.Keepers.Pairing.GetBadgeUsedCu(ts.Ctx, badgeUsedCuMapKey)
		require.True(t, found)
		require.Equal(t, cuSum, badgeUsedCuMapEntry.UsedCu)
	}
}

func TestIntOverflow(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair
	consumerAcct, consumerAddr := ts.GetAccount(common.CONSUMER, 0)
	provider1Acc, provider1Addr := ts.GetAccount(common.PROVIDER, 0)

	spec2 := ts.spec
	spec2.Index = "mock2"
	ts.AddSpec("mock2", spec2)

	err := ts.StakeProvider(provider1Acc.GetVaultAddr(), provider1Addr, spec2, testStake)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	// The TotalCuLimit needs to be higher than the CU used
	// The EpochCuLimit needs to be smaller than the CU used
	whalePolicy := planstypes.Policy{
		TotalCuLimit:       400,
		EpochCuLimit:       200,
		MaxProvidersToPair: 3,
		GeolocationProfile: 1,
	}

	whalePlan := planstypes.Plan{
		Index:                    "whale",
		Description:              "whale",
		Type:                     "rpc",
		Block:                    ts.BlockHeight(),
		Price:                    sdk.NewCoin(commonconsts.TestTokenDenom, sdk.NewInt(1000)),
		AllowOveruse:             true,
		OveruseRate:              10,
		AnnualDiscountPercentage: 20,
		PlanPolicy:               whalePolicy,
		ProjectsLimit:            10,
	}

	whalePlan2 := planstypes.Plan{
		Index:                    "whale2",
		Description:              "whale2",
		Type:                     "rpc",
		Block:                    ts.BlockHeight(),
		Price:                    sdk.NewCoin(commonconsts.TestTokenDenom, sdk.NewInt(1001)),
		AllowOveruse:             true,
		OveruseRate:              10,
		AnnualDiscountPercentage: 20,
		PlanPolicy:               whalePolicy,
		ProjectsLimit:            10,
	}

	whalePlan = ts.AddPlan("whale", whalePlan).Plan("whale")
	whalePlan2 = ts.AddPlan("whale2", whalePlan2).Plan("whale2")
	ts.AdvanceBlock()

	// Buy the whale plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, whalePlan.Index, 1, false, false)
	require.NoError(t, err)
	ts.AdvanceEpoch()

	relaySessionCus := []struct {
		Cu uint64
	}{
		{Cu: 60},
		{Cu: 70},
		{Cu: 60},
		{Cu: 60},
		{Cu: 60},
		{Cu: 60},
		{Cu: 10},
	}

	for i, relaySessionCu := range relaySessionCus {
		relaySession := ts.newRelaySession(provider1Addr, uint64(i), relaySessionCu.Cu, ts.BlockHeight(), 0)
		if i != 0 {
			relaySession.SpecId = spec2.Index
		}
		sig, err := sigs.Sign(consumerAcct.SK, *relaySession)
		relaySession.Sig = sig
		require.NoError(t, err)

		_, err = ts.TxPairingRelayPayment(provider1Addr, relaySession)
		require.NoError(t, err)
	}

	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, whalePlan2.Index, 1, false, false)
	require.NoError(t, err)
	for i := 0; i < 20; i++ {
		ts.AdvanceEpoch()
	}
}

func TestPairingCaching(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(3, 3, 0) // 3 provider, 3 client, default providers-to-pair

	ts.AdvanceEpoch()

	relayNum := uint64(0)
	totalCU := uint64(0)
	// trigger relay payment with cache
	for i := 0; i < 3; i++ {
		relays := []*types.RelaySession{}
		_, provider1Addr := ts.GetAccount(common.PROVIDER, i)
		for i := 0; i < 3; i++ {
			consumerAcct, _ := ts.GetAccount(common.CONSUMER, i)
			totalCU = 0
			for i := 0; i < 50; i++ {
				totalCU += uint64(i)
				relaySession := ts.newRelaySession(provider1Addr, relayNum, uint64(i), ts.BlockHeight(), 0)
				sig, err := sigs.Sign(consumerAcct.SK, *relaySession)
				relaySession.Sig = sig
				require.NoError(t, err)
				relays = append(relays, relaySession)
				relayNum++
			}
		}
		_, err := ts.TxPairingRelayPayment(provider1Addr, relays...)
		require.NoError(t, err)
	}

	pecs := ts.Keepers.Pairing.GetAllProviderEpochCuStore(ts.Ctx)
	require.Len(t, pecs, 3)

	UniquePayments := ts.Keepers.Pairing.GetAllUniqueEpochSessionStore(ts.Ctx)
	require.Len(t, UniquePayments, 3*3*50)

	storages := ts.Keepers.Pairing.GetAllProviderConsumerEpochCuStore(ts.Ctx)
	require.Len(t, storages, 3*3)

	for i := 0; i < 3; i++ {
		consumerAcct, _ := ts.GetAccount(common.CONSUMER, i)
		project, err := ts.GetProjectForDeveloper(consumerAcct.Addr.String(), ts.BlockHeight())
		require.NoError(t, err)
		require.Equal(t, totalCU*3, project.UsedCu)

		sub, err := ts.QuerySubscriptionCurrent(consumerAcct.Addr.String())
		require.NoError(t, err)
		require.Equal(t, totalCU*3, sub.Sub.MonthCuTotal-sub.Sub.MonthCuLeft)
	}
}
