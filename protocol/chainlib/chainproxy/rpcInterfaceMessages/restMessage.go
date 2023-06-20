package rpcInterfaceMessages

import (
	"encoding/json"
	"strings"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/parser"
)

type RestMessage struct {
	Msg      []byte
	Path     string
	SpecPath string
	chainproxy.BaseMessage
}

// GetParams will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (cp RestMessage) GetParams() interface{} {
	var parsedMethod string
	idx := strings.Index(cp.Path, "?")
	if idx == -1 {
		parsedMethod = cp.Path
	} else {
		parsedMethod = cp.Path[0:idx]
	}

	objectSpec := strings.Split(cp.SpecPath, "/")
	objectPath := strings.Split(parsedMethod, "/")

	var parameters []interface{}

	for index, element := range objectSpec {
		if strings.Contains(element, "{") {
			parameters = append(parameters, objectPath[index])
		}
	}

	return parameters
}

func (rm *RestMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	return rm.SetLatestBlockWithHeader(latestBlock, modifyContent)
	// if !done else we need a different setter
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
