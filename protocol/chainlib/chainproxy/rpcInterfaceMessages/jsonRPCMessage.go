package rpcInterfaceMessages

import (
	"fmt"

	"github.com/goccy/go-json"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/parser"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"
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

func (jm *JsonrpcMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return string(reply.Result)
}

// get msg hash byte array containing all the relevant information for a unique request. (headers / api / params)
func (jm *JsonrpcMessage) GetRawRequestHash() ([]byte, error) {
	headers := jm.GetHeaders()
	headersByteArray, err := json.Marshal(headers)
	if err != nil {
		utils.LavaFormatError("Failed marshalling headers on jsonRpc message", err, utils.LogAttr("headers", headers))
		return []byte{}, err
	}

	methodByteArray := []byte(jm.Method)

	paramsByteArray, err := json.Marshal(jm.Params)
	if err != nil {
		utils.LavaFormatError("Failed marshalling params on jsonRpc message", err, utils.LogAttr("headers", jm.Params))
		return []byte{}, err
	}
	return sigs.HashMsg(append(append(methodByteArray, paramsByteArray...), headersByteArray...)), nil
}

// returns if error exists and
func (jm JsonrpcMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	result := &JsonrpcMessage{}
	err := json.Unmarshal(data, result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling CheckError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}
	if result.Error == nil { // no error
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

func (jm *JsonrpcMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	return false
}

func (jm JsonrpcMessage) NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error) {
	msg := &JsonrpcMessage{}
	err := json.Unmarshal(input, msg)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling JsonrpcMessage", err, utils.Attribute{Key: "input", Value: input})
	}

	return ParsableRPCInput{Result: msg.Result, Error: msg.Error}, nil
}

func (jm JsonrpcMessage) GetParams() interface{} {
	return jm.Params
}

func (jm JsonrpcMessage) GetMethod() string {
	return jm.Method
}

func (jm JsonrpcMessage) GetResult() json.RawMessage {
	if jm.Error != nil {
		utils.LavaFormatWarning("GetResult() Request got an error from the node", nil, utils.Attribute{Key: "error", Value: jm.Error})
	}
	return jm.Result
}

func (jm JsonrpcMessage) GetID() json.RawMessage {
	return jm.ID
}

func (jm JsonrpcMessage) GetError() *rpcclient.JsonError {
	return jm.Error
}

func (jm JsonrpcMessage) ParseBlock(inp string) (int64, error) {
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
			// failed parsing both as batch and jsonrpc return the first unmarshal error, unless the first character is "["
			if len(data) > 0 && data[0] == '[' {
				return nil, errBatch
			}
			return nil, err
		}
		return batch, nil
	}
	if msg.ID == nil {
		msg.ID = []byte("null")
	}
	return []JsonrpcMessage{msg}, nil
}

type JsonrpcBatchMessage struct {
	batch []rpcclient.BatchElemWithId
	chainproxy.BaseMessage
}

func (jbm *JsonrpcBatchMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}

// on batches we don't want to calculate the batch hash as its impossible to get the args
// we will just return false so retry wont trigger.
func (jbm JsonrpcBatchMessage) GetRawRequestHash() ([]byte, error) {
	return nil, WontCalculateBatchHash
}

func (jbm *JsonrpcBatchMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	return false
}

func (jbm *JsonrpcBatchMessage) GetBatch() []rpcclient.BatchElemWithId {
	return jbm.batch
}

func (jbm JsonrpcBatchMessage) GetParams() interface{} {
	return [][]byte{}
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
