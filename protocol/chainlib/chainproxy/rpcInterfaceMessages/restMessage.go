package rpcInterfaceMessages

import (
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
)

type RestMessage struct {
	Msg      []byte
	Path     string
	SpecPath string
	chainproxy.BaseMessage
}

func (jm RestMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	if httpStatusCode >= 200 && httpStatusCode <= 300 { // valid code
		return false, ""
	}
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
func (cp RestMessage) GetParams() interface{} {
	urlObj, err := url.Parse(cp.Path)
	if err != nil {
		return nil
	}
	parsedMethod := urlObj.Path
	objectSpec := strings.Split(cp.SpecPath, "/")
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
func (cp RestMessage) GetResult() json.RawMessage {
	return nil
}

func (cp RestMessage) GetMethod() string {
	return cp.Path
}

func (cp RestMessage) GetID() json.RawMessage {
	return nil
}

// ParseBlock parses default block number from string to int
func (cp RestMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
