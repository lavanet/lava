package rpcInterfaceMessages

import (
	"fmt"

	"github.com/goccy/go-json"

	"errors"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/parser"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/sigs"
)

var ErrFailedToConvertMessage = errors.New("failed to convert a message")

// BatchNodeErrorOnAny controls batch request error detection:
// - false (default): batch is an error only if ALL sub-requests failed
// - true: batch is an error if ANY sub-request failed (strict mode)
var BatchNodeErrorOnAny = false

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
	// Use a temporary struct that omits the Result field to avoid allocating memory for large results
	// when we only need to check for errors
	var result struct {
		Error *rpcclient.JsonError `json:"error,omitempty"`
	}
	err := json.Unmarshal(data, &result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling CheckError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}
	if result.Error == nil { // no error
		return false, ""
	}
	if result.Error.Message == "" {
		return false, ""
	}

	return true, result.Error.Message
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

	// Clear the large Result field from source after conversion
	rpcMsg.Result = nil

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
	msgs, _, err := ParseJsonRPCMsgWithBatchFlag(data)
	return msgs, err
}

// ParseJsonRPCMsgWithBatchFlag parses JSON-RPC message(s) and returns whether the input
// was a batch request (JSON array). This distinction matters for single-element batches
// like [{"id":1,"method":"getblockhash","params":[100]}] which must be treated as batch
// requests and receive array responses per the JSON-RPC spec.
func ParseJsonRPCMsgWithBatchFlag(data []byte) (msgRet []JsonrpcMessage, isBatch bool, err error) {
	// Strip UTF-8 BOM if present — some clients/proxies prepend it.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Check if the data is a JSON array (batch request) by looking at the first non-whitespace byte.
	// This must be done before unmarshaling because json.Unmarshal into a single struct may
	// silently succeed on a single-element array, losing the batch context.
	firstByte := firstNonWhitespaceByte(data)
	isBatch = firstByte == '['
	if isBatch {
		var batch []JsonrpcMessage
		err = json.Unmarshal(data, &batch)
		if err != nil {
			return nil, true, err
		}
		return batch, true, nil
	}

	var msg JsonrpcMessage
	err = json.Unmarshal(data, &msg)
	if err != nil {
		// Single-object unmarshal failed — try batch as a fallback in case our
		// first-byte heuristic was wrong (e.g. unexpected leading bytes).
		var batch []JsonrpcMessage
		if errBatch := json.Unmarshal(data, &batch); errBatch == nil {
			return batch, true, nil
		}
		return nil, false, err
	}
	if msg.ID == nil {
		msg.ID = []byte("null")
	}
	return []JsonrpcMessage{msg}, false, nil
}

// firstNonWhitespaceByte returns the first byte in data that is not
// a JSON whitespace character (space, tab, newline, carriage return).
// Returns 0 if data is empty or all whitespace.
func firstNonWhitespaceByte(data []byte) byte {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return b
		}
	}
	return 0
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
// Behavior controlled by BatchNodeErrorOnAny flag:
// - false (default): batch is an error only if ALL sub-requests failed
// - true: batch is an error if ANY sub-request failed (strict mode)
func CheckResponseErrorForJsonRpcBatch(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	// Use a temporary struct that checks for both error and result presence
	var result []struct {
		Error  *rpcclient.JsonError `json:"error,omitempty"`
		Result json.RawMessage      `json:"result,omitempty"`
	}
	err := json.Unmarshal(data, &result)
	if err != nil {
		utils.LavaFormatWarning("Failed unmarshalling CheckError", err, utils.LogAttr("data", string(data)))
		return false, ""
	}

	hasAnySuccess := false
	hasAnyError := false
	aggregatedErrors := ""

	for _, batchResult := range result {
		if batchResult.Error == nil && len(batchResult.Result) > 0 {
			hasAnySuccess = true
		}
		if batchResult.Error != nil && batchResult.Error.Message != "" {
			hasAnyError = true
			if aggregatedErrors != "" {
				aggregatedErrors += ",-," // add a unique comma separator between results
			}
			aggregatedErrors += batchResult.Error.Message
		}
	}

	// BatchNodeErrorOnAny=true (strict): error if ANY sub-request failed
	// BatchNodeErrorOnAny=false (default): error only if ALL sub-requests failed
	if BatchNodeErrorOnAny {
		// Strict mode: any error means batch failed
		return hasAnyError, aggregatedErrors
	}

	// Default mode: only error if no successes at all
	if hasAnySuccess {
		return false, ""
	}
	return aggregatedErrors != "", aggregatedErrors
}
