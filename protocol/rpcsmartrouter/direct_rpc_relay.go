package rpcsmartrouter

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/goccy/go-json"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/parser"
	pairingtypes "github.com/lavanet/lava/v5/types/relay"
	"github.com/lavanet/lava/v5/utils"
)

// DirectRPCRelaySender handles sending relay requests directly to RPC endpoints
// (bypassing the Lava provider-relay protocol)
type DirectRPCRelaySender struct {
	directConnection    lavasession.DirectRPCConnection
	endpointName        string             // Sanitized endpoint name (no API keys)
	originalRequestData []byte             // Original request bytes (for batch support)
	chainFamily         common.ChainFamily // Chain family for Tier 2 classification (-1 if unknown)
}

// maxResponseSizeForBlockExtraction is the threshold above which we skip JSON parsing
// for block height extraction. Responses larger than this (e.g. debug_traceTransaction,
// trace_replayBlockTransactions) are too expensive to unmarshal and rarely contain
// useful block height info. The ChainTracker independently maintains per-endpoint
// block tracking, so skipping extraction here is safe.
const maxResponseSizeForBlockExtraction = 1 * 1024 * 1024 // 1 MB

// extractBlockHeightFromJSONResponse extracts block height using spec-driven parsing.
// This works for any API interface (EVM, Tendermint, etc.) by using the chain's
// parse directives defined in the spec. Falls back to EVM-specific parsing for
// backwards compatibility when parse directive is not available.
func extractBlockHeightFromJSONResponse(
	responseData []byte,
	chainMessage chainlib.ChainMessage,
) int64 {
	// Guard: skip block extraction for very large responses (e.g. debug/trace).
	// Parsing multi-MB responses is extremely expensive (CPU + GC pressure) and
	// these responses rarely contain block height info. Per-endpoint ChainTracker
	// provides block tracking independently as a fallback.
	if len(responseData) > maxResponseSizeForBlockExtraction {
		utils.LavaFormatDebug("skipping block extraction for large response",
			utils.LogAttr("response_size", len(responseData)),
			utils.LogAttr("threshold", maxResponseSizeForBlockExtraction),
			utils.LogAttr("method", chainMessage.GetApi().Name),
		)
		return 0
	}

	// First try spec-driven parsing (works for all API interfaces including Tendermint)
	parseDirective := chainMessage.GetParseDirective()
	if parseDirective != nil {
		parserInput, err := chainlib.FormatResponseForParsing(
			&pairingtypes.RelayReply{Data: responseData},
			chainMessage,
		)
		if err == nil {
			parsedInput := parser.ParseBlockFromReply(
				parserInput,
				parseDirective.ResultParsing,
				parseDirective.Parsers,
			)
			if block := parsedInput.GetBlock(); block > 0 {
				return block
			}
		}
	}

	// Fallback to EVM-specific parsing for backwards compatibility
	return extractBlockHeightFromEVMResponse(responseData, chainMessage.GetApi().Name)
}

