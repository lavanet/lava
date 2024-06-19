package lavaprotocol

import (
	"encoding/json"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/utils"
)

func CraftEmptyRPCResponseFromGenericMessage(message rpcInterfaceMessages.GenericMessage) (*rpcInterfaceMessages.RPCResponse, error) {
	createRPCResponse := func(rawId json.RawMessage) (*rpcInterfaceMessages.RPCResponse, error) {
		jsonRpcId, err := rpcInterfaceMessages.IdFromRawMessage(rawId)
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}

		jsonResponse := &rpcInterfaceMessages.RPCResponse{
			JSONRPC: "2.0",
			ID:      jsonRpcId,
			Result:  nil,
			Error:   nil,
		}

		return jsonResponse, nil
	}

	var err error
	var rpcResponse *rpcInterfaceMessages.RPCResponse
	if hasID, ok := message.(interface{ GetID() json.RawMessage }); ok {
		rpcResponse, err = createRPCResponse(hasID.GetID())
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	} else {
		rpcResponse, err = createRPCResponse([]byte("1"))
		if err != nil {
			return nil, utils.LavaFormatError("failed creating jsonrpc id", err)
		}
	}

	return rpcResponse, nil
}
