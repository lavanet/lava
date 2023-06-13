package keeper_test

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptionypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

var (
	balance int64 = 100000000
	stake   int64 = 100000
)

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	providers []*common.Account
	clients   []*common.Account
	spec      spectypes.Spec
	plan      plantypes.Plan
}

func createStubRequest(relaySession *types.RelaySession) *types.RelayRequest {
	req := &types.RelayRequest{
		RelaySession: relaySession,
		RelayData:    &types.RelayPrivateData{Data: []byte("stub-data")},
	}
	return req
}

func (ts *testStruct) addClient(amount int) error {
	for i := 0; i < amount; i++ {
		sk, address := sigs.GenerateFloatingKey()
		ts.clients = append(ts.clients, &common.Account{SK: sk, Addr: address})
		err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
		if err != nil {
			return err
		}

		_, err = ts.servers.SubscriptionServer.Buy(ts.ctx, &subscriptionypes.MsgBuy{Creator: address.String(), Consumer: address.String(), Index: ts.plan.Index, Duration: 1})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ts *testStruct) addProvider(amount int) error {
	return ts.addProviderGeolocation(amount, 1)
}

func (ts *testStruct) addProviderGeolocation(amount int, geolocation uint64) error {
	for i := 0; i < amount; i++ {
		sk, address := sigs.GenerateFloatingKey()
		ts.providers = append(ts.providers, &common.Account{SK: sk, Addr: address})
		err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
		if err != nil {
			return err
		}
		endpoints := []epochstoragetypes.Endpoint{}
		endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: geolocation})
		_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: geolocation, Endpoints: endpoints})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ts *testStruct) getProvider(addr string) *common.Account {
	for _, provider := range ts.providers {
		if provider.Addr.String() == addr {
			return provider
		}
	}
	return nil
}

// Test that a provider payment is valid if he asked it within the chain's memory, and check the relay payment object (RPO)
func TestRelayPaymentMemoryTransferAfterEpochChange(t *testing.T) {
	// setup testnet with mock spec, a staked client and a staked provider
	ts := setupForPaymentTest(t)

	// Advance epoch to apply client and provider stake
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	firstEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// Get epochsToSave, and epochBlocks
	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), firstEpoch)
	require.Nil(t, err)
	epochBlocks, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), firstEpoch)
	require.Nil(t, err)

	// The last block offset in each epoch is epochBlocks-1
	lastBlockInEpoch := epochBlocks - 1

	// define tests - different epoch+blocks, valid tells if the payment request should work
	tests := []struct {
		name            string
		epochsToAdvance uint64 // number of epochs to advance in the test
		blocksToAdvance uint64 // number of blocks to advance in the test
		valid           bool
	}{
		{"PaymentFirstEpoch", 0, 0, true},                                // first block of current epoch
		{"PaymentEndOfMemoryEpochFirstBlock", epochsToSave, 0, true},     // first block of end of memory epoch
		{"PaymentEndOfMemoryEpochLastBlock", 0, lastBlockInEpoch, true},  // last block of end of memory epoch
		{"PaymentOutOfMemoryEpochFirstBlock", 0, 1, false},               // first block of out of memory epoch (end of memory+1)
		{"PaymentOutOfMemoryEpochLastBlock", 0, lastBlockInEpoch, false}, // last block of out of memory epoch (end of memory+1)
	}

	sessionCounter := uint64(0)
	for _, tt := range tests {
		sessionCounter += 1
		t.Run(tt.name, func(t *testing.T) {
			// Advance epochs according to the epochsToAdvance
			if tt.epochsToAdvance != 0 {
				for i := 0; i < int(tt.epochsToAdvance); i++ {
					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
			}

			// Advance blocks according to the blocksToAdvance
			if tt.blocksToAdvance != 0 {
				for i := 0; i < int(tt.blocksToAdvance); i++ {
					ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)
				}
			}

			// Create relay request that was done in the first epoch. Change session ID each iteration to avoid double spending error (provider asks reward for the same transaction twice)
			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)
			relaySession.Epoch = int64(firstEpoch)
			relaySession.SessionId = sessionCounter

			// Sign and send the payment requests
			sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
			relaySession.Sig = sig
			require.Nil(t, err)

			// Request payment (helper function validates the balances and verifies if we should get an error through valid)
			var Relays []*types.RelaySession
			Relays = append(Relays, relaySession)
			relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, true, tt.valid, ts.clients[0].Addr, ts.providers[0].Addr)

			// Check the RPO exists (shouldn't exist after epochsToSave+1 passes)
			verifyRelayPaymentObjects(t, ts, relaySession, tt.valid)
		})
	}
}

