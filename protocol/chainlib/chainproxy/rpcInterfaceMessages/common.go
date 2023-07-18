package rpcInterfaceMessages

import (
	"encoding/json"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/parser"
)

type ParsableRPCInput struct {
	Result json.RawMessage
	chainproxy.BaseMessage
}

func (pri ParsableRPCInput) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (pri ParsableRPCInput) GetParams() interface{} {
	return nil
}

func (pri ParsableRPCInput) GetResult() json.RawMessage {
	return pri.Result
}
