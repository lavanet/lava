package rpcsmartrouter

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// DirectRPCRelaySender handles sending relay requests directly to RPC endpoints
// (bypassing the Lava provider-relay protocol)
type DirectRPCRelaySender struct {
	directConnection lavasession.DirectRPCConnection
	endpointName     string // Sanitized endpoint name (no API keys)
}

// extractLatestBlockFromResponse attempts to extract the latest block number from RPC response
// Returns 0 if the method doesn't include block information
func extractLatestBlockFromResponse(responseData []byte, method string) int64 {
	// Parse response JSON
	var jsonResponse struct {
		Result interface{} `json:"result"`
	}

	if err := json.Unmarshal(responseData, &jsonResponse); err != nil {
		return 0 // Can't parse, return 0
	}

	switch method {
	case "eth_blockNumber":
		// Response: {"result": "0x12a7b5c"}
		if hexStr, ok := jsonResponse.Result.(string); ok {
			if len(hexStr) > 2 && hexStr[:2] == "0x" {
				if block, err := strconv.ParseInt(hexStr[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getBlockByNumber", "eth_getBlockByHash":
		// Response: {"result": {"number": "0x12a7b5c", ...}}
		if blockObj, ok := jsonResponse.Result.(map[string]interface{}); ok {
			if numberHex, ok := blockObj["number"].(string); ok && len(numberHex) > 2 {
				if block, err := strconv.ParseInt(numberHex[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getTransactionReceipt":
		// Response: {"result": {"blockNumber": "0x12a7b5c", ...}}
		if receiptObj, ok := jsonResponse.Result.(map[string]interface{}); ok {
			if blockNumHex, ok := receiptObj["blockNumber"].(string); ok && len(blockNumHex) > 2 {
				if block, err := strconv.ParseInt(blockNumHex[2:], 16, 64); err == nil {
					return block
				}
			}
		}

	case "eth_getLogs":
		// Response: {"result": [{"blockNumber": "0x12a7b5c"}, ...]}
		if logsArray, ok := jsonResponse.Result.([]interface{}); ok && len(logsArray) > 0 {
			if firstLog, ok := logsArray[0].(map[string]interface{}); ok {
				if blockNumHex, ok := firstLog["blockNumber"].(string); ok && len(blockNumHex) > 2 {
					if block, err := strconv.ParseInt(blockNumHex[2:], 16, 64); err == nil {
						return block
					}
				}
			}
		}
	}

	return 0 // Method doesn't return block info or couldn't parse
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

	default:
		return nil, fmt.Errorf("unsupported API interface for direct RPC: %s", apiCollection.CollectionData.ApiInterface)
	}
}

// sendJSONRPCRelay handles JSON-RPC requests (Phase 3 implementation)
func (d *DirectRPCRelaySender) sendJSONRPCRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	// Create context with timeout
	requestCtx, cancel := context.WithTimeout(ctx, relayTimeout)
	defer cancel()

	// STEP 1: Get the RPC message and marshal to JSON bytes
	// The chainMessage already contains the parsed and validated request
	rpcMessage := chainMessage.GetRPCMessage()
	requestData, err := json.Marshal(rpcMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC message: %w", err)
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

	// STEP 2: Extract headers from RPC message
	headers := rpcMessage.GetHeaders()
	headerMap := make(map[string]string)
	for _, header := range headers {
		headerMap[header.Name] = header.Value
	}

	// STEP 3: Send request (protocol-agnostic transport)
	startTime := time.Now()
	responseData, err := d.directConnection.SendRequest(requestCtx, requestData, headerMap)
	latency := time.Since(startTime)

	// Extract HTTP status code from error if present
	statusCode := 200
	if err != nil {
		// Check if this is an HTTP status error
		if httpErr, ok := err.(*lavasession.HTTPStatusError); ok {
			statusCode = httpErr.StatusCode
			responseData = httpErr.Body // Use body from error (may contain error details)

			utils.LavaFormatDebug("direct RPC request returned HTTP error",
				utils.LogAttr("endpoint", endpointIdentifier),
				utils.LogAttr("protocol", d.directConnection.GetProtocol()),
				utils.LogAttr("status", statusCode),
				utils.LogAttr("error", err.Error()),
				utils.LogAttr("latency", latency),
			)

			// Map error to user-friendly format
			mappedErr := MapDirectRPCError(err, d.directConnection.GetProtocol())
			// For 5xx errors, return error immediately (node issue)
			if statusCode >= 500 {
				return nil, mappedErr
			}
			// For 4xx errors, continue to check RPC error (might be valid RPC error response)
		} else {
			// Non-HTTP error (timeout, connection refused, etc.)
			utils.LavaFormatDebug("direct RPC request failed",
				utils.LogAttr("endpoint", endpointIdentifier),
				utils.LogAttr("protocol", d.directConnection.GetProtocol()),
				utils.LogAttr("error", err.Error()),
				utils.LogAttr("latency", latency),
			)

			// Map error to user-friendly format
			mappedErr := MapDirectRPCError(err, d.directConnection.GetProtocol())
			return nil, mappedErr
		}
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
	// Use configured endpoint name (not full URL) to avoid leaking API keys
	providerAddress := d.endpointName
	if providerAddress == "" {
		// Fallback to sanitized URL if name not provided
		providerAddress = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	// Extract latest block from response if possible (for QoS sync tracking)
	latestBlockFromResponse := extractLatestBlockFromResponse(responseData, chainMessage.GetApi().Name)

	result := &common.RelayResult{
		Reply: &pairingtypes.RelayReply{
			Data:        responseData,
			LatestBlock: latestBlockFromResponse, // ✅ Set LatestBlock (provider parity)
		},
		Finalized:  true,       // Direct responses don't need consensus
		StatusCode: statusCode, // Actual HTTP status code (200, 400, 429, 500, etc.)
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerAddress, // Use configured name (no API keys)
		},
		IsNodeError: hasError, // Mark if response contains RPC error
	}

	return result, nil
}

// sendRESTRelay handles REST API requests (Phase 4 corrected implementation)
func (d *DirectRPCRelaySender) sendRESTRelay(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayTimeout time.Duration,
) (*common.RelayResult, error) {
	requestCtx, cancel := context.WithTimeout(ctx, relayTimeout)
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

	// ✅ CORRECTION 1: HTTP method from parser (already validated)
	httpMethod := apiCollection.CollectionData.Type
	if httpMethod == "" {
		return nil, fmt.Errorf("HTTP method not set by REST parser")
	}

	// Extract headers as Metadata (preserves delete semantics)
	var headers []pairingtypes.Metadata
	headers = restMessage.GetHeaders()

	// ✅ CORRECTION 6: Robust URL joining
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
		Headers:     headers,  // ✅ CORRECTION 4: Use Metadata (preserves delete semantics)
		ContentType: "application/json",
	})
	latency := time.Since(startTime)

	// Handle transport errors
	if err != nil {
		return nil, MapDirectRPCError(err, d.directConnection.GetProtocol())
	}

	// ✅ CORRECTION 5: Proper error classification (don't treat all 4xx as node errors)
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

	// ✅ CORRECTION 2: Convert response headers to metadata
	responseMetadata := convertHTTPHeadersToMetadata(response.Headers)

	// Build result (include body even for 4xx/5xx!)
	providerAddress := d.endpointName
	if providerAddress == "" {
		providerAddress = sanitizeEndpointURL(d.directConnection.GetURL())
	}

	result := &common.RelayResult{
		Reply: &pairingtypes.RelayReply{
			Data:     response.Body,    // ✅ Include body even for errors!
			Metadata: responseMetadata, // ✅ Include headers
		},
		Finalized:  true,
		StatusCode: response.StatusCode,
		ProviderInfo: common.ProviderInfo{
			ProviderAddress: providerAddress,
		},
		IsNodeError: isNodeError, // ✅ Correct transport-level classification
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

// joinURLPath joins base URL and path robustly (handles slashes and query params correctly)
func joinURLPath(base, path string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Resolve reference (handles double slashes, missing slashes, query params)
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
