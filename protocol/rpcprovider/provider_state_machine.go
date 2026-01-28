package rpcprovider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
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

	// used to implement deterministic "head on first request" behavior in test mode
	headReturned atomic.Bool
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

func (psm *ProviderStateMachine) SendNodeMessage(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) (*chainlib.RelayReplyWrapper, time.Duration, error) {
	hash, hashErr := chainMsg.GetRawRequestHash()
	requestHashString := ""
	// Batch requests intentionally return WontCalculateBatchHash - this is expected, not a warning.
	// Only log warning for unexpected hash failures on single requests.
	isBatchRequest := errors.Is(hashErr, rpcInterfaceMessages.WontCalculateBatchHash)
	if hashErr != nil && !isBatchRequest {
		utils.LavaFormatWarning("Failed converting message to hash", hashErr, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
	} else if hashErr == nil {
		requestHashString = string(hash)
	}

	var replyWrapper *chainlib.RelayReplyWrapper
	var isNodeError bool
	var errorMessage string
	var err error
	emptyTime := 0 * time.Millisecond
	for retryAttempt := 0; retryAttempt <= psm.numberOfRetries; retryAttempt++ {
		sendTime := time.Now()

		// Check if this is a test mode request
		isTestMode, _ := ctx.Value(TestModeContextKey{}).(bool)
		if isTestMode && psm.testModeConfig != nil && psm.testModeConfig.TestMode {
			replyWrapper, err = psm.generateTestResponseWithAvailability(ctx, chainMsg, request)
		} else {
			// Original behavior - send to real node
			replyWrapper, _, _, _, _, err = psm.relaySender.SendNodeMsg(ctx, nil, chainMsg, request.RelayData.Extensions)
		}
		latency := time.Since(sendTime)
		if err != nil {
			// Preserve gRPC status errors (used by test mode availability failures) without wrapping,
			// so the consumer can see a stable failure code. Wrap all other errors for logging context.
			if _, ok := grpcstatus.FromError(err); ok {
				return nil, emptyTime, err
			}
			return nil, emptyTime, utils.LavaFormatError("Sending chainMsg failed", err, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return nil, emptyTime, utils.LavaFormatError("Relay Wrapper returned nil without an error", nil, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if debugLatency {
			utils.LavaFormatDebug("node reply received", utils.LogAttr("attempt", retryAttempt), utils.LogAttr("timeTaken", latency), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		// Check for node errors
		isNodeError, errorMessage = chainMsg.CheckResponseError(replyWrapper.RelayReply.Data, replyWrapper.StatusCode)

		// Batch requests cannot be hashed (intentionally), so return immediately with latency.
		// No retries are possible for batch requests since we can't cache the hash.
		if isBatchRequest {
			return replyWrapper, latency, nil
		}

		// For single requests that failed to hash unexpectedly, we can't perform retries.
		if requestHashString == "" {
			utils.LavaFormatWarning("Failed to hash request, shouldn't happen", nil, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
			break // We can't perform the retries as we failed fetching the request hash.
		}
		if !isNodeError {
			// Successful relay, remove it from the cache if we have it and return a valid response.
			go psm.relayRetriesManager.RemoveHashFromCache(requestHashString)
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

func clampProb01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// generateTestResponseWithAvailability wraps generateTestResponse with an availability gate.
// If the request falls into the unavailable bucket, it returns a gRPC-level error so the consumer
// treats it as a relay failure (availability sample = 0).
func (psm *ProviderStateMachine) generateTestResponseWithAvailability(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) (*chainlib.RelayReplyWrapper, error) {
	// Resolve method name
	apiMethod := ""
	if chainMsg != nil && chainMsg.GetApi() != nil {
		apiMethod = chainMsg.GetApi().Name
	}

	// Get test response config (or defaults)
	testResponse, exists := psm.testModeConfig.Responses[apiMethod]
	if !exists {
		// If method isn't configured, behave as always-available by default.
		return psm.generateTestResponse(ctx, chainMsg, request), nil
	}

	availability := 1.0
	if testResponse.Availability != nil {
		availability = clampProb01(*testResponse.Availability)
	}
	// Gate on availability first
	if rand.Float64() > availability {
		return nil, grpcstatus.Error(codes.Unavailable, "test mode availability failure")
	}

	return psm.generateTestResponse(ctx, chainMsg, request), nil
}

func clampNonNegativeInt64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func (psm *ProviderStateMachine) sleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
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

	// Apply per-method artificial delay (optional). This influences consumer-measured relay latency.
	delayMs := testResponse.DelayMs
	delayJitterMs := testResponse.DelayJitterMs
	delayMs = clampNonNegativeInt64(delayMs)
	delayJitterMs = clampNonNegativeInt64(delayJitterMs)
	if delayMs > 0 || delayJitterMs > 0 {
		jitter := int64(0)
		if delayJitterMs > 0 {
			jitter = rand.Int63n(delayJitterMs + 1)
		}
		totalDelay := time.Duration(delayMs+jitter) * time.Millisecond
		// Clamp delay to remaining time, if a deadline exists, to avoid hanging beyond caller timeout.
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < 0 {
				remaining = 0
			}
			if totalDelay > remaining {
				totalDelay = remaining
			}
		}
		psm.sleepWithContext(ctx, totalDelay)
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

	// Compute LatestBlock; precedence: method-level override if set, else default.
	latestBlock := int64(12345)
	latestBlockJitter := int64(0)
	if testResponse.LatestBlock != 0 || testResponse.LatestBlockJitter != 0 {
		latestBlock = testResponse.LatestBlock
		latestBlockJitter = testResponse.LatestBlockJitter
	} else if psm.testModeConfig != nil && psm.testModeConfig.HeadBlock != 0 {
		// Provider-level deterministic head/gap behavior (per provider file).
		head := psm.testModeConfig.HeadBlock
		gap := psm.testModeConfig.GapBlocks
		if gap < 0 {
			gap = 0
		}

		if psm.testModeConfig.HeadOnFirstRequest && !psm.headReturned.Load() {
			// Try to mark as returned; if racing, it is OK if a couple of concurrent first requests return head.
			psm.headReturned.Store(true)
			latestBlock = head
		} else {
			latestBlock = head - gap
		}
	}
	latestBlockJitter = clampNonNegativeInt64(latestBlockJitter)
	if latestBlockJitter > 0 {
		latestBlock += rand.Int63n(latestBlockJitter + 1)
	}
	if latestBlock < 0 {
		latestBlock = 0
	}

	return &chainlib.RelayReplyWrapper{
		RelayReply: &pairingtypes.RelayReply{
			Data:        []byte(responseData),
			LatestBlock: latestBlock,
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
