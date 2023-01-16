package rpcconsumer

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/apilib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	MaxRelayRetries = 3
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
	requiredResponses      int
	finalizationConsensus  lavaprotocol.FinalizationConsensus
}

type RelayResult struct {
	request         *pairingtypes.RelayRequest
	reply           *pairingtypes.RelayReply
	providerAddress string
	replyServer     *pairingtypes.Relayer_RelaySubscribeClient
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	consumerStateTracker ConsumerStateTrackerInf,
	consumerSessionManager *lavasession.ConsumerSessionManager,
	PublicAddress string,
	requiredResponses int,
	cache *performance.Cache, // optional
) (err error) {
	rpccs.consumerSessionManager = consumerSessionManager
	rpccs.listenEndpoint = listenEndpoint
	rpccs.cache = cache
	rpccs.consumerTxSender = consumerStateTracker
	rpccs.requiredResponses = requiredResponses
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
	// construct the common data for a relay message, common data is identical across multiple sends and data reliability
	// sends a relay message to a provider
	// compares the result with other providers if defined so
	// compares the response with other consumer wallets if defined so
	// asynchronously sends data reliability if necessary

	apiMessage, err := rpccs.apiParser.ParseMsg(url, []byte(req), connectionType)
	if err != nil {
		return nil, nil, err
	}
	// Unmarshal request
	var unwantedProviders = map[string]struct{}{}

	// do this in a loop with retry attempts, configurable via a flag, limited by the number of providers in CSM
	relayRequestCommonData := lavaprotocol.NewRelayRequestCommonData(rpccs.listenEndpoint.ChainID, connectionType, url, []byte(req), apiMessage.RequestedBlock())

	relayResults := []*RelayResult{}
	relayErrors := []error{}
	for retries := 0; retries < MaxRelayRetries; retries++ {
		//TODO: make this async between different providers
		relayResult, err := rpccs.sendRelayToProvider(ctx, apiMessage, relayRequestCommonData, dappID, &unwantedProviders)
		if relayResult.providerAddress != "" {
			unwantedProviders[relayResult.providerAddress] = struct{}{}
		}
		if err != nil {
			relayErrors = append(relayErrors, err)
			if lavasession.PairingListEmptyError.Is(err) {
				// if we ran out of pairings because unwantedProviders is too long or validProviders is too short, continue to reply handling code
				break
			}
			// decide if we should break here if its something retry won't solve
			utils.LavaFormatDebug("could not send relay to provider", &map[string]string{"error": err.Error()})
			continue
		}
		relayResults = append(relayResults, relayResult)
		if len(relayResults) >= rpccs.requiredResponses {
			break
		}
	}

	for _, relayResult := range relayResults {
		go rpccs.sendDataReliabilityRelayIfApplicable(ctx, relayResult) // runs asynchronously
	}

	// TODO: secure, go over relay results to find discrepancies and choose majority, or trigger a second wallet relay
	if len(relayResults) == 0 {
		return nil, nil, utils.LavaFormatError("Failed all retries", nil, &map[string]string{"errors": fmt.Sprintf("Errors: %+v", relayErrors)})
	}
	// TODO: when secure code works, this won't be needed
	returnedResult := relayResults[0]

	return returnedResult.reply, nil, nil
}

