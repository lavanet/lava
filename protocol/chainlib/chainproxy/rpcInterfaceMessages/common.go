package rpcInterfaceMessages

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/goccy/go-json"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/parser"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

var WontCalculateBatchHash = sdkerrors.New("Wont calculate batch hash", 892, "wont calculate batch message hash") // on batches we just wont calculate hashes, meaning we wont retry.

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

type GenericMessage interface {
	GetHeaders() []pairingtypes.Metadata
	DisableErrorHandling()
}
