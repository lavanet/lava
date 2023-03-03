package chainproxy

import (
	"encoding/json"

	"github.com/lavanet/lava/relayer/parser"
)

const (
	LavaErrorCode       = 555
	InternalErrorString = "Internal Error"
)

type CustomParsingMessage interface {
	NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error)
}

type DefaultRPCInput struct {
	Result json.RawMessage
}

func (dri DefaultRPCInput) GetParams() interface{} {
	return nil
}

func (dri DefaultRPCInput) GetResult() json.RawMessage {
	return dri.Result
}

func (dri DefaultRPCInput) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func DefaultParsableRPCInput(input json.RawMessage) parser.RPCInput {
	return DefaultRPCInput{Result: input}
}
