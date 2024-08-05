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
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavaprotocol"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/v2/protocol/statetracker"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

type mockFilter struct{}

func (m mockFilter) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filtered []pairingtypes.Metadata, overwriteReqBlock string, ignoredMetadata []pairingtypes.Metadata) {
	return metadata, "", []pairingtypes.Metadata{}
}

func TestFullFlowReliabilityCompare(t *testing.T) {
	t.Run("test", func(t *testing.T) {
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
			CuSum:              20,
			LatestRelayCu:      10, // set by GetSessions cuNeededForSession
			QoSInfo:            lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
			SessionId:          123,
			Parent:             nil,
			RelayNum:           1,
			LatestBlock:        epoch,
			EndpointConnection: nil,
			BlockListed:        false, // if session lost sync we blacklist it.
		}
		singleConsumerSession2 := &lavasession.SingleConsumerSession{
			CuSum:              200,
			LatestRelayCu:      100, // set by GetSessions cuNeededForSession
			QoSInfo:            lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
			SessionId:          456,
			Parent:             nil,
			RelayNum:           5,
			LatestBlock:        epoch,
			EndpointConnection: nil,
			BlockListed:        false, // if session lost sync we blacklist it.
		}
		metadataValue := make([]pairingtypes.Metadata, 1)
		metadataValue[0] = pairingtypes.Metadata{
			Name:  "banana",
			Value: "55",
		}
		relayRequestData := lavaprotocol.NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue, "", nil)
		require.Equal(t, relayRequestData.Metadata, metadataValue)
		relay, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
		require.NoError(t, err)

		// provider checks
		extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
		require.NoError(t, err)
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
		err = lavaprotocol.VerifyRelayReply(ctx, reply, relay, provider_address.String())
		require.NoError(t, err)
		_, _, err = lavaprotocol.VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
		require.NoError(t, err)

		relayResult := &common.RelayResult{
			Request:      relay,
			Reply:        reply,
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider_address.String()},
			ReplyServer:  nil,
			Finalized:    true,
		}

		// now send this to another provider
		relayRequestDataDR := lavaprotocol.NewRelayData(ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, 0, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata, "", nil)
		relayDR, err := lavaprotocol.ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
		require.NoError(t, err)

		// provider checks
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
		require.NoError(t, err)
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
		err = lavaprotocol.VerifyRelayReply(ctx, replyDR, relayDR, providerDR_address.String())
		require.NoError(t, err)
		_, _, err = lavaprotocol.VerifyFinalizationData(replyDR, relayDR, providerDR_address.String(), consumer_address, int64(0), 0)
		require.NoError(t, err)
		relayResultDR := &common.RelayResult{
			Request:      relayDR,
			Reply:        replyDR,
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider_address.String()},
			ReplyServer:  nil,
			Finalized:    true,
		}

		conflict := lavaprotocol.VerifyReliabilityResults(ctx, relayResult, relayResultDR, nil, mockFilter{})
		require.Nil(t, conflict)
	})
}

type mockTx struct {
	callbackCommit func(voteID string, voteData *reliabilitymanager.VoteData, specID string)
	callbackReveal func(voteID string, voteData *reliabilitymanager.VoteData, specID string)
}

func (m mockTx) SendVoteCommitment(voteID string, voteData *reliabilitymanager.VoteData, specID string) error {
	m.callbackCommit(voteID, voteData, specID)
	return nil
}

