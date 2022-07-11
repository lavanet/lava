package keeper_test

import (
	"context"
	"fmt"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	utils "github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type account struct {
	SK   *btcSecp256k1.PrivateKey
	Addr sdk.AccAddress
}

const NUM_OF_PROVIDERS = 5

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	Providers []account
	spec      spectypes.Spec
	consumer  account
}

func setupForConflictTests(t *testing.T) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	//init keepers state
	//setup consumer
	var balance int64 = 100000
	ts.consumer.SK, ts.consumer.Addr = sigs.GenerateFloatingKey()
	ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.consumer.Addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))

	//setup providers
	for i := 0; i < NUM_OF_PROVIDERS; i++ {
		var tempProviderAccount account
		tempProviderAccount.SK, tempProviderAccount.Addr = sigs.GenerateFloatingKey()
		ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), tempProviderAccount.Addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
		ts.Providers = append(ts.Providers, tempProviderAccount)
	}

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	//stake consumer
	var stake int64 = 1000
	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)
	_, err := ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: ts.consumer.Addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)

	//stake providers
	endpoints := []epochtypes.Endpoint{}
	endpoints = append(endpoints, epochtypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
	for _, provider := range ts.Providers {
		_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: provider.Addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
		require.Nil(t, err)
	}

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	return ts
}

func TestDetection(t *testing.T) {
	ts := setupForConflictTests(t)

	tests := []struct {
		name string
	}{
		{"HappyFlow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg conflicttypes.MsgDetection
			msg.Creator = ts.consumer.Addr.String()
			msg.ResponseConflict = &conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}, ConflictRelayData1: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}}
			msg.ResponseConflict.ConflictRelayData0.Request.ApiId = 0
			msg.ResponseConflict.ConflictRelayData0.Request.ApiUrl = ""
			msg.ResponseConflict.ConflictRelayData0.Request.BlockHeight = sdk.UnwrapSDKContext(ts.ctx).BlockHeight()
			msg.ResponseConflict.ConflictRelayData0.Request.ChainID = ts.spec.Index
			msg.ResponseConflict.ConflictRelayData0.Request.CuSum = 0
			msg.ResponseConflict.ConflictRelayData0.Request.Data = []byte("DUMMYREQUEST")
			msg.ResponseConflict.ConflictRelayData0.Request.Provider = ts.Providers[0].Addr.String()
			msg.ResponseConflict.ConflictRelayData0.Request.QoSReport = &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}
			msg.ResponseConflict.ConflictRelayData0.Request.RelayNum = 1
			msg.ResponseConflict.ConflictRelayData0.Request.SessionId = 1
			msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock = 100
			msg.ResponseConflict.ConflictRelayData0.Request.DataReliability = &types.VRFData{}
			msg.ResponseConflict.ConflictRelayData1.Request = msg.ResponseConflict.ConflictRelayData0.Request
			msg.ResponseConflict.ConflictRelayData1.Request.Provider = ts.Providers[1].Addr.String()

			sig, err := sigs.SignRelay(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData0.Request)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData0.Request.Sig = sig

			sig, err = sigs.SignRelay(ts.consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Request.Sig = sig

			msg.ResponseConflict.ConflictRelayData0.Reply.Nonce = 10
			msg.ResponseConflict.ConflictRelayData0.Reply.FinalizedBlocksHashes = []byte{}
			msg.ResponseConflict.ConflictRelayData0.Reply.LatestBlock = msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock + int64(ts.spec.FinalizationCriteria)
			msg.ResponseConflict.ConflictRelayData0.Reply.Data = []byte("DUMMYREPLY")
			msg.ResponseConflict.ConflictRelayData0.Reply.SigBlocks = []byte{}

			msg.ResponseConflict.ConflictRelayData1.Reply = msg.ResponseConflict.ConflictRelayData0.Reply

			sig, err = sigs.SignRelayResponse(ts.Providers[0].SK, msg.ResponseConflict.ConflictRelayData0.Reply, msg.ResponseConflict.ConflictRelayData0.Request)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData0.Reply.Sig = sig

			sig, err = sigs.SignRelayResponse(ts.Providers[1].SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request)
			require.Nil(t, err)
			msg.ResponseConflict.ConflictRelayData1.Reply.Sig = sig

			_, err = ts.servers.ConflictServer.Detection(ts.ctx, &msg)
			require.Nil(t, err)
			sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()
			fmt.Printf("yarom event \n %+v ", sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())
		})
	}
}
