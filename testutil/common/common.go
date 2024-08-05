package common

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	conflictconstruct "github.com/lavanet/lava/v2/x/conflict/types/construct"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func CreateNewAccount(ctx context.Context, keepers testkeeper.Keepers, balance int64) (acc sigs.Account) {
	acc = sigs.GenerateDeterministicFloatingKey(testkeeper.Randomizer)
	testkeeper.Randomizer.Inc()
	coins := sdk.NewCoins(sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.NewInt(balance)))
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), acc.Addr, coins)
	return
}

func StakeAccount(t *testing.T, ctx context.Context, keepers testkeeper.Keepers, servers testkeeper.Servers, acc sigs.Account, spec spectypes.Spec, stake int64, validator sigs.Account) {
	endpoints := []epochstoragetypes.Endpoint{}
	for _, collection := range spec.ApiCollections {
		endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", ApiInterfaces: []string{collection.CollectionData.ApiInterface}, Geolocation: 1})
	}
	_, err := servers.PairingServer.StakeProvider(ctx, &types.MsgStakeProvider{Creator: acc.Addr.String(), Address: acc.Addr.String(), ChainID: spec.Index, Amount: sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints, DelegateLimit: sdk.NewCoin(keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ctx)), sdk.ZeroInt()), DelegateCommission: 100, Validator: sdk.ValAddress(validator.Addr).String(), Description: MockDescription()})
	require.NoError(t, err)
}

func BuySubscription(ctx context.Context, keepers testkeeper.Keepers, servers testkeeper.Servers, acc sigs.Account, plan string) {
	servers.SubscriptionServer.Buy(ctx, &subscriptiontypes.MsgBuy{Creator: acc.Addr.String(), Consumer: acc.Addr.String(), Index: plan, Duration: 1})
}

func MockDescription() stakingtypes.Description {
	return stakingtypes.NewDescription("prov", "iden", "web", "sec", "details")
}

func BuildRelayRequest(ctx context.Context, provider string, contentHash []byte, cuSum uint64, spec string, qos *types.QualityOfServiceReport) *types.RelaySession {
	return BuildRelayRequestWithBadge(ctx, provider, contentHash, uint64(1), cuSum, spec, qos, nil)
}

func BuildRelayRequestWithSession(ctx context.Context, provider string, contentHash []byte, sessionId uint64, cuSum uint64, spec string, qos *types.QualityOfServiceReport) *types.RelaySession {
	return BuildRelayRequestWithBadge(ctx, provider, contentHash, sessionId, cuSum, spec, qos, nil)
}

func BuildRelayRequestWithBadge(ctx context.Context, provider string, contentHash []byte, sessionId uint64, cuSum uint64, spec string, qos *types.QualityOfServiceReport, badge *types.Badge) *types.RelaySession {
	relaySession := &types.RelaySession{
		Provider:    provider,
		ContentHash: contentHash,
		SessionId:   sessionId,
		SpecId:      spec,
		CuSum:       cuSum,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QosReport:   qos,
		LavaChainId: sdk.UnwrapSDKContext(ctx).BlockHeader().ChainID,
		Badge:       badge,
	}
	if qos != nil {
		qos.ComputeQoS()
	}
	return relaySession
}

func CreateMsgDetectionTest(ctx context.Context, consumer, provider0, provider1 sigs.Account, spec spectypes.Spec) (detectionMsg *conflicttypes.MsgDetection, reply1, reply2 *types.RelayReply, errRet error) {
	msg := &conflicttypes.MsgDetection{}
	msg.Creator = consumer.Addr.String()
	// request 0
	msg.ResponseConflict = &conflicttypes.ResponseConflict{ConflictRelayData0: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &conflicttypes.ReplyMetadata{}}, ConflictRelayData1: &conflicttypes.ConflictRelayData{Request: &types.RelayRequest{}, Reply: &conflicttypes.ReplyMetadata{}}}
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
		ContentHash: sigs.HashMsg(msg.ResponseConflict.ConflictRelayData0.Request.RelayData.GetContentHashData()),
		SessionId:   uint64(1),
		SpecId:      spec.Index,
		CuSum:       0,
		Epoch:       sdk.UnwrapSDKContext(ctx).BlockHeight(),
		RelayNum:    0,
		QosReport:   &types.QualityOfServiceReport{Latency: sdk.OneDec(), Availability: sdk.OneDec(), Sync: sdk.OneDec()},
	}

	sig, err := sigs.Sign(consumer.SK, *msg.ResponseConflict.ConflictRelayData0.Request.RelaySession)
	if err != nil {
		return msg, nil, nil, err
	}

	msg.ResponseConflict.ConflictRelayData0.Request.RelaySession.Sig = sig

	// request 1
	temp, _ := msg.ResponseConflict.ConflictRelayData0.Request.Marshal()
	msg.ResponseConflict.ConflictRelayData1.Request.Unmarshal(temp)
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Provider = provider1.Addr.String()
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = []byte{}
	sig, err = sigs.Sign(consumer.SK, *msg.ResponseConflict.ConflictRelayData1.Request.RelaySession)
	if err != nil {
		return msg, nil, nil, err
	}
	msg.ResponseConflict.ConflictRelayData1.Request.RelaySession.Sig = sig

	// reply 0
	reply := &types.RelayReply{
		Data:                  []byte("DUMMYREPLY"),
		Sig:                   sig,
		LatestBlock:           msg.ResponseConflict.ConflictRelayData0.Request.RelayData.RequestBlock + int64(spec.BlockDistanceForFinalizedData),
		FinalizedBlocksHashes: []byte{},
		SigBlocks:             sig,
		Metadata:              []types.Metadata{},
	}
	relayExchange := types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData0.Request, *reply)
	sig, err = sigs.Sign(provider0.SK, relayExchange)
	if err != nil {
		return msg, nil, nil, err
	}
	reply.Sig = sig

	relayFinalization := types.NewRelayFinalization(types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData0.Request, *reply), consumer.Addr)
	sigBlocks, err := sigs.Sign(provider0.SK, relayFinalization)
	if err != nil {
		return msg, nil, nil, err
	}
	reply.SigBlocks = sigBlocks
	msg.ResponseConflict.ConflictRelayData0 = conflictconstruct.ConstructConflictRelayData(reply, msg.ResponseConflict.ConflictRelayData0.Request)
	// reply 1
	temp, _ = reply.Marshal()
	reply2 = &types.RelayReply{}
	reply2.Unmarshal(temp)
	reply2.Data = append(reply2.Data, []byte("DIFF")...)
	relayExchange2 := types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData1.Request, *reply2)
	sig, err = sigs.Sign(provider1.SK, relayExchange2)
	if err != nil {
		return msg, nil, nil, err
	}
	reply2.Sig = sig
	relayFinalization2 := types.NewRelayFinalization(types.NewRelayExchange(*msg.ResponseConflict.ConflictRelayData1.Request, *reply2), consumer.Addr)
	sigBlocks, err = sigs.Sign(provider1.SK, relayFinalization2)
	if err != nil {
		return msg, nil, nil, err
	}
	reply2.SigBlocks = sigBlocks
	msg.ResponseConflict.ConflictRelayData1 = conflictconstruct.ConstructConflictRelayData(reply2, msg.ResponseConflict.ConflictRelayData1.Request)
	return msg, reply, reply2, err
}