func (m mockTx) SendVoteReveal(voteID string, voteData *reliabilitymanager.VoteData, specID string) error {
	m.callbackReveal(voteID, voteData, specID)
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
	t.Run("test", func(t *testing.T) {
		specId := "LAV1"
		ts := chainlib.SetupForTests(t, 3, specId, "../../../")
		// consumer
		consumer_sk, consumer_address := ts.Consumer.SK, ts.Consumer.Addr
		// provider
		provider_sk, provider_address := ts.Providers[0].SK, ts.Providers[0].Addr
		// second provider (DR)
		providerDR_sk, providerDR_address := ts.Providers[1].SK, ts.Providers[1].Addr
		unwrappedCtx := sdk.UnwrapSDKContext(ts.Ctx)
		epoch := int64(ts.Keepers.Epochstorage.GetEpochStart(unwrappedCtx))
		replyDataBuf := []byte(`{"reply": "REPLY-STUB"}`)
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, string(replyDataBuf))
		})
		chainParser, chainProxy, chainFetcher, closeServer, _, err := chainlib.CreateChainLibMocks(ts.Ctx, specId, spectypes.APIInterfaceRest, serverHandler, "../../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		require.NotNil(t, chainParser)
		require.NotNil(t, chainProxy)
		require.NotNil(t, chainFetcher)

		consumerSesssionWithProvider := &lavasession.ConsumerSessionsWithProvider{}
		singleConsumerSession := &lavasession.SingleConsumerSession{
			CuSum:              20,
			LatestRelayCu:      10, // set by GetSessions cuNeededForSession
			QoSInfo:            lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
			SessionId:          123,
			Parent:             nil,
			RelayNum:           1,
			LatestBlock:        epoch,
			EndpointConnection: nil,
			BlockListed:        false, // if session lost sync we blacklist it.
		}
		singleConsumerSession2 := &lavasession.SingleConsumerSession{
			CuSum:              200,
			LatestRelayCu:      100, // set by GetSessions cuNeededForSession
			QoSInfo:            lavasession.QoSReport{LastQoSReport: &pairingtypes.QualityOfServiceReport{}},
			SessionId:          456,
			Parent:             consumerSesssionWithProvider,
			RelayNum:           5,
			LatestBlock:        epoch,
			EndpointConnection: nil,
			BlockListed:        false, // if session lost sync we blacklist it.
		}
		metadataValue := make([]pairingtypes.Metadata, 1)
		metadataValue[0] = pairingtypes.Metadata{
			Name:  "banana",
			Value: "55",
		}
		chainMessage, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/latest", []byte{}, "GET", metadataValue, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		reqBlock, _ := chainMessage.RequestedBlock()
		relayRequestData := lavaprotocol.NewRelayData(ts.Ctx, "GET", "/cosmos/base/tendermint/v1beta1/blocks/latest", []byte{}, 0, reqBlock, spectypes.APIInterfaceRest, chainMessage.GetRPCMessage().GetHeaders(), "", nil)

		relay, err := lavaprotocol.ConstructRelayRequest(ts.Ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
		require.NoError(t, err)

		// provider checks
		extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
		require.NoError(t, err)
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
		err = lavaprotocol.VerifyRelayReply(ts.Ctx, reply, relay, provider_address.String())
		require.NoError(t, err)
		_, _, err = lavaprotocol.VerifyFinalizationData(reply, relay, provider_address.String(), consumer_address, int64(0), 0)
		require.NoError(t, err)

		relayResult := &common.RelayResult{
			Request:      relay,
			Reply:        reply,
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider_address.String()},
			ReplyServer:  nil,
			Finalized:    true,
		}

		relayExchange := pairingtypes.NewRelayExchange(*relay, *reply)
		allDataHash := sigs.HashMsg(relayExchange.DataToSign())
		utils.LavaFormatDebug("honest provider allDataHash", utils.Attribute{Key: "hash", Value: fmt.Sprintf("%#v", allDataHash)})

		// now send this to another provider
		relayRequestDataDR := lavaprotocol.NewRelayData(ts.Ctx, relay.RelayData.ConnectionType, relay.RelayData.ApiUrl, relay.RelayData.Data, 0, relay.RelayData.RequestBlock, relay.RelayData.ApiInterface, relay.RelayData.Metadata, "", nil)
		relayDR, err := lavaprotocol.ConstructRelayRequest(ts.Ctx, consumer_sk, "lava", specId, relayRequestDataDR, providerDR_address.String(), singleConsumerSession2, epoch, []*pairingtypes.ReportedProvider{{Address: "stub"}})
		require.NoError(t, err)

		// provider checks
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(relayDR.RelaySession)
		require.NoError(t, err)
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
		err = lavaprotocol.VerifyRelayReply(ts.Ctx, replyDR, relayDR, providerDR_address.String())
		require.NoError(t, err)
		_, _, err = lavaprotocol.VerifyFinalizationData(replyDR, relayDR, providerDR_address.String(), consumer_address, int64(0), 0)
		require.NoError(t, err)
		relayResultDR := &common.RelayResult{
			Request:      relayDR,
			Reply:        replyDR,
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider_address.String()},
			ReplyServer:  nil,
			Finalized:    true,
		}

		conflict := lavaprotocol.VerifyReliabilityResults(ts.Ctx, relayResult, relayResultDR, chainMessage.GetApiCollection(), chainParser)
		require.NotNil(t, conflict)
		msg := conflicttypes.NewMsgDetection(consumer_address.String(), nil, conflict, nil)

		cb := func() error {
			_, err = ts.Servers.ConflictServer.Detection(ts.Ctx, msg)
			require.NoError(t, err)
			return err
		}
		txm := &txSenderMock{cb: cb}
		consumerStateTracker := &statetracker.ConsumerStateTracker{ConsumerTxSenderInf: txm}
		err = consumerStateTracker.TxConflictDetection(ts.Ctx, nil, conflict, nil, singleConsumerSession2.Parent) // report first time
		require.NoError(t, err)
		err = consumerStateTracker.TxConflictDetection(ts.Ctx, nil, conflict, nil, singleConsumerSession2.Parent) // make sure we dont report 2nd time
		require.NoError(t, err)

		_, err = ts.Servers.ConflictServer.Detection(ts.Ctx, msg) // validate reporting 2nd time returns an error.
		require.Error(t, err)

		lastEvent := sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events()[len(sdk.UnwrapSDKContext(ts.Ctx).EventManager().Events())-1]
		event := terderminttypes.Event(lastEvent)
		require.Equal(t, utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName, event.Type)
		votingProvider := ts.Providers[2]

		voteParams, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
		require.NoError(t, err)
		require.Equal(t, specId, voteParams.ChainID)

		commitCalled := false
		revealCalled := false
		sendVoteCommit := func(voteID string, vote *reliabilitymanager.VoteData, specID string) {
			commitCalled = true
			msg := conflicttypes.NewMsgConflictVoteCommit(votingProvider.Addr.String(), voteID, vote.CommitHash)
			_, err := ts.Servers.ConflictServer.ConflictVoteCommit(ts.Ctx, msg)
			require.NoError(t, err)
		}
		sendVoteReveal := func(voteID string, vote *reliabilitymanager.VoteData, specID string) {
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
	})
}
