package keeper_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestUnresponsivenessStressTest(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 50
	testProviderAmount := 5
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	// get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))

	// check which const is larger
	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < largerConst+recommendedEpochNumToCollectPayment; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// create unresponsive data list of the first 100 providers being unresponsive
	var unresponsiveDataList [][]byte
	unresponsiveProviderAmount := 1
	for i := 0; i < unresponsiveProviderAmount; i++ {
		unresponsiveData, err := json.Marshal([]string{ts.providers[i].Addr.String()})
		require.Nil(t, err)
		unresponsiveDataList = append(unresponsiveDataList, unresponsiveData)
	}

	// create relay requests for that contain complaints about providers with indices 0-100
	relayEpoch := sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		var Relays []*types.RelaySession

		// Get pairing for the client to pick a valid provider
		providersStakeEntries, err := ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[clientIndex].Addr)
		require.Nil(t, err)
		providerIndex := rand.Intn(len(providersStakeEntries))
		providerAddress := providersStakeEntries[providerIndex].Address
		providerSdkAddress, err := sdk.AccAddressFromBech32(providerAddress)
		require.Nil(t, err)

		// create relay request
		relayRequest := &types.RelaySession{
			Provider:              providerAddress,
			ContentHash:           []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(0),
			SpecId:                ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits*10 + uint64(clientIndex),
			Epoch:                 relayEpoch,
			RelayNum:              0,
			UnresponsiveProviders: unresponsiveDataList[clientIndex%unresponsiveProviderAmount], // create the complaint
		}

		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relayRequest)

		// send relay payment and check the funds did transfer normally
		payAndVerifyBalance(t, ts, types.MsgRelayPayment{Creator: providerAddress, Relays: Relays}, true, true, ts.clients[clientIndex].Addr, providerSdkAddress)
	}

	// advance enough epochs so the unresponsive providers will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}
	for i := uint64(0); i < largerConst; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// go over unresponsive providers
	for i := 0; i < unresponsiveProviderAmount; i++ {
		// test the providers has been unstaked
		_, unstakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), ts.providers[i].Addr)
		require.True(t, unstakeStoragefound)
		_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.providers[i].Addr)
		require.False(t, stakeStorageFound)

		// validate the complainers CU field in the unresponsive provider's providerPaymentStorage has been reset after being punished (note we use the epoch from the relay because that is when it got reported)
		providerPaymentStorageKey := ts.keepers.Pairing.GetProviderPaymentStorageKey(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, uint64(relayEpoch), ts.providers[i].Addr)
		providerPaymentStorage, found := ts.keepers.Pairing.GetProviderPaymentStorage(sdk.UnwrapSDKContext(ts.ctx), providerPaymentStorageKey)
		require.Equal(t, true, found)
		require.Equal(t, uint64(0), providerPaymentStorage.GetComplainersTotalCu())
	}

	// go over responsive providers - make sure they are still staked
	for i := unresponsiveProviderAmount; i < testProviderAmount; i++ {
		// test the providers hasn't been unstaked
		_, unstakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), ts.providers[i].Addr)
		require.False(t, unstakeStoragefound)
		_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.providers[i].Addr)
		require.True(t, stakeStorageFound)
	}
}

