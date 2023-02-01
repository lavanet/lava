package chainproxy

import (
	"encoding/json"

	"github.com/lavanet/lava/relayer/parser"
)

type RestMessage struct {
	Msg  []byte
	Path string
}

func (cp RestMessage) GetParams() interface{} {
	return nil
}

func (cp RestMessage) GetResult() json.RawMessage {
	return nil
}

func (cp RestMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