func setupForPaymentTest(t *testing.T) *testStruct {
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create mock spec, stake a client and a provider
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	ts.plan = common.CreateMockPlan()
	err := ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)
	require.Nil(t, err)

	err = ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)

	return ts
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
			ts := setupForPaymentTest(t) // reset the keepers state before each state

			err := ts.addClient(1)
			require.Nil(t, err)
			err = ts.addProvider(1)
			require.Nil(t, err)
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)
			relaySession.Epoch = sdk.UnwrapSDKContext(ts.ctx).BlockHeight() + tt.blockTime

			sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
			relaySession.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelaySession
			Relays = append(Relays, relaySession)

			payment := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}

			payAndVerifyBalance(t, ts, payment, true, tt.valid, ts.clients[0].Addr, ts.providers[0].Addr)
		})
	}
}

func TestRelayPaymentOverUse(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	maxcu := ts.plan.PlanPolicy.EpochCuLimit

	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), maxcu*2, ts.spec.Name, nil)
	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)
	// TODO: currently over use is returning an error and doesnt get to balance zero. we will fix it in the future so this can be uncommented.
	// balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.Error(t, err)
	// TODO: currently over use is returning an error and doesnt get to balance zero. we will fix it in the future so this can be uncommented.
	// balance = balance - ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
	// require.Zero(t, balance)
}

func setupClientsAndProvidersForUnresponsiveness(t *testing.T, amountOfClients int, amountOfProviders int) (ts *testStruct) {
	ts = &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// Create mock spec, stake a client and a provider
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	ts.plan = common.CreateMockPlan()
	ts.plan.PlanPolicy.MaxProvidersToPair = 6
	err := ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)
	require.Nil(t, err)

	err = ts.addClient(amountOfClients)
	require.Nil(t, err)
	err = ts.addProvider(amountOfProviders)
	require.Nil(t, err)
	return ts
}

// only one epoch is not enough for the unstaking to happen need atleast two epochs in the past
func TestRelayPaymentNotUnstakingProviderForUnresponsivenessIfNoEpochInformation(t *testing.T) {
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].Addr.String()})
	require.Nil(t, err)
	var Relays []*types.RelaySession
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData // create the complaint
		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relaySession)
	}
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.Nil(t, err)
	// testing that the provider wasnt unstaked.
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.providers[1].Addr)
	require.True(t, stakeStorageFound)
}

func TestRelayPaymentUnstakingProviderForUnresponsivenessWithBadDataInput(t *testing.T) {
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)
	for i := 0; i < 2; i++ { // move to epoch 3 so we can check enough epochs in the past
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test multiple bad data types
	unresponsiveProvidersData := make([]([]byte), 4)
	inputData := []interface{}{
		[]int{1, 2, 3, 4, 5},
		[]string{"bad", "data", "cosmosBadAddress"},
		"cosmosBadAddress",
		[]byte("cosmosBadAddress"),
	}
	// badData2, err := json.Marshal([]string{"bad", "data", "cosmosBadAddress"}) // test bad data
	var err error
	for i := 0; i < testClientAmount; i++ {
		badData, err := json.Marshal(inputData[i])
		require.Nil(t, err)
		unresponsiveProvidersData[i] = badData
	}
	require.Nil(t, err)
	var Relays []*types.RelaySession
	var totalCu uint64
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData[clientIndex]
		totalCu += relaySession.CuSum

		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relaySession)
	}

	balanceProviderBeforePayment := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.Nil(t, err)
	balanceProviderAfterPayment := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	reward := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(totalCu)).TruncateInt64()
	// testing reward + before == after
	require.Equal(t, balanceProviderAfterPayment, reward+balanceProviderBeforePayment)
}