// extractBlockHeightFromEVMResponse extracts block height from EVM JSON-RPC responses.
// This is a fallback for EVM chains when spec-driven parsing doesn't yield results.
//
// Only methods known to return block height info are parsed. All other methods
// (e.g. debug_traceTransaction, trace_replayBlockTransactions) return 0 immediately
// without any JSON parsing, avoiding expensive full
func extractBlockHeightFromEVMResponse(responseData []byte, method string) int64 {
	switch method {
	case "eth_blockNumber":
		result, err := unmarshalEVMResponseData(responseData)
		if err != nil {
			return 0
		}
		// Response: {"result": "0x12a7b5c"}
		if hexStr, ok := result.(string); ok {
			if len(hexStr) > 2 && hexStr[:2] == "0x" {
				if block, err := strconv.ParseInt(hexStr[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getBlockByNumber", "eth_getBlockByHash":
		result, err := unmarshalEVMResponseData(responseData)
		if err != nil {
			return 0
		}
		// Response: {"result": {"number": "0x12a7b5c", ...}}
		if blockObj, ok := result.(map[string]interface{}); ok {
			if numberHex, ok := blockObj["number"].(string); ok && len(numberHex) > 2 {
				if block, err := strconv.ParseInt(numberHex[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getTransactionReceipt":
		result, err := unmarshalEVMResponseData(responseData)
		if err != nil {
			return 0
		}
		// Response: {"result": {"blockNumber": "0x12a7b5c", ...}}
		if receiptObj, ok := result.(map[string]interface{}); ok {
			if blockNumHex, ok := receiptObj["blockNumber"].(string); ok && len(blockNumHex) > 2 {
				if block, err := strconv.ParseInt(blockNumHex[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getLogs":
		// Response: {"result": [{"blockNumber": "0x12a7b5c"}, ...]}
		result, err := unmarshalEVMResponseData(responseData)
		if err != nil {
			return 0
		}
		if logsArray, ok := result.([]interface{}); ok && len(logsArray) > 0 {
			if firstLog, ok := logsArray[0].(map[string]interface{}); ok {
				if blockNumHex, ok := firstLog["blockNumber"].(string); ok && len(blockNumHex) > 2 {
					if block, err := strconv.ParseInt(blockNumHex[2:], 16, 64); err == nil {
						return block
					}
				}
			}
		}
	}

	utils.LavaFormatDebug("EVM fallback: no block height for method",
		utils.LogAttr("method", method),
		utils.LogAttr("response_size", len(responseData)),
	)
	return 0
}

func unmarshalEVMResponseData(responseData []byte) (interface{}, error) {
	var jsonResponse struct {
		Result interface{} `json:"result"`
	}
	if err := json.Unmarshal(responseData, &jsonResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	}
	return jsonResponse.Result, nil
}

// extractBlockHeightFromGRPCResponse extracts block height from gRPC response using spec-driven parsing.
// This follows the same pattern as ChainFetcher.FetchLatestBlockNum() for consistency.
// Returns 0 if parsing fails or no block height is available in the response.
func extractBlockHeightFromGRPCResponse(
	responseData []byte,
	chainMessage chainlib.ChainMessage,
) int64 {
	// Get parse directive from chain message (contains spec-defined parsing rules)
	parseDirective := chainMessage.GetParseDirective()
	if parseDirective == nil {
		return 0
	}

	// Format response for parsing using the chainMessage's RPC type
	parserInput, err := chainlib.FormatResponseForParsing(
		&pairingtypes.RelayReply{Data: responseData},
		chainMessage,
	)
	if err != nil {
		utils.LavaFormatTrace("failed to format gRPC response for block parsing",
			utils.LogAttr("error", err))
		return 0
	}

	// Parse block height using spec-driven rules (same as ChainFetcher)
	parsedInput := parser.ParseBlockFromReply(
		parserInput,
		parseDirective.ResultParsing,
		parseDirective.Parsers,
	)

	return parsedInput.GetBlock()
}

// createGRPCFormatter creates a grpcurl.Formatter for converting protobuf messages to JSON.
// This is needed for parsing gRPC responses to extract block heights.
// The formatter uses the method descriptor's output type to properly format the response.
func createGRPCFormatter(methodDesc *desc.MethodDescriptor) grpcurl.Formatter {
	return func(msg proto.Message) (string, error) {
		// The message should be a dynamic.Message from protoreflect
		dynMsg, ok := msg.(*dynamic.Message)
		if !ok {
			// Try to marshal using standard JSON for other proto types
			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				return "", fmt.Errorf("failed to marshal proto message: %w", err)
			}
			return string(jsonBytes), nil
		}

		// Use dynamic message's JSON marshaling (matches grpcurl behavior)
		jsonBytes, err := dynMsg.MarshalJSON()
		if err != nil {
			return "", fmt.Errorf("failed to marshal dynamic message to JSON: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// sanitizeEndpointURL removes sensitive information (API keys, tokens) from URLs
// Returns just the hostname, or a configured name if available
func sanitizeEndpointURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If parsing fails, return a generic identifier
		return "endpoint"
	}

	// Return just the hostname (no path, no query params which might contain API keys)
	host := parsed.Host
	if host == "" {
		host = parsed.Hostname()
	}

	// If hostname is also empty, use scheme + opaque
	if host == "" {
		return "endpoint"
	}

	return host
}

// SendDirectRelay sends a relay request directly to an RPC endpoint
// SendDirectRelay routes to the appropriate protocol handler (JSON-RPC or REST)
func (d *DirectRPCRelaySender) SendDirectRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	// Branch based on API interface
	apiCollection := chainMessage.GetApiCollection()

	switch apiCollection.CollectionData.ApiInterface {
	case "jsonrpc", "tendermintrpc":
		return d.sendJSONRPCRelay(ctx, chainMessage, relayTimeout)

	case "rest":
		return d.sendRESTRelay(ctx, chainMessage, relayTimeout)

	case "grpc":
		return d.sendGRPCRelay(ctx, chainMessage, relayTimeout)

	default:
		return nil, fmt.Errorf("unsupported API interface for direct RPC: %s", apiCollection.CollectionData.ApiInterface)
	}
}

// sendJSONRPCRelay handles JSON-RPC requests (Phase 3 implementation)
// Fixed to address gaps: header semantics, timeout overrides, response headers
func (d *DirectRPCRelaySender) sendJSONRPCRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	// Use NodeUrl.LowerContextTimeoutWithDuration for per-endpoint timeout overrides
	// This allows operators to configure extended timeouts for heavy RPCs (debug_traceTransaction, etc.)
	nodeUrl := d.directConnection.GetNodeUrl()
	requestCtx, cancel := nodeUrl.LowerContextTimeoutWithDuration(ctx, relayTimeout)
	defer cancel()

	// STEP 1: Use original request bytes (supports batch requests)
	// For batch requests, we MUST use the original JSON because JsonrpcBatchMessage
	// has a non-exported 'batch' field that cannot be marshaled.
	// This approach works for both single and batch requests.
	requestData := d.originalRequestData
	if len(requestData) == 0 {
		// Fallback: marshal the RPC message (should not happen in normal flow)
		rpcMessage := chainMessage.GetRPCMessage()
		var err error
		requestData, err = json.Marshal(rpcMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal RPC message: %w", err)
		}
	}

	// Use sanitized endpoint identifier for logging (no API keys)
	endpointIdentifier := d.endpointName
	if endpointIdentifier == "" {
		endpointIdentifier = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	utils.LavaFormatTrace("sending direct RPC request",
		utils.LogAttr("endpoint", endpointIdentifier),
		utils.LogAttr("protocol", d.directConnection.GetProtocol()),
		utils.LogAttr("method", chainMessage.GetApi().Name),
		utils.LogAttr("timeout", relayTimeout),
	)

	// Use DoHTTPRequest with []Metadata (preserves duplicates, supports delete semantics)
	// Instead of map[string]string which loses duplicates and doesn't support delete (empty = delete)
	httpDoer, ok := d.directConnection.(lavasession.HTTPDirectRPCDoer)
	if !ok {
		return nil, fmt.Errorf("connection does not support HTTP requests (protocol: %s)", d.directConnection.GetProtocol())
	}

	// Build HTTP request params (same as REST path for consistency)
	httpParams := lavasession.HTTPRequestParams{
		Method:      "POST",
		URL:         nodeUrl.Url,
		Body:        requestData,
		Headers:     chainMessage.GetRPCMessage().GetHeaders(), // Preserves duplicates, empty value = delete
		ContentType: "application/json",
	}

	// STEP 3: Send request using DoHTTPRequest (returns headers + body)
	startTime := time.Now()
	response, err := httpDoer.DoHTTPRequest(requestCtx, httpParams)
	latency := time.Since(startTime)

	if err != nil {
		utils.LavaFormatDebug("direct RPC request failed",
			utils.LogAttr("endpoint", endpointIdentifier),
			utils.LogAttr("protocol", d.directConnection.GetProtocol()),
			utils.LogAttr("error", err.Error()),
			utils.LogAttr("latency", latency),
		)
		return nil, classifyAndWrap(err, d.chainFamily, common.TransportJsonRPC)
	}

	statusCode := response.StatusCode
	responseData := response.Body

	// Handle HTTP error status codes
	if statusCode >= 500 {
		utils.LavaFormatDebug("direct RPC request returned server error",
			utils.LogAttr("endpoint", endpointIdentifier),
			utils.LogAttr("status", statusCode),
			utils.LogAttr("latency", latency),
		)
		httpErr := &lavasession.HTTPStatusError{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d", statusCode),
			Body:       responseData,
		}
		return nil, classifyAndWrap(httpErr, d.chainFamily, common.TransportJsonRPC)
	}

	utils.LavaFormatTrace("direct RPC request succeeded",
		utils.LogAttr("endpoint", endpointIdentifier),
		utils.LogAttr("latency", latency),
		utils.LogAttr("status_code", statusCode),
		utils.LogAttr("response_size", len(responseData)),
	)

	// STEP 4: Check response for errors using chainMessage (with actual HTTP status)
	hasError, errorMessage := chainMessage.CheckResponseError(responseData, statusCode)
	if hasError {
		utils.LavaFormatDebug("RPC response contains error",
			utils.LogAttr("endpoint", endpointIdentifier),
			utils.LogAttr("error", errorMessage),
		)
		// Still return the response - the caller will handle the RPC error
	}

	// STEP 5: Build RelayResult compatible with existing smart router flow
	providerAddress := d.endpointName
	if providerAddress == "" {
		providerAddress = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	// Extract latest block from response if possible (for QoS sync tracking)
	// Uses spec-driven parsing for all API interfaces (EVM, Tendermint, etc.)
	latestBlockFromResponse := extractBlockHeightFromJSONResponse(responseData, chainMessage)

	// Convert response headers to metadata (same as REST path)
	// This enables Provider-Latest-Block, lava-identified-node-error, and upstream hints
	responseMetadata := convertHTTPHeadersToMetadata(response.Headers)

	result := &common.RelayResult{
		Reply: &pairingtypes.RelayReply{
			Data:        responseData,
			LatestBlock: latestBlockFromResponse,
			Metadata:    responseMetadata, // Response headers now included
		},
		Finalized:  true,
		StatusCode: statusCode,
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerAddress,
		},
		IsNodeError: hasError,
	}
	if hasError {
		// JSON-RPC node errors come back as HTTP 200 with the real code inside
		// error.code. Prefer the body code over the HTTP status so registry
		// code-based matchers (e.g. -32700, -32602) fire.
		errorCode := statusCode
		if jsonrpcCode := common.ExtractJSONRPCErrorCode(responseData); jsonrpcCode != 0 {
			errorCode = jsonrpcCode
		}
		result.IsNonRetryable = common.IsNonRetryableNodeErrorWithContext(d.chainFamily, common.TransportJsonRPC, errorCode, errorMessage)
	}

	return result, nil
}

// sendRESTRelay handles REST API requests (Phase 4 corrected implementation)
func (d *DirectRPCRelaySender) sendRESTRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	// Use NodeUrl.LowerContextTimeoutWithDuration for per-endpoint timeout overrides
	nodeUrl := d.directConnection.GetNodeUrl()
	requestCtx, cancel := nodeUrl.LowerContextTimeoutWithDuration(ctx, relayTimeout)
	defer cancel()

	// Get RPC message
	rpcMessage := chainMessage.GetRPCMessage()

	restMessage, ok := rpcMessage.(*rpcInterfaceMessages.RestMessage)
	if !ok {
		return nil, fmt.Errorf("expected RestMessage for REST API, got %T", rpcMessage)
	}

	// REST path (including query string) is stored on the message.
	restPath := restMessage.Path
	restBody := restMessage.Msg

	// Get API collection
	apiCollection := chainMessage.GetApiCollection()

	// HTTP method from parser (already validated)
	httpMethod := apiCollection.CollectionData.Type
	if httpMethod == "" {
		return nil, fmt.Errorf("HTTP method not set by REST parser")
	}

	// Extract headers as Metadata (preserves delete semantics)
	headers := restMessage.GetHeaders()

	// Robust URL joining
	baseURL := d.directConnection.GetURL()
	fullURL, err := joinURLPath(baseURL, restPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build REST URL: %w", err)
	}

	// Type-assert to HTTP doer
	httpDoer, ok := d.directConnection.(lavasession.HTTPDirectRPCDoer)
	if !ok {
		return nil, fmt.Errorf("connection doesn't support REST (HTTP)")
	}

	// Send request
	startTime := time.Now()
	response, err := httpDoer.DoHTTPRequest(requestCtx, lavasession.HTTPRequestParams{
		Method:      httpMethod,
		URL:         fullURL,
		Body:        restBody, // Send body as-is (for POST/PUT)
		Headers:     headers,  // Use Metadata (preserves delete semantics)
		ContentType: "application/json",
	})
	latency := time.Since(startTime)

	// Handle transport errors
	if err != nil {
		return nil, classifyAndWrap(err, d.chainFamily, common.TransportREST)
	}

	// Proper error classification (don't treat all 4xx as node errors)
	var isNodeError bool
	switch {
	case response.StatusCode >= 500:
		isNodeError = true // Server error
	case response.StatusCode == 429:
		isNodeError = false // Rate limit (not node issue)
	case response.StatusCode >= 400:
		isNodeError = false // Client error
	default:
		isNodeError = false // Success
	}

	// Let the chain message parse domain-specific REST errors (e.g. Cosmos tx errors on HTTP 200).
	// NOTE: This should NOT be treated as "node error" by default; it is typically a request/application error.
	hasError, errorMessage := chainMessage.CheckResponseError(response.Body, response.StatusCode)
	if hasError && errorMessage != "" {
		utils.LavaFormatDebug("REST response contains error",
			utils.LogAttr("endpoint", d.endpointName),
			utils.LogAttr("error", errorMessage),
		)
	}

	// Convert response headers to metadata
	responseMetadata := convertHTTPHeadersToMetadata(response.Headers)

	// Build result (include body even for 4xx/5xx!)
	providerAddress := d.endpointName
	if providerAddress == "" {
		providerAddress = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	result := &common.RelayResult{
		Reply: &pairingtypes.RelayReply{
			Data:     response.Body,    // Include body even for errors!
			Metadata: responseMetadata, // Include headers
		},
		Finalized:  true,
		StatusCode: response.StatusCode,
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerAddress,
		},
		IsNodeError: isNodeError, // Correct transport-level classification
	}
	if hasError {
		result.IsNonRetryable = common.IsNonRetryableNodeErrorWithContext(d.chainFamily, common.TransportREST, response.StatusCode, errorMessage)
	}

	utils.LavaFormatTrace("REST request completed",
		utils.LogAttr("method", httpMethod),
		utils.LogAttr("status", response.StatusCode),
		utils.LogAttr("response_size", len(response.Body)),
		utils.LogAttr("latency", latency),
		utils.LogAttr("is_node_error", isNodeError),
	)

	return result, nil
}

// sendGRPCRelay handles gRPC requests (Phase 6 implementation)
// Supports Cosmos SDK, Solana Geyser, Sui, Aptos, Flow, and other gRPC-based chains
func (d *DirectRPCRelaySender) sendGRPCRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	// Apply per-endpoint timeout override
	nodeUrl := d.directConnection.GetNodeUrl()
	requestCtx, cancel := nodeUrl.LowerContextTimeoutWithDuration(ctx, relayTimeout)
	defer cancel()

	// Get RPC message (contains the gRPC method path and request data)
	rpcMessage := chainMessage.GetRPCMessage()

	// For gRPC, cast to GrpcMessage to access the Msg field
	grpcMessage, ok := rpcMessage.(*rpcInterfaceMessages.GrpcMessage)
	if !ok {
		return nil, fmt.Errorf("expected GrpcMessage for gRPC API, got %T", rpcMessage)
	}

	// For gRPC, the API name or GrpcMessage.Path contains the full method path
	methodPath := grpcMessage.Path
	if methodPath == "" {
		methodPath = chainMessage.GetApi().Name
	}
	if methodPath == "" {
		return nil, fmt.Errorf("gRPC method path not set")
	}

	// Get request data (can be JSON or binary proto)
	requestData := grpcMessage.Msg

	// Build headers with required gRPC method header
	headers := make(map[string]string)
	headers[lavasession.GRPCMethodHeader] = methodPath

	// Add any additional headers from the RPC message
	for _, meta := range grpcMessage.GetHeaders() {
		if meta.Value != "" { // Empty value means delete (not applicable for gRPC)
			headers[meta.Name] = meta.Value
		}
	}

	// Use sanitized endpoint identifier for logging
	endpointIdentifier := d.endpointName
	if endpointIdentifier == "" {
		endpointIdentifier = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	utils.LavaFormatTrace("sending direct gRPC request",
		utils.LogAttr("endpoint", endpointIdentifier),
		utils.LogAttr("method", methodPath),
		utils.LogAttr("timeout", relayTimeout),
	)

	// Send gRPC request via DirectRPCConnection
	startTime := time.Now()
	response, err := d.directConnection.SendRequest(requestCtx, requestData, headers)
	latency := time.Since(startTime)

	if err != nil {
		utils.LavaFormatDebug("direct gRPC request failed",
			utils.LogAttr("endpoint", endpointIdentifier),
			utils.LogAttr("method", methodPath),
			utils.LogAttr("error", err.Error()),
			utils.LogAttr("latency", latency),
		)

		// Check if it's a gRPC status error (use comma-ok idiom to avoid panic)
		grpcErr, isGRPCErr := err.(*lavasession.GRPCStatusError)

		if isGRPCErr && response != nil {
			// gRPC error with status code - might contain valid error response
			// The response.Data contains the error details in JSON format
			return &common.RelayResult{
				Reply: &pairingtypes.RelayReply{
					Data:     response.Data,                                   // Error response in JSON format
					Metadata: convertHTTPHeadersToMetadata(response.Metadata), // Include metadata even for errors
				},
				Finalized: true,
				ProviderInfo: common.ProviderInfo{
					ProviderAddress: endpointIdentifier,
				},
				IsNodeError: grpcErr.Code >= 13, // INTERNAL and above are node errors
			}, nil
		}

		return nil, classifyAndWrap(err, d.chainFamily, common.TransportGRPC)
	}

	utils.LavaFormatTrace("direct gRPC request succeeded",
		utils.LogAttr("endpoint", endpointIdentifier),
		utils.LogAttr("method", methodPath),
		utils.LogAttr("latency", latency),
		utils.LogAttr("response_size", len(response.Data)),
	)

	// Check for errors in response using chainMessage
	hasError, errorMessage := chainMessage.CheckResponseError(response.Data, response.StatusCode)
	if hasError {
		utils.LavaFormatDebug("gRPC response contains error",
			utils.LogAttr("endpoint", endpointIdentifier),
			utils.LogAttr("method", methodPath),
			utils.LogAttr("error", errorMessage),
		)
	}

	// Set parsing data on grpcMessage for block height extraction (QoS sync tracking)
	// This is required because FormatResponseForParsing needs the method descriptor and formatter
	// to properly parse the binary protobuf response into JSON for block extraction.
	// The descriptor is cached by GRPCDirectRPCConnection during SendRequest.
	if descriptorProvider, ok := d.directConnection.(lavasession.GRPCDescriptorProvider); ok {
		if methodDesc := descriptorProvider.GetCachedMethodDescriptor(methodPath); methodDesc != nil {
			formatter := createGRPCFormatter(methodDesc)
			grpcMessage.SetParsingData(methodDesc, formatter)
		}
	}

	// Extract block height from gRPC response using spec-driven parsing (for QoS sync tracking)
	latestBlockFromResponse := extractBlockHeightFromGRPCResponse(response.Data, chainMessage)

	// Build result with response metadata
	providerAddress := d.endpointName
	if providerAddress == "" {
		providerAddress = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	result := &common.RelayResult{
		Reply: &pairingtypes.RelayReply{
			Data:        response.Data,
			LatestBlock: latestBlockFromResponse,
			Metadata:    convertHTTPHeadersToMetadata(response.Metadata), // Include gRPC response metadata
		},
		Finalized:  true,
		StatusCode: response.StatusCode,
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerAddress,
		},
		IsNodeError: hasError,
	}
	if hasError {
		result.IsNonRetryable = common.IsNonRetryableNodeErrorWithContext(d.chainFamily, common.TransportGRPC, response.StatusCode, errorMessage)
	}

	return result, nil
}

// joinURLPath joins base URL and path robustly (handles slashes and query params correctly).
// When path is absolute (starts with /), it is appended to the base URL's path so that
// base paths like /gateway/lava/rest/KEY are preserved (ResolveReference would replace them).
func joinURLPath(base, path string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// If path is absolute, append it to base path instead of replacing (preserves e.g. /gateway/lava/rest/KEY).
	if strings.HasPrefix(pathURL.Path, "/") {
		basePath := strings.TrimSuffix(baseURL.Path, "/")
		relPath := strings.TrimPrefix(pathURL.Path, "/")
		if relPath != "" {
			baseURL.Path = basePath + "/" + relPath
		} else {
			baseURL.Path = basePath
		}
		baseURL.RawPath = "" // let EscapedPath() derive from Path
		if pathURL.RawQuery != "" {
			baseURL.RawQuery = pathURL.RawQuery
		}
		return baseURL.String(), nil
	}

	// Relative path: use ResolveReference (handles ., .., query params)
	return baseURL.ResolveReference(pathURL).String(), nil
}

// convertHTTPHeadersToMetadata converts http.Header to pairingtypes.Metadata
func convertHTTPHeadersToMetadata(headers map[string][]string) []pairingtypes.Metadata {
	metadata := make([]pairingtypes.Metadata, 0, len(headers))
	for name, values := range headers {
		if len(values) > 0 {
			// Use first value (most headers are single-value)
			metadata = append(metadata, pairingtypes.Metadata{
				Name:  name,
				Value: values[0],
			})
		}
	}
	return metadata
}
