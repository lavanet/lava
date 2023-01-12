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

func TestRelayPaymentUnstakingProviderForUnresponsiveness(t *testing.T) {
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)

	for i := 0; i < 2; i++ { // move to epoch 3 so we can check enough epochs in the past
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
	staked_amount, _, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	balanceProvideratBeforeStake := staked_amount.Stake.Amount.Int64() + ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].address, epochstoragetypes.TokenDenom).Amount.Int64()

	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].address.String()})
	require.Nil(t, err)
	var Relays []*types.RelayRequest
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints
		relayRequest := &types.RelayRequest{
			Provider:              ts.providers[0].address.String(),
			ApiUrl:                "",
			Data:                  []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(1),
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
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.Nil(t, err)
	// testing that the provider was unstaked. and checking his balance after many epochs
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

	OriginalBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(sdk.UnwrapSDKContext(ts.ctx), OriginalBlockHeight)
	require.Nil(t, err)
	for { // move to epoch 13 so we can check balance at the end
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
		blockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
		if blockHeight > blocksToSave+OriginalBlockHeight {
			break
		}
	}
	// validate that the provider is no longer unstaked. and stake was returned.
	_, unStakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.False(t, unStakeStoragefound)
	// also that the provider wasnt returned to stake pool
	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

	balanceProviderAfterUnstakeMoneyReturned := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].address, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Equal(t, balanceProvideratBeforeStake, balanceProviderAfterUnstakeMoneyReturned)
}

func TestRelayPaymentUnstakingProviderForUnresponsivenessContinueComplainingAfterUnstake(t *testing.T) {
	testClientAmount := 4
	testProviderAmount := 2
	ts := setupClientsAndProvidersForUnresponsiveness(t, testClientAmount, testProviderAmount)
	for i := 0; i < 2; i++ { // move to epoch 3 so we can check enough epochs in the past
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	unresponsiveProvidersData, err := json.Marshal([]string{ts.providers[1].address.String()})
	require.Nil(t, err)
	var Relays []*types.RelayRequest
	for clientIndex := 0; clientIndex < testClientAmount; clientIndex++ { // testing testClientAmount of complaints

		relayRequest := &types.RelayRequest{
			Provider:              ts.providers[0].address.String(),
			ApiUrl:                "",
			Data:                  []byte(ts.spec.Apis[0].Name),
			SessionId:             uint64(1),
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
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.Nil(t, err)
	// testing that the provider wasnt unstaked.
	_, unStakeStoragefound, _ := ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unStakeStoragefound)
	_, stakeStorageFound, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)

	// continue reporting provider after unstake
	for i := 0; i < 2; i++ { // move to epoch 5
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

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
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: RelaysAfter})
	require.Nil(t, err)

	_, stakeStorageFound, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.spec.Name, ts.providers[1].address)
	require.False(t, stakeStorageFound)
	_, unStakeStoragefound, _ = ts.keepers.Epochstorage.UnstakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey, ts.providers[1].address)
	require.True(t, unStakeStoragefound)

	// validating number of appearances for unstaked provider in unstake storage (if more than once found, throw an error)
	storage, foundStorage := ts.keepers.Epochstorage.GetStakeStorageUnstake(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ProviderKey)
	require.True(t, foundStorage)
	var numberOfAppearances int
	for _, stored := range storage.StakeEntries {
		if stored.Address == ts.providers[1].address.String() {
			numberOfAppearances += 1
		}
	}
	require.Equal(t, numberOfAppearances, 1)
}
