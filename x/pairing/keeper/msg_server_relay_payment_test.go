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
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/rpc/core"
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
	_, ts.proAddr = sigs.GenerateFloatingKey()
	ts.clientSK, ts.clientAddr = sigs.GenerateFloatingKey()

	var balance int64 = 100000
	ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(balance))))
	ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.clientAddr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(balance))))

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)
	_, err := ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: ts.clientAddr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin("stake", sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)

	endpoints := []epochtypes.Endpoint{}
	endpoints = append(endpoints, epochtypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
	_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: ts.proAddr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin("stake", sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	return ts
}

func TestRelayPaymentBlockTime(t *testing.T) {
	ts := setupForPaymentTest(t)

	blockstore := testkeeper.MockBlockStore{}
	blockstore.SetHeight(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	core.SetEnvironment(&core.Environment{BlockStore: &blockstore})

	tests := []struct {
		name      string
		blockTime int64
		valid     bool
	}{
		{"HappyFlow", sdk.UnwrapSDKContext(ts.ctx).BlockHeight(), true},
		{"OldBlock", sdk.UnwrapSDKContext(ts.ctx).BlockHeight() - 1, false},
		{"NewBlock", sdk.UnwrapSDKContext(ts.ctx).BlockHeight() + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRequest := &types.RelayRequest{
				Provider:        ts.proAddr.String(),
				ApiUrl:          "",
				Data:            []byte(ts.spec.Apis[0].Name),
				SessionId:       uint64(1),
				ChainID:         ts.spec.Name,
				CuSum:           ts.spec.Apis[0].ComputeUnits * 10,
				BlockHeight:     tt.blockTime,
				RelayNum:        0,
				RequestBlock:    -1,
				DataReliability: nil,
			}

			sig, err := sigs.SignRelay(ts.clientSK, *relayRequest)
			relayRequest.Sig = sig
			require.Nil(t, err)

			var Relays []*types.RelayRequest
			Relays = append(Relays, relayRequest)

			balance := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, "stake").Amount.Int64()

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.proAddr.String(), Relays: Relays})

			if tt.valid {
				require.Nil(t, err)

				mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
				want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

				require.Equal(t, balance+want.TruncateInt64(),
					ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.proAddr, "stake").Amount.Int64())
			} else {
				require.NotNil(t, err)
			}

		})
	}
}
