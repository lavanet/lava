package keeper_test

import (
	"context"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/coniks-sys/coniks-go/crypto/vrf"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	utils "github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

var (
	balance int64 = 100000000
	stake   int64 = 100000
)

type account struct {
	secretKey *btcSecp256k1.PrivateKey
	address   sdk.AccAddress
	vrfSk     vrf.PrivateKey
	vrfPk     vrf.PublicKey
}

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	providers []*account
	clients   []*account
	spec      spectypes.Spec
}

func (ts *testStruct) addClient(amount int) error {
	for i := 0; i < amount; i++ {
		sk, address := sigs.GenerateFloatingKey()
		vrfSk, vrfPk, _ := utils.GeneratePrivateVRFKey()
		ts.clients = append(ts.clients, &account{secretKey: sk, address: address, vrfSk: vrfSk, vrfPk: vrfPk})
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
		ts.providers = append(ts.providers, &account{secretKey: sk, address: address})
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

func (ts *testStruct) getProvider(addr string) *account {
	for _, provider := range ts.providers {
		if provider.address.String() == addr {
			return provider
		}
	}
	return nil
}

func setupForPaymentTest(t *testing.T) *testStruct {
	ts := &testStruct{
		providers: make([]*account, 0),
		clients:   make([]*account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochstoragetypes.DefaultGenesis().EpochDetails)

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
			ts.spec = common.CreateMockSpec()
			ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
			err := ts.addClient(1)
			require.Nil(t, err)
			err = ts.addProvider(1)
			require.Nil(t, err)
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			relayRequest := &types.RelayRequest{
				Provider:        ts.providers[0].address.String(),
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

			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balanceProvider := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, found, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
			require.Equal(t, true, found)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})

			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

				require.Equal(t, balanceProvider+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
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
	entry, err := ts.keepers.Epochstorage.GetStakeEntryForClientEpoch(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clients[0].address, epoch)
	require.Nil(t, err)

	maxcu, err := ts.keepers.Pairing.GetAllowedCU(sdk.UnwrapSDKContext(ts.ctx), entry)
	require.Nil(t, err)

	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
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

	sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.Nil(t, err)
	balance = balance - ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Zero(t, balance)
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
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
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

	sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)
	relayRequest2 := *relayRequest
	Relays = append(Relays, &relayRequest2)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
	require.NotNil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(cuSum))
	require.Equal(t, balance+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())
	burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
	newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
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

	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
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

	sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       int64
	}{
		{"ModifiedProvider", ts.clients[0].address.String(), ts.spec.Apis[0].ComputeUnits * 10, 1},
		{"ModifiedCU", ts.providers[0].address.String(), ts.spec.Apis[0].ComputeUnits * 9, 1},
		{"ModifiedID", ts.providers[0].address.String(), ts.spec.Apis[0].ComputeUnits * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			relayRequest.Provider = tt.provider
			relayRequest.CuSum = tt.cu
			relayRequest.SessionId = uint64(tt.id)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})

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

	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
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

	sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	relay := *relayRequest
	Relays = append(Relays, &relay)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
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

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
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
				Provider:        ts.providers[0].address.String(),
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

			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
			if tt.valid {
				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(cuSum))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
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

			relayRequest := &types.RelayRequest{
				Provider:        ts.providers[0].address.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(1),
				ChainID:         ts.spec.Name,
				CuSum:           cuSum,
				BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
				RelayNum:        0,
				RequestBlock:    -1,
				QoSReport:       QoS,
				DataReliability: nil,
			}
			QoS.ComputeQoS()
			sig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			relay := *relayRequest
			Relays = append(Relays, &relay)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].address.String(), Relays: Relays})
			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				score, err := QoS.ComputeQoS()
				require.Nil(t, err)

				want := mint.MulInt64(int64(cuSum))
				want = want.Mul(score.Mul(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx))).Add(sdk.OneDec().Sub(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx)))))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[0].address, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochstoragetypes.ClientKey, ts.spec.Index, ts.clients[0].address)
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
			err := ts.addClient(1)
			require.Nil(t, err)
			err = ts.addProvider(100)
			require.Nil(t, err)
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			cuSum := ts.spec.Apis[0].ComputeUnits * 10

			QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
			relayRequest := &types.RelayRequest{
				Provider:        ts.providers[0].address.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(1),
				ChainID:         ts.spec.Name,
				CuSum:           cuSum,
				BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
				RelayNum:        0,
				RequestBlock:    -1,
				QoSReport:       QoS,
				DataReliability: nil,
			}
			QoS.ComputeQoS()

			relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
			require.Nil(t, err)

			currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
			var index0 int64
			var providers []epochstoragetypes.StakeEntry
			var relayReply *types.RelayReply
			var nonce uint32
			// increasing the nonce changes the hash of the reply which in turn produces a different vrfRes resulting to a different index
			for {
				relayReply = &types.RelayReply{
					Nonce: nonce,
				}
				relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
				require.Nil(t, err)

				vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, currentEpoch)

				index0 = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

				providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relayRequest.ChainID, ts.clients[0].address)
				require.Nil(t, err)

				if providers[index0].Address != ts.providers[0].address.String() {
					break
				} else {
					nonce += 1
				}
			}
			vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false, currentEpoch)
			dataReliability0 := &types.VRFData{
				Differentiator: false,
				VrfValue:       vrf_res0,
				VrfProof:       vrf_proof0,
				ProviderSig:    relayReply.Sig,
				AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
				QueryHash:      utils.CalculateQueryHash(*relayRequest),
				Sig:            nil,
			}
			dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].secretKey, dataReliability0)
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
			relayRequestWithDataReliability0 := &types.RelayRequest{
				Provider:        providers[index0].Address,
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(1),
				ChainID:         ts.spec.Name,
				CuSum:           cuSum,
				BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
				RelayNum:        0,
				RequestBlock:    -1,
				DataReliability: dataReliability0,
				QoSReport:       QoSDR,
			}
			QoSDR.ComputeQoS()
			relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequestWithDataReliability0)
			require.Nil(t, err)

			provider := ts.getProvider(providers[index0].Address)
			relaysRequests := []*types.RelayRequest{relayRequestWithDataReliability0}

			balanceBefore := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.address, epochstoragetypes.TokenDenom).Amount.Int64()
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
			if tt.valid {
				require.Nil(t, err)

				balanceAfter := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.address, epochstoragetypes.TokenDenom).Amount.Int64()

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
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10

	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
		QoSReport:       QoS,
	}
	QoS.ComputeQoS()

	relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32

	wrongProviderIndex := 1
