package rpcconsumer

import (
	"context"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/protocol/apilib"
	"github.com/lavanet/lava/protocol/lavaprotocol"
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
	rpcConsumerLogs        *lavaprotocol.RPCConsumerLogs
	cache                  *performance.Cache
	privKey                *btcec.PrivateKey
	consumerTxSender       ConsumerStateTrackerInf
	requiredResponses      int
	finalizationConsensus  *lavaprotocol.FinalizationConsensus
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
	privKey *btcec.PrivateKey,
	cache *performance.Cache, // optional
) (err error) {
	rpccs.consumerSessionManager = consumerSessionManager
	rpccs.listenEndpoint = listenEndpoint
	rpccs.cache = cache
	rpccs.consumerTxSender = consumerStateTracker
	rpccs.requiredResponses = requiredResponses
	pLogs, err := lavaprotocol.NewRPCConsumerLogs()
	if err != nil {
		utils.LavaFormatFatal("failed creating RPCConsumer logs", err, nil)
	}
	rpccs.rpcConsumerLogs = pLogs
	rpccs.privKey = privKey
	apiParser, err := apilib.NewApiParser(listenEndpoint.ApiInterface)
	if err != nil {
		return err
	}
	rpccs.apiParser = apiParser
	consumerStateTracker.RegisterApiParserForSpecUpdates(ctx, apiParser)
	finalizationConsensus := &lavaprotocol.FinalizationConsensus{}
	consumerStateTracker.RegisterFinalizationConsensusForUpdates(ctx, finalizationConsensus)
	rpccs.finalizationConsensus = finalizationConsensus
	apiListener, err := apilib.NewApiListener(ctx, listenEndpoint, rpccs)
	if err != nil {
		return err
	}
	go apiListener.Serve()
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
	relayResult = &RelayResult{providerAddress: providerPublicAddress}
	if err != nil {
		return relayResult, err
	}
	privKey := rpccs.privKey
	chainID := rpccs.listenEndpoint.ChainID
	relayRequest, err := lavaprotocol.ConstructRelayRequest(ctx, privKey, chainID, relayRequestCommonData, providerPublicAddress, singleConsumerSession, int64(epoch), reportedProviders)
	if err != nil {
		return relayResult, err
	}
	relayResult.request = relayRequest
	endpointClient := *singleConsumerSession.Endpoint.Client

	if isSubscription {
		return rpccs.relaySubscriptionInner(ctx, endpointClient, singleConsumerSession, relayResult)
	}

	// try using cache before sending relay
	var reply *pairingtypes.RelayReply

	reply, err = rpccs.cache.GetEntry(ctx, relayRequest, apiMessage.GetInterface().Interface, nil, chainID, false) // caching in the portal doesn't care about hashes, and we don't have data on finalization yet
	if err == nil && reply != nil {
		// Info was fetched from cache, so we don't need to change the state
		// so we can return here, no need to update anything and calculate as this info was fetched from the cache
		relayResult.reply = reply
		err = rpccs.consumerSessionManager.OnSessionUnUsed(singleConsumerSession)
		return relayResult, err
	}

	// cache failed, move on to regular relay
	if performance.NotConnectedError.Is(err) {
		utils.LavaFormatError("cache not connected", err, nil)
	}

	// get here only if performed a regular relay successfully
	relayResult, relayLatency, finalized, err := rpccs.relayInner(ctx, endpointClient, singleConsumerSession, relayResult)
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return relayResult, utils.LavaFormatError("failed relay onSessionFailure errored", errReport, &map[string]string{"original error": err.Error()})
		}
	}
	expectedBH, numOfProviders := rpccs.finalizationConsensus.ExpectedBlockHeight(rpccs.apiParser)
	err = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, epoch, reply.LatestBlock, apiMessage.GetServiceApi().ComputeUnits, relayLatency, expectedBH, numOfProviders, rpccs.consumerSessionManager.GetAtomicPairingAddressesLength()) // session done successfully

	// set cache in a non blocking call
	go func() {
		err2 := rpccs.cache.SetEntry(ctx, relayRequest, apiMessage.GetInterface().Interface, nil, chainID, dappID, reply, finalized) // caching in the portal doesn't care about hashes
		if err2 != nil && !performance.NotInitialisedError.Is(err2) {
			utils.LavaFormatWarning("error updating cache with new entry", err2, nil)
		}
	}()
	return relayResult, err
}

func (rpccs *RPCConsumerServer) relayInner(ctx context.Context, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *RelayResult) (relayResultRet *RelayResult, relayLatency time.Duration, finalized bool, err error) {
	existingSessionLatestBlock := singleConsumerSession.LatestBlock // we read it now because singleConsumerSession is locked, and later it's not
	relaySentTime := time.Now()
	connectCtx, cancel := context.WithTimeout(ctx, lavaprotocol.GetTimePerCu(singleConsumerSession.LatestRelayCu)+lavaprotocol.AverageWorldLatency)
	defer cancel()
	relayRequest := relayResult.request
	providerPublicAddress := relayResult.providerAddress
	reply, err := endpointClient.Relay(connectCtx, relayRequest)
	relayLatency = time.Since(relaySentTime)
	if err != nil {
		return relayResult, 0, false, err
	}
	relayResult.reply = reply
	lavaprotocol.UpdateRequestedBlock(relayRequest, reply) // update relay request requestedBlock to the provided one in case it was arbitrary
	_, _, blockDistanceForFinalizedData := rpccs.apiParser.ChainBlockStats()
	finalized = spectypes.IsFinalizedBlock(relayRequest.RequestBlock, reply.LatestBlock, blockDistanceForFinalizedData)
	err = lavaprotocol.VerifyRelayReply(reply, relayRequest, providerPublicAddress)
	if err != nil {
		return relayResult, 0, false, err
	}

	// TODO: response data sanity, check its under an expected format add that format to spec

	if rpccs.apiParser.DataReliabilityEnabled() {
		finalizedBlocks, err := lavaprotocol.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, existingSessionLatestBlock, blockDistanceForFinalizedData)
		if err != nil {
			if lavaprotocol.ProviderFinzalizationDataAccountabilityError.Is(err) {
				// TODO: report this provider with conflict detection transaction
				// TODO: add necessary data (maybe previous proofs)
				go rpccs.consumerTxSender.ReportProviderForFinalizationData(ctx, reply)
			}
			return relayResult, 0, false, err
		}

		err = rpccs.finalizationConsensus.CheckFinalizedHashes(int64(blockDistanceForFinalizedData), providerPublicAddress, reply.LatestBlock, finalizedBlocks, relayRequest, reply)
		if err != nil {
			return relayResult, 0, false, err
		}
	}
	return relayResult, relayLatency, finalized, nil
}

