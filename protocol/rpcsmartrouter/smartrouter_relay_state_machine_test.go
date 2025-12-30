package rpcsmartrouter

import (
	context "context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	lavasession "github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
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

type SmartRouterRelaySenderMock struct {
	retValue    error
	tickerValue time.Duration
}

func (srsm *SmartRouterRelaySenderMock) getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration) {
	if srsm.tickerValue != 0 {
		return time.Second * 50000, srsm.tickerValue
	}
	return time.Second * 50000, 100 * time.Millisecond
}

// SmartRouterRelaySenderMockWithTimeout is a mock with configurable processing timeout
type SmartRouterRelaySenderMockWithTimeout struct {
	SmartRouterRelaySenderMock
	processingTimeout time.Duration
}

func (srsm *SmartRouterRelaySenderMockWithTimeout) getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration) {
	return srsm.processingTimeout, 10 * time.Second
}

func (srsm *SmartRouterRelaySenderMock) GetChainIdAndApiInterface() (string, string) {
	return "testUno", "testDos"
}

func (srsm *SmartRouterRelaySenderMock) ParseRelay(
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := relaycore.NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, relaycore.RelayProcessorMetrics, relaycore.RelayProcessorMetrics, relaycore.RelayRetriesManagerInstance, NewSmartRouterRelayStateMachine(ctx, usedProviders, &SmartRouterRelaySenderMock{retValue: nil}, protocolMessage, nil, false, relaycore.RelayProcessorMetrics), qos.NewQoSManager())

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
				relaycore.SendProtocolError(relayProcessor, "lava@test", time.Millisecond*1, fmt.Errorf("bad"))
			case 1:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				relaycore.SendNodeError(relayProcessor, "lava2@test", time.Millisecond*1)
			case 2:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				relaycore.SendNodeError(relayProcessor, "lava2@test", time.Millisecond*1)
			case 3:
				require.False(t, task.IsDone())
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				relaycore.SendSuccessResp(relayProcessor, "lava4@test", time.Millisecond*1)
			case 4:
				require.True(t, task.IsDone())
				results, _ := relayProcessor.HasRequiredNodeResults(1)
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := relaycore.NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, relaycore.RelayProcessorMetrics, relaycore.RelayProcessorMetrics, relaycore.RelayRetriesManagerInstance, NewSmartRouterRelayStateMachine(ctx, usedProviders, &SmartRouterRelaySenderMock{retValue: nil, tickerValue: 10 * time.Second}, protocolMessage, nil, false, relaycore.RelayProcessorMetrics), qos.NewQoSManager())

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
				require.Error(t, task.Err)
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)
		relayProcessor := relaycore.NewRelayProcessor(
			ctx,
			common.DefaultQuorumParams,
			consistency,
			relaycore.RelayProcessorMetrics,
			relaycore.RelayProcessorMetrics,
			relaycore.RelayRetriesManagerInstance,
			NewSmartRouterRelayStateMachine(
				ctx,
				usedProviders,
				&SmartRouterRelaySenderMock{retValue: nil, tickerValue: 10 * time.Second},
				protocolMessage,
				nil,
				false,
				relaycore.RelayProcessorMetrics,
			),
			qos.NewQoSManager(),
		)

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
				relaycore.SendNodeErrorJsonRpc(relayProcessor, "lava2@test", time.Millisecond*1)
			case 1:
				require.False(t, task.IsDone())
				require.True(t,
					lavaslices.ContainsPredicate(
						task.RelayState.GetProtocolMessage().GetExtensions(),
						func(predicate *spectypes.Extension) bool { return predicate.Name == "archive" }),
				)
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				relaycore.SendSuccessRespJsonRpc(relayProcessor, "lava4@test", time.Millisecond*1)
			case 2:
				require.True(t, task.IsDone())
				results, _ := relayProcessor.HasRequiredNodeResults(1)
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

