package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/testutil/common"
	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
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
	_, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Name, provider2)
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
	_, stakeStorageFound := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Name, provider2)
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

// TestUpdateReputationEpochQosScore tests the update of the reputation's epoch qos score
// Scenarios:
//  1. provider1 sends relay -> its reputation is updated (epoch score and time last updated),
//     also, provider2 reputation is not updated
func TestUpdateReputationEpochQosScore(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 2 providers, 1 client, default providers-to-pair

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	_, provider2 := ts.GetAccount(common.PROVIDER, 1)
	qos := &types.QualityOfServiceReport{
		Latency:      sdk.ZeroDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.ZeroDec(),
	}

	res, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := res.Sub.Cluster

	// set default reputations for both providers. Advance epoch to change the current block time
	ts.Keepers.Pairing.SetReputation(ts.Ctx, ts.spec.Index, cluster, provider1, types.NewReputation(ts.Ctx))
	ts.Keepers.Pairing.SetReputation(ts.Ctx, ts.spec.Index, cluster, provider2, types.NewReputation(ts.Ctx))
	ts.AdvanceEpoch()

	// send relay payment msg from provider1
	relaySession := ts.newRelaySession(provider1, 0, 100, ts.BlockHeight(), 1)
	relaySession.QosExcellenceReport = qos
	sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig

	payment := types.MsgRelayPayment{
		Creator: provider1,
		Relays:  []*types.RelaySession{relaySession},
	}
	ts.relayPaymentWithoutPay(payment, true)

	// get both providers reputation: provider1 should have its epoch score and time last updated changed,
	// provider2 should have nothing change from the default
	r1, found := ts.Keepers.Pairing.GetReputation(ts.Ctx, ts.spec.Index, cluster, provider1)
	require.True(t, found)
	r2, found := ts.Keepers.Pairing.GetReputation(ts.Ctx, ts.spec.Index, cluster, provider2)
	require.True(t, found)

	require.Greater(t, r1.TimeLastUpdated, r2.TimeLastUpdated)
	epochScore1, err := r1.EpochScore.Score.Resolve()
	require.NoError(t, err)
	epochScore2, err := r2.EpochScore.Score.Resolve()
	require.NoError(t, err)
	variance1, err := r1.EpochScore.Variance.Resolve()
	require.NoError(t, err)
	variance2, err := r2.EpochScore.Variance.Resolve()
	require.NoError(t, err)
	require.True(t, epochScore1.LT(epochScore2)) // score is lower because QoS is excellent
	require.True(t, variance1.GT(variance2))     // variance is higher because the QoS is significantly differnet from DefaultQos

	entry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider1)
	require.True(t, found)
	require.True(t, entry.Stake.IsEqual(r1.Stake))
}

// TestUpdateReputationEpochQosScoreTruncation tests the following scenarios:
// 1. stabilization period has not passed -> no truncation
// 2. stabilization period passed -> with truncation (score update is smaller than the first one)
// note, this test works since we use a bad QoS report (compared to default) so we know that the score should
// increase (which is considered worse)
func TestUpdateReputationEpochQosScoreTruncation(t *testing.T) {
	// these will be used to compare the score change with/without truncation
	scoreUpdates := []sdk.Dec{}

	// we set the stabilization period to 2 epochs time. Advancing one epoch means we won't truncate,
	// advancing 3 means we will truncate.
	epochsToAdvance := []uint64{1, 3}

	for i := range epochsToAdvance {
		ts := newTester(t)
		ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

		consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
		_, provider1 := ts.GetAccount(common.PROVIDER, 0)
		qos := &types.QualityOfServiceReport{
			Latency:      sdk.NewDec(1000),
			Availability: sdk.OneDec(),
			Sync:         sdk.NewDec(1000),
		}

		resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
		require.NoError(t, err)
		cluster := resQCurrent.Sub.Cluster

		// set stabilization period to be 2*epoch time
		resQParams, err := ts.Keepers.Pairing.Params(ts.GoCtx, &types.QueryParamsRequest{})
		require.NoError(t, err)
		resQParams.Params.ReputationVarianceStabilizationPeriod = int64(ts.EpochTimeDefault().Seconds())
		ts.Keepers.Pairing.SetParams(ts.Ctx, resQParams.Params)

		// set default reputation
		ts.Keepers.Pairing.SetReputation(ts.Ctx, ts.spec.Index, cluster, provider1, types.NewReputation(ts.Ctx))

		// advance epochs
		ts.AdvanceEpochs(epochsToAdvance[i])

		// send relay payment msg from provider1
		relaySession := ts.newRelaySession(provider1, 0, 100, ts.BlockHeight(), 1)
		relaySession.QosExcellenceReport = qos
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: provider1,
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)

		// get update of epoch score
		r, found := ts.Keepers.Pairing.GetReputation(ts.Ctx, ts.spec.Index, cluster, provider1)
		require.True(t, found)
		epochScoreNoTruncation, err := r.EpochScore.Score.Resolve()
		require.NoError(t, err)
		defaultEpochScore, err := types.ZeroQosScore.Score.Resolve()
		require.NoError(t, err)
		scoreUpdates = append(scoreUpdates, epochScoreNoTruncation.Sub(defaultEpochScore))
	}

	// require that the score update that was not truncated is larger than the one that was truncated
	require.True(t, scoreUpdates[0].GT(scoreUpdates[1]))
}

