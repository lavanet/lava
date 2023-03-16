package common

import (
	"context"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/coniks-sys/coniks-go/crypto/vrf"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type Account struct {
	SK    *btcSecp256k1.PrivateKey
	Addr  sdk.AccAddress
	VrfSk vrf.PrivateKey
	VrfPk vrf.PublicKey
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
		Description:              "plan for testing",
		Type:                     "rpc",
		Block:                    100,
		Price:                    sdk.NewCoin("ulava", sdk.NewInt(100)),
		ComputeUnits:             1000,
		ComputeUnitsPerEpoch:     100,
		MaxProvidersToPair:       3,
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

func BuildRelayRequest(ctx context.Context, provider string, contentHash []byte, cuSum uint64, spec string, qos *types.QualityOfServiceReport) *types.RelaySession {
	relaySession := &types.RelaySession{
		Provider:    provider,
		ContentHash: contentHash,
		SessionId:   uint64(1),
		SpecID:      spec,
		CuSum:       cuSum,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QoSReport:   qos,
		LavaChainId: sdk.UnwrapSDKContext(ctx).BlockHeader().ChainID,
	}
	if qos != nil {
		qos.ComputeQoS()
	}
	return relaySession
}

func CreateMsgDetection(ctx context.Context, consumer Account, provider0 Account, provider1 Account, spec spectypes.Spec) (conflicttypes.MsgDetection, error) {
	var msg conflicttypes.MsgDetection
	msg.Creator = consumer.Addr.String()
	// request 0
	msg.ResponseConflict = &conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}, ConflictRelayData1: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &types.RelayReply{}}}
	msg.ResponseConflict.ConflictRelayData0.Request.RelayData = &types.RelayPrivateData{
		ConnectionType: "",
		ApiUrl:         "",
		Data:           []byte("DUMMYREQUEST"),
		RequestBlock:   100,
		ApiInterface:   "",
		Salt:           []byte{1},
	}
	msg.ResponseConflict.ConflictRelayData0.Request.RelaySession = &types.RelaySession{
		Provider:    provider0.Addr.String(),
		ContentHash: sigs.CalculateContentHashForRelayData(msg.ResponseConflict.ConflictRelayData0.Request.RelayData),
		SessionId:   uint64(1),
		SpecID:      spec.Index,
		CuSum:       0,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QoSReport:   &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()},
	}

	msg.ResponseConflict.ConflictRelayData0.Request.DataReliability = nil

	sig, err := sigs.SignRelay(consumer.SK, *msg.ResponseConflict.ConflictRelayData0.Request.RelaySession)
	if err != nil {
		return msg, err
	}

	msg.ResponseConflict.ConflictRelayData0.Request.RelaySession.Sig = sig

	// request 1
	temp, _ := msg.ResponseConflict.ConflictRelayData0.Request.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Request.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Provider = provider1.Addr.String()
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
	sig, err = sigs.SignRelay(consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request.RelaySession)
	if err != nil {
		return msg, err
	}
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = sig

	// reply 0
	msg.ResponseConflict.ConflictRelayData0.Reply.Nonce = 10
	msg.ResponseConflict.ConflictRelayData0.Reply.FinalizedBlocksHashes = []byte{}
	msg.ResponseConflict.ConflictRelayData0.Reply.LatestBlock = msg.ResponseConflict.ConflictRelayData0.Request.RelayData.RequestBlock + int64(spec.BlockDistanceForFinalizedData)
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
