package rpcconsumer

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/protocol/consumerstatetracker"
	"github.com/lavanet/lava/protocol/rpcconsumer/apiparser"
	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RPCConsumerServer struct {
	txSender *consumerstatetracker.TxSender
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *RPCEndpoint,
	consumerStateTracker *consumerstatetracker.ConsumerStateTracker,
	consumerSessionManager *lavasession.ConsumerSessionManager) (err error) {

	rpccs.txSender = consumerStateTracker.TxSender
	apiParser, err := NewApiParser(listenEndpoint.ApiInterface)
	if err != nil {
		return err
	}
	consumerStateTracker.RegisterApiParserForSpecUpdates(ctx, apiParser)

	return nil
}

func NewApiParser(apiInterface string) (apiParser apiparser.APIParser, err error) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return apiparser.NewJrpcAPIParser()
	case spectypes.APIInterfaceTendermintRPC:
		return apiparser.NewTendermintRpcAPIParser()
	case spectypes.APIInterfaceRest:
		return apiparser.NewRestAPIParser()
	case spectypes.APIInterfaceGrpc:
		return apiparser.NewGrpcAPIParser()
	}
	return nil, fmt.Errorf("chain proxy for apiInterface (%s) not found", apiInterface)
}