// TestUpdateReputationEpochQosScoreTruncation tests the following scenario:
// 1. relay num is the reputation update weight. More relays = bigger update
func TestUpdateReputationEpochQosScoreRelayNumWeight(t *testing.T) {
	// these will be used to compare the score change with high/low relay numbers
	scoreUpdates := []sdk.Dec{}

	// we set the stabilization period to 2 epochs time. Advancing one epoch means we won't truncate,
	// advancing 3 means we will truncate.
	relayNums := []uint64{100, 1}

	for i := range relayNums {
		ts := newTester(t)
		ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

		consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
		_, provider1 := ts.GetAccount(common.PROVIDER, 0)
		qos := &types.QualityOfServiceReport{
			Latency:      sdk.NewDec(1000),
			Availability: sdk.OneDec(),
			Sync:         sdk.NewDec(1000),
		}

		resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
		require.NoError(t, err)
		cluster := resQCurrent.Sub.Cluster

		// set stabilization period to be 2*epoch time to avoid truncation
		resQParams, err := ts.Keepers.Pairing.Params(ts.GoCtx, &types.QueryParamsRequest{})
		require.NoError(t, err)
		resQParams.Params.ReputationVarianceStabilizationPeriod = int64(ts.EpochTimeDefault().Seconds())
		ts.Keepers.Pairing.SetParams(ts.Ctx, resQParams.Params)

		// set default reputation
		ts.Keepers.Pairing.SetReputation(ts.Ctx, ts.spec.Index, cluster, provider1, types.NewReputation(ts.Ctx))
		ts.AdvanceEpoch()

		// send relay payment msg from provider1
		relaySession := ts.newRelaySession(provider1, 0, 100, ts.BlockHeight(), relayNums[i])
		relaySession.QosExcellenceReport = qos
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: provider1,
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)

		// get update of epoch score
		r, found := ts.Keepers.Pairing.GetReputation(ts.Ctx, ts.spec.Index, cluster, provider1)
		require.True(t, found)
		epochScoreNoTruncation, err := r.EpochScore.Score.Resolve()
		require.NoError(t, err)
		defaultEpochScore, err := types.ZeroQosScore.Score.Resolve()
		require.NoError(t, err)
		scoreUpdates = append(scoreUpdates, epochScoreNoTruncation.Sub(defaultEpochScore))
	}

	// require that the score update that was with 1000 relay num is larger than the one with one relay num
	require.True(t, scoreUpdates[0].GT(scoreUpdates[1]))
}

