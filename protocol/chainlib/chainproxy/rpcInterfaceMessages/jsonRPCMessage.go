package rpcInterfaceMessages

import (
	"fmt"

	"github.com/goccy/go-json"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/parser"
	"github.com/lavanet/lava/v2/utils"
)

var ErrFailedToConvertMessage = sdkerrors.New("RPC error", 1000, "failed to convert a message")

type JsonrpcMessage struct {
	Version                string               `json:"jsonrpc,omitempty"`
	ID                     json.RawMessage      `json:"id,omitempty"`
	Method                 string               `json:"method,omitempty"`
	Params                 interface{}          `json:"params,omitempty"`
	Error                  *rpcclient.JsonError `json:"error,omitempty"`
	Result                 json.RawMessage      `json:"result,omitempty"`
	chainproxy.BaseMessage `json:"-"`
}

// returns if error exists and
func (jm JsonrpcMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	result := &JsonrpcMessage{}
	err := json.Unmarshal(data, result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling CheckError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}
	if result.Error == nil {
		return false, ""
	}
	return result.Error.Message != "", result.Error.Message
}

func ConvertJsonRPCMsg(rpcMsg *rpcclient.JsonrpcMessage) (*JsonrpcMessage, error) {
	// Return an error if the message was not sent
	if rpcMsg == nil {
		return nil, ErrFailedToConvertMessage
	}

	msg := &JsonrpcMessage{
		Version: rpcMsg.Version,
		ID:      rpcMsg.ID,
		Method:  rpcMsg.Method,
		Error:   rpcMsg.Error,
		Result:  rpcMsg.Result,
	}

	if rpcMsg.Params != nil {
		msg.Params = rpcMsg.Params
	}

	return msg, nil
}

func ConvertBatchElement(batchElement rpcclient.BatchElemWithId) (JsonrpcMessage, error) {
	var JsonError *rpcclient.JsonError
	var ok bool
	if batchElement.Error != nil {
		JsonError, ok = batchElement.Error.(*rpcclient.JsonError)
		if !ok {
			return JsonrpcMessage{}, batchElement.Error
		}
	}
	var result json.RawMessage
	if batchElement.Result != nil {
		resultRef, ok := batchElement.Result.(*json.RawMessage)
		if !ok {
			return JsonrpcMessage{}, batchElement.Error
		}
		result = *resultRef
	}
	msg := JsonrpcMessage{
		Version: rpcclient.Vsn,
		ID:      batchElement.ID,
		Error:   JsonError,
		Result:  result,
	}

	return msg, nil
}

func (gm *JsonrpcMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	return false
}

func (gm JsonrpcMessage) NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error) {
	msg := &JsonrpcMessage{}
	err := json.Unmarshal(input, msg)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling JsonrpcMessage", err, utils.Attribute{Key: "input", Value: input})
	}

	// Make sure the response does not have an error
	if msg.Error != nil && msg.Result == nil {
		return nil, utils.LavaFormatError("response is an error message", msg.Error)
	}
	return ParsableRPCInput{Result: msg.Result}, nil
}

func (cp JsonrpcMessage) GetParams() interface{} {
	return cp.Params
}

func (cp JsonrpcMessage) GetMethod() string {
	return cp.Method
}

func (cp JsonrpcMessage) GetResult() json.RawMessage {
	if cp.Error != nil {
		utils.LavaFormatWarning("GetResult() Request got an error from the node", nil, utils.Attribute{Key: "error", Value: cp.Error})
	}
	return cp.Result
}

func (cp JsonrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func ParseJsonRPCMsg(data []byte) (msgRet []JsonrpcMessage, err error) {
	// connectionType is currently only used in rest API.
	// Unmarshal request
	var msg JsonrpcMessage
	err = json.Unmarshal(data, &msg)
	if err != nil {
		// we failed unmarshaling
		// try to parse a batch
		var batch []JsonrpcMessage
		errBatch := json.Unmarshal(data, &batch)
		if errBatch != nil {
			// failed parsing both as batch and jsonrpc return the first unmarshal error, unless the first charqacter is "["
			if len(data) > 0 && data[0] == '[' {
				return nil, errBatch
			}
			return nil, err
		}
		return batch, nil
	}
	return []JsonrpcMessage{msg}, nil
}

type JsonrpcBatchMessage struct {
	batch []rpcclient.BatchElemWithId
	chainproxy.BaseMessage
}

func (jbm *JsonrpcBatchMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	return false
}

func (jbm *JsonrpcBatchMessage) GetBatch() []rpcclient.BatchElemWithId {
	return jbm.batch
}

func NewBatchMessage(msgs []JsonrpcMessage) (JsonrpcBatchMessage, error) {
	batch := make([]rpcclient.BatchElemWithId, len(msgs))
	for idx, msg := range msgs {
		switch params := msg.Params.(type) {
		case []interface{}, map[string]interface{}, nil:
		default:
			return JsonrpcBatchMessage{}, fmt.Errorf("invalid params in batch, batching only supports empty, ordered or dictionary arguments  %s %+v", msg.Method, params)
		}
		element, err := rpcclient.NewBatchElementWithId(msg.Method, msg.Params, &json.RawMessage{}, msg.ID)
		if err != nil {
			return JsonrpcBatchMessage{}, err
		}
		batch[idx] = element
	}
	return JsonrpcBatchMessage{batch: batch}, nil
}

// returns if error exists and
func CheckResponseErrorForJsonRpcBatch(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	result := []JsonrpcMessage{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling CheckError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}
	aggregatedResults := ""
	numberOfBatchElements := len(result)
	for idx, batchResult := range result {
		if batchResult.Error == nil {
			continue
		}
		if batchResult.Error.Message != "" {
			aggregatedResults += batchResult.Error.Message
			if idx < numberOfBatchElements-1 {
				aggregatedResults += ",-," // add a unique comma separator between results
			}
		}
	}
	return aggregatedResults != "", aggregatedResults
}
