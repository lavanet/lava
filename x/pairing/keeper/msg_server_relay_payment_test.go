package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	ctx             context.Context
	keepers         *testkeeper.Keepers
	servers         *testkeeper.Servers
	providerAccount common.Account
	clientAccount   common.Account
	spec            spectypes.Spec
}

func setupForPaymentTest(t *testing.T) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	//init keepers state

	var balance int64 = 100000
	ts.clientAccount = common.CreateNewAccount(ts.ctx, *ts.keepers, balance)
	ts.providerAccount = common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, ts.clientAccount, ts.spec, stake, false)
	common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, ts.providerAccount, ts.spec, stake, true)

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
				Provider:        ts.providerAccount.Addr.String(),
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

			sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balanceProvider := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, found, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)
			require.Equal(t, true, found)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})

			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

				require.Equal(t, balanceProvider+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)
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
	entry, err := ts.keepers.Epochstorage.GetStakeEntryForClientEpoch(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Name, ts.clientAccount.Addr, epoch)
	require.Nil(t, err)

	maxcu, err := ts.keepers.Pairing.GetAllowedCUForBlock(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()), entry)
	require.Nil(t, err)

	relayRequest := &types.RelayRequest{
		Provider:        ts.providerAccount.Addr.String(),
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

	sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
	require.Nil(t, err)
	balance = balance - ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Zero(t, balance)
}

func TestRelayPaymentDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	cuSum := ts.spec.GetApis()[0].ComputeUnits * 10
	relayRequest := &types.RelayRequest{
		Provider:        ts.providerAccount.Addr.String(),
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

	sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)
	relayRequest2 := *relayRequest
	Relays = append(Relays, &relayRequest2)

	balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
	require.NotNil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(cuSum))
	require.Equal(t, balance+want.TruncateInt64(),
		ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64())

	burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
	newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)
	require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

}

func TestRelayPaymentDataModification(t *testing.T) {
	ts := setupForPaymentTest(t)

	relayRequest := &types.RelayRequest{
		Provider:        ts.providerAccount.Addr.String(),
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

	sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	tests := []struct {
		name     string
		provider string
		cu       uint64
		id       int64
	}{
		{"ModifiedProvider", ts.clientAccount.Addr.String(), ts.spec.Apis[0].ComputeUnits * 10, 1},
		{"ModifiedCU", ts.providerAccount.Addr.String(), ts.spec.Apis[0].ComputeUnits * 9, 1},
		{"ModifiedID", ts.providerAccount.Addr.String(), ts.spec.Apis[0].ComputeUnits * 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			relayRequest.Provider = tt.provider
			relayRequest.CuSum = tt.cu
			relayRequest.SessionId = uint64(tt.id)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})

			require.NotNil(t, err)
		})
	}
}

func TestRelayPaymentDelayedDoubleSpending(t *testing.T) {
	ts := setupForPaymentTest(t)

	relayRequest := &types.RelayRequest{
		Provider:        ts.providerAccount.Addr.String(),
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

	sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	var Relays []*types.RelayRequest
	relay := *relayRequest
	Relays = append(Relays, &relay)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
	require.Nil(t, err)

	epochToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

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

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
			require.NotNil(t, err)

		})
	}
}

func TestRelayPaymentOldEpochs(t *testing.T) {
	ts := setupForPaymentTest(t)
	currBlock := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	epochsToSave, err := ts.keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ts.ctx), currBlock)
	require.Nil(t, err)
	blocksInEpoch, err := ts.keepers.Epochstorage.EpochBlocks(sdk.UnwrapSDKContext(ts.ctx), currBlock)
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
		{"Epoch", 1, 1, true},                               //current -1*epoch
		{"Memory-Epoch", 2, int64(epochsToSave - 1), true},  //current -epoch to save + 1
		{"Memory", 3, int64(epochsToSave), true},            //current - epochToSave
		{"Memory+Epoch", 4, int64(epochsToSave + 1), false}, //current - epochToSave - 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cuSum := ts.spec.Apis[0].ComputeUnits * 10
			relayRequest := &types.RelayRequest{
				Provider:        ts.providerAccount.Addr.String(),
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

			sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
			if tt.valid {
				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(cuSum))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)
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
		availebility sdk.Dec
		latency      sdk.Dec
		sync         sdk.Dec
		valid        bool
	}{
		{"InvalidLatency", sdk.NewDecWithPrec(2, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), false},
		{"InvalidAvailebility", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(2, 0), sdk.NewDecWithPrec(1, 0), false},
		{"Invalidsync", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(2, 0), false},
		{"PerfectScore", sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), true},
		{"MediumScore", sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(1, 0), sdk.NewDecWithPrec(1, 0), true},
		{"ZeroScore", sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupForPaymentTest(t)

			cuSum := ts.spec.Apis[0].ComputeUnits * 10
			QoS := &types.QualityOfServiceReport{Latency: tt.latency, Availability: tt.availebility, Sync: tt.sync}

			relayRequest := &types.RelayRequest{
				Provider:        ts.providerAccount.Addr.String(),
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
			sig, err := sigs.SignRelay(ts.clientAccount.SK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			relay := *relayRequest
			Relays = append(Relays, &relay)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64()
			stakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providerAccount.Addr.String(), Relays: Relays})
			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				score, err := QoS.ComputeQoS()
				require.Nil(t, err)

				want := mint.MulInt64(int64(cuSum))
				want = want.Mul(score.Mul(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx))).Add(sdk.OneDec().Sub(ts.keepers.Pairing.QoSWeight(sdk.UnwrapSDKContext(ts.ctx)))))
				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providerAccount.Addr, epochstoragetypes.TokenDenom).Amount.Int64())

				burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
				newStakeClient, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clientAccount.Addr)
				require.Equal(t, stakeClient.Stake.Amount.Int64()-burn.TruncateInt64(), newStakeClient.Stake.Amount.Int64())

			} else {
				require.NotNil(t, err)
			}

		})
	}
}
