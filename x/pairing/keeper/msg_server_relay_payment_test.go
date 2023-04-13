package keeper_test

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	projecttypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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

func createStubRequest(relaySession *types.RelaySession, dataReliability *types.VRFData) *types.RelayRequest {
	req := &types.RelayRequest{
		RelaySession:    relaySession,
		RelayData:       &types.RelayPrivateData{Data: []byte("stub-data")},
		DataReliability: dataReliability,
	}
	return req
}

func (ts *testStruct) addClient(amount int) error {
	for i := 0; i < amount; i++ {
		sk, address := sigs.GenerateFloatingKey()
		vrfSk, vrfPk, _ := utils.GeneratePrivateVRFKey()
		ts.clients = append(ts.clients, &common.Account{SK: sk, Addr: address, VrfSk: vrfSk, VrfPk: vrfPk})
		err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
		if err != nil {
			return err
		}
		vrfPkStr := &utils.VrfPubKey{}
		vrfPkStr.Unmarshal(vrfPk)
		_, err = ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPkStr.String()})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ts *testStruct) addProvider(amount int) error {
	for i := 0; i < amount; i++ {
		sk, address := sigs.GenerateFloatingKey()
		ts.providers = append(ts.providers, &common.Account{SK: sk, Addr: address})
		err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
		if err != nil {
			return err
		}
		endpoints := []epochstoragetypes.Endpoint{}
		endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
		_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
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
			payAndVerifyBalance(t, ts, relayPaymentMessage, tt.valid, ts.clients[0].Addr, ts.providers[0].Addr)

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
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)

	ts.plan = common.CreateMockPlan()
	err = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)
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
			ts.spec = common.CreateMockSpec()
			ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
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

			balanceProvider := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, found, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
			require.Equal(t, true, found)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})

			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

				require.Equal(t, balanceProvider+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))
				newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
				require.Nil(t, err)
				require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestRelayPaymentOverUse(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(1)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	epoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	entry, err := ts.keepers.Epochstorage.GetStakeEntryForClientEpoch(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[0].Addr, epoch)
	require.Nil(t, err)

	maxcu, err := ts.keepers.Pairing.GetAllowedCUForBlock(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()), entry)
	require.Nil(t, err)

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

	err := ts.addClient(amountOfClients)
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
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].Addr)
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
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].Addr)
	require.False(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].Addr)
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

	cuSum := ts.spec.GetApis()[0].ComputeUnits * 10
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.spec.Apis[0].ComputeUnits*10, ts.spec.Name, nil)

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relaySession)
	relaySession.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)
	relayRequest2 := *relaySession
	Relays = append(Relays, &relayRequest2)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
	require.NotNil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(cuSum))
	require.Equal(t, balance+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64())
	burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
	newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
	require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())
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

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
			if tt.valid {
				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(cuSum))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
				require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			} else {
				require.NotNil(t, err)
			}
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

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})
			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				score, err := QoS.ComputeQoS()
				require.Nil(t, err)

				want := mint.MulInt64(int64(cuSum))
				want = want.Mul(score.Mul(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx))).Add(sdk.OneDec().Sub(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx)))))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
				require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			} else {
				require.NotNil(t, err)
			}
		})
	}
}