// TestUpdateReputationScores tests that on epoch start: score is modified, epoch score is zeroed time last
// updated is modified, and reputation stake changed to current stake. We do this with the following scenario:
// 1. we set half life factor to be one epoch time
// 2. we set a provider with default QoS score, let him send a relay and advance epoch
// 3. we check expected result
func TestReputationUpdateOnEpochStart(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	_, provider1 := ts.GetAccount(common.PROVIDER, 0)
	qos := &types.QualityOfServiceReport{
		Latency:      sdk.NewDec(1000),
		Availability: sdk.OneDec(),
		Sync:         sdk.NewDec(1000),
	}

	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	// set half life factor to be epoch time
	resQParams, err := ts.Keepers.Pairing.Params(ts.GoCtx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	resQParams.Params.ReputationHalfLifeFactor = int64(ts.EpochTimeDefault().Seconds())
	ts.Keepers.Pairing.SetParams(ts.Ctx, resQParams.Params)

	// set default reputation and keep some properties for future comparison
	ts.Keepers.Pairing.SetReputation(ts.Ctx, ts.spec.Index, cluster, provider1, types.NewReputation(ts.Ctx))
	creationTime := ts.BlockTime().UTC().Unix()
	entry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider1)
	require.True(t, found)
	stake := sdk.NewCoin(entry.Stake.Denom, entry.EffectiveStake().AddRaw(1))

	// stake some more coins and advance epoch
	err = ts.StakeProvider(entry.Vault, entry.Address, ts.spec, entry.EffectiveStake().Int64())
	require.NoError(t, err)
	ts.AdvanceEpoch()

	// send relay payment msg from provider1
	relaySession := ts.newRelaySession(provider1, 0, 100, ts.BlockHeight(), 10)
	relaySession.QosExcellenceReport = qos
	sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig

	payment := types.MsgRelayPayment{
		Creator: provider1,
		Relays:  []*types.RelaySession{relaySession},
	}
	ts.relayPaymentWithoutPay(payment, true)

	// advance epoch and check reputation for expected results
	ts.AdvanceEpoch()
	reputation, found := ts.Keepers.Pairing.GetReputation(ts.Ctx, ts.spec.Index, cluster, provider1)
	require.True(t, found)
	require.False(t, reputation.Score.Equal(types.ZeroQosScore))
	require.True(t, reputation.EpochScore.Equal(types.ZeroQosScore))
	require.Equal(t, ts.BlockTime().UTC().Unix(), reputation.TimeLastUpdated)
	require.Equal(t, creationTime, reputation.CreationTime)
	require.NotEqual(t, reputation.CreationTime, reputation.TimeLastUpdated)

	entry, found = ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, ts.spec.Index, provider1)
	require.True(t, found)
	stakeAfterUpdate := sdk.NewCoin(entry.Stake.Denom, entry.EffectiveStake())

	require.False(t, stakeAfterUpdate.IsEqual(stake))
	require.True(t, reputation.Stake.IsEqual(stakeAfterUpdate))
}

