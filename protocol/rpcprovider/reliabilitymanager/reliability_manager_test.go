package reliabilitymanager_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
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

type testStruct struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	Providers []common.Account
	spec      spectypes.Spec
	plan      plantypes.Plan
	consumer  common.Account
}

func setupForConflictTests(t *testing.T, numOfProviders int, specID string) testStruct {
	ts := testStruct{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	// init keepers state
	var balance int64 = 100000000000

	// setup consumer
	ts.consumer = common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	// setup providers
	for i := 0; i < numOfProviders; i++ {
		ts.Providers = append(ts.Providers, common.CreateNewAccount(ts.ctx, *ts.keepers, balance))
	}
	getToTopMostPath := "../../../"
	sdkContext := sdk.UnwrapSDKContext(ts.ctx)
	spec, err := testkeeper.GetASpec(specID, getToTopMostPath, &sdkContext, &ts.keepers.Spec)
	if err != nil {
		require.NoError(t, err)
	}
	ts.spec = spec
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	ts.plan = common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)

	var stake int64 = 50000000000

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

type mockTx struct {
	callbackCommit func(voteID string, voteData *reliabilitymanager.VoteData)
	callbackReveal func(voteID string, voteData *reliabilitymanager.VoteData)
}

func (m mockTx) SendVoteCommitment(voteID string, voteData *reliabilitymanager.VoteData) error {
	m.callbackCommit(voteID, voteData)
	return nil
}
func (m mockTx) SendVoteReveal(voteID string, voteData *reliabilitymanager.VoteData) error {
	m.callbackReveal(voteID, voteData)
	return nil
}

func TestFullFlowReliabilityConflict(t *testing.T) {
	specId := "LAV1"
	ts := setupForConflictTests(t, 3, specId)
	// consumer
	consumer_sk, consumer_address := ts.consumer.SK, ts.consumer.Addr
	// provider
	provider_sk, provider_address := ts.Providers[0].SK, ts.Providers[0].Addr
	// second provider (DR)
	providerDR_sk, providerDR_address := ts.Providers[1].SK, ts.Providers[1].Addr
	unwrapedCtx := sdk.UnwrapSDKContext(ts.ctx)
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
	relayRequestData := lavaprotocol.NewRelayData(ts.ctx, "GET", "/blocks/latest", []byte{}, spectypes.LATEST_BLOCK, spectypes.APIInterfaceRest, metadataValue)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := lavaprotocol.ConstructRelayRequest(ts.ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []byte("stubbytes"))
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
	relayRequestDataDR := lavaprotocol.NewRelayData(ts.ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata)
	relayDR, err := lavaprotocol.ConstructRelayRequest(ts.ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []byte("stubbytes"))
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
	conflict := lavaprotocol.VerifyReliabilityResults(ts.ctx, relayResult, relayResultDR)
	require.NotNil(t, conflict)
	msg := conflicttypes.NewMsgDetection(consumer_address.String(), nil, conflict, nil)
	_, err = ts.servers.ConflictServer.Detection(ts.ctx, msg)
	require.Nil(t, err)
	lastEvent := sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1]
	event := terderminttypes.Event(lastEvent)
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName, event.Type)
	votingProvider := ts.Providers[2]

	voteParams, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
	require.Nil(t, err)
	require.Equal(t, specId, voteParams.ChainID)
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, replyDataBuf)
	})
	chainParser, chainProxy, chainFetcher, closeServer, err := chainlib.CreateChainLibMocks(ts.ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../../")
	if closeServer != nil {
		defer closeServer()
	}
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)
	commitCalled := false
	revealCalled := false
	sendVoteCommit := func(voteID string, vote *reliabilitymanager.VoteData) {
		commitCalled = true
		msg := conflicttypes.NewMsgConflictVoteCommit(votingProvider.Addr.String(), voteID, vote.CommitHash)
		_, err := ts.servers.ConflictServer.ConflictVoteCommit(ts.ctx, msg)
		require.NoError(t, err)
	}
	sendVoteReveal := func(voteID string, vote *reliabilitymanager.VoteData) {
		revealCalled = true
		msg := conflicttypes.NewMsgConflictVoteReveal(votingProvider.Addr.String(), voteID, vote.Nonce, vote.RelayDataHash)
		_, err := ts.servers.ConflictServer.ConflictVoteReveal(ts.ctx, msg)
		require.NoError(t, err)
	}
	mockTxSender := mockTx{callbackCommit: sendVoteCommit, callbackReveal: sendVoteReveal}

	// provider 3 now needs to vote
	reliabilityManage := reliabilitymanager.NewReliabilityManager(nil, mockTxSender, votingProvider.Addr.String(), chainProxy, chainParser)
	// trigger commit event handling
	err = reliabilityManage.VoteHandler(voteParams, 1)
	require.NoError(t, err)
	// commit called
	require.True(t, commitCalled)
	for uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()) < voteParams.VoteDeadline {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	event = terderminttypes.Event(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1])
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteRevealEventName, event.Type)

	voteID, voteDeadline, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
	require.NoError(t, err)
	require.Equal(t, voteParams.VoteID, voteID)

	// trigger reveal event handling
	reliabilityManage.VoteHandler(&reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: voteDeadline, ParamsType: reliabilitymanager.RevealVoteType}, 1)
	// commit called
	require.True(t, revealCalled)

	// resolve vote
	for uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()) < voteDeadline {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
	event = terderminttypes.Event(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.ctx).EventManager().Events())-1])
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteResolvedEventName, event.Type)
}