// Data Reliability test for field corruption
func TestRelayPaymentDataReliability(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "Honest", valid: true},
		{name: "VrfValueNil", valid: false},
		{name: "VrfProofNil", valid: false},
		{name: "ProviderSigNil", valid: false},
		{name: "AllDataHashNil", valid: false},
		{name: "QueryHashNil", valid: false},
		{name: "SigNil", valid: false},
		{name: "VrfValueCorrupt", valid: false},
		{name: "VrfProofCorrupt", valid: false},
		{name: "ProviderSigCorrupt", valid: false},
		{name: "AllDataHashCorrupt", valid: false},
		{name: "QueryHashCorrupt", valid: false},
		{name: "SigCorrupt", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupForPaymentTest(t)

			ts.spec = common.CreateMockSpec()
			ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
			params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
			params.ServicersToPairCount = 100
			ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
			ts.keepers.Epochstorage.PushFixatedParams(sdk.UnwrapSDKContext(ts.ctx), 0, 0) // we need that in order for the param set to take effect
			err := ts.addClient(1)
			require.Nil(t, err)
			err = ts.addProvider(100)
			require.Nil(t, err)
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			cuSum := ts.spec.Apis[0].ComputeUnits * 10

			QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
			relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
			relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
			require.Nil(t, err)

			currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
			var index0 int64
			var providers []epochstoragetypes.StakeEntry
			var relayReply *types.RelayReply
			var nonce uint32
			// increasing the nonce changes the hash of the reply which in turn produces a different vrfRes resulting to a different index
			relayRequest := createStubRequest(relaySession, nil)
			for {
				relayReply = &types.RelayReply{
					Nonce: nonce,
				}

				relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].SK, relayReply, relayRequest)
				require.Nil(t, err)

				vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, currentEpoch)
				require.Nil(t, err)

				index0, err = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)
				require.Nil(t, err)

				providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relaySession.SpecId, ts.clients[0].Addr)
				require.Nil(t, err)

				if providers[index0].Address != ts.providers[0].Addr.String() {
					break
				} else {
					nonce += 1
				}
			}
			vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, false, currentEpoch)
			dataReliability0 := &types.VRFData{
				ChainId:        relayRequest.RelaySession.SpecId,
				Epoch:          relayRequest.RelaySession.Epoch,
				Differentiator: false,
				VrfValue:       vrf_res0,
				VrfProof:       vrf_proof0,
				ProviderSig:    relayReply.Sig,
				AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
				QueryHash:      utils.CalculateQueryHash(*relayRequest.RelayData),
				Sig:            nil,
			}
			dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].SK, dataReliability0)
			require.Nil(t, err)

			switch tt.name {
			case "VrfValueNil":
				dataReliability0.VrfValue = nil
			case "VrfProofNil":
				dataReliability0.VrfProof = nil
			case "ProviderSigNil":
				dataReliability0.ProviderSig = nil
			case "AllDataHashNil":
				dataReliability0.AllDataHash = nil
			case "QueryHashNil":
				dataReliability0.QueryHash = nil
			case "SigNil":
				dataReliability0.Sig = nil
			case "VrfValueCorrupt":
				dataReliability0.VrfValue = dataReliability0.VrfValue[0 : len(dataReliability0.VrfValue)-1]
			case "VrfProofCorrupt":
				dataReliability0.VrfProof = dataReliability0.VrfProof[0 : len(dataReliability0.VrfProof)-1]
			case "ProviderSigCorrupt":
				dataReliability0.ProviderSig = dataReliability0.ProviderSig[0 : len(dataReliability0.ProviderSig)-1]
			case "AllDataHashCorrupt":
				dataReliability0.AllDataHash = dataReliability0.AllDataHash[0 : len(dataReliability0.AllDataHash)-1]
			case "QueryHashCorrupt":
				dataReliability0.QueryHash = dataReliability0.QueryHash[0 : len(dataReliability0.QueryHash)-1]
			case "SigCorrupt":
				dataReliability0.Sig = dataReliability0.Sig[0 : len(dataReliability0.Sig)-1]
			}

			QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
			relayRequestWithDataReliability0 := common.BuildRelayRequest(ts.ctx, providers[index0].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoSDR)
			relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequestWithDataReliability0)
			require.Nil(t, err)
			provider := ts.getProvider(providers[index0].Address)
			relaysRequests := []*types.RelaySession{relayRequestWithDataReliability0}
			dataReliabilities := []*types.VRFData{dataReliability0}
			balanceBefore := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
			if tt.valid {
				require.Nil(t, err)

				balanceAfter := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.Addr, epochstoragetypes.TokenDenom).Amount.Int64()

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(cuSum))
				reward := want.MustFloat64() * (1 + params.DataReliabilityReward.MustFloat64())
				require.Equal(t, balanceBefore+int64(reward), balanceAfter)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