// TestUpdateReputationScoresMap checks that after calling UpdateReputationsForEpochStart() we get a sorted (by QoS score)
// map[chain+cluster]stakeProviderScores in ascending order. We do this with the following scenario:
// 1. have 4 providers: 2 on chain "mockspec", 3 on chain "mockspec1" (one provider is staked on both)
// 2. let the providers have different reputation scores and different stakes and call updateReputationsScores()
// 3. expect to see two map keys (for the two chains) and a list of providers in ascending order by QoS score
func TestUpdateReputationScoresSortedMap(t *testing.T) {
	ts := newTester(t)

	// create provider addresses
	_, p1 := ts.AddAccount(common.PROVIDER, 0, testStake)
	_, p2 := ts.AddAccount(common.PROVIDER, 1, testStake)
	_, p3 := ts.AddAccount(common.PROVIDER, 2, testStake)
	_, p4 := ts.AddAccount(common.PROVIDER, 3, testStake)

	// set reputations like described above
	providers := []string{p1, p2, p1, p3, p4} // p1 will be "staked" on two chains
	specs := []string{"mockspec", "mockspec", "mockspec1", "mockspec1", "mockspec1"}
	stakes := []math.Int{sdk.NewInt(1), sdk.NewInt(2), sdk.NewInt(3), sdk.NewInt(4), sdk.NewInt(5)}
	zeroFrac := types.Frac{Num: sdk.ZeroDec(), Denom: sdk.MaxSortableDec}
	epochScores := []types.QosScore{
		{Score: types.Frac{Num: sdk.NewDec(6), Denom: sdk.NewDec(3)}, Variance: zeroFrac},  // score = 2
		{Score: types.Frac{Num: sdk.NewDec(9), Denom: sdk.NewDec(3)}, Variance: zeroFrac},  // score = 3
		{Score: types.Frac{Num: sdk.NewDec(3), Denom: sdk.NewDec(3)}, Variance: zeroFrac},  // score = 1
		{Score: types.Frac{Num: sdk.NewDec(12), Denom: sdk.NewDec(3)}, Variance: zeroFrac}, // score = 4
		{Score: types.Frac{Num: sdk.NewDec(9), Denom: sdk.NewDec(3)}, Variance: zeroFrac},  // score = 3
	}
	for i := range providers {
		reputation := types.NewReputation(ts.Ctx)
		reputation.EpochScore = epochScores[i]
		reputation.Score = types.ZeroQosScore
		reputation.Stake = sdk.NewCoin(ts.TokenDenom(), stakes[i])
		ts.Keepers.Pairing.SetReputation(ts.Ctx, specs[i], "cluster", providers[i], reputation)
	}

	// create expected map (supposed to be sorted by score)
	expected := map[string]keeper.StakeProviderScores{
		"mockspec cluster": {
			TotalStake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(3)), // 1+2
			ProviderScores: []keeper.ProviderQosScore{
				{
					Provider: p1,
					Score:    types.QosScore{Score: types.Frac{Num: sdk.NewDec(6), Denom: sdk.NewDec(3)}, Variance: zeroFrac},
					Stake:    sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(1)),
				},
				{
					Provider: p2,
					Score:    types.QosScore{Score: types.Frac{Num: sdk.NewDec(9), Denom: sdk.NewDec(3)}, Variance: zeroFrac},
					Stake:    sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(2)),
				},
			},
		},

		"mockspec1 cluster": {
			TotalStake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(12)), // 3+4+5
			ProviderScores: []keeper.ProviderQosScore{
				{
					Provider: p1,
					Score:    types.QosScore{Score: types.Frac{Num: sdk.NewDec(3), Denom: sdk.NewDec(3)}, Variance: zeroFrac},
					Stake:    sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(3)),
				},
				{
					Provider: p4,
					Score:    types.QosScore{Score: types.Frac{Num: sdk.NewDec(9), Denom: sdk.NewDec(3)}, Variance: zeroFrac},
					Stake:    sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(5)),
				},
				{
					Provider: p3,
					Score:    types.QosScore{Score: types.Frac{Num: sdk.NewDec(12), Denom: sdk.NewDec(3)}, Variance: zeroFrac},
					Stake:    sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(4)),
				},
			},
		},
	}

	// call UpdateReputationsForEpochStart() and check the scores map
	scores, err := ts.Keepers.Pairing.UpdateReputationsForEpochStart(ts.Ctx)
	require.NoError(t, err)

	for chainCluster, stakeProviderScores := range scores {
		expectedScores := expected[chainCluster]
		require.True(t, expectedScores.TotalStake.IsEqual(stakeProviderScores.TotalStake))
		for i := range stakeProviderScores.ProviderScores {
			require.Equal(t, expectedScores.ProviderScores[i].Provider, stakeProviderScores.ProviderScores[i].Provider)
			require.True(t, expectedScores.ProviderScores[i].Stake.IsEqual(stakeProviderScores.ProviderScores[i].Stake))

			expectedScore, err := expectedScores.ProviderScores[i].Score.Score.Resolve()
			require.NoError(t, err)
			score, err := stakeProviderScores.ProviderScores[i].Score.Score.Resolve()
			require.NoError(t, err)
			require.True(t, expectedScore.RoundInt().Equal(score.RoundInt()))
		}
	}
}

// TestReputationPairingScoreBenchmark tests that the benchmark value after calling getBenchmarkReputationScore()
// is the score of the last provider we iterate on when aggregating stake to pass ReputationPairingScoreBenchmarkStakeThreshold.
//
//	We do this with the following scenario:
//	1. have 5 providers with varying stakes -> providers with 10% of stake should have max scores, the others should have
//	   pairing scores that differ by their QoS scores difference
func TestReputationPairingScoreBenchmark(t *testing.T) {
	ts := newTester(t)
	totalStake := sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(1000)) // threshold for benchmark will be 100ulava

	// create provider addresses
	_, p1 := ts.AddAccount(common.PROVIDER, 0, testStake)
	_, p2 := ts.AddAccount(common.PROVIDER, 1, testStake)
	_, p3 := ts.AddAccount(common.PROVIDER, 2, testStake)
	_, p4 := ts.AddAccount(common.PROVIDER, 3, testStake)
	_, p5 := ts.AddAccount(common.PROVIDER, 4, testStake)

	qosScores := createQosScoresForBenchmarkTest()

	// create a StakeProviderScores object with ProviderScores list which is descending by score
	sps := keeper.StakeProviderScores{
		TotalStake: totalStake,
		ProviderScores: []keeper.ProviderQosScore{
			{Provider: p1, Score: qosScores[0], Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(1))},
			{Provider: p2, Score: qosScores[1], Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(19))},
			{Provider: p3, Score: qosScores[2], Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(40))},
			{Provider: p4, Score: qosScores[3], Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(60))},
			{Provider: p5, Score: qosScores[4], Stake: sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(1000))},
		},
	}

	// the benchmark should be equal to qosScores[3] since p4 is the last provider on which the
	// stake aggregation should complete (1+19+40+60=100)
	benchmark, err := ts.Keepers.Pairing.GetBenchmarkReputationScore(sps)
	require.NoError(t, err)
	expectedBenchmark, err := qosScores[3].Score.Resolve()
	require.NoError(t, err)
	require.True(t, benchmark.Equal(expectedBenchmark))
}