func (rpccs *RPCConsumerServer) sendRelayToProvider(
	ctx context.Context,
	apiMessage apilib.APIMessage,
	relayRequestCommonData lavaprotocol.RelayRequestCommonData,
	dappID string,
	unwantedProviders *map[string]struct{},
) (relayResult *RelayResult, errRet error) {
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
	// in case connection totally fails, update unresponsive providers in ConsumerSessionManager

	isSubscription := apiMessage.GetInterface().Category.Subscription
	// Get Session. we get session here so we can use the epoch in the callbacks
	singleConsumerSession, epoch, providerPublicAddress, reportedProviders, err := rpccs.consumerSessionManager.GetSession(ctx, apiMessage.GetServiceApi().ComputeUnits, *unwantedProviders)
	if err != nil {
		return nil, err
	}
	privKey := rpccs.privKey
	chainID := rpccs.listenEndpoint.ChainID
	relayRequest, err := lavaprotocol.ConstructRelayRequest(ctx, privKey, chainID, relayRequestCommonData, providerPublicAddress, singleConsumerSession, int64(epoch), reportedProviders)
	if err != nil {
		return nil, err
	}
	relayResult = &RelayResult{request: relayRequest, providerAddress: providerPublicAddress}
	c := *singleConsumerSession.Endpoint.Client

	connectCtx, cancel := context.WithTimeout(ctx, lavaprotocol.GetTimePerCu(singleConsumerSession.LatestRelayCu)+lavaprotocol.AverageWorldLatency)
	defer cancel()

	var replyServer pairingtypes.Relayer_RelaySubscribeClient
	var reply *pairingtypes.RelayReply

	relaySentTime := time.Now()
	if isSubscription {
		replyServer, err = c.RelaySubscribe(ctx, relayRequest)
		if err != nil {
			return nil, err
		}
		relayResult.replyServer = &replyServer
	} else {
		reply, err = rpccs.cache.GetEntry(ctx, relayRequest, apiMessage.GetInterface().Interface, nil, chainID, false) // caching in the portal doesn't care about hashes, and we don't have data on finalization yet
		if err != nil || reply == nil {
			if performance.NotConnectedError.Is(err) {
				utils.LavaFormatError("cache not connected", err, nil)
			}
			reply, err = c.Relay(connectCtx, relayRequest)
			relayResult.reply = reply
		} else {
			// Info was fetched from cache, so we don't need to change the state
			// so we can return here, no need to update anything and calculate as this info was fetched from the cache
			relayResult.reply = reply
			return relayResult, nil
		}
	}
	currentLatency := time.Since(relaySentTime)
	_ = currentLatency
	if !isSubscription {
		// update relay request requestedBlock to the provided one in case it was arbitrary
		lavaprotocol.UpdateRequestedBlock(relayRequest, reply)
		finalized := spectypes.IsFinalizedBlock(relayRequest.RequestBlock, reply.LatestBlock, rpccs.apiParser.GetBlockDistanceForFinalizedData())
		err = lavaprotocol.VerifyRelayReply(reply, relayRequest, providerPublicAddress)
		if err != nil {
			return nil, err
		}
		// TODO: response sanity, check its under an expected format add that format to spec
		if rpccs.apiParser.DataReliabilityEnabled() {
			finalizedBlocks, err := lavaprotocol.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, singleConsumerSession.LatestBlock, rpccs.apiParser.GetBlockDistanceForFinalizedData())
			if err != nil {
				if lavaprotocol.ProviderFinzalizationDataAccountabilityError.Is(err) {
					// TODO: report this provider with conflict detection transaction
					// TODO: add necessary data (maybe previous proofs)
					go rpccs.consumerTxSender.ReportProviderForFinalizationData(ctx, reply)
				}
				return nil, err
			}
			// Compare finalized block hashes with previous providers
			// Looks for discrepancy with current epoch providers
			// if no conflicts, insert into consensus and break
			// create new consensus group if no consensus matched
			// check for discrepancy with old epoch
			err = rpccs.finalizationConsensus.CheckFinalizedHashes(int64(rpccs.apiParser.GetBlockDistanceForFinalizedData()), providerPublicAddress, singleConsumerSession.LatestBlock, finalizedBlocks, relayRequest, reply)
			if err != nil {
				return nil, err
			}
		}
		err := rpccs.cache.SetEntry(ctx, relayRequest, apiMessage.GetInterface().Interface, nil, chainID, dappID, reply, finalized) // caching in the portal doesn't care about hashes
		if err != nil && !performance.NotInitialisedError.Is(err) {
			utils.LavaFormatWarning("error updating cache with new entry", err, nil)
		}

	}

	return relayResult, nil

}

func (rpccs *RPCConsumerServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, relayResult *RelayResult) {
	// Data reliability:
	// handle data reliability VRF random value check with the lavaprotocol package
	// asynchronous: if applicable, get a data reliability session from ConsumerSessionManager
	// construct a data reliability relay message with lavaprotocol package
	// sign the data reliability relay message with the lavaprotocol package
	// send the data reliability relay message with the lavaprotocol grpc service
	// check validity of the data reliability response with the lavaprotocol package
	// compare results for both relays, if there is a difference send a detection tx with both requests and both responses
}
