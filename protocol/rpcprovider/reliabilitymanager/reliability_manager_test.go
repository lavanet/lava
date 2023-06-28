package reliabilitymanager_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	terderminttypes "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	plantypes "github.com/lavanet/lava/x/plans/types"
)

func TestFullFlowReliabilityCompare(t *testing.T) {
	ctx := context.Background()
	// consumer
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	// provider
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	// second provider (DR)
	providerDR_sk, providerDR_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:                       20,
		LatestRelayCu:               10, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   123,
		Client:                      nil,
		RelayNum:                    1,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	singleConsumerSession2 := &lavasession.SingleConsumerSession{
		CuSum:                       200,
		LatestRelayCu:               100, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   456,
		Client:                      nil,
		RelayNum:                    5,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "banana",
		Value: "55",
	}
	relayRequestData := lavaprotocol.NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []byte("stubbytes"))
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.CalculateContentHashForRelayData(relay.RelayData)))
	latestBlock := int64(123)
	// provider handling the response
	finalizedBlockHashes := map[int64]interface{}{latestBlock: "AAA"}
	replyDataBuf := []byte("REPLY-STUB")
	reply := &pairingtypes.RelayReply{Data: replyDataBuf}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = latestBlock
	reply, err = lavaprotocol.SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply, true)
	require.NoError(t, err)
	err = lavaprotocol.VerifyRelayReply(reply, relay, provider_address.String())
	require.NoError(t, err)
	_, _, err = lavaprotocol.VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)

	relayResult := &lavaprotocol.RelayResult{
		Request:         relay,
		Reply:           reply,
		ProviderAddress: provider_address.String(),
		ReplyServer:     nil,
		Finalized:       true,
	}

	// now send this to another provider
	relayRequestDataDR := lavaprotocol.NewRelayData(ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata)
	relayDR, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []byte("stubbytes"))
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relayDR.RelaySession.ContentHash, sigs.CalculateContentHashForRelayData(relayDR.RelayData)))
	latestBlock = int64(123)
	// provider handling the response
	finalizedBlockHashes = map[int64]interface{}{latestBlock: "AAA"}
	replyDR := &pairingtypes.RelayReply{Data: replyDataBuf}
	jsonStr, err = json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	replyDR.FinalizedBlocksHashes = jsonStr
	replyDR.LatestBlock = latestBlock
	replyDR, err = lavaprotocol.SignRelayResponse(extractedConsumerAddress, *relayDR, providerDR_sk, replyDR, true)
	require.NoError(t, err)
	err = lavaprotocol.VerifyRelayReply(replyDR, relayDR, providerDR_address.String())
	require.NoError(t, err)
	_, _, err = lavaprotocol.VerifyFinalizationData(replyDR, relayDR, providerDR_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)
	relayResultDR := &lavaprotocol.RelayResult{
		Request:         relayDR,
		Reply:           replyDR,
		ProviderAddress: providerDR_address.String(),
		ReplyServer:     nil,
		Finalized:       true,
	}
	conflict := lavaprotocol.VerifyReliabilityResults(ctx, relayResult, relayResultDR)
	require.Nil(t, conflict)
}