// client sends data reliability to a different provider collaborating to get more rewards
func TestRelayPaymentDataReliabilityWrongProvider(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
	params.ServicersToPairCount = 100
	ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
	ts.keepers.Epochstorage.PushFixatedParams(sdk.UnwrapSDKContext(ts.ctx), 0, 0) // we need that in order for the param set to take effect
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10

	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
	relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32

	wrongProviderIndex := 1
	relayRequest := createStubRequest(relaySession, nil)
GetWrongProvider:
	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].SK, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, currentEpoch)

		index0, _ = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)
		index1, _ := utils.GetIndexForVrf(vrfRes1, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relaySession.SpecId, ts.clients[0].Addr)
		require.Nil(t, err)
		// two providers returned by GetIndexForVrf and the provider getting tested need 1 more to perform this test properly
		require.Greater(t, len(providers), 3)
		nonce += 1
		// recalculate if for some reason vrf function returned the provider getting tested's index (should not happen but just to be safe)
		if providers[index0].Address == ts.providers[0].Addr.String() {
			continue
		} else if providers[index1].Address == ts.providers[0].Addr.String() {
			continue
		}
		// loop through all the providers and use the first one that is not getting asked to do vrf
		for i := 1; i < len(providers); i++ {
			if i == int(index0) || i == int(index1) {
				continue
			} else {
				wrongProviderIndex = i
				break GetWrongProvider
			}
		}

	}
	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		ChainId:        relayRequest.RelaySession.SpecId,
		Epoch:          relayRequest.RelaySession.Epoch,
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest.RelayData),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].SK, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := common.BuildRelayRequest(ts.ctx, providers[wrongProviderIndex].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoSDR)
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[wrongProviderIndex].Address)
	relaysRequests := []*types.RelaySession{relayRequestWithDataReliability0}
	dataReliabilities := []*types.VRFData{dataReliability0}
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
	require.NotNil(t, err)
}

// provider attempts to do a data reliability even though it is not triggered (below the threshold)
func TestRelayPaymentDataReliabilityBelowReliabilityThreshold(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.spec.ReliabilityThreshold = 0
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
	params.ServicersToPairCount = 5
	ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
	ts.keepers.Epochstorage.PushFixatedParams(sdk.UnwrapSDKContext(ts.ctx), 0, 0) // we need that in order for the param set to take effect
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(5)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
	relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
	require.Nil(t, err)

	relayRequest := createStubRequest(relaySession, nil)

	var relayReply *types.RelayReply
	var nonce uint32
	relayReply = &types.RelayReply{
		Nonce: nonce,
	}
	relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].SK, relayReply, relayRequest)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, currentEpoch)

	index0, _ := utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)
	index1, _ := utils.GetIndexForVrf(vrfRes1, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

	require.Equal(t, index0, int64(-1))
	require.Equal(t, index1, int64(-1))
	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		ChainId:        relayRequest.RelaySession.SpecId,
		Epoch:          relayRequest.RelaySession.Epoch,
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest.RelayData),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].SK, dataReliability0)
	require.Nil(t, err)

	// make all providers send a datareliability payment request. Everyone should fail
	for _, provider := range ts.providers {
		QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
		relayRequestWithDataReliability0 := common.BuildRelayRequest(ts.ctx, provider.Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoSDR)
		relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequestWithDataReliability0)
		require.Nil(t, err)

		relaysRequests := []*types.RelaySession{relayRequestWithDataReliability0}
		dataReliabilities := []*types.VRFData{dataReliability0}
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
		require.NotNil(t, err)
	}
}

// provider crafts datareliability with a client he has access to
func TestRelayPaymentDataReliabilityDifferentClientSign(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
	params.ServicersToPairCount = 100
	ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
	ts.keepers.Epochstorage.PushFixatedParams(sdk.UnwrapSDKContext(ts.ctx), 0, 0) // we need that in order for the param set to take effect
	err := ts.addClient(2)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)
	QoS.ComputeQoS()
	relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32
	relayRequest := createStubRequest(relaySession, nil)

	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].SK, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, currentEpoch)

		index0, _ = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relaySession.SpecId, ts.clients[0].Addr)
		require.Nil(t, err)

		if providers[index0].Address != ts.providers[0].Addr.String() {
			break
		}
		nonce += 1
	}

	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		ChainId:        relayRequest.RelaySession.SpecId,
		Epoch:          relayRequest.RelaySession.Epoch,
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest.RelayData),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[1].SK, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := common.BuildRelayRequest(ts.ctx, providers[index0].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoSDR)
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[1].SK, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[index0].Address)
	relaysRequests := []*types.RelaySession{relayRequestWithDataReliability0}
	dataReliabilities := []*types.VRFData{dataReliability0}
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
	require.NotNil(t, err)
}

