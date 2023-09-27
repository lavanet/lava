package reliabilitymanager_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	terderminttypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/statetracker"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type mockFilter struct{}

func (m mockFilter) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filtered []pairingtypes.Metadata, overwriteReqBlock string, ignoredMetadata []pairingtypes.Metadata) {
	return metadata, "", []pairingtypes.Metadata{}
}

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
	relayRequestData := lavaprotocol.NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue, "", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
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
	relayRequestDataDR := lavaprotocol.NewRelayData(ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata, "", nil)
	relayDR, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relayDR.RelaySession.ContentHash, sigs.HashMsg(relayDR.RelayData.GetContentHashData())))
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

	conflict := lavaprotocol.VerifyReliabilityResults(ctx, relayResult, relayResultDR, nil, mockFilter{})
	require.Nil(t, conflict)
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

type txSenderMock struct {
	cb func() error
}

func (tsm *txSenderMock) TxSenderConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict) error {
	if tsm.cb == nil {
		return fmt.Errorf("No cb")
	}
	return tsm.cb()
}

func TestFullFlowReliabilityConflict(t *testing.T) {
	specId := "LAV1"
	ts := chainlib.SetupForTests(t, 3, specId, "../../../")
	// consumer
	consumer_sk, consumer_address := ts.Consumer.SK, ts.Consumer.Addr
	// provider
	provider_sk, provider_address := ts.Providers[0].SK, ts.Providers[0].Addr
	// second provider (DR)
	providerDR_sk, providerDR_address := ts.Providers[1].SK, ts.Providers[1].Addr
	unwrapedCtx := sdk.UnwrapSDKContext(ts.Ctx)
	epoch := int64(ts.Keepers.Epochstorage.GetEpochStart(unwrapedCtx))
	replyDataBuf := []byte(`{"reply": "REPLY-STUB"}`)
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(replyDataBuf))
	})
	chainParser, chainProxy, chainFetcher, closeServer, err := chainlib.CreateChainLibMocks(ts.Ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../../", nil)
	if closeServer != nil {
		defer closeServer()
	}
	require.NoError(t, err)
	require.NotNil(t, chainParser)
	require.NotNil(t, chainProxy)
	require.NotNil(t, chainFetcher)

	consumerSesssionWithProvider := &lavasession.ConsumerSessionsWithProvider{}
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
		Client:                      consumerSesssionWithProvider,
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
	chainMessage, err := chainParser.ParseMsg("/blocks/latest", []byte{}, "GET", metadataValue, 0)
	require.NoError(t, err)
	reqBlock, _ := chainMessage.RequestedBlock()
	relayRequestData := lavaprotocol.NewRelayData(ts.Ctx, "GET", "/blocks/latest", []byte{}, reqBlock, spectypes.APIInterfaceRest, chainMessage.GetRPCMessage().GetHeaders(), "", nil)

	relay, err := lavaprotocol.ConstructRelayRequest(ts.Ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
	latestBlock := int64(123)
	// provider handling the response
	finalizedBlockHashes := map[int64]interface{}{latestBlock: "AAA"}

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

	relayExchange := pairingtypes.NewRelayExchange(*relay, *reply)
	allDataHash := sigs.HashMsg(relayExchange.DataToSign())
	utils.LavaFormatDebug("honest provider allDataHash", utils.Attribute{Key: "hash", Value: fmt.Sprintf("%#v", allDataHash)})

	// now send this to another provider
	relayRequestDataDR := lavaprotocol.NewRelayData(ts.Ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata, "", nil)
	relayDR, err := lavaprotocol.ConstructRelayRequest(ts.Ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
	require.Nil(t, err)

	// provider checks
	extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
	require.Nil(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relayDR.RelaySession.ContentHash, sigs.HashMsg(relayDR.RelayData.GetContentHashData())))
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

	conflict := lavaprotocol.VerifyReliabilityResults(ts.Ctx, relayResult, relayResultDR, chainMessage.GetApiCollection(), chainParser)
	require.NotNil(t, conflict)
	msg := conflicttypes.NewMsgDetection(consumer_address.String(), nil, conflict, nil)

	cb := func() error {
		_, err = ts.Servers.ConflictServer.Detection(ts.Ctx, msg)
		require.Nil(t, err)
		return err
	}
	txm := &txSenderMock{cb: cb}
	consumerStateTracker := &statetracker.ConsumerStateTracker{ConsumerTxSenderInf: txm}
	err = consumerStateTracker.TxConflictDetection(ts.Ctx, nil, conflict, nil, singleConsumerSession2.Client) // report first time
	require.NoError(t, err)
	err = consumerStateTracker.TxConflictDetection(ts.Ctx, nil, conflict, nil, singleConsumerSession2.Client) // make sure we dont report 2nd time
	require.NoError(t, err)

	_, err = ts.Servers.ConflictServer.Detection(ts.Ctx, msg) // validate reporting 2nd time returns an error.
	require.Error(t, err)

	lastEvent := sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events())-1]
	event := terderminttypes.Event(lastEvent)
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName, event.Type)
	votingProvider := ts.Providers[2]

	voteParams, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
	require.Nil(t, err)
	require.Equal(t, specId, voteParams.ChainID)

	commitCalled := false
	revealCalled := false
	sendVoteCommit := func(voteID string, vote *reliabilitymanager.VoteData) {
		commitCalled = true
		msg := conflicttypes.NewMsgConflictVoteCommit(votingProvider.Addr.String(), voteID, vote.CommitHash)
		_, err := ts.Servers.ConflictServer.ConflictVoteCommit(ts.Ctx, msg)
		require.NoError(t, err)
	}
	sendVoteReveal := func(voteID string, vote *reliabilitymanager.VoteData) {
		revealCalled = true
		msg := conflicttypes.NewMsgConflictVoteReveal(votingProvider.Addr.String(), voteID, vote.Nonce, vote.RelayDataHash)
		_, err := ts.Servers.ConflictServer.ConflictVoteReveal(ts.Ctx, msg)
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
	for uint64(sdk.UnwrapSDKContext(ts.Ctx).BlockHeight()) < voteParams.VoteDeadline {
		ts.Ctx = testkeeper.AdvanceEpoch(ts.Ctx, ts.Keepers)
	}

	event = terderminttypes.Event(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events())-1])
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteRevealEventName, event.Type)

	voteID, voteDeadline, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
	require.NoError(t, err)
	require.Equal(t, voteParams.VoteID, voteID)

	// trigger reveal event handling
	reliabilityManage.VoteHandler(&reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: voteDeadline, ParamsType: reliabilitymanager.RevealVoteType}, 1)
	// commit called
	require.True(t, revealCalled)

	// resolve vote
	for uint64(sdk.UnwrapSDKContext(ts.Ctx).BlockHeight()) < voteDeadline {
		ts.Ctx = testkeeper.AdvanceEpoch(ts.Ctx, ts.Keepers)
	}
	event = terderminttypes.Event(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events())-1])
	require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteResolvedEventName, event.Type)
	require.True(t, func() bool {
		for _, attr := range event.Attributes {
			if attr.Key == "winner" {
				require.Equal(t, provider_address.String(), attr.Value)
				return true
			}
		}
		return false
	}())
}
