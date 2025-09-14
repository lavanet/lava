package rpcprovider

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

type ProviderRelaySender interface {
	SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage chainlib.ChainMessageForSend, extensions []string) (relayReply *chainlib.RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error)
}

type ProviderStateMachine struct {
	relayRetriesManager lavaprotocol.RelayRetriesManagerInf
	chainId             string
	relaySender         ProviderRelaySender
	numberOfRetries     int
	testModeConfig      *TestModeConfig
}

func NewProviderStateMachine(chainId string, relayRetriesManager lavaprotocol.RelayRetriesManagerInf, relaySender ProviderRelaySender, numberOfRetries int, testModeConfig *TestModeConfig) *ProviderStateMachine {
	return &ProviderStateMachine{
		relayRetriesManager: relayRetriesManager,
		chainId:             chainId,
		relaySender:         relaySender,
		numberOfRetries:     numberOfRetries,
		testModeConfig:      testModeConfig,
	}
}

func (psm *ProviderStateMachine) SendNodeMessage(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) (*chainlib.RelayReplyWrapper, error) {
	hash, err := chainMsg.GetRawRequestHash()
	requestHashString := ""
	if err != nil {
		utils.LavaFormatWarning("Failed converting message to hash", err, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
	} else {
		requestHashString = string(hash)
	}

	var replyWrapper *chainlib.RelayReplyWrapper
	var isNodeError bool
	var errorMessage string
	for retryAttempt := 0; retryAttempt <= psm.numberOfRetries; retryAttempt++ {
		sendTime := time.Now()

		// Check if this is a test mode request
		isTestMode, _ := ctx.Value(TestModeContextKey{}).(bool)
		if isTestMode && psm.testModeConfig != nil && psm.testModeConfig.TestMode {
			replyWrapper = psm.generateTestResponse(ctx, chainMsg, request)
			err = nil // No error in test mode
		} else {
			// Original behavior - send to real node
			replyWrapper, _, _, _, _, err = psm.relaySender.SendNodeMsg(ctx, nil, chainMsg, request.RelayData.Extensions)
		}

		if err != nil {
			return nil, utils.LavaFormatError("Sending chainMsg failed", err, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return nil, utils.LavaFormatError("Relay Wrapper returned nil without an error", nil, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if debugLatency {
			utils.LavaFormatDebug("node reply received", utils.LogAttr("attempt", retryAttempt), utils.LogAttr("timeTaken", time.Since(sendTime)), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		// Check for node errors
		isNodeError, errorMessage = chainMsg.CheckResponseError(replyWrapper.RelayReply.Data, replyWrapper.StatusCode)

		// Failed fetching hash return the reply.
		if requestHashString == "" {
			utils.LavaFormatWarning("Failed to hash request, shouldn't happen", nil, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
			break // We can't perform the retries as we failed fetching the request hash.
		}
		if !isNodeError {
			// Successful relay, remove it from the cache if we have it and return a valid response.
			go psm.relayRetriesManager.RemoveHashFromCache(requestHashString)
			return replyWrapper, nil
		}

		// Check if this is an unsupported method error OR if we're using a default API and got an error
		isDefaultApi := false
		if chainMsg != nil && chainMsg.GetApi() != nil {
			isDefaultApi = strings.HasPrefix(chainMsg.GetApi().Name, chainlib.DefaultApiName)
		}
		//Anna - check with Avi
		if chainlib.IsUnsupportedMethodErrorMessage(errorMessage) {
			//if chainlib.IsUnsupportedMethodErrorMessage(errorMessage) || (isDefaultApi && isNodeError) {
			// Extract method name if available
			methodName := ""
			apiInterface := ""
			if chainMsg != nil && chainMsg.GetApi() != nil {
				methodName = chainMsg.GetApi().Name
				if chainMsg.GetApiCollection() != nil {
					apiInterface = chainMsg.GetApiCollection().CollectionData.ApiInterface
				}
			}

			// Comprehensive structured logging
			logMessage := "unsupported method error detected - returning error to consumer"
			if isDefaultApi && !chainlib.IsUnsupportedMethodErrorMessage(errorMessage) {
				logMessage = "default API error detected - returning error to consumer"
			}
			utils.LavaFormatInfo(logMessage,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("error", errorMessage),
				utils.LogAttr("method", methodName),
				utils.LogAttr("api_interface", apiInterface),
				utils.LogAttr("chain_id", psm.chainId),
				utils.LogAttr("url", request.RelayData.ApiUrl),
				utils.LogAttr("consumer_address", request.RelayData.GetMetadata()),
				utils.LogAttr("session_id", request.RelaySession.SessionId),
				utils.LogAttr("relay_num", request.RelaySession.RelayNum),
				utils.LogAttr("spec_id", request.RelaySession.SpecId),
				utils.LogAttr("cu_sum", request.RelaySession.CuSum),
				utils.LogAttr("request_block", request.RelayData.RequestBlock),
				utils.LogAttr("seen_block", request.RelayData.SeenBlock),
				utils.LogAttr("timestamp", time.Now().UTC()),
				utils.LogAttr("retry_attempt", retryAttempt),
				utils.LogAttr("request_data", string(request.RelayData.Data)),
				utils.LogAttr("status_code", replyWrapper.StatusCode),
				utils.LogAttr("is_default_api", isDefaultApi),
			)

			// Return an UnsupportedMethodError to the consumer so they don't increment their CU counter
			unsupportedError := chainlib.NewUnsupportedMethodError(errors.New(errorMessage), methodName)
			return nil, unsupportedError
		}

		// On the first retry, check if this hash has already failed previously
		if retryAttempt == 0 && psm.relayRetriesManager.CheckHashInCache(requestHashString) {
			utils.LavaFormatTrace("received node error, request hash was already in cache, skipping retry")
			break
		}
		utils.LavaFormatTrace("Errored Node Message, retrying message", utils.LogAttr("retry", retryAttempt))
	}

	if isNodeError && requestHashString != "" {
		utils.LavaFormatTrace("failed all relay retries for message", utils.LogAttr("hash", requestHashString))
		go psm.relayRetriesManager.AddHashToCache(requestHashString)
	}
	return replyWrapper, nil
}

// generateTestResponse generates a test response based on the configured probabilities
func (psm *ProviderStateMachine) generateTestResponse(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) *chainlib.RelayReplyWrapper {
	// Get the API method name
	apiMethod := ""
	if chainMsg != nil && chainMsg.GetApi() != nil {
		apiMethod = chainMsg.GetApi().Name
	}

	// Get test response configuration for this method
	testResponse, exists := psm.testModeConfig.Responses[apiMethod]
	if !exists {
		// Default response if method not configured
		testResponse = TestResponse{
			SuccessReply:       `{"jsonrpc":"2.0","id":1,"result":"default_success"}`,
			ErrorReply:         `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Server error"}}`,
			RateLimitReply:     `{"jsonrpc":"2.0","id":1,"error":{"code":429,"message":"Too many requests"}}`,
			SuccessProbability: 1.0,
		}
	}

	// Generate response based on probabilities
	randValue := rand.Float64()
	var responseData string
	var statusCode int

	if randValue < testResponse.SuccessProbability {
		responseData = testResponse.SuccessReply
		statusCode = 200
	} else if randValue < testResponse.SuccessProbability+testResponse.ErrorProbability {
		responseData = testResponse.ErrorReply
		statusCode = 500
	} else {
		responseData = testResponse.RateLimitReply
		statusCode = 429
	}

	utils.LavaFormatDebug("Generated test response",
		utils.LogAttr("apiMethod", apiMethod),
		utils.LogAttr("randValue", randValue),
		utils.LogAttr("statusCode", statusCode),
		utils.LogAttr("GUID", ctx))

	return &chainlib.RelayReplyWrapper{
		RelayReply: &pairingtypes.RelayReply{
			Data:        []byte(responseData),
			LatestBlock: 12345, // Mock block number
		},
		StatusCode: statusCode,
	}
}