// In this test we will test the protection from unstaking if the amount of previous serices*2 is greater than complaints
func TestRelayPaymentNotUnstakingProviderForUnresponsivenessBecauseOfServices(t *testing.T) {
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // after payment move one epoch to stake

	var RelaysForUnresponsiveProviderInFirstTwoEpochs []*types.RelaySession
	for i := 0; i < 2; i++ { // move to epoch 3 so we can check enough epochs in the past
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[1].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

		sig, err := sigs.SignRelay(ts.clients[i].SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(t, err)
		RelaysForUnresponsiveProviderInFirstTwoEpochs = []*types.RelaySession{relaySession} // each epoch get one service
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[1].Addr.String(), Relays: RelaysForUnresponsiveProviderInFirstTwoEpochs})
		require.Nil(t, err)
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // after payment move one epoch
	}

	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].Addr.String()})
	require.Nil(t, err)
	var Relays []*types.RelaySession
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)
		relaySession.UnresponsiveProviders = unresponsiveProvidersData

		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relaySession)
	}
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.Nil(t, err)

	// testing that the provider wasnt unstaked.
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.providers[1].Addr)
	require.True(t, stakeStorageFound)
}

func TestRelayPaymentDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)
	relayRequest2 := *relaySession
	Relays = append(Relays, &relayRequest2)

	payment := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}
	payAndVerifyBalance(t, ts, payment, true, false, ts.clients[0].Addr, ts.providers[0].Addr)
}

func TestRelayPaymentDataModification(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       int64
	}{
		{"ModifiedProvider", ts.clients[0].Addr.String(), ts.spec.Apis[0].ComputeUnits * 10, 1},
		{"ModifiedCU", ts.providers[0].Addr.String(), ts.spec.Apis[0].ComputeUnits * 9, 1},
		{"ModifiedID", ts.providers[0].Addr.String(), ts.spec.Apis[0].ComputeUnits * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession.Provider = tt.provider
			relaySession.CuSum = tt.cu
			relaySession.SessionId = uint64(tt.id)

			var Relays []*types.RelaySession
			Relays = append(Relays, relaySession)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})

			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentDelayedDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelaySession
	relay := *relaySession
	Relays = append(Relays, &relay)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.Nil(t, err)

	epochToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	tests := []struct {
		name    string
		advance uint64
	}{
		{"Epoch", 1},
		{"Memory-Epoch", epochToSave - 1},
		{"Memory", 1},       // epochToSave
		{"Memory+Epoch", 1}, // epochToSave + 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < int(tt.advance); i++ {
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			}

			var Relays []*types.RelaySession
			relay := *relaySession
			Relays = append(Relays, &relay)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentOldEpochs(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)
	blocksInEpoch, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	for i := 0; i < int(epochsToSave+1); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	tests := []struct {
		name  string
		sid   uint64
		epoch int64
		valid bool
	}{
		{"Epoch", 1, 1, true},                               // current -1*epoch
		{"Memory-Epoch", 2, int64(epochsToSave - 1), true},  // current -epoch to save + 1
		{"Memory", 3, int64(epochsToSave), true},            // current - epochToSave
		{"Memory+Epoch", 4, int64(epochsToSave + 1), false}, // current - epochToSave - 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cuSum := ts.spec.Apis[0].ComputeUnits * 10
			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, nil)
			relaySession.SessionId = tt.sid
			relaySession.Epoch = sdk.UnwrapSDKContext(ts.ctx).BlockHeight() - int64(blocksInEpoch)*tt.epoch

			sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
			relaySession.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelaySession
			Relays = append(Relays, relaySession)

			payment := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}

			payAndVerifyBalance(t, ts, payment, true, tt.valid, ts.clients[0].Addr, ts.providers[0].Addr)
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
			ts := setupForPaymentTest(t)

			ts.spec = common.CreateMockSpec()
			ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
			err := ts.addClient(1)
			require.Nil(t, err)
			err = ts.addProvider(1)
			require.Nil(t, err)
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			cuSum := ts.spec.Apis[0].ComputeUnits * 10
			QoS := &types.QualityOfServiceReport{Latency: tt.latency, Availability: tt.availability, Sync: tt.sync}

			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
			sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
			relaySession.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelaySession
			relay := *relaySession
			Relays = append(Relays, &relay)

			payment := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}

			payAndVerifyBalance(t, ts, payment, true, tt.valid, ts.clients[0].Addr, ts.providers[0].Addr)
		})
	}
}