GetWrongProvider:
	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, currentEpoch)

		index0 = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)
		index1 := utils.GetIndexForVrf(vrfRes1, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relayRequest.ChainID, ts.clients[0].address)
		require.Nil(t, err)
		// two providers returned by GetIndexForVrf and the provider getting tested need 1 more to perform this test properly
		require.Greater(t, len(providers), 3)
		nonce += 1
		// recalculate if for some reason vrf function returned the provider getting tested's index (should not happen but just to be safe)
		if providers[index0].Address == ts.providers[0].address.String() {
			continue
		} else if providers[index1].Address == ts.providers[0].address.String() {
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
	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].secretKey, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := &types.RelayRequest{
		Provider:        providers[wrongProviderIndex].Address,
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: dataReliability0,
		QoSReport:       QoSDR,
	}
	QoSDR.ComputeQoS()
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[wrongProviderIndex].Address)
	relaysRequests := []*types.RelayRequest{relayRequestWithDataReliability0}

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
	require.NotNil(t, err)
}

// provider attempts to do a datareliability even though it is not triggered (below the threshold)
func TestRelayPaymentDataReliabilityBelowReliabilityThreshold(t *testing.T) {
	ts := setupForPaymentTest(t)

	ts.spec = common.CreateMockSpec()
	ts.spec.ReliabilityThreshold = 0
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)
	params := ts.keepers.Pairing.GetParams(sdk.UnwrapSDKContext(ts.ctx))
	params.ServicersToPairCount = 5
	ts.keepers.Pairing.SetParams(sdk.UnwrapSDKContext(ts.ctx), params)
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(5)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
		QoSReport:       QoS,
	}
	QoS.ComputeQoS()
	relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	require.Nil(t, err)

	var relayReply *types.RelayReply
	var nonce uint32
	relayReply = &types.RelayReply{
		Nonce: nonce,
	}
	relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, currentEpoch)

	index0 := utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)
	index1 := utils.GetIndexForVrf(vrfRes1, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

	require.Equal(t, index0, int64(-1))
	require.Equal(t, index1, int64(-1))
	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].secretKey, dataReliability0)
	require.Nil(t, err)

	// make all providers send a datareliability payment request. Everyone should fail
	for _, provider := range ts.providers {
		QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
		relayRequestWithDataReliability0 := &types.RelayRequest{
			Provider:        provider.address.String(),
			ApiUrl:          "",
			Data:            []byte(ts.spec.Apis[0].Name),
			SessionId:       uint64(1),
			ChainID:         ts.spec.Name,
			CuSum:           cuSum,
			BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			RelayNum:        0,
			RequestBlock:    -1,
			DataReliability: dataReliability0,
			QoSReport:       QoSDR,
		}
		QoSDR.ComputeQoS()
		relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequestWithDataReliability0)
		require.Nil(t, err)

		relaysRequests := []*types.RelayRequest{relayRequestWithDataReliability0}

		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
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
	err := ts.addClient(2)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
		QoSReport:       QoS,
	}
	QoS.ComputeQoS()
	relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32
	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, currentEpoch)

		index0 = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relayRequest.ChainID, ts.clients[0].address)
		require.Nil(t, err)

		if providers[index0].Address != ts.providers[0].address.String() {
			break
		}
		nonce += 1
	}

	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[1].secretKey, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := &types.RelayRequest{
		Provider:        providers[index0].Address,
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: dataReliability0,
		QoSReport:       QoSDR,
	}
	QoSDR.ComputeQoS()
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[1].secretKey, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[index0].Address)
	relaysRequests := []*types.RelayRequest{relayRequestWithDataReliability0}

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
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
	err := ts.addClient(1)
	require.Nil(t, err)
	err = ts.addProvider(100)
	require.Nil(t, err)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	cuSum := ts.spec.Apis[0].ComputeUnits * 10
	QoS := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].address.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
		QoSReport:       QoS,
	}
	QoS.ComputeQoS()

	relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	require.Nil(t, err)

	currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
	var index0 int64
	var providers []epochstoragetypes.StakeEntry
	var relayReply *types.RelayReply
	var nonce uint32
	for {
		relayReply = &types.RelayReply{
			Nonce: nonce,
		}
		relayReply.Sig, err = sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
		require.Nil(t, err)

		vrfRes0, _ := utils.CalculateVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, currentEpoch)

		index0 = utils.GetIndexForVrf(vrfRes0, uint32(ts.keepers.Pairing.ServicersToPairCount(sdk.UnwrapSDKContext(ts.ctx))), ts.spec.ReliabilityThreshold)

		providers, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), relayRequest.ChainID, ts.clients[0].address)
		require.Nil(t, err)

		if providers[index0].Address != ts.providers[0].address.String() {
			break
		} else {
			nonce += 1
		}
	}

	vrf_res0, vrf_proof0 := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false, currentEpoch)
	dataReliability0 := &types.VRFData{
		Differentiator: false,
		VrfValue:       vrf_res0,
		VrfProof:       vrf_proof0,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest),
		Sig:            nil,
	}
	dataReliability0.Sig, err = sigs.SignVRFData(ts.clients[0].secretKey, dataReliability0)
	require.Nil(t, err)

	QoSDR := &types.QualityOfServiceReport{Latency: sdk.NewDecWithPrec(1, 0), Availability: sdk.NewDecWithPrec(1, 0), Sync: sdk.NewDecWithPrec(1, 0)}
	relayRequestWithDataReliability0 := &types.RelayRequest{
		Provider:        providers[index0].Address,
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: dataReliability0,
		QoSReport:       QoSDR,
	}
	QoSDR.ComputeQoS()
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	provider := ts.getProvider(providers[index0].Address)
	relaysRequests := []*types.RelayRequest{relayRequestWithDataReliability0}

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
	require.Nil(t, err)

	// Advance Epoch and set block height and resign the tx
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	relayRequestWithDataReliability0.BlockHeight = sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
	relayRequestWithDataReliability0.SessionId = uint64(2)
	relayRequestWithDataReliability0.Sig, err = sigs.SignRelay(ts.clients[0].secretKey, *relayRequestWithDataReliability0)
	require.Nil(t, err)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: provider.address.String(), Relays: relaysRequests})
	require.NotNil(t, err)
}
