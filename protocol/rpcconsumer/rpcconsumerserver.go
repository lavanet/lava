package rpcconsumer

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/consumerstatetracker"
	"github.com/lavanet/lava/protocol/rpcconsumer/apilib"
	"github.com/lavanet/lava/relayer/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type RPCConsumerServer struct {
	txSender *consumerstatetracker.TxSender
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	consumerStateTracker *consumerstatetracker.ConsumerStateTracker,
	consumerSessionManager *lavasession.ConsumerSessionManager,
) (err error) {
	rpccs.txSender = consumerStateTracker.TxSender
	apiParser, err := apilib.NewApiParser(listenEndpoint.ApiInterface)
	if err != nil {
		return err
	}
	consumerStateTracker.RegisterApiParserForSpecUpdates(ctx, apiParser)
	apiListener, err := apilib.NewApiListener(ctx, listenEndpoint, apiParser, rpccs)
	if err != nil {
		return err
	}
	apiListener.Serve()
	return nil
}

func (rpccs *RPCConsumerServer) SendRelay(
	ctx context.Context,
	privKey *btcec.PrivateKey,
	url string,
	req string,
	connectionType string,
	dappID string,
) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