// Helper function to perform payment and verify the balances (if valid, provider's balance should increase and consumer should decrease)
func payAndVerifyBalance(t *testing.T, ts *testStruct, relayPaymentMessage types.MsgRelayPayment, validConsumer bool, validPayment bool, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress) {
	_ctx := sdk.UnwrapSDKContext(ts.ctx)

	// Get provider's stake and consumer's project before payment
	balance := ts.keepers.BankKeeper.GetBalance(_ctx, providerAddress, epochstoragetypes.TokenDenom).Amount.Int64()
	proj, err := ts.keepers.Projects.GetProjectForDeveloper(_ctx, clientAddress.String(), uint64(_ctx.BlockHeight()))
	originalProjectUsedCu := proj.UsedCu
	if !validConsumer {
		require.NotNil(t, err)
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &relayPaymentMessage)
		require.NotNil(t, err)
		return
	}
	// valid consumer
	require.Nil(t, err)

	sub, found := ts.keepers.Subscription.GetSubscription(sdk.UnwrapSDKContext(ts.ctx), proj.Subscription)
	require.True(t, found)
	originalSubCuLeft := sub.MonthCuLeft

	// perform payment
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &relayPaymentMessage)
	if !validPayment {
		require.NotNil(t, err)
		return
	}
	// valid payment
	require.Nil(t, err)

	// calculate total used CU
	var totalCuUsed uint64
	var totalPaid uint64
	for _, relay := range relayPaymentMessage.Relays {
		cuUsed := relay.CuSum
		totalCuUsed += cuUsed
		if relay.QosReport != nil {
			score, err := relay.QosReport.ComputeQoS()
			require.Nil(t, err)

			cuUsed = (score.Mul(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx))).Add(sdk.OneDec().Sub(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx))))).MulInt64(int64(cuUsed)).TruncateInt().Uint64()
		}
		totalPaid += cuUsed
	}

	// verify provider's balance
	mint := ts.keepers.Pairing.MintCoinsPerCU(_ctx)
	want := mint.MulInt64(int64(totalPaid))
	require.Equal(t, balance+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(_ctx, providerAddress, epochstoragetypes.TokenDenom).Amount.Int64())

	// verify each project balance (project used cu should increase and its corresponding subscription cu left should decrease)
	proj, err = ts.keepers.Projects.GetProjectForDeveloper(_ctx, clientAddress.String(), uint64(_ctx.BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, originalProjectUsedCu+totalCuUsed, proj.UsedCu)

	sub, found = ts.keepers.Subscription.GetSubscription(_ctx, proj.Subscription)
	require.True(t, found)
	require.Equal(t, originalSubCuLeft-totalCuUsed, sub.MonthCuLeft)
}

func TestEpochPaymentDeletion(t *testing.T) {
	ts := setupForPaymentTest(t) // reset the keepers state before each state
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)

	payment := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}

	payAndVerifyBalance(t, ts, payment, true, true, ts.clients[0].Addr, ts.providers[0].Addr)

	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	for i := 0; i < int(epochsToSave+1); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	ans, err := ts.keepers.Pairing.EpochPaymentsAll(ts.ctx, &types.QueryAllEpochPaymentsRequest{})
	require.Nil(t, err)
	require.Equal(t, 0, len(ans.EpochPayments))
}

// Test that checks that after the consumer uses some CU it's updated in its project and subscription
func TestCuUsageInProjectsAndSubscription(t *testing.T) {
	ts := setupForPaymentTest(t)
	_ctx := sdk.UnwrapSDKContext(ts.ctx)
	subkeeper := ts.keepers.Subscription
	projKeeper := ts.keepers.Projects

	subscriptionOwner := ts.clients[0].Addr.String()
	proj1SK, projectAdmin1 := sigs.GenerateFloatingKey()
	_, projectAdmin2 := sigs.GenerateFloatingKey()

	err := subkeeper.CreateSubscription(_ctx, subscriptionOwner, subscriptionOwner, ts.plan.Index, 1)
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectData := projecttypes.ProjectData{
		Name:    "proj1",
		Enabled: true,
		ProjectKeys: []projecttypes.ProjectKey{
			projecttypes.ProjectDeveloperKey(projectAdmin1.String()),
		},
		Policy: &projecttypes.Policy{
			GeolocationProfile: uint64(1),
			MaxProvidersToPair: 3,
			TotalCuLimit:       1000,
			EpochCuLimit:       100,
		},
	}
	err = subkeeper.AddProjectToSubscription(_ctx, subscriptionOwner, projectData)
	require.Nil(t, err)

	projectData.Name = "proj2"
	projectData.ProjectKeys = []projecttypes.ProjectKey{
		projecttypes.ProjectDeveloperKey(projectAdmin2.String()),
	}
	err = subkeeper.AddProjectToSubscription(_ctx, subscriptionOwner, projectData)
	require.Nil(t, err)

	// create a signed relay request
	cuSum := uint64(1)
	QoS := types.QualityOfServiceReport{
		Latency:      sdk.OneDec(),
		Availability: sdk.OneDec(),
		Sync:         sdk.OneDec(),
	}
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, &QoS)

	relaySession.Sig, err = sigs.SignRelay(proj1SK, *relaySession)
	require.Nil(t, err)

	// Request payment (helper function validates the balances and verifies if we should get an error through valid)
	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)
	relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}
	payAndVerifyBalance(t, ts, relayPaymentMessage, true, true, projectAdmin1, ts.providers[0].Addr)

	sub, found := subkeeper.GetSubscription(_ctx, subscriptionOwner)
	require.True(t, found)

	proj1, err := projKeeper.GetProjectForDeveloper(_ctx, projectAdmin1.String(), uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, sub.MonthCuTotal-sub.MonthCuLeft, proj1.UsedCu)

	proj2, err := projKeeper.GetProjectForDeveloper(_ctx, projectAdmin2.String(), uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.NotEqual(t, sub.MonthCuTotal-sub.MonthCuLeft, proj2.UsedCu)
}

