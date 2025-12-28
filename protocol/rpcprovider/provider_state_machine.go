package rpcprovider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
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

const debugDetailedLatencyPSM = true // Enable detailed timing logs for provider state machine

func (psm *ProviderStateMachine) SendNodeMessage(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) (*chainlib.RelayReplyWrapper, time.Duration, error) {
	sendNodeMsgStart := time.Now()
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
	emptyTime := 0 * time.Millisecond
	for retryAttempt := 0; retryAttempt <= psm.numberOfRetries; retryAttempt++ {
		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] SendNodeMessage - retry loop iteration start",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("retryAttempt", retryAttempt),
				utils.LogAttr("timeSinceStart", time.Since(sendNodeMsgStart)),
			)
		}

		sendTime := time.Now()

		// Check if this is a test mode request
		isTestMode, _ := ctx.Value(TestModeContextKey{}).(bool)
		if isTestMode && psm.testModeConfig != nil && psm.testModeConfig.TestMode {
			replyWrapper = psm.generateTestResponse(ctx, chainMsg, request)
			err = nil // No error in test mode
		} else {
			// Original behavior - send to real node
			if debugDetailedLatencyPSM {
				utils.LavaFormatDebug("[Timing] calling relaySender.SendNodeMsg",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("retryAttempt", retryAttempt),
					utils.LogAttr("timeSinceStart", time.Since(sendNodeMsgStart)),
				)
			}
			actualSendStart := time.Now()
			replyWrapper, _, _, _, _, err = psm.relaySender.SendNodeMsg(ctx, nil, chainMsg, request.RelayData.Extensions)
			if debugDetailedLatencyPSM {
				utils.LavaFormatDebug("[Timing] relaySender.SendNodeMsg - actual call duration",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("retryAttempt", retryAttempt),
					utils.LogAttr("actualDuration", time.Since(actualSendStart)),
				)
			}
		}
		latency := time.Since(sendTime)
		postProcessStart := time.Now() // Start timing post-processing
		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] relaySender.SendNodeMsg returned",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("retryAttempt", retryAttempt),
				utils.LogAttr("latency", latency),
				utils.LogAttr("isError", err != nil),
			)
		}

		validationStart := time.Now()
		if err != nil {
			return nil, emptyTime, utils.LavaFormatError("Sending chainMsg failed", err, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return nil, emptyTime, utils.LavaFormatError("Relay Wrapper returned nil without an error", nil, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}
		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] SendNodeMessage - reply validation completed",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("timeTaken", time.Since(validationStart)),
				utils.LogAttr("retryAttempt", retryAttempt),
			)
		}

		if debugLatency {
			utils.LavaFormatDebug("node reply received", utils.LogAttr("attempt", retryAttempt), utils.LogAttr("timeTaken", latency), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] SendNodeMessage - about to call CheckResponseError",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("timeSincePostProcessStart", time.Since(postProcessStart)),
				utils.LogAttr("timeSinceValidationStart", time.Since(validationStart)),
				utils.LogAttr("retryAttempt", retryAttempt),
			)
		}

		// Check for node errors
		checkErrorStart := time.Now()
		isNodeError, errorMessage = chainMsg.CheckResponseError(replyWrapper.RelayReply.Data, replyWrapper.StatusCode)
		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] SendNodeMessage - CheckResponseError completed",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("timeTaken", time.Since(checkErrorStart)),
				utils.LogAttr("isNodeError", isNodeError),
				utils.LogAttr("dataSize", len(replyWrapper.RelayReply.Data)),
			)
		}

		// Failed fetching hash return the reply.
		hashCheckStart := time.Now()
		if requestHashString == "" {
			utils.LavaFormatWarning("Failed to hash request, shouldn't happen", nil, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
			break // We can't perform the retries as we failed fetching the request hash.
		}
		if debugDetailedLatencyPSM {
			utils.LavaFormatDebug("[Timing] SendNodeMessage - hash string check completed",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("timeTaken", time.Since(hashCheckStart)),
				utils.LogAttr("timeFromCheckError", time.Since(checkErrorStart)),
				utils.LogAttr("hashStringEmpty", requestHashString == ""),
			)
		}

		nodeErrorCheckStart := time.Now()
		if !isNodeError {
			if debugDetailedLatencyPSM {
				utils.LavaFormatDebug("[Timing] SendNodeMessage - entering success path (no node error)",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("timeTaken", time.Since(nodeErrorCheckStart)),
					utils.LogAttr("totalSoFar", time.Since(postProcessStart)),
				)
			}

			// Successful relay, remove it from the cache if we have it and return a valid response.
			beforeCacheRemove := time.Now()
			go psm.relayRetriesManager.RemoveHashFromCache(requestHashString)
			if debugDetailedLatencyPSM {
				utils.LavaFormatDebug("[Timing] SendNodeMessage - cache remove goroutine spawned",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("timeTaken", time.Since(beforeCacheRemove)),
					utils.LogAttr("totalSoFar", time.Since(postProcessStart)),
				)
			}

			beforeFinalReturn := time.Now()
			if debugDetailedLatencyPSM {
				totalPostProcessing := time.Since(postProcessStart)
				utils.LavaFormatDebug("[Timing] SendNodeMessage - total post-processing completed (success path)",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("postProcessingTime", totalPostProcessing),
					utils.LogAttr("nodeLatency", latency),
					utils.LogAttr("overhead", totalPostProcessing),
					utils.LogAttr("retryAttempt", retryAttempt),
					utils.LogAttr("timeInFinalLogging", time.Since(beforeFinalReturn)),
				)
			}
			return replyWrapper, latency, nil
		}

		// Check if this is an unsupported method error based on known patterns/status codes
		isUnsupported := chainlib.IsUnsupportedMethodErrorMessage(errorMessage)
		if !isUnsupported && chainMsg != nil && chainMsg.GetApiCollection() != nil {
			if strings.EqualFold(chainMsg.GetApiCollection().CollectionData.ApiInterface, "rest") {
				if replyWrapper.StatusCode == http.StatusNotFound || replyWrapper.StatusCode == http.StatusMethodNotAllowed {
					isUnsupported = true
				}
			}
		}

		if isUnsupported {
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
			)

			// Return an UnsupportedMethodError to the consumer so they don't increment their CU counter
			unsupportedError := chainlib.NewUnsupportedMethodError(errors.New(errorMessage), methodName)
			return nil, emptyTime, unsupportedError
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
	return replyWrapper, emptyTime, nil
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
			SuccessReply:           `{"jsonrpc":"2.0","id":1,"result":"default_success"}`,
			ErrorReply:             `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Server error"}}`,
			RateLimitReply:         `{"jsonrpc":"2.0","id":1,"error":{"code":429,"message":"Too many requests"}}`,
			UnsupportedMethodReply: "", // Will be generated dynamically
			SuccessProbability:     0.85,
			ErrorProbability:       0.1,
			RateLimitProbability:   0.03,
			UnsupportedProbability: 0.02,
		}
	}

	// Generate response based on probabilities
	randValue := rand.Float64()
	var responseData string
	var statusCode int

	// ENHANCED: Add unsupported method simulation as the first check
	if randValue < testResponse.UnsupportedProbability {
		// Generate realistic unsupported method response based on API interface
		responseData, statusCode = psm.generateUnsupportedMethodResponse(chainMsg, apiMethod, testResponse.UnsupportedMethodReply)
	} else if randValue < testResponse.UnsupportedProbability+testResponse.SuccessProbability {
		responseData = testResponse.SuccessReply
		statusCode = 200
	} else if randValue < testResponse.UnsupportedProbability+testResponse.SuccessProbability+testResponse.ErrorProbability {
		responseData = testResponse.ErrorReply
		statusCode = 500
	} else if randValue < testResponse.UnsupportedProbability+testResponse.SuccessProbability+testResponse.ErrorProbability+testResponse.RateLimitProbability {
		responseData = testResponse.RateLimitReply
		statusCode = 429
	} else {
		// Fallback to success if probabilities don't add up to 1.0
		responseData = testResponse.SuccessReply
		statusCode = 200
	}

	// Get provider address from the request
	providerAddress := ""
	if request != nil && request.RelaySession != nil {
		providerAddress = request.RelaySession.Provider
	}

	utils.LavaFormatDebug("Generated test response",
		utils.LogAttr("apiMethod", apiMethod),
		utils.LogAttr("providerAddress", providerAddress),
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

// generateUnsupportedMethodResponse creates realistic unsupported method responses
// based on the API interface type, ensuring consistency with production error detection
func (psm *ProviderStateMachine) generateUnsupportedMethodResponse(chainMsg chainlib.ChainMessage, apiMethod string, configuredReply string) (string, int) {
	// If a specific unsupported reply is configured, use it
	if configuredReply != "" {
		return configuredReply, 500
	}

	// Otherwise, generate API-appropriate unsupported method responses
	if chainMsg == nil || chainMsg.GetApiCollection() == nil {
		return `{"error":"method not supported"}`, 500
	}

	apiInterface := strings.ToLower(chainMsg.GetApiCollection().CollectionData.ApiInterface)

	switch apiInterface {
	case "jsonrpc", "tendermintrpc":
		// Generate JSON-RPC method not found error (code -32601)
		// This matches the pattern that chainlib.IsUnsupportedMethodErrorMessage() detects
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found: %s"}}`, apiMethod), 200

	case "rest":
		// Generate REST not found error that will trigger HTTP status code detection
		return fmt.Sprintf(`{"error":"Endpoint not found: %s","message":"The requested endpoint does not exist"}`, apiMethod), 404

	case "grpc":
		// Generate gRPC unimplemented error
		return fmt.Sprintf(`{"error":"Method not implemented: %s","code":12}`, apiMethod), 500

	default:
		// Generic unsupported method error with patterns that will be detected
		return fmt.Sprintf(`{"error":"Method not supported: %s"}`, apiMethod), 500
	}
}
