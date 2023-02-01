package chainproxy

import (
	"encoding/json"

	"github.com/lavanet/lava/relayer/parser"
)

type GrpcMessage struct {
	Msg  []byte
	Path string
}

func (cp GrpcMessage) GetParams() interface{} {
	return nil
}

func (cp GrpcMessage) GetResult() json.RawMessage {
	return nil
}

func (cp GrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