// TestProcessingContextTimeoutEnforcement tests that no new retries are spawned after processingCtx expires
// This test validates the fix for the bug where processingCtx.Done() case was not being selected,
// causing 30+ seconds of wasted retries even though the context had expired.
func TestProcessingContextTimeoutEnforcement(t *testing.T) {
	t.Run("NoRetriesAfterTimeout", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)

		// Create mock with SHORT processing timeout (100ms) to test timeout enforcement
		mockSender := &SmartRouterRelaySenderMockWithTimeout{
			processingTimeout: 100 * time.Millisecond, // 100ms processing timeout
		}

		relayProcessor := relaycore.NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, relaycore.RelayProcessorMetrics, relaycore.RelayProcessorMetrics, relaycore.RelayRetriesManagerInstance, NewSmartRouterRelayStateMachine(ctx, usedProviders, mockSender, protocolMessage, nil, false, relaycore.RelayProcessorMetrics), qos.NewQoSManager())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5) // Overall test timeout
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}}

		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)

		taskNumber := 0
		startTime := time.Now()

		for task := range relayTaskChannel {
			elapsed := time.Since(startTime)
			t.Logf("Task %d received at %v", taskNumber, elapsed)

			if task.IsDone() {
				t.Logf("Task %d is Done (error: %v), ending test", taskNumber, task.Err)
				break
			}

			// Add provider and simulate failure (no results)
			usedProviders.AddUsed(consumerSessionsMap, nil)
			relayProcessor.UpdateBatch(nil)

			// Simulate various errors to trigger retries
			if taskNumber%2 == 0 {
				relaycore.SendProtocolError(relayProcessor, "lava@test", time.Millisecond*1, fmt.Errorf("simulated protocol error"))
			} else {
				relaycore.SendNodeError(relayProcessor, "lava@test", time.Millisecond*1)
			}

			// Sleep slightly to allow WaitForResults to detect the error and return
			time.Sleep(10 * time.Millisecond)
			taskNumber++
		}

		elapsedTime := time.Since(startTime)

		t.Logf("Total tasks spawned: %d", taskNumber)
		t.Logf("Total elapsed time: %v", elapsedTime)

		// CRITICAL ASSERTION: After 100ms timeout, no new tasks should be spawned
		// We expect very few tasks (1-5) because the timeout should stop retries quickly
		// Before the fix, this would spawn 10+ tasks over several seconds
		require.LessOrEqual(t, taskNumber, 5,
			"Expected maximum 5 tasks before timeout, but got %d. This suggests processingCtx timeout is not being enforced!",
			taskNumber)

		// Verify timeout was respected (should be close to 100ms, not multiple seconds)
		require.Less(t, elapsedTime, 500*time.Millisecond,
			"Expected test to complete within 500ms (timeout + buffer), but took %v. This suggests retries continued after timeout!",
			elapsedTime)
	})
}

// TestProcessingContextStillValidAllowsRetries tests that retries continue when context is still valid
// This ensures our fix doesn't break the normal retry mechanism
func TestProcessingContextStillValidAllowsRetries(t *testing.T) {
	t.Run("RetriesContinueWhenContextValid", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)

		// LONG processing timeout (10 seconds) - retries should continue
		relayProcessor := relaycore.NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, relaycore.RelayProcessorMetrics, relaycore.RelayProcessorMetrics, relaycore.RelayRetriesManagerInstance, NewSmartRouterRelayStateMachine(ctx, usedProviders, &SmartRouterRelaySenderMock{retValue: nil, tickerValue: 10 * time.Second}, protocolMessage, nil, false, relaycore.RelayProcessorMetrics), qos.NewQoSManager())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2) // Test timeout
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}}

		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)

		taskNumber := 0

		for task := range relayTaskChannel {
			t.Logf("Task %d received", taskNumber)

			if taskNumber >= 5 {
				// After 5 retries, send success to end the test
				usedProviders.AddUsed(consumerSessionsMap, nil)
				relayProcessor.UpdateBatch(nil)
				relaycore.SendSuccessResp(relayProcessor, "lava@test", time.Millisecond*1)
				time.Sleep(20 * time.Millisecond)

				if task.IsDone() {
					t.Log("Got Done task after success, test complete")
					break
				}
				taskNumber++
				continue
			}

			if task.IsDone() {
				t.Logf("Task %d is Done unexpectedly (error: %v)", taskNumber, task.Err)
				break
			}

			// Simulate failure to trigger retry
			usedProviders.AddUsed(consumerSessionsMap, nil)
			relayProcessor.UpdateBatch(nil)
			relaycore.SendNodeError(relayProcessor, "lava@test", time.Millisecond*1)
			time.Sleep(10 * time.Millisecond)

			taskNumber++
		}

		// ASSERTION: With valid context, we should have been able to retry multiple times
		require.GreaterOrEqual(t, taskNumber, 5,
			"Expected at least 5 retries when context is valid, but only got %d", taskNumber)
	})
}