func TestBadgeValidation(t *testing.T) {
	ts := setupForPaymentTest(t)
	subkeeper := ts.keepers.Subscription

	// apply pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectDeveloper := *ts.clients[0]
	badgeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 100000)

	err := subkeeper.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String(), projectDeveloper.Addr.String(), ts.plan.Index, 1)
	require.Nil(t, err)

	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}

	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits, ts.spec.Name, QoS)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	// make the context's lava chain ID a non-empty string (as usually defined in most tests)
	lavaChainID := "lavanet"
	_ctx := sdk.UnwrapSDKContext(ts.ctx)
	_ctxBlockHeader := _ctx.BlockHeader()
	_ctxBlockHeader.ChainID = lavaChainID
	_ctx = _ctx.WithBlockHeader(_ctxBlockHeader)
	ts.ctx = sdk.WrapSDKContext(_ctx)

	badgeCuAllocation := ts.spec.Apis[0].ComputeUnits * 2 // times 2 to have enough CU for the two tests that succeed

	tests := []struct {
		name         string
		badgeAddress sdk.AccAddress // should be the badge user (in happy flow)
		relaySigner  common.Account // should be the badge user (in happy flow)
		badgeSigner  common.Account // should be the badge granter, i.e. project developer (in happy flow)
		epoch        uint64
		lavaChainID  string
		valid        bool
	}{
		{"happy flow", badgeUser.Addr, badgeUser, projectDeveloper, currentEpoch, lavaChainID, true},
		{"badge address != badge user", projectDeveloper.Addr, badgeUser, projectDeveloper, currentEpoch, lavaChainID, false},
		{"relay signer != badge user", badgeUser.Addr, projectDeveloper, projectDeveloper, currentEpoch, lavaChainID, false},
		{"badge signer != project developer", badgeUser.Addr, badgeUser, badgeUser, currentEpoch, lavaChainID, false},
		{"badge epoch != relay epoch", badgeUser.Addr, badgeUser, projectDeveloper, currentEpoch - 1, lavaChainID, false},
		{"badge lava chain id != relay lava chain id", badgeUser.Addr, badgeUser, projectDeveloper, currentEpoch, "dummy-lavanet", false},
		{"badge epoch != relay epoch (epoch passed)", badgeUser.Addr, badgeUser, projectDeveloper, currentEpoch, lavaChainID, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// this test was put to make sure that we only compare the relay's and badge's epoch fields (without any
			// relation to the advancement of blocks). So this test should have no errors
			if tt.name == "badge epoch != relay epoch (epoch passed)" {
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)

				// remove past payments to avoid double spending error (first test had a successful payment)
				err = ts.keepers.Pairing.RemoveAllEpochPaymentsForBlock(sdk.UnwrapSDKContext(ts.ctx), tt.epoch)
				require.Nil(t, err)
			}
			badge := types.CreateBadge(badgeCuAllocation, tt.epoch, tt.badgeAddress, tt.lavaChainID, []byte{})
			sig, err := sigs.SignBadge(tt.badgeSigner.SK, *badge)
			require.Nil(t, err)
			badge.ProjectSig = sig

			relaySession.Badge = badge
			relaySession.LavaChainId = tt.lavaChainID

			relaySession.Sig, err = sigs.SignRelay(tt.relaySigner.SK, *relaySession)
			require.Nil(t, err)

			var Relays []*types.RelaySession
			Relays = append(Relays, relaySession)
			relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}

			validConsumer := true
			if !tt.badgeSigner.Addr.Equals(projectDeveloper.Addr) {
				validConsumer = false
			}
			payAndVerifyBalance(t, ts, relayPaymentMessage, validConsumer, tt.valid, tt.badgeSigner.Addr, ts.providers[0].Addr)
		})
	}
}

