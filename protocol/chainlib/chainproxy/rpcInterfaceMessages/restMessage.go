package rpcInterfaceMessages

import (
	"encoding/json"

	"github.com/lavanet/lava/protocol/parser"
)

type RestMessage struct {
	Msg  []byte
	Path string
}

// GetParams will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (cp RestMessage) GetParams() interface{} {
	return nil
}

// GetResult will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (cp RestMessage) GetResult() json.RawMessage {
	return nil
}

// ParseBlock parses default block number from string to int
func (cp RestMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}
