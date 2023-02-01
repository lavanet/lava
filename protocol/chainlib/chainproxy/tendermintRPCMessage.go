package chainproxy

import (
	"encoding/json"

	"github.com/lavanet/lava/relayer/parser"
)

type TendermintrpcMessage struct {
	JsonrpcMessage
	Path string
}

func (cp TendermintrpcMessage) GetParams() interface{} {
	return cp.Params
}

func (cp TendermintrpcMessage) GetResult() json.RawMessage {
	return cp.Result
}

func (cp TendermintrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
