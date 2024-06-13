package rpcInterfaceMessages

import (
	"github.com/goccy/go-json"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/parser"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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

func (pri ParsableRPCInput) GetMethod() string {
	return ""
}

func (pri ParsableRPCInput) GetResult() json.RawMessage {
	return pri.Result
}

func (pri ParsableRPCInput) GetID() json.RawMessage {
	return nil
}

type GenericMessage interface {
	GetHeaders() []pairingtypes.Metadata
	DisableErrorHandling()
	GetParams() interface{}
}