// TestProcessingContextRaceCondition tests the exact race condition from the bug:
// When both gotResults and processingCtx.Done() are ready simultaneously,
// we should respect the timeout and not spawn new retries
func TestProcessingContextRaceCondition(t *testing.T) {
	t.Run("TimeoutWinsRaceWithGotResults", func(t *testing.T) {
		ctx := context.Background()
		serverHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		consistency := relaycore.NewConsistency(specId)
		usedProviders := lavasession.NewUsedProviders(nil)

		// Short timeout (200ms) to create the race condition
		mockSender := &SmartRouterRelaySenderMockWithTimeout{
			processingTimeout: 200 * time.Millisecond,
		}

		relayProcessor := relaycore.NewRelayProcessor(ctx, common.DefaultQuorumParams, consistency, relaycore.RelayProcessorMetrics, relaycore.RelayProcessorMetrics, relaycore.RelayRetriesManagerInstance, NewSmartRouterRelayStateMachine(ctx, usedProviders, mockSender, protocolMessage, nil, false, relaycore.RelayProcessorMetrics), qos.NewQoSManager())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.NoError(t, ctx.Err())
		require.Nil(t, canUse)

		consumerSessionsMap := lavasession.ConsumerSessionsMap{"lava@test": &lavasession.SessionInfo{}}

		relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
		require.NoError(t, err)

		startTime := time.Now()
		taskTimestamps := []time.Duration{}
		tasksAfterTimeout := 0

		for task := range relayTaskChannel {
			elapsed := time.Since(startTime)
			taskTimestamps = append(taskTimestamps, elapsed)
			taskNum := len(taskTimestamps)

			t.Logf("Task %d at %v", taskNum, elapsed)

			// Count tasks spawned AFTER the 200ms timeout
			if elapsed > 200*time.Millisecond {
				tasksAfterTimeout++
				t.Logf("⚠️  Task %d spawned AFTER timeout! (%v after start)", taskNum, elapsed)
			}

			if task.IsDone() {
				t.Logf("Task %d is Done, test ending", taskNum)
				break
			}

			// Rapidly fail to create the race condition
			usedProviders.AddUsed(consumerSessionsMap, nil)
			relayProcessor.UpdateBatch(nil)
			relaycore.SendNodeError(relayProcessor, "lava@test", time.Millisecond*1)
			// Small sleep to let WaitForResults return
			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("Task timeline: %v", taskTimestamps)
		t.Logf("Tasks spawned after 200ms timeout: %d", tasksAfterTimeout)

		// CRITICAL ASSERTION: No tasks should be spawned after the 200ms timeout
		// Before the fix, tasks would continue spawning for seconds after timeout
		require.Equal(t, 0, tasksAfterTimeout,
			"Expected ZERO tasks after 200ms timeout, but %d tasks were spawned after timeout! This is the bug we're fixing.",
			tasksAfterTimeout)
	})
}
