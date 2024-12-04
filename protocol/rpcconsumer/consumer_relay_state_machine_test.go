package rpcconsumer

import (
	context "context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	common "github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	lavasession "github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

type PolicySt struct {
	addons       []string
	extensions   []string
	apiInterface string
}

func (a PolicySt) GetSupportedAddons(string) ([]string, error) {
	return a.addons, nil
}

func (a PolicySt) GetSupportedExtensions(string) ([]epochstoragetypes.EndpointService, error) {
	ret := []epochstoragetypes.EndpointService{}
	for _, ext := range a.extensions {
		ret = append(ret, epochstoragetypes.EndpointService{Extension: ext, ApiInterface: a.apiInterface})
	}
	return ret, nil
}

type ConsumerRelaySenderMock struct {
	retValue    error
	tickerValue time.Duration
}

func (crsm *ConsumerRelaySenderMock) getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration) {
	if crsm.tickerValue != 0 {
		return time.Second * 50000, crsm.tickerValue
	}
	return time.Second * 50000, 100 * time.Millisecond
}

func (crsm *ConsumerRelaySenderMock) GetChainIdAndApiInterface() (string, string) {
	return "testUno", "testDos"
}

func (crsm *ConsumerRelaySenderMock) ParseRelay(
	ctx context.Context,
	url string,
	req string,
	connectionType string,
	dappID string,
	consumerIp string,
	metadata []pairingtypes.Metadata,
) (protocolMessage chainlib.ProtocolMessage, err error) {
	foundArchive := false
	for _, md := range metadata {
		if md.Value == "archive" {
			foundArchive = true
		}
	}
	if !foundArchive {
		utils.LavaFormatFatal("misuse in mocked parse relay", nil)
	}
	serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the incoming request and provide the desired response
		w.WriteHeader(http.StatusOK)
	})
	specId := "NEAR"
	chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceJsonRPC, serverHandler, nil, "../../", []string{"archive"})
	defer closeServer()
	policy := PolicySt{
		addons:       []string{},
		extensions:   []string{"archive"},
		apiInterface: spectypes.APIInterfaceJsonRPC,
	}
	chainParser.SetPolicy(policy, specId, spectypes.APIInterfaceJsonRPC)
	chainMsg, err := chainParser.ParseMsg(url, []byte(req), connectionType, metadata, extensionslib.ExtensionInfo{LatestBlock: 0, ExtensionOverride: []string{"archive"}})
	relayRequestData := lavaprotocol.NewRelayData(ctx, connectionType, url, []byte(req), 0, -2, spectypes.APIInterfaceJsonRPC, chainMsg.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMsg), common.GetExtensionNames(chainMsg.GetExtensions()))
	protocolMessage = chainlib.NewProtocolMessage(chainMsg, nil, relayRequestData, dappID, consumerIp)
	return protocolMessage, nil
}

func TestConsumerStateMachineHappyFlow(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		dappId := "dapp"
		consumerIp := "123.11"
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, dappId, consumerIp)
		consistency := NewConsumerConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, consistency, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &ConsumerRelaySenderMock{retValue: nil}, protocolMessage, nil, false, relayProcessorMetrics))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}

		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)
		taskNumber := 0
		for task := range relayTaskChannel {
			switch taskNumber {
			case 0:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendProtocolError(relayProcessor, "lava@test", time.Millisecond*1, fmt.Errorf("bad"))
			case 1:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendNodeError(relayProcessor, "lava2@test", time.Millisecond*1)
			case 2:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendNodeError(relayProcessor, "lava2@test", time.Millisecond*1)
			case 3:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendSuccessResp(relayProcessor, "lava4@test", time.Millisecond*1)
			case 4:
				require.True(t, task.IsDone())
				results, _ := relayProcessor.HasRequiredNodeResults()
				require.True(t, results)
				returnedResult, err := relayProcessor.ProcessingResult()
				require.NoError(t, err)
				require.Equal(t, string(returnedResult.Reply.Data), "ok")
				require.Equal(t, http.StatusOK, returnedResult.StatusCode)
				return // end test.
			}
			taskNumber++
		}
	})
}