func TestAddressEpochBadgeMap(t *testing.T) {
	ts := setupForPaymentTest(t)
	subkeeper := ts.keepers.Subscription

	// apply pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectDeveloper := *ts.clients[0]
	badgeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 100000)

	err := subkeeper.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String(), projectDeveloper.Addr.String(), ts.plan.Index, 1)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))

	badge := types.CreateBadge(10, currentEpoch, badgeUser.Addr, "", []byte{})
	sig, err := sigs.SignBadge(projectDeveloper.SK, *badge)
	require.Nil(t, err)
	badge.ProjectSig = sig

	// create 5 identical relays. Assign the badge only to the first one
	var relays []*types.RelaySession
	for i := 0; i < 5; i++ {
		QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), 1, ts.spec.Name, QoS)
		if i == 0 {
			relaySession.Badge = badge
		}

		// change session ID to avoid double spending
		relaySession.SessionId += uint64(i)
		relaySession.Sig, err = sigs.SignRelay(badgeUser.SK, *relaySession)
		require.Nil(t, err)

		relays = append(relays, relaySession)
	}
	relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: relays}
	payAndVerifyBalance(t, ts, relayPaymentMessage, true, true, projectDeveloper.Addr, ts.providers[0].Addr)
}

// Test:
// 1. basic badge CU enforcement works (can't ask more CU than badge's cuAllocation)
// 2. claiming payment for relays with different session IDs in different relay payment messages works
func TestBadgeCuAllocationEnforcement(t *testing.T) {
	ts := setupForPaymentTest(t)
	subkeeper := ts.keepers.Subscription

	// apply pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectDeveloper := *ts.clients[0]
	badgeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 100000)

	err := subkeeper.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String(), projectDeveloper.Addr.String(), ts.plan.Index, 1)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	plan, err := ts.keepers.Subscription.GetPlanFromSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String())
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, currentEpoch, badgeUser.Addr, "", []byte{})
	sig, err := sigs.SignBadge(projectDeveloper.SK, *badge)
	require.Nil(t, err)
	badge.ProjectSig = sig

	QoS := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}
	usedCuSoFar := uint64(0)

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
	for it, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), tt.cuSum, ts.spec.Name, QoS)
			relaySession.SessionId = uint64(it)
			relaySession.Sig, err = sigs.SignRelay(badgeUser.SK, *relaySession)
			require.Nil(t, err)

			relaySession.Badge = badge

			relays := []*types.RelaySession{relaySession}

			relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, true, tt.valid, projectDeveloper.Addr, ts.providers[0].Addr)

			if tt.valid {
				usedCuSoFar += tt.cuSum
			}
			badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, ts.providers[0].Addr.String())
			badgeUsedCuMapEntry, found := ts.keepers.Pairing.GetBadgeUsedCu(sdk.UnwrapSDKContext(ts.ctx), badgeUsedCuMapKey)
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
	ts := setupForPaymentTest(t)
	subkeeper := ts.keepers.Subscription

	// apply pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectDeveloper := *ts.clients[0]
	badgeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 100000)

	err := subkeeper.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String(), projectDeveloper.Addr.String(), ts.plan.Index, 1)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	plan, err := ts.keepers.Subscription.GetPlanFromSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String())
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit

	badge := types.CreateBadge(badgeCuAllocation, currentEpoch, badgeUser.Addr, "", []byte{})
	sig, err := sigs.SignBadge(projectDeveloper.SK, *badge)
	require.Nil(t, err)
	badge.ProjectSig = sig

	QoS := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}
	relayNum := 5
	cuSum := badgeCuAllocation / (uint64(relayNum) * 2)

	badgeUsedCuExpiry := ts.keepers.Pairing.BadgeUsedCuExpiry(sdk.UnwrapSDKContext(ts.ctx), *badge)

	tests := []struct {
		name            string
		blocksToAdvance uint64
		valid           bool
	}{
		{"valid backwards claim payment", badgeUsedCuExpiry - currentEpoch - 1, true},
		{"invalid backwards claim payment", 2, false}, // cuSum is valid (not passing badgeCuAllocation), but we've passed the backward payment period
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relays := []*types.RelaySession{}
			for i := 0; i < relayNum; i++ {
				relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
				relaySession.Epoch = int64(currentEpoch) // BuildRelayRequest always takes the current blockHeight, which is not desirable in this test
				relaySession.SessionId = uint64(i)
				relaySession.Sig, err = sigs.SignRelay(badgeUser.SK, *relaySession)
				require.Nil(t, err)

				if i == 0 {
					relaySession.Badge = badge
				}

				relays = append(relays, relaySession)
			}

			ts.ctx = testkeeper.AdvanceBlocks(ts.ctx, ts.keepers, int(tt.blocksToAdvance))

			relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: relays}
			payAndVerifyBalance(t, ts, relayPaymentMessage, true, tt.valid, projectDeveloper.Addr, ts.providers[0].Addr)

			// verify that the badgeUsedCu entry was deleted after it expired (and has the right value of used cu before expiring)
			badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, ts.providers[0].Addr.String())
			badgeUsedCuMapEntry, found := ts.keepers.Pairing.GetBadgeUsedCu(sdk.UnwrapSDKContext(ts.ctx), badgeUsedCuMapKey)
			currentBlock := sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
			if currentBlock > int64(badgeUsedCuExpiry) {
				require.False(t, found)
			} else {
				require.True(t, found)
				require.Equal(t, cuSum*uint64(relayNum), badgeUsedCuMapEntry.UsedCu)
			}
		})
	}
}