func TestFullFlowReliabilityConflict(t *testing.T) {

	ts := setupForConflictTests(t, 3)
	ctx := ts.ctx
	// consumer
	consumer_sk, consumer_address := ts.consumer.SK, ts.consumer.Addr
	// provider
	provider_sk, provider_address := ts.Providers[0].SK, ts.Providers[0].Addr
	// second provider (DR)
	providerDR_sk, providerDR_address := ts.Providers[1].SK, ts.Providers[1].Addr
	specId := ts.spec.Index
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	epoch := int64(ts.keepers.Epochstorage.GetEpochStart(unwrapedCtx))
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:                       20,
		LatestRelayCu:               10, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   123,
		Client:                      nil,
		RelayNum:                    1,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	singleConsumerSession2 := &lavasession.SingleConsumerSession{
		CuSum:                       200,
		LatestRelayCu:               100, // set by GetSessions cuNeededForSession
		QoSInfo:                     lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
		SessionId:                   456,
		Client:                      nil,
		RelayNum:                    5,
		LatestBlock:                 epoch,
		Endpoint:                    nil,
		BlockListed:                 false, // if session lost sync we blacklist it.
		ConsecutiveNumberOfFailures: 0,     // number of times this session has failed
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "banana",
		Value: "55",
	}
	relayRequestData := lavaprotocol.NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []byte("stubbytes"))
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.CalculateContentHashForRelayData(relay.RelayData)))
	latestBlock := int64(123)
	// provider handling the response
	finalizedBlockHashes := map[int64]interface{}{latestBlock: "AAA"}
	replyDataBuf := []byte("REPLY-STUB")
	reply := &pairingtypes.RelayReply{Data: replyDataBuf}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = latestBlock
	reply, err = lavaprotocol.SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply, true)
	require.NoError(t, err)
	err = lavaprotocol.VerifyRelayReply(reply, relay, provider_address.String())
	require.NoError(t, err)
	_, _, err = lavaprotocol.VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)

	relayResult := &lavaprotocol.RelayResult{
		Request:         relay,
		Reply:           reply,
		ProviderAddress: provider_address.String(),
		ReplyServer:     nil,
		Finalized:       true,
	}

	// now send this to another provider
	relayRequestDataDR := lavaprotocol.NewRelayData(ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata)
	relayDR, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []byte("stubbytes"))
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relayDR.RelaySession.ContentHash, sigs.CalculateContentHashForRelayData(relayDR.RelayData)))
	latestBlock = int64(123)
	// provider handling the response
	finalizedBlockHashes = map[int64]interface{}{latestBlock: "AAA"}
	maliciousReply := []byte("Gimme-your-lava")
	replyDR := &pairingtypes.RelayReply{Data: maliciousReply}
	jsonStr, err = json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	replyDR.FinalizedBlocksHashes = jsonStr
	replyDR.LatestBlock = latestBlock
	replyDR, err = lavaprotocol.SignRelayResponse(extractedConsumerAddress, *relayDR, providerDR_sk, replyDR, true)
	require.NoError(t, err)
	err = lavaprotocol.VerifyRelayReply(replyDR, relayDR, providerDR_address.String())
	require.NoError(t, err)
	_, _, err = lavaprotocol.VerifyFinalizationData(replyDR, relayDR, providerDR_address.String(), consumer_address, int64(0), 0)
	require.NoError(t, err)
	relayResultDR := &lavaprotocol.RelayResult{
		Request:         relayDR,
		Reply:           replyDR,
		ProviderAddress: providerDR_address.String(),
		ReplyServer:     nil,
		Finalized:       true,
	}
	conflict := lavaprotocol.VerifyReliabilityResults(ctx, relayResult, relayResultDR)
	require.NotNil(t, conflict)
	msg := conflicttypes.NewMsgDetection(consumer_address.String(), nil, conflict, nil)
	_, err = ts.servers.ConflictServer.Detection(ts.ctx, msg)
	require.Nil(t, err)
	lastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	event := terderminttypes.Event(lastEvent)
	voteParams, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
	require.Nil(t, err)
	require.Equal(t, specId, voteParams.ChainID)
}

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	Providers []common.Account
	spec      spectypes.Spec
	plan      plantypes.Plan
	consumer  common.Account
}

func setupForConflictTests(t *testing.T, numOfProviders int) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	// init keepers state
	var balance int64 = 100000

	// setup consumer
	ts.consumer = common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	// setup providers
	for i := 0; i < numOfProviders; i++ {
		ts.Providers = append(ts.Providers, common.CreateNewAccount(ts.ctx, *ts.keepers, balance))
	}

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	var stake int64 = 1000

	// subscribe consumer
	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, ts.consumer, ts.plan.Index)

	// stake providers
	for _, provider := range ts.Providers {
		common.StakeAccount(t, ts.ctx, *ts.keepers, *ts.servers, provider, ts.spec, stake)
	}

	// advance for the staking to be valid
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	return ts
}