// provider resends the same data reliability on the next epoch
func TestRelayPaymentDataReliabilityDoubleSpendDifferentEpoch(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
	params.ServicersToPairCount = 100
	ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
	ts.keepers.Epochstorage.PushFixatedParams(sdk.UnwrapSDKContext(ts.ctx), 0, 0) // we need that in order for the param set to take effect
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relaySession := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoS)

	relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32
	relayRequest := createStubRequest(relaySession, nil)
	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].SK, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, currentEpoch)

		index0, _ = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCountRaw(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relaySession.SpecId, ts.clients[0].Addr)
		require.Nil(t, err)

		if providers[index0].Address != ts.providers[0].Addr.String() {
			break
		} else {
			nonce += 1
		}
	}

	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest.RelayData, relayReply, ts.clients[0].VrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		ChainId:        relayRequest.RelaySession.SpecId,
		Epoch:          relayRequest.RelaySession.Epoch,
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest.RelayData),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].SK, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := common.BuildRelayRequest(ts.ctx, providers[index0].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, QoSDR)
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[index0].Address)
	relaysRequests := []*types.RelaySession{relayRequestWithDataReliability0}
	dataReliabilities := []*types.VRFData{dataReliability0}
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
	require.Nil(t, err)

	// Advance Epoch and set block height and resign the tx
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	relayRequestWithDataReliability0.Epoch = sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	relayRequestWithDataReliability0.SessionId = uint64(2)
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.Addr.String(), Relays: relaysRequests, VRFs: dataReliabilities})
	require.NotNil(t, err)
}

// Helper function to perform payment and verify the balances (if valid, provider's balance should increase and consumer should decrease)
func payAndVerifyBalance(t *testing.T, ts *testStruct, relayPaymentMessage types.MsgRelayPayment, valid bool, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress) {
	// Get provider's and consumer's before payment
	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, clientAddress)

	// perform payment
	_, err := ts.servers.PairingServer.RelayPayment(ts.ctx, &relayPaymentMessage)
	if valid {
		require.Nil(t, err)
		// payment is valid, provider's balance should increase
		mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
		want := mint.MulInt64(int64(relayPaymentMessage.GetRelays()[0].CuSum)) // The compensation for a single query
		require.Equal(t, balance+want.TruncateInt64(),
			ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), providerAddress, epochstoragetypes.TokenDenom).Amount.Int64())

		// payment is valid, consumer's balance should decrease
		burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(relayPaymentMessage.GetRelays()[0].CuSum))
		newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, clientAddress)
		require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

	} else {
		// payment is not valid, should result in an error
		require.NotNil(t, err)
	}
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

	balanceProvider := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, found, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
	require.Equal(t, true, found)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays})

	require.Nil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

	require.Equal(t, balanceProvider+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].Addr, epochstoragetypes.TokenDenom).Amount.Int64())

	burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))
	newStakeClient, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].Addr)
	require.Nil(t, err)
	require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

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

	err := ts.addClient(1)
	require.Nil(t, err)

	subscriptionOwner := ts.providers[0].Addr.String()
	projectAdmin1 := ts.clients[0].Addr.String()
	projectAdmin2 := ts.clients[1].Addr.String()

	err = subkeeper.CreateSubscription(_ctx, subscriptionOwner, subscriptionOwner, ts.plan.Index, 1, "")
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	projectData := projecttypes.ProjectData{
		Name:        "proj1",
		Description: "description",
		Enabled:     true,
		ProjectKeys: []projecttypes.ProjectKey{
			{
				Key:   projectAdmin1,
				Types: []projecttypes.ProjectKey_KEY_TYPE{projecttypes.ProjectKey_DEVELOPER},
			}},
		Policy: nil,
	}
	err = subkeeper.AddProjectToSubscription(_ctx, subscriptionOwner, projectData)
	require.Nil(t, err)

	projectData.Name = "proj2"
	projectData.ProjectKeys = []projecttypes.ProjectKey{{
		Key:   projectAdmin2,
		Types: []projecttypes.ProjectKey_KEY_TYPE{projecttypes.ProjectKey_DEVELOPER},
	}}
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

	relaySession.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relaySession)
	require.Nil(t, err)

	// Request payment (helper function validates the balances and verifies if we should get an error through valid)
	var Relays []*types.RelaySession
	Relays = append(Relays, relaySession)
	relayPaymentMessage := types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: Relays}
	payAndVerifyBalance(t, ts, relayPaymentMessage, true, ts.clients[0].Addr, ts.providers[0].Addr)

	sub, found := subkeeper.GetSubscription(_ctx, subscriptionOwner)
	require.True(t, found)

	proj1, _, err := projKeeper.GetProjectForDeveloper(_ctx, projectAdmin1, uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, sub.MonthCuTotal-sub.MonthCuLeft, proj1.UsedCu)

	proj2, _, err := projKeeper.GetProjectForDeveloper(_ctx, projectAdmin2, uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.NotEqual(t, sub.MonthCuTotal-sub.MonthCuLeft, proj2.UsedCu)
}