func (rpccs *RPCConsumerServer) relaySubscriptionInner(ctx context.Context, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *RelayResult) (relayResultRet *RelayResult, err error) {
	// relaySentTime := time.Now()
	replyServer, err := endpointClient.RelaySubscribe(ctx, relayResult.request)
	// relayLatency := time.Since(relaySentTime) // TODO: use subscription QoS
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return relayResult, utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport, &map[string]string{"original error": err.Error()})
		}
		return relayResult, err
	}
	relayResult.replyServer = &replyServer
	err = rpccs.consumerSessionManager.OnSessionDoneIncreaseRelayAndCu(singleConsumerSession)
	return relayResult, err
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

	// sendReliabilityRelay := func(singleConsumerSession *lavasession.SingleConsumerSession, providerAddress string, differentiator bool) (relay_rep *pairingtypes.RelayReply, relay_req *pairingtypes.RelayRequest, err error) {
	// 	var dataReliabilityLatency time.Duration
	// 	s.VrfSkMu.Lock()
	// 	vrf_res, vrf_proof := utils.ProveVrfOnRelay(request, reply, s.VrfSk, differentiator, sessionEpoch)
	// 	s.VrfSkMu.Unlock()
	// 	dataReliability := &pairingtypes.VRFData{
	// 		Differentiator: differentiator,
	// 		VrfValue:       vrf_res,
	// 		VrfProof:       vrf_proof,
	// 		ProviderSig:    reply.Sig,
	// 		AllDataHash:    sigs.AllDataHash(reply, request),
	// 		QueryHash:      utils.CalculateQueryHash(*request), // calculated from query body anyway, but we will use this on payment
	// 		Sig:            nil,                                // calculated in cb_send_reliability
	// 	}
	// 	relay_rep, relay_req, dataReliabilityLatency, err = cb_send_reliability(singleConsumerSession, dataReliability, providerAddress)
	// 	if err != nil {
	// 		errRet := s.consumerSessionManager.OnDataReliabilitySessionFailure(singleConsumerSession, err)
	// 		if errRet != nil {
	// 			return nil, nil, utils.LavaFormatError("OnDataReliabilitySessionFailure Error", errRet, &map[string]string{"sendReliabilityError": err.Error()})
	// 		}
	// 		return nil, nil, utils.LavaFormatError("sendReliabilityRelay Could not get reply to reliability relay from provider", err, &map[string]string{"Address": providerAddress})
	// 	}

	// 	expectedBH, numOfProviders := s.ExpectedBlockHeight()
	// 	err = s.consumerSessionManager.OnDataReliabilitySessionDone(singleConsumerSession, relay_rep.LatestBlock, singleConsumerSession.LatestRelayCu, dataReliabilityLatency, expectedBH, numOfProviders, s.GetProvidersCount())
	// 	return relay_rep, relay_req, err
	// }

	// checkReliability := func() {
	// 	numberOfReliabilitySessions := len(dataReliabilitySessions)
	// 	if numberOfReliabilitySessions > supportedNumberOfVRFs {
	// 		utils.LavaFormatError("Trying to use DataReliability with more than two vrf sessions, currently not supported", nil, &map[string]string{"number_of_DataReliabilitySessions": strconv.Itoa(numberOfReliabilitySessions)})
	// 		return
	// 	} else if numberOfReliabilitySessions == 0 {
	// 		return
	// 	}
	// 	// apply first request and reply to dataReliabilityVerifications
	// 	originalDataReliabilityResult := &DataReliabilityResult{reply: reply, relayRequest: request, providerPublicAddress: providerPubAddress}
	// 	dataReliabilityVerifications := make([]*DataReliabilityResult, 0)

	// 	for _, dataReliabilitySession := range dataReliabilitySessions {
	// 		reliabilityReply, reliabilityRequest, err := sendReliabilityRelay(dataReliabilitySession.singleConsumerSession, dataReliabilitySession.providerPublicAddress, dataReliabilitySession.uniqueIdentifier)
	// 		if err == nil && reliabilityReply != nil {
	// 			dataReliabilityVerifications = append(dataReliabilityVerifications,
	// 				&DataReliabilityResult{
	// 					reply:                 reliabilityReply,
	// 					relayRequest:          reliabilityRequest,
	// 					providerPublicAddress: dataReliabilitySession.providerPublicAddress,
	// 				})
	// 		}
	// 	}
	// 	if len(dataReliabilityVerifications) > 0 {
	// 		s.verifyReliabilityResults(originalDataReliabilityResult, dataReliabilityVerifications, numberOfReliabilitySessions)
	// 	}
	// }
	// go checkReliability()

}
