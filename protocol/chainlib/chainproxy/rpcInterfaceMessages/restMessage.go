package rpcInterfaceMessages

import (
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/parser"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
)

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
	// First, try to parse as Cosmos SDK transaction response
	// Cosmos SDK returns HTTP 200 for both success and error transactions
	// We need to check the tx_response.code field to determine if it's an error
	if httpStatusCode >= 200 && httpStatusCode <= 300 {
		// Try to parse as Cosmos transaction response
		type CosmosTxResponse struct {
			TxResponse struct {
				Code   int    `json:"code"`
				RawLog string `json:"raw_log"`
			} `json:"tx_response"`
		}

		var txResp CosmosTxResponse
		if err := json.Unmarshal(data, &txResp); err == nil {
			// If we successfully parsed a tx_response structure and it has an error
			if txResp.TxResponse.Code != 0 {
				// Non-zero code means error in Cosmos SDK
				return true, txResp.TxResponse.RawLog
			}
		}
		// If it's not a Cosmos tx response or code is 0 (success), continue with normal flow
		return false, ""
	}

	// For non-2xx HTTP status codes, check for error in response body
	result := make(map[string]interface{}, 0)
	err := json.Unmarshal(data, &result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling RestMessage CheckResponseError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}
	// make sure we have both message and code for an error message.
	if errMsg, okMessage := result["message"].(string); okMessage {
		if _, okCode := result["code"]; okCode {
			return true, errMsg
		}
		utils.LavaFormatWarning("found only message without code in returned result", nil, utils.LogAttr("result", result))
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