// Test that in the same epoch I can use the total CU allocation of the badge multiple times when talking to different providers
func TestBadgeDifferentProvidersCuAllocation(t *testing.T) {
	ts := setupForPaymentTest(t)
	err := ts.addProvider(1)
	require.Nil(t, err)

	subkeeper := ts.keepers.Subscription

	// apply pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectDeveloper := *ts.clients[0]
	badgeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 100000)

	err = subkeeper.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String(), projectDeveloper.Addr.String(), ts.plan.Index, 1)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	plan, err := ts.keepers.Subscription.GetPlanFromSubscription(sdk.UnwrapSDKContext(ts.ctx), projectDeveloper.Addr.String())
	require.Nil(t, err)
	badgeCuAllocation := plan.PlanPolicy.EpochCuLimit / 2

	badge := types.CreateBadge(badgeCuAllocation, currentEpoch, badgeUser.Addr, "", []byte{})
	sig, err := sigs.SignBadge(projectDeveloper.SK, *badge)
	require.Nil(t, err)
	badge.ProjectSig = sig

	QoS := &types.QualityOfServiceReport{
		Latency:      sdk.NewDecWithPrec(1, 0),
		Availability: sdk.NewDecWithPrec(1, 0),
		Sync:         sdk.NewDecWithPrec(1, 0),
	}

	cuSum := badgeCuAllocation

	for i := 0; i < 2; i++ {
		relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[i].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
		relaySession.Epoch = int64(currentEpoch) // BuildRelayRequest always takes the current blockHeight, which is not desirable in this test
		relaySession.Sig, err = sigs.SignRelay(badgeUser.SK, *relaySession)
		require.Nil(t, err)

		relaySession.Badge = badge

		relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[i].Addr.String(), Relays: []*types.RelaySession{relaySession}}
		payAndVerifyBalance(t, ts, relayPaymentMessage, true, true, projectDeveloper.Addr, ts.providers[i].Addr)

		badgeUsedCuMapKey := types.BadgeUsedCuKey(badge.ProjectSig, ts.providers[i].Addr.String())
		badgeUsedCuMapEntry, found := ts.keepers.Pairing.GetBadgeUsedCu(sdk.UnwrapSDKContext(ts.ctx), badgeUsedCuMapKey)
		require.True(t, found)
		require.Equal(t, cuSum, badgeUsedCuMapEntry.UsedCu)
	}
}