// createQosScoresForBenchmarkTest is a helper function to create an array of QoS scores
func createQosScoresForBenchmarkTest() []types.QosScore {
	nums := []math.LegacyDec{sdk.NewDec(1000), sdk.NewDec(100), sdk.NewDec(90), sdk.NewDec(40), sdk.NewDec(10)}
	zeroFrac := types.Frac{Num: sdk.ZeroDec(), Denom: sdk.MaxSortableDec}
	qosScores := []types.QosScore{}
	for _, num := range nums {
		scoreFrac, err := types.NewFrac(num, sdk.OneDec())
		if err != nil {
			panic(err)
		}
		qs := types.NewQosScore(scoreFrac, zeroFrac)
		qosScores = append(qosScores, qs)
	}
	return qosScores
}

// TestReputationPairingScore tests that on epoch start:
// 1. The reputation pairing score is set.
// 2. Providers that didn't send relays get the default reputation pairing score.
// 2. If a provider has a bad QoS report, it yields a bad reputation pairing score.
// We test it by having 4 providers:
//
//	p1: high stake and great QoS (will be used as benchmark),
//	p2: low stake and good QoS score
//	p3: same low stake and bad QoS score
//	p4: high stake that doesn't send relays.
//
// We check that all 4 have reputation pairing score: p1 with max score, p2 with better score than p3, p4
// has no reputation pairing score.
func TestReputationPairingScore(t *testing.T) {
	ts := newTester(t)
	ts, reports := ts.setupForReputation(false)

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	minStake := ts.spec.MinStakeProvider.Amount.Int64()
	stakes := []int64{minStake * 5, minStake + 10, minStake + 10, minStake * 2}
	providers := []string{}
	for i := 0; i < 4; i++ {
		_, provider := ts.AddAccount(common.PROVIDER, i, testBalance)
		providers = append(providers, provider)
	}
	for i := 0; i < 4; i++ {
		// create providers
		err = ts.StakeProvider(providers[i], providers[i], ts.spec, stakes[i])
		require.NoError(t, err)
	}
	// advance epoch to apply pairing
	ts.AdvanceEpoch()

	// send relays (except for last one)
	for i := 0; i < 3; i++ {
		relaySession := ts.newRelaySession(providers[i], 0, 100, ts.BlockHeight(), 10)
		relaySession.QosExcellenceReport = &reports[i]
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: providers[i],
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)
	}

	// advance epoch to update pairing scores
	ts.AdvanceEpoch()

	// check results
	pairingScores := []math.LegacyDec{}
	for i := range providers {
		score, found := ts.Keepers.Pairing.GetReputationScore(ts.Ctx, ts.spec.Index, cluster, providers[i])
		if i < 3 {
			require.True(t, found)
			pairingScores = append(pairingScores, score)
		} else {
			require.False(t, found)
		}
	}
	require.True(t, pairingScores[0].Equal(types.MaxReputationPairingScore))
	require.True(t, pairingScores[1].GT(pairingScores[2]))
}

