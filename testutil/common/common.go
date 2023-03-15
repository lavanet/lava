package common

import (
	"context"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type Account struct {
	SK   *btcSecp256k1.PrivateKey
	Addr sdk.AccAddress
}

func CreateMockSpec() spectypes.Spec {
	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.ReliabilityThreshold = 4294967295
	spec.BlockDistanceForFinalizedData = 0
	spec.DataReliabilityEnabled = true
	spec.MinStakeClient = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(100))
	spec.MinStakeProvider = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1000))
	apiInterface := spectypes.ApiInterface{Interface: "mockInt", Type: "GET"}
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: []spectypes.ApiInterface{apiInterface}})
	spec.BlockDistanceForFinalizedData = 0
	return spec
}

func CreateMockPlan() plantypes.Plan {
	plan := plantypes.Plan{
		Index:                    "mockPlan",
		Name:                     "mock plan",
		Description:              "plan for testing",
		Type:                     "rpc",
		Block:                    100,
		Price:                    sdk.NewCoin("ulava", sdk.NewInt(100)),
		ComputeUnits:             1000,
		ComputeUnitsPerEpoch:     100,
		ServicersToPair:          3,
		AllowOveruse:             true,
		OveruseRate:              10,
		AnnualDiscountPercentage: 20,
	}

	return plan
}

func CreateNewAccount(ctx context.Context, keepers testkeeper.Keepers, balance int64) (acc Account) {
	acc.SK, acc.Addr = sigs.GenerateFloatingKey()
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), acc.Addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	return
}

func StakeAccount(t *testing.T, ctx context.Context, keepers testkeeper.Keepers, servers testkeeper.Servers, acc Account, spec spectypes.Spec, stake int64, isProvider bool) {
	if isProvider {
		endpoints := []epochstoragetypes.Endpoint{}
		endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", UseType: spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
		_, err := servers.PairingServer.StakeProvider(ctx, &types.MsgStakeProvider{Creator: acc.Addr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
		require.Nil(t, err)
	} else {
		_, pk, _ := utils.GeneratePrivateVRFKey()
		vrfPk := &utils.VrfPubKey{}
		vrfPk.Unmarshal(pk)
		_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: acc.Addr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
		require.Nil(t, err)
	}
}

func CreateMsgDetection(ctx context.Context, consumer Account, provider0 Account, provider1 Account, spec spectypes.Spec) (conflicttypes.MsgDetection, error) {
	var msg conflicttypes.MsgDetection
	msg.Creator = consumer.Addr.String()
	// request 0
	msg.ResponseConflict = &conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}, ConflictRelayData1: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}}
	msg.ResponseConflict.ConflictRelayData0.Request.ConnectionType = ""
	msg.ResponseConflict.ConflictRelayData0.Request.ApiUrl = ""
	msg.ResponseConflict.ConflictRelayData0.Request.BlockHeight = sdk.UnwrapSDKContext(ctx).BlockHeight()
	msg.ResponseConflict.ConflictRelayData0.Request.ChainID = spec.Index
	msg.ResponseConflict.ConflictRelayData0.Request.CuSum = 0
	msg.ResponseConflict.ConflictRelayData0.Request.Data = []byte("DUMMYREQUEST")
	msg.ResponseConflict.ConflictRelayData0.Request.Provider = provider0.Addr.String()
	msg.ResponseConflict.ConflictRelayData0.Request.QoSReport = &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()}
	msg.ResponseConflict.ConflictRelayData0.Request.RelayNum = 1
	msg.ResponseConflict.ConflictRelayData0.Request.SessionId = 1
	msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock = 100
	msg.ResponseConflict.ConflictRelayData0.Request.DataReliability = nil
	msg.ResponseConflict.ConflictRelayData0.Request.Sig = []byte{}

	sig, err := sigs.SignRelay(consumer.SK, *msg.ResponseConflict.ConflictRelayData0.Request)
	if err != nil {
		return msg, err
	}

	msg.ResponseConflict.ConflictRelayData0.Request.Sig = sig

	// request 1
	temp, _ := msg.ResponseConflict.ConflictRelayData0.Request.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Request.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Request.Provider = provider1.Addr.String()
	msg.ResponseConflict.ConflictRelayData1.Request.Sig = []byte{}
	sig, err = sigs.SignRelay(consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData1.Request.Sig = sig

	// reply 0
	msg.ResponseConflict.ConflictRelayData0.Reply.Nonce = 10
	msg.ResponseConflict.ConflictRelayData0.Reply.FinalizedBlocksHashes = []byte{}
	msg.ResponseConflict.ConflictRelayData0.Reply.LatestBlock = msg.ResponseConflict.ConflictRelayData0.Request.RequestBlock + int64(spec.BlockDistanceForFinalizedData)
	msg.ResponseConflict.ConflictRelayData0.Reply.Data = []byte("DUMMYREPLY")
	sig, err = sigs.SignRelayResponse(provider0.SK, msg.ResponseConflict.ConflictRelayData0.Reply, msg.ResponseConflict.ConflictRelayData0.Request)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData0.Reply.Sig = sig
	sigBlocks, err := sigs.SignResponseFinalizationData(provider0.SK, msg.ResponseConflict.ConflictRelayData0.Reply, msg.ResponseConflict.ConflictRelayData0.Request, consumer.Addr)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData0.Reply.SigBlocks = sigBlocks

	// reply 1
	temp, _ = msg.ResponseConflict.ConflictRelayData0.Reply.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Reply.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Reply.Data = append(msg.ResponseConflict.ConflictRelayData1.Reply.Data, []byte("DIFF")...)
	sig, err = sigs.SignRelayResponse(provider1.SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData1.Reply.Sig = sig
	sigBlocks, err = sigs.SignResponseFinalizationData(provider1.SK, msg.ResponseConflict.ConflictRelayData1.Reply, msg.ResponseConflict.ConflictRelayData1.Request, consumer.Addr)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData1.Reply.SigBlocks = sigBlocks

	return msg, nil
}
