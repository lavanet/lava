package rpcInterfaceMessages

import (
	"encoding/json"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
)

var ErrFailedToConvertMessage = sdkerrors.New("RPC error", 1000, "failed to convert a message")

type JsonrpcMessage struct {
	Version string               `json:"jsonrpc,omitempty"`
	ID      json.RawMessage      `json:"id,omitempty"`
	Method  string               `json:"method,omitempty"`
	Params  interface{}          `json:"params,omitempty"`
	Error   *rpcclient.JsonError `json:"error,omitempty"`
	Result  json.RawMessage      `json:"result,omitempty"`
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

func (gm JsonrpcMessage) NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error) {
	msg := &JsonrpcMessage{}
	err := json.Unmarshal(input, msg)
	if err != nil {
		return nil, utils.LavaFormatError("failed unmarshaling JsonrpcMessage", err, &map[string]string{"input": string(input)})
	}
	return ParsableRPCInput{Result: msg.Result}, nil
}

func (cp JsonrpcMessage) GetParams() interface{} {
	return cp.Params
}

func (cp JsonrpcMessage) GetResult() json.RawMessage {
	return cp.Result
}

func (cp JsonrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func ParseJsonRPCMsg(data []byte) (msgRet *JsonrpcMessage, err error) {
	// connectionType is currently only used in rest API.
	// Unmarshal request
	var msg JsonrpcMessage
	err = json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