// TestReputationPairingScoreWithinRange tests that the reputation pairing score is always between
// MinReputationPairingScore and MaxReputationPairingScore.
// We test it by setting two providers with extreme QoS reports: one very good, and one very bad.
// We expect that both pairing scores will be in the expected range.
func TestReputationPairingScoreWithinRange(t *testing.T) {
	ts := newTester(t)
	ts, reports := ts.setupForReputation(false)

	greatQos := reports[GreatQos]
	badQos := reports[BadQos]

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	providers := []string{}
	for i := 0; i < 2; i++ {
		_, provider := ts.AddAccount(common.PROVIDER, i, testBalance)
		providers = append(providers, provider)
	}

	minStake := ts.spec.MinStakeProvider.Amount.Int64()
	for i := 0; i < 2; i++ {
		// create providers
		err = ts.StakeProvider(providers[i], providers[i], ts.spec, minStake)
		require.NoError(t, err)
	}
	// advance epoch to apply pairing
	ts.AdvanceEpoch()

	// send relays
	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(providers[i], 0, 100, ts.BlockHeight(), 10)
		report := greatQos
		if i == 0 {
			report = badQos
		}
		relaySession.QosExcellenceReport = &report
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: providers[i],
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)
	}

	// advance epoch to update pairing scores
	ts.AdvanceEpoch()

	// check results are withing the expected range
	for i := range providers {
		score, found := ts.Keepers.Pairing.GetReputationScore(ts.Ctx, ts.spec.Index, cluster, providers[i])
		require.True(t, found)
		if score.LT(types.MinReputationPairingScore) || score.GT(types.MaxReputationPairingScore) {
			require.FailNow(t, "score is not within expected pairing score range")
		}
	}
}

// TestReputationPairingScoreZeroQosScores tests that if all providers have a QoS score of zero,
// they all get the max reputation pairing score.
// We test it by having two providers with zero QoS score. We expect that both will have max reputation
// pairing score.
func TestReputationPairingScoreZeroQosScores(t *testing.T) {
	ts := newTester(t)
	ts, _ = ts.setupForReputation(false)

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	providers := []string{}
	for i := 0; i < 2; i++ {
		_, provider := ts.AddAccount(common.PROVIDER, i, testBalance)
		providers = append(providers, provider)
	}

	minStake := ts.spec.MinStakeProvider.Amount.Int64()
	for i := 0; i < 2; i++ {
		// create providers
		err = ts.StakeProvider(providers[i], providers[i], ts.spec, minStake)
		require.NoError(t, err)
	}
	// advance epoch to apply pairing
	ts.AdvanceEpoch()

	// send relays
	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(providers[i], 0, 100, ts.BlockHeight(), 10)
		perfectQosReport := types.QualityOfServiceReport{
			Latency:      sdk.ZeroDec(),
			Availability: sdk.OneDec(),
			Sync:         sdk.ZeroDec(),
		}
		relaySession.QosExcellenceReport = &perfectQosReport
		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: providers[i],
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)
	}

	// advance epoch to update pairing scores
	ts.AdvanceEpoch()

	// check results are max pairing score
	for i := range providers {
		score, found := ts.Keepers.Pairing.GetReputationScore(ts.Ctx, ts.spec.Index, cluster, providers[i])
		require.True(t, found)
		require.True(t, score.Equal(types.MaxReputationPairingScore))
	}
}

// TestReputationPairingScoreFixation tests that the reputation pairing score is saved in a fixation store
// and can be fetched using a block argument.
// We test this by making two providers, one with high stake and great QoS score and one with low stake and
// bad QoS score. We get the bad provider pairing score and send another relay with a slightly improved QoS
// score. We expect that fetching the pairing score from the past will be lower than the current one
func TestReputationPairingScoreFixation(t *testing.T) {
	ts := newTester(t)
	ts, reports := ts.setupForReputation(false)

	greatQos := reports[GreatQos]
	goodQos := reports[GoodQos]
	badQos := reports[BadQos]

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	providers := []string{}
	for i := 0; i < 2; i++ {
		_, provider := ts.AddAccount(common.PROVIDER, i, testBalance)
		providers = append(providers, provider)
	}

	minStake := ts.spec.MinStakeProvider.Amount.Int64()
	for i := 0; i < 2; i++ {
		// create providers
		err = ts.StakeProvider(providers[i], providers[i], ts.spec, minStake)
		require.NoError(t, err)
	}
	// advance epoch to apply pairing
	ts.AdvanceEpoch()

	// send relays
	for i := 0; i < 2; i++ {
		relaySession := ts.newRelaySession(providers[i], 0, 100, ts.BlockHeight(), 10)

		if i == 0 {
			relaySession.QosExcellenceReport = &greatQos
		} else {
			relaySession.QosExcellenceReport = &badQos
		}

		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.NoError(t, err)
		relaySession.Sig = sig

		payment := types.MsgRelayPayment{
			Creator: providers[i],
			Relays:  []*types.RelaySession{relaySession},
		}
		ts.relayPaymentWithoutPay(payment, true)
	}

	// advance epoch to update pairing scores
	ts.AdvanceEpoch()
	badQosBlock := ts.BlockHeight()

	// send another relay with provider1 with a better QoS report
	relaySession := ts.newRelaySession(providers[1], 0, 100, ts.BlockHeight(), 10)
	relaySession.QosExcellenceReport = &goodQos
	sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
	require.NoError(t, err)
	relaySession.Sig = sig

	payment := types.MsgRelayPayment{
		Creator: providers[1],
		Relays:  []*types.RelaySession{relaySession},
	}
	ts.relayPaymentWithoutPay(payment, true)

	// advance epoch to update pairing scores
	ts.AdvanceEpoch()
	goodQosBlock := ts.BlockHeight()

	// fetch pairing score for both badQosBlock and goodQosBlock
	// verify that the pairing score changes for the better
	badQosPairingScore, _, found := ts.Keepers.Pairing.GetReputationScoreForBlock(ts.Ctx, ts.spec.Index, cluster, providers[1], badQosBlock)
	require.True(t, found)
	goodQosPairingScore, _, found := ts.Keepers.Pairing.GetReputationScoreForBlock(ts.Ctx, ts.spec.Index, cluster, providers[1], goodQosBlock)
	require.True(t, found)
	require.True(t, goodQosPairingScore.GT(badQosPairingScore))
}

