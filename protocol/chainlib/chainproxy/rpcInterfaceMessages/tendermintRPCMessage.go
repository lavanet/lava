package rpcInterfaceMessages

import (
	"fmt"
	"reflect"

	"github.com/goccy/go-json"

	tenderminttypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
)

type TendermintrpcMessage struct {
	JsonrpcMessage
	Path string
}

func (tm TendermintrpcMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	params, err := json.Marshal(tm.GetParams())
	if err != nil {
		utils.LavaFormatWarning("failed marshaling params", err, utils.LogAttr("request", tm))
		return ""
	}
	return string(params)
}

func (tm TendermintrpcMessage) GetParams() interface{} {
	return tm.Params
}

func (tm TendermintrpcMessage) GetResult() json.RawMessage {
	if tm.Error != nil {
		utils.LavaFormatWarning("GetResult() Request got an error from the node", nil, utils.Attribute{Key: "error", Value: tm.Error})
	}
	return tm.Result
}

func (tm TendermintrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func GetTendermintRPCError(jsonError *rpcclient.JsonError) (*tenderminttypes.RPCError, error) {
	// Guard that the jsonError exists
	//nolint
	if jsonError == nil {
		return nil, nil
	}

	var rpcError *tenderminttypes.RPCError

	errData := ""
	var ok bool

	// Make sure jsonError.Data exists
	if jsonError.Data != nil {
		errData, ok = (jsonError.Data).(string)
		if !ok {
			return nil, utils.LavaFormatError("(rpcMsg.Error.Data).(string) conversion failed", nil, utils.Attribute{Key: "data", Value: jsonError.Data})
		}
	}

	rpcError = &tenderminttypes.RPCError{
		Code:    jsonError.Code,
		Message: jsonError.Message,
		Data:    errData,
	}
	return rpcError, nil
}

func ConvertErrorToRPCError(errString string, code int) *tenderminttypes.RPCError {
	var rpcError *tenderminttypes.RPCError
	unmarshalError := json.Unmarshal([]byte(errString), &rpcError)
	if unmarshalError != nil || (rpcError.Data == "" && rpcError.Message == "") {
		utils.LavaFormatWarning("Failed unmarshalling error tendermintrpc", unmarshalError, utils.Attribute{Key: "err", Value: errString})
		rpcError = &tenderminttypes.RPCError{
			Code:    code,
			Message: "Rpc Error",
			Data:    errString,
		}
	}
	return rpcError
}

type jsonrpcId interface {
	isJSONRPCID()
}

// JSONRPCStringID a wrapper for JSON-RPC string IDs
type JSONRPCStringID string

func (JSONRPCStringID) isJSONRPCID()      {}
func (id JSONRPCStringID) String() string { return string(id) }

// JSONRPCIntID a wrapper for JSON-RPC integer IDs
type JSONRPCIntID int

func (JSONRPCIntID) isJSONRPCID()      {}
func (id JSONRPCIntID) String() string { return fmt.Sprintf("%d", id) }

func IdFromRawMessage(rawID json.RawMessage) (jsonrpcId, error) {
	var idInterface interface{}
	err := json.Unmarshal(rawID, &idInterface)
	if err != nil {
		return nil, utils.LavaFormatError("failed to unmarshal id from response", err, utils.Attribute{Key: "id", Value: string(rawID)})
	}

	switch id := idInterface.(type) {
	case string:
		return JSONRPCStringID(id), nil
	case float64:
		// json.Unmarshal uses float64 for all numbers
		return JSONRPCIntID(int(id)), nil
	case nil:
		return jsonrpcId(nil), nil
	default:
		typ := reflect.TypeOf(id)
		return nil, utils.LavaFormatError("failed to unmarshal id not a string or float", err, []utils.Attribute{{Key: "id", Value: string(rawID)}, {Key: "id type", Value: typ}}...)
	}
}

type RPCResponse struct {
	JSONRPC string                    `json:"jsonrpc"`
	ID      jsonrpcId                 `json:"id,omitempty"`
	Result  json.RawMessage           `json:"result,omitempty"`
	Error   *tenderminttypes.RPCError `json:"error,omitempty"`
}

func ConvertTendermintMsg(rpcMsg *rpcclient.JsonrpcMessage) (*RPCResponse, error) {
	// Return an error if the message was not sent
	if rpcMsg == nil {
		return nil, ErrFailedToConvertMessage
	}
	rpcError, err := GetTendermintRPCError(rpcMsg.Error)
	if err != nil {
		return nil, err
	}

	jsonid, err := IdFromRawMessage(rpcMsg.ID)
	if err != nil {
		return nil, err
	}
	msg := &RPCResponse{
		JSONRPC: rpcMsg.Version,
		ID:      jsonid,
		Result:  rpcMsg.Result,
		Error:   rpcError,
	}

	return msg, nil
}

func ConvertToTendermintError(errString string, inputInfo []byte) string {
	var msg JsonrpcMessage
	err := json.Unmarshal(inputInfo, &msg)
	if err == nil {
		id, errId := IdFromRawMessage(msg.ID)
		if errId != nil {
			utils.LavaFormatError("error idFromRawMessage", errId)
			return chainproxy.InternalErrorString
		}
		res, merr := json.Marshal(&RPCResponse{
			JSONRPC: msg.Version,
			ID:      id,
			Error:   ConvertErrorToRPCError(errString, chainproxy.LavaErrorCode),
		})
		if merr != nil {
			utils.LavaFormatError("convertToTendermintError json.Marshal", merr)
			return chainproxy.InternalErrorString
		}
		return string(res)
	}
	utils.LavaFormatError("error convertToTendermintError", err)
	return chainproxy.InternalErrorString
}