// Test to measure the time the check for unresponsiveness every epoch start takes
func TestUnstakingProviderForUnresponsiveness(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 1
	testProviderAmount := 10
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	// get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))

	// check which const is larger
	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < largerConst+recommendedEpochNumToCollectPayment; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// find two providers in the pairing
	pairingProviders, err := ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[0].Addr)
	require.NoError(t, err)
	provider0_addr := sdk.MustAccAddressFromBech32(pairingProviders[0].Address)
	provider1_addr := sdk.MustAccAddressFromBech32(pairingProviders[1].Address)

	// get provider1's balance before the stake
	staked_amount, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	balanceProvideratBeforeStake := staked_amount.Stake.Amount.Int64() + ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider1_addr, epochstoragetypes.TokenDenom).Amount.Int64()

	// create unresponsive data that includes provider1 being unresponsive
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		var Relays []*types.RelaySession
		relayRequest := &types.RelaySession{
			Provider:              provider0_addr.String(),
			ContentHash:           []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(0),
			SpecId:                ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits*10 + uint64(clientIndex),
			Epoch:                 relayEpoch,
			RelayNum:              0,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}

		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relayRequest)

		// send relay payment and check the funds did transfer normally
		payAndVerifyBalance(t, ts, types.MsgRelayPayment{Creator: provider0_addr.String(), Relays: Relays}, true, true, ts.clients[clientIndex].Addr, provider0_addr)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}
	for i := uint64(0); i < largerConst; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test the unresponsive provider1 has been unstaked
	_, unstakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider1_addr)
	require.True(t, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)

	// validate the complainers CU field in provider1's providerPaymentStorage has been reset after being punished (note we use the epoch from the relay because that is when it got reported)
	providerPaymentStorageKey := ts.keepers.Pairing.GetProviderPaymentStorageKey(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, uint64(relayEpoch), provider1_addr)
	providerPaymentStorage, found := ts.keepers.Pairing.GetProviderPaymentStorage(sdk.UnwrapSDKContext(ts.ctx), providerPaymentStorageKey)
	require.Equal(t, true, found)
	require.Equal(t, uint64(0), providerPaymentStorage.GetComplainersTotalCu())

	// test the responsive provider0 hasn't been unstaked
	_, unstakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider0_addr)
	require.False(t, unstakeStoragefound)
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider0_addr)
	require.True(t, stakeStorageFound)

	// advance enough epochs so the current block will be deleted (advance more than the chain's memory - blocksToSave)
	OriginalBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(sdk.UnwrapSDKContext(ts.ctx), OriginalBlockHeight)
	require.Nil(t, err)
	for {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		blockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
		if blockHeight > blocksToSave+OriginalBlockHeight {
			break
		}
	}

	// validate that the provider is no longer unstaked
	_, unstakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider1_addr)
	require.False(t, unstakeStoragefound)

	// also validate that the provider hasn't returned to the stake pool
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)

	// validate that the provider's balance after the unstake is the same as before he staked
	balanceProviderAfterUnstakeMoneyReturned := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider1_addr, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Equal(t, balanceProvideratBeforeStake, balanceProviderAfterUnstakeMoneyReturned)
}