// TestReputationPairingScoreStakeAggregation tests that the benchmark is determined by stake. We test the following
// scenarios:
//  1. Have 2 providers with equal stake -> pairing scores should differ by their QoS scores difference
//  2. Have 2 providers one with high score and low stake, the other with low score and high stake -> both should get max
//     pairing score
func TestReputationPairingScoreStakeAggregation(t *testing.T) {
	ts := newTester(t)
	ts, reports := ts.setupForReputation(false)

	greatQos := reports[GreatQos]
	goodQos := reports[GoodQos]
	badQos := reports[BadQos]

	consumerAcc, consumer := ts.GetAccount(common.CONSUMER, 0)
	resQCurrent, err := ts.QuerySubscriptionCurrent(consumer)
	require.NoError(t, err)
	cluster := resQCurrent.Sub.Cluster

	providers := []string{}
	for i := 0; i < 2; i++ {
		_, provider := ts.AddAccount(common.PROVIDER, i, testBalance)
		providers = append(providers, provider)
	}
	minStake := ts.spec.MinStakeProvider.Amount.Int64()

	tests := []struct {
		name            string
		stakes          []int64
		reports         []types.QualityOfServiceReport
		differentScores bool
	}{
		{"equal stake, different QoS", []int64{minStake, minStake}, []types.QualityOfServiceReport{goodQos, badQos}, true},
		{"high score + low stake, low score + high stake", []int64{minStake, minStake * 20}, []types.QualityOfServiceReport{greatQos, badQos}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 2; i++ {
				// create providers
				err = ts.StakeProvider(providers[i], providers[i], ts.spec, tt.stakes[i])
				require.NoError(t, err)
			}
			// advance epoch to apply pairing
			ts.AdvanceEpoch()

			// send relays
			for i := 0; i < 2; i++ {
				relaySession := ts.newRelaySession(providers[i], 0, 100, ts.BlockHeight(), 10)
				relaySession.QosExcellenceReport = &tt.reports[i]
				sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
				require.NoError(t, err)
				relaySession.Sig = sig

				payment := types.MsgRelayPayment{
					Creator: providers[i],
					Relays:  []*types.RelaySession{relaySession},
				}
				ts.relayPaymentWithoutPay(payment, true)
			}

			// advance epoch to update pairing scores
			ts.AdvanceEpoch()

			// get pairing scores and check results
			pairingScores := []math.LegacyDec{}
			for i := range providers {
				score, found := ts.Keepers.Pairing.GetReputationScore(ts.Ctx, ts.spec.Index, cluster, providers[i])
				require.True(t, found)
				pairingScores = append(pairingScores, score)
			}

			if tt.differentScores {
				require.True(t, pairingScores[0].GT(pairingScores[1]))
			} else {
				require.True(t, pairingScores[0].Equal(pairingScores[1]))
				require.True(t, pairingScores[0].Equal(types.MaxReputationPairingScore))
			}
		})
	}
}