func TestConsumerStateMachineExhaustRetries(t *testing.T) {
	t.Run("retries", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "LAV1"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceRest, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		chainMsg, err := chainParser.ParseMsg("/cosmos/base/tendermint/v1beta1/blocks/17", nil, http.MethodGet, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		dappId := "dapp"
		consumerIp := "123.11"
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, nil, dappId, consumerIp)
		consistency := NewConsumerConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(ctx, 1, consistency, relayProcessorMetrics, relayProcessorMetrics, relayRetriesManagerInstance, NewRelayStateMachine(ctx, usedProviders, &ConsumerRelaySenderMock{retValue: nil, tickerValue: 100 * time.Second}, protocolMessage, nil, false, relayProcessorMetrics))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())

		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)
		taskNumber := 0
		for task := range relayTaskChannel {
			switch taskNumber {
			case 0, 1, 2, 3:
				require.False(t, task.IsDone())
				relayProcessor.UpdateBatch(fmt.Errorf("failed sending message"))
			case 4:
				require.True(t, task.IsDone())
				require.Error(t, task.err)
				return
			}
			taskNumber++
		}
	})
}

func TestConsumerStateMachineArchiveRetry(t *testing.T) {
	t.Run("retries_archive", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			w.WriteHeader(http.StatusOK)
		})
		specId := "NEAR"
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, specId, spectypes.APIInterfaceJsonRPC, serverHandler, nil, "../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)

		params, _ := json.Marshal([]string{"5NFtBbExnjk4TFXpfXhJidcCm5KYPk7QCY51nWiwyQNU"})
		id, _ := json.Marshal(1)
		reqBody := rpcclient.JsonrpcMessage{
			Version: "2.0",
			Method:  "block", // Query latest block
			Params:  params,  // Use "final" to get the latest final block
			ID:      id,
		}

		// Convert request to JSON
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			log.Fatalf("Error marshalling request: %v", err)
		}

		chainMsg, err := chainParser.ParseMsg("", jsonData, http.MethodPost, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
		require.NoError(t, err)
		dappId := "dapp"
		consumerIp := "123.11"
		reqBlock, _ := chainMsg.RequestedBlock()
		var seenBlock int64 = 0

		relayRequestData := lavaprotocol.NewRelayData(ctx, http.MethodPost, "", jsonData, seenBlock, reqBlock, spectypes.APIInterfaceJsonRPC, chainMsg.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMsg), common.GetExtensionNames(chainMsg.GetExtensions()))
		protocolMessage := chainlib.NewProtocolMessage(chainMsg, nil, relayRequestData, dappId, consumerIp)
		consistency := NewConsumerConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := NewRelayProcessor(
			ctx,
			1,
			consistency,
			relayProcessorMetrics,
			relayProcessorMetrics,
			relayRetriesManagerInstance,
			NewRelayStateMachine(
				ctx,
				usedProviders,
				&ConsumerRelaySenderMock{retValue: nil, tickerValue: 100 * time.Second},
				protocolMessage,
				nil,
				false,
				relayProcessorMetrics,
			))

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}, "lava@test2": &lavasession.SessionInfo{}}
		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)
		taskNumber := 0
		for task := range relayTaskChannel {
			switch taskNumber {
			case 0:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendNodeErrorJsonRpc(relayProcessor, "lava2@test", time.Millisecond*1)
			case 1:
				require.False(t, task.IsDone())
				require.True(t,
					lavaslices.ContainsPredicate(
						task.relayState.GetProtocolMessage().GetExtensions(),
						func(predicate *spectypes.Extension) bool { return predicate.Name == "archive" }),
				)
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				sendSuccessRespJsonRpc(relayProcessor, "lava4@test", time.Millisecond*1)
			case 2:
				require.True(t, task.IsDone())
				results, _ := relayProcessor.HasRequiredNodeResults()
				require.True(t, results)
				returnedResult, err := relayProcessor.ProcessingResult()
				require.NoError(t, err)
				require.Equal(t, string(returnedResult.Reply.Data), `{"jsonrpc":"2.0","id":1,"result":{"result":"success"}}`)
				require.Equal(t, http.StatusOK, returnedResult.StatusCode)
				fmt.Println(relayProcessor.GetProtocolMessage().GetExtensions())
				return // end test.
			}
			taskNumber++
		}
	})
}
