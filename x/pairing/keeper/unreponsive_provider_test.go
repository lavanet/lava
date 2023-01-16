package keeper_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test to measure the time the check for unresponsiveness every epoch start takes
func TestRelayPaymentUnstakingProviderForUnresponsiveness(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < testkeeper.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+testkeeper.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS+ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx)); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// get provider1's balance before the stake
	staked_amount, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	balanceProvideratBeforeStake := staked_amount.Stake.Amount.Int64() + ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].address, epochstoragetypes.TokenDenom).Amount.Int64()

	// create unresponsive data that includes provider1 being unresponsive
	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].address.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	var Relays []*types.RelayRequest
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		relayRequest := &types.RelayRequest{
			Provider:              ts.providers[0].address.String(),
			ApiUrl:                "",
			Data:                  []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(0),
			ChainID:               ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits*10 + uint64(clientIndex),
			BlockHeight:           sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			RelayNum:              0,
			RequestBlock:          -1,
			DataReliability:       nil,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}

		sig, err := sigs.SignRelay(ts.clients[clientIndex].secretKey, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relayRequest)
	}

	// send the relay requests (provider gets payment)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.Nil(t, err)

	// advance enough epochs so the unresponsive provider will be punished (the check happens every epoch start, the complaints will be accounted for after EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+RecommendedEpochNumToCollectPayment+1 epochs from the payment epoch)
	for i := uint64(0); i < testkeeper.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test the provider has been unstaked
	_, unstakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unstakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

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
	_, unstakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.False(t, unstakeStoragefound)

	// also validate that the provider hasn't returned to the stake pool
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

	// validate that the provider's balance after the unstake is the same as before he staked
	balanceProviderAfterUnstakeMoneyReturned := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].address, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Equal(t, balanceProvideratBeforeStake, balanceProviderAfterUnstakeMoneyReturned)
}

func TestRelayPaymentUnstakingProviderForUnresponsivenessContinueComplainingAfterUnstake(t *testing.T) {
	// setup test for unresponsiveness
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	// advance enough epochs so we can check punishment due to unresponsiveness (if the epoch is too early, there's no punishment)
	for i := uint64(0); i < testkeeper.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+testkeeper.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS+ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx)); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// create unresponsive data that includes provider1 being unresponsive
	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].address.String()})
	require.Nil(t, err)

	// create relay requests for provider0 that contain complaints about provider1
	var Relays []*types.RelayRequest
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints

		relayRequest := &types.RelayRequest{
			Provider:              ts.providers[0].address.String(),
			ApiUrl:                "",
			Data:                  []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(0),
			ChainID:               ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits * 10,
			BlockHeight:           sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			RelayNum:              0,
			RequestBlock:          -1,
			DataReliability:       nil,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}

		sig, err := sigs.SignRelay(ts.clients[clientIndex].secretKey, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		Relays = append(Relays, relayRequest)
	}

	// send the relay requests (provider gets payment)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.Nil(t, err)

	// advance enough epochs so the unresponsive provider will be punished (the check happens every epoch start, the complaints will be accounted for after EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+RecommendedEpochNumToCollectPayment+1 epochs from the payment epoch)
	for i := uint64(0); i < testkeeper.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER+ts.keepers.Pairing.RecommendedEpochNumToCollectPayment(sdk.UnwrapSDKContext(ts.ctx))+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// test the provider has been unstaked
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

	// advance some more epochs
	for i := 0; i < 2; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// create more relay requests for provider0 that contain complaints about provider1 (note, sessionID changed)
	var RelaysAfter []*types.RelayRequest
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints

		relayRequest := &types.RelayRequest{
			Provider:              ts.providers[0].address.String(),
			ApiUrl:                "",
			Data:                  []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(2),
			ChainID:               ts.spec.Name,
			CuSum:                 ts.spec.Apis[0].ComputeUnits * 10,
			BlockHeight:           sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			RelayNum:              0,
			RequestBlock:          -1,
			DataReliability:       nil,
			UnresponsiveProviders: unresponsiveProvidersData, // create the complaint
		}
		sig, err := sigs.SignRelay(ts.clients[clientIndex].secretKey, *relayRequest)
		relayRequest.Sig = sig
		require.Nil(t, err)
		RelaysAfter = append(RelaysAfter, relayRequest)
	}

	// send the relay requests (provider gets payment)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: RelaysAfter})
	require.Nil(t, err)

	// test the provider is still unstaked
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)
	_, unStakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unStakeStoragefound)

	// get the current unstake storage
	storage, foundStorage := ts.keepers.Epochstorage.GetStakeStorageUnstake(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey)
	require.True(t, foundStorage)

	// validate the punished provider is not shown twice (or more) in the unstake storage
	var numberOfAppearances int
	for _, stored := range storage.StakeEntries {
		if stored.Address == ts.providers[1].address.String() {
			numberOfAppearances += 1
		}
	}
	require.Equal(t, numberOfAppearances, 1)
}