func TestUnstakingProviderForUnresponsivenessContinueComplainingAfterUnstake(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 1
	testProviderAmount := 3
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	// get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))

	// check which const is larger
	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < largerConst+ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx)); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// find two providers in the pairing
	pairingProviders, err := ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[0].Addr)
	require.NoError(t, err)
	provider0_addr := sdk.MustAccAddressFromBech32(pairingProviders[0].Address)
	provider1_addr := sdk.MustAccAddressFromBech32(pairingProviders[1].Address)
	// create unresponsive data that includes provider1 being unresponsive
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := sdk.UnwrapSDKContext(ts.ctx).BlockHeight()

	var Relays []*types.RelaySession
	relayRequest := &types.RelaySession{
		Provider:              provider0_addr.String(),
		ContentHash:           []byte(ts.spec.Apis[0].Name),
		SessionId:             uint64(0),
		SpecId:                ts.spec.Name,
		CuSum:                 ts.spec.Apis[0].ComputeUnits * 10,
		Epoch:                 relayEpoch,
		RelayNum:              0,
		UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
	}

	sig, err := sigs.SignRelay(ts.clients[0].SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)
	Relays = append(Relays, relayRequest)

	// send relay payment and check the funds did transfer normally
	payAndVerifyBalance(t, ts, types.MsgRelayPayment{Creator: provider0_addr.String(), Relays: Relays}, true, true, ts.clients[0].Addr, provider0_addr)

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}
	for i := uint64(0); i < largerConst; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test the provider has been unstaked
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider1_addr)
	require.True(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)

	// validate the complainers CU field in provider1's providerPaymentStorage has been reset after being punished (note we use the epoch from the relay because that is when it got reported)
	providerPaymentStorageKey := ts.keepers.Pairing.GetProviderPaymentStorageKey(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, uint64(relayEpoch), provider1_addr)
	providerPaymentStorage, found := ts.keepers.Pairing.GetProviderPaymentStorage(sdk.UnwrapSDKContext(ts.ctx), providerPaymentStorageKey)
	require.Equal(t, true, found)
	require.Equal(t, uint64(0), providerPaymentStorage.GetComplainersTotalCu())

	// advance some more epochs
	for i := 0; i < 2; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// create more relay requests for provider0 that contain complaints about provider1 (note, sessionID changed)
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		var RelaysAfter []*types.RelaySession
		relayRequest := &types.RelaySession{
			Provider:              provider0_addr.String(),
			ContentHash:           []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(2),
			SpecId:                ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits * 10,
			Epoch:                 sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			RelayNum:              0,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}
		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		RelaysAfter = append(RelaysAfter, relayRequest)

		// send relay payment and check the funds did transfer normally
		payAndVerifyBalance(t, ts, types.MsgRelayPayment{Creator: provider0_addr.String(), Relays: RelaysAfter}, true, true, ts.clients[clientIndex].Addr, provider0_addr)
	}

	// test the provider is still unstaked
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	require.False(t, stakeStorageFound)
	_, unStakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider1_addr)
	require.True(t, unStakeStoragefound)

	// get the current unstake storage
	storage, foundStorage := ts.keepers.Epochstorage.GetStakeStorageUnstake(sdk.UnwrapSDKContext(ts.ctx))
	require.True(t, foundStorage)

	// validate the punished provider is not shown twice (or more) in the unstake storage
	var numberOfAppearances int
	for _, stored := range storage.StakeEntries {
		if stored.Address == provider1_addr.String() {
			numberOfAppearances += 1
		}
	}
	require.Equal(t, numberOfAppearances, 1)
}

func TestNotUnstakingProviderForUnresponsivenessWithMinProviders(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 1
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)
	err := ts.addProviderGeolocation(2, 2)
	require.Nil(t, err)

	// get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))

	// check which const is larger
	largerConst := types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER
	if largerConst < types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS {
		largerConst = types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS
	}

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < largerConst+recommendedEpochNumToCollectPayment; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// find two providers in the pairing
	pairingProviders, err := ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[0].Addr)
	require.NoError(t, err)
	provider0_addr := sdk.MustAccAddressFromBech32(pairingProviders[0].Address)
	provider1_addr := sdk.MustAccAddressFromBech32(pairingProviders[1].Address)

	// create unresponsive data that includes provider1 being unresponsive
	unresponsiveProvidersData, err := json.Marshal([]string{provider1_addr.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	relayEpoch := sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		var Relays []*types.RelaySession
		relayRequest := &types.RelaySession{
			Provider:              provider0_addr.String(),
			ContentHash:           []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(0),
			SpecId:                ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits*10 + uint64(clientIndex),
			Epoch:                 relayEpoch,
			RelayNum:              0,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}

		sig, err := sigs.SignRelay(ts.clients[clientIndex].SK, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relayRequest)

		// send relay payment and check the funds did transfer normally
		payAndVerifyBalance(t, ts, types.MsgRelayPayment{Creator: provider0_addr.String(), Relays: Relays}, true, true, ts.clients[clientIndex].Addr, provider0_addr)
	}

	// advance enough epochs so the unresponsive provider will be punished
	if largerConst < recommendedEpochNumToCollectPayment {
		largerConst = recommendedEpochNumToCollectPayment
	}
	for i := uint64(0); i < largerConst; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test the unresponsive provider1 has been unstaked
	_, unstakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), provider1_addr)
	require.False(t, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, provider1_addr)
	require.True(t, stakeStorageFound)
}
