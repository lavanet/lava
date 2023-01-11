package rpcconsumer

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/rpcconsumer/apilib"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

// implements Relay Sender interfaced and uses an APIListener to get it called
type RPCConsumerServer struct {
	apiParser              apilib.APIParser
	consumerSessionManager *lavasession.ConsumerSessionManager
	listenEndpoint         *lavasession.RPCEndpoint
	portalLogs             *chainproxy.PortalLogs
	cache                  *performance.Cache
	privKey                *btcec.PrivateKey
	consumerTxSender       ConsumerStateTrackerInf
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	consumerStateTracker ConsumerStateTrackerInf,
	consumerSessionManager *lavasession.ConsumerSessionManager,
	PublicAddress string,
	cache *performance.Cache, // optional
) (err error) {
	rpccs.consumerSessionManager = consumerSessionManager
	rpccs.listenEndpoint = listenEndpoint
	rpccs.cache = cache
	rpccs.consumerTxSender = consumerStateTracker
	apiParser, err := apilib.NewApiParser(listenEndpoint.ApiInterface)
	if err != nil {
		return err
	}
	rpccs.apiParser = apiParser
	consumerStateTracker.RegisterApiParserForSpecUpdates(ctx, apiParser)
	apiListener, err := apilib.NewApiListener(ctx, listenEndpoint, rpccs)
	if err != nil {
		return err
	}
	apiListener.Serve()
	return nil
}

func (rpccs *RPCConsumerServer) SendRelay(
	ctx context.Context,

	url string,
	req string,
	connectionType string,
	dappID string,
) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, error) {

	// gets the relay request data from the APIListener
	// parses the request into an APIMessage, and validating it corresponds to the spec currently in use
	// get a session for the relay from the ConsumerSessionManager
	// construct a relay message with lavaprotocol package, include QoS and jail providers
	// sign the relay message with the lavaprotocol package
	// send the relay message with the lavaprotocol grpc service
	// handle the response verification with the lavaprotocol package
	// handle data reliability provider finalization data with the lavaprotocol package
	// if necessary send detection tx for breach of data reliability provider finalization data
	// handle data reliability hashes consensus checks with the lavaprotocol package
	// if necessary send detection tx for hashes consensus mismatch
	// handle QoS updates
	// handle data reliability VRF random value check with the lavaprotocol package
	// asynchronous: if applicable, get a data reliability session from ConsumerSessionManager
	// construct a data reliability relay message with lavaprotocol package
	// sign the data reliability relay message with the lavaprotocol package
	// send the data reliability relay message with the lavaprotocol grpc service
	// check validity of the data reliability response with the lavaprotocol package
	// compare results for both relays, if there is a difference send a detection tx with both requests and both responses
	// in case connection totally fails, update unresponsive providers in ConsumerSessionManager

	apiMessage, err := rpccs.apiParser.ParseMsg(url, []byte(req), connectionType)
	if err != nil {
		return nil, nil, err
	}
	// Unmarshal request
	_ = apiMessage
	// isSubscription := apiMessage.GetInterface().Category.Subscription
	// blockHeight := int64(-1) // to sync reliability blockHeight in case it changes
	// requestedBlock := int64(0)

	// // Get Session. we get session here so we can use the epoch in the callbacks
	// singleConsumerSession, epoch, providerPublicAddress, reportedProviders, err := rpccs.consumerSessionManager.GetSession(ctx, apiMessage.GetServiceApi().ComputeUnits, nil)
	if err != nil {
		return nil, nil, err
	}
	return nil, nil, fmt.Errorf("not implemented")
}
