package rpcInterfaceMessages

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/parser"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

var WontCalculateBatchHash = sdkerrors.New("Wont calculate batch hash", 892, "wont calculate batch message hash") // on batches we just wont calculate hashes, meaning we wont retry.

type ParsableRPCInput struct {
	Result json.RawMessage
	Error  *rpcclient.JsonError
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

func (pri ParsableRPCInput) GetError() *rpcclient.JsonError {
	return pri.Error
}

type GenericMessage interface {
	GetHeaders() []pairingtypes.Metadata
	DisableErrorHandling()
	GetParams() interface{}
}
