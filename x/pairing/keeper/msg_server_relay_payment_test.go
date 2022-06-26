package keeper_test

import (
	"context"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	utils "github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	ctx        context.Context
	keepers    *testkeeper.Keepers
	servers    *testkeeper.Servers
	proSK      *btcSecp256k1.PrivateKey
	proAddr    sdk.AccAddress
	clientSK   *btcSecp256k1.PrivateKey
	clientAddr sdk.AccAddress
	spec       spectypes.Spec
}

func setupForPaymentTest(t *testing.T) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	//init keepers state
	ts.proSK, ts.proAddr = sigs.GenerateFloatingKey()
	ts.clientSK, ts.clientAddr = sigs.GenerateFloatingKey()

	var balance int64 = 100000
	ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.clientAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)
	_, err := ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: ts.clientAddr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)

	endpoints := []epochtypes.Endpoint{}
	endpoints = append(endpoints, epochtypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
	_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: ts.proAddr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
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

			ts := setupForPaymentTest(t) //reset the keepers state before each state

			relayRequest := &types.RelayRequest{
				Provider:        ts.proAddr.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(1),
				ChainID:         ts.spec.Name,
				CuSum:           ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight() + tt.blockTime,
				RelayNum:        0,
				RequestBlock:    -1,
				DataReliability: nil,
			}

			sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balanceProvider := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, found, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)
			require.Equal(t, true, found)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})

			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

				require.Equal(t, balanceProvider+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)
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

	epoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	entry, err := ts.keepers.Epochstorage.GetStakeEntryForClientEpoch(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clientAddr, epoch)
	require.Nil(t, err)

	maxcu, err := ts.keepers.Pairing.GetAllowedCU(sdk.UnwrapSDKContext(ts.ctx), entry)
	require.Nil(t, err)

	relayRequest := &types.RelayRequest{
		Provider:        ts.proAddr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           maxcu * 2,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64()

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})
	require.Nil(t, err)
	balance = balance - ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Zero(t, balance)
}

func TestRelayPaymentDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	cuSum := ts.spec.GetApis()[0].ComputeUnits * 10
	relayRequest := &types.RelayRequest{
		Provider:        ts.proAddr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)
	relayRequest2 := *relayRequest
	Relays = append(Relays, &relayRequest2)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})
	require.NotNil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(cuSum))
	require.Equal(t, balance+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64())

	burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
	newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)
	require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

}

func TestRelayPaymentDataModification(t *testing.T) {
	ts := setupForPaymentTest(t)

	relayRequest := &types.RelayRequest{
		Provider:        ts.proAddr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           ts.spec.Apis[0].ComputeUnits * 10,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       int64
	}{
		{"ModifiedProvider", ts.clientAddr.String(), ts.spec.Apis[0].ComputeUnits * 10, 1},
		{"ModifiedCU", ts.proAddr.String(), ts.spec.Apis[0].ComputeUnits * 9, 1},
		{"ModifiedID", ts.proAddr.String(), ts.spec.Apis[0].ComputeUnits * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			relayRequest.Provider = tt.provider
			relayRequest.CuSum = tt.cu
			relayRequest.SessionId = uint64(tt.id)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})

			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentDelayedDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	relayRequest := &types.RelayRequest{
		Provider:        ts.proAddr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           ts.spec.Apis[0].ComputeUnits * 10,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	relay := *relayRequest
	Relays = append(Relays, &relay)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})
	require.Nil(t, err)

	epochToSave := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx))

	tests := []struct {
		name    string
		advance uint64
	}{
		{"Epoch", 1},
		{"Memory-Epoch", epochToSave - 1},
		{"Memory", 1},       //epochToSave
		{"Memory+Epoch", 1}, //epochToSave + 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for i := 0; i < int(tt.advance); i++ {
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			}

			var Relays []*types.RelayRequest
			relay := *relayRequest
			Relays = append(Relays, &relay)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})
			require.NotNil(t, err)

		})
	}
}

func TestRelayPaymentOldEpochs(t *testing.T) {
	ts := setupForPaymentTest(t)

	epochsToSave := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx))
	blocksInEpoch := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx))
	for i := 0; i < int(epochsToSave+1); i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	tests := []struct {
		name  string
		sid   uint64
		epoch int64
		valid bool
	}{
		{"Epoch", 1, 1, true},                               //current -1*epoch
		{"Memory-Epoch", 2, int64(epochsToSave - 1), true},  //current -epoch to save + 1
		{"Memory", 3, int64(epochsToSave), true},            //current - epochToSave
		{"Memory+Epoch", 4, int64(epochsToSave + 1), false}, //current - epochToSave - 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cuSum := ts.spec.Apis[0].ComputeUnits * 10
			relayRequest := &types.RelayRequest{
				Provider:        ts.proAddr.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       tt.sid,
				ChainID:         ts.spec.Name,
				CuSum:           cuSum,
				BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight() - int64(blocksInEpoch)*tt.epoch,
				RelayNum:        0,
				RequestBlock:    -1,
				DataReliability: nil,
			}

			sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})
			if tt.valid {
				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(cuSum))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAddr)
				require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			} else {
				require.NotNil(t, err)
			}

		})
	}
}
