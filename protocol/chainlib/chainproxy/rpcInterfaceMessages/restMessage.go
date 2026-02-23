package rpcInterfaceMessages

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/parser"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
)

const (
	cosmosSDKSuccessCode = 0 // Cosmos SDK uses 0 for success, non-zero for errors
)

// cosmosTxResponse represents Cosmos SDK transaction broadcast response structure
type cosmosTxResponse struct {
	TxResponse struct {
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	} `json:"tx_response"`
}

type RestMessage struct {
	Msg      []byte
	Path     string
	SpecPath string
	chainproxy.BaseMessage
}

func (rm *RestMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}

// get msg hash byte array containing all the relevant information for a unique request. (headers / api / params)
func (rm *RestMessage) GetRawRequestHash() ([]byte, error) {
	headers := rm.GetHeaders()
	headersByteArray, err := json.Marshal(headers)
	if err != nil {
		utils.LavaFormatError("Failed marshalling headers on jsonRpc message", err, utils.LogAttr("headers", headers))
		return []byte{}, err
	}
	pathByteArray := []byte(rm.Path)
	return sigs.HashMsg(append(append(pathByteArray, rm.Msg...), headersByteArray...)), nil
}

func (jm RestMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	// Treat 5xx and 429 as node errors (triggers retries)
	if httpStatusCode >= 500 || httpStatusCode == 429 {
		// Server error or rate limit - treat as node error for retry logic
		errorMsg := extractErrorMessage(data, httpStatusCode)
		return true, errorMsg
	}

	// Check Cosmos SDK transaction errors (HTTP 2xx with error code in JSON)
	if httpStatusCode >= 200 && httpStatusCode < 300 {
		if hasError, errMsg := checkCosmosTxError(data); hasError {
			return hasError, errMsg
		}
		// HTTP status code is 2xx and no Cosmos SDK error, treat as success
		return false, ""
	}

	// 4xx (except 429) are client errors - NOT node errors
	// Return false so state machine doesn't retry (client error won't succeed on retry)
	return false, ""
}

// checkCosmosTxError detects errors in Cosmos SDK transaction responses
// Cosmos returns HTTP 200 for both success and failed txs - must check tx_response.code
func checkCosmosTxError(data []byte) (bool, string) {
	var txResp cosmosTxResponse
	if err := json.Unmarshal(data, &txResp); err != nil {
		return false, "" // Not a Cosmos tx response, treat as success
	}

	// Non-zero code indicates error in Cosmos SDK
	if txResp.TxResponse.Code != cosmosSDKSuccessCode {
		return true, txResp.TxResponse.RawLog
	}

	return false, ""
}

// extractErrorMessage attempts to extract error message from response body
// Tries common fields in order: "message", "error", raw body (truncated), or fallback to status
func extractErrorMessage(data []byte, httpStatusCode int) string {
	// Try to parse as JSON and extract error message
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		// Try common error field names
		if msg, ok := result["message"].(string); ok && msg != "" {
			return msg
		}
		if msg, ok := result["error"].(string); ok && msg != "" {
			return msg
		}
		// Try nested error.message
		if errorObj, ok := result["error"].(map[string]interface{}); ok {
			if msg, ok := errorObj["message"].(string); ok && msg != "" {
				return msg
			}
		}
	}

	// Fallback: use raw body (truncated to 1KB)
	bodyStr := string(data)
	if len(bodyStr) > 1024 {
		bodyStr = bodyStr[:1024] + "..."
	}
	if bodyStr != "" {
		return bodyStr
	}

	// Final fallback: use HTTP status code
	return fmt.Sprintf("HTTP %d", httpStatusCode)
}

// checkGenericRESTError detects errors in generic REST API responses (deprecated - kept for backward compatibility)
// Expects format: {"message": "error text", "code": <error_code>}
func checkGenericRESTError(data []byte) (bool, string) {
	result := make(map[string]interface{})
	if err := json.Unmarshal(data, &result); err != nil {
		utils.LavaFormatWarning("Failed unmarshalling REST error response", err, utils.LogAttr("data", string(data)))
		return false, ""
	}

	// Valid error requires both message and code fields
	if errMsg, ok := result["message"].(string); ok {
		if _, hasCode := result["code"]; hasCode {
			return true, errMsg
		}
		utils.LavaFormatWarning("found message without code in REST response", nil, utils.LogAttr("result", result))
	}

	return false, ""
}

// GetParams will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (rm RestMessage) GetParams() interface{} {
	urlObj, err := url.Parse(rm.Path)
	if err != nil {
		return nil
	}
	parsedMethod := urlObj.Path
	objectSpec := strings.Split(rm.SpecPath, "/")
	objectPath := strings.Split(parsedMethod, "/")

	parameters := map[string]interface{}{}

	for index, element := range objectSpec {
		if strings.Contains(element, "{") {
			element = strings.Trim(element, "{}")
			parameters[element] = objectPath[index]
		}
	}
	for key, values := range urlObj.Query() {
		parameters[key] = strings.Join(values, ",")
	}
	if len(parameters) == 0 {
		return nil
	}
	return parameters
}

func (rm *RestMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	// return rm.SetLatestBlockWithHeader(latestBlock, modifyContent)
	// removed until behavior inconsistency with the cosmos sdk header is solved
	return false
	// if !done else we need a different setter
}

// GetResult will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (rm RestMessage) GetResult() json.RawMessage {
	return nil
}

func (rm RestMessage) GetMethod() string {
	return rm.Path
}

func (rm RestMessage) GetID() json.RawMessage {
	return nil
}

func (rm RestMessage) GetError() *rpcclient.JsonError {
	return nil
}

// ParseBlock parses default block number from string to int
func (rm RestMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
