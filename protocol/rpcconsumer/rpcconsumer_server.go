package rpcconsumer

import (
	"context"
	"errors"
	"math/rand"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/performance"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	MaxRelayRetries = 4
)

var NoResponseTimeout = sdkerrors.New("NoResponseTimeout Error", 685, "timeout occurred while waiting for providers responses")

// implements Relay Sender interfaced and uses an ChainListener to get it called
type RPCConsumerServer struct {
	chainParser            chainlib.ChainParser
	consumerSessionManager *lavasession.ConsumerSessionManager
	listenEndpoint         *lavasession.RPCEndpoint
	rpcConsumerLogs        *metrics.RPCConsumerLogs
	cache                  *performance.Cache
	privKey                *btcec.PrivateKey
	consumerTxSender       ConsumerTxSender
	requiredResponses      int
	finalizationConsensus  *lavaprotocol.FinalizationConsensus
	lavaChainID            string
	consumerAddress        sdk.AccAddress
	consumerServices       map[string]struct{}
}

type ConsumerTxSender interface {
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict, conflictHandler lavaprotocol.ConflictHandlerInterface) error
	GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	consumerStateTracker ConsumerStateTrackerInf,
	chainParser chainlib.ChainParser,
	finalizationConsensus *lavaprotocol.FinalizationConsensus,
	consumerSessionManager *lavasession.ConsumerSessionManager,
	requiredResponses int,
	privKey *btcec.PrivateKey,
	lavaChainID string,
	cache *performance.Cache, // optional
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	consumerAddress sdk.AccAddress,
) (err error) {
	rpccs.consumerSessionManager = consumerSessionManager
	rpccs.listenEndpoint = listenEndpoint
	rpccs.cache = cache
	rpccs.consumerTxSender = consumerStateTracker
	rpccs.requiredResponses = requiredResponses
	rpccs.lavaChainID = lavaChainID
	rpccs.rpcConsumerLogs = rpcConsumerLogs
	rpccs.privKey = privKey
	rpccs.chainParser = chainParser
	rpccs.finalizationConsensus = finalizationConsensus
	rpccs.consumerAddress = consumerAddress
	consumerPolicy, err := rpccs.consumerTxSender.GetConsumerPolicy(ctx, consumerAddress.String(), listenEndpoint.ChainID)
	if err != nil {
		return err
	}
	consumerAddons, err := consumerPolicy.GetSupportedAddons(listenEndpoint.ChainID)
	if err != nil {
		return err
	}
	consumerExtensions, err := consumerPolicy.GetSupportedExtensions(listenEndpoint.ChainID)
	if err != nil {
		return err
	}
	rpccs.consumerServices = make(map[string]struct{})
	for _, consumerAddon := range consumerAddons {
		rpccs.consumerServices[consumerAddon] = struct{}{}
	}
	for _, consumerExtension := range consumerExtensions {
		// store only relevant apiInterface extensions
		if consumerExtension.ApiInterface == listenEndpoint.ApiInterface {
			rpccs.consumerServices[consumerExtension.Extension] = struct{}{}
		}
	}
	rpccs.chainParser.SetConfiguredExtensions(rpccs.consumerServices) // configure possible extensions as set by the policy
	chainListener, err := chainlib.NewChainListener(ctx, listenEndpoint, rpccs, rpcConsumerLogs, chainParser)
	if err != nil {
		return err
	}
	go chainListener.Serve(ctx)
	// we trigger a latest block call to get some more information on our providers
	go rpccs.sendInitialRelays(MaxRelayRetries)
	return nil
}

// sending a few latest blocks relays to providers in order to have some data on the providers when relays start arriving
func (rpccs *RPCConsumerServer) sendInitialRelays(count int) {
	// only start after everythign is initialized
	reinitializedChan := make(chan bool)
	// check consumer session manager
	go func() {
		for {
			if rpccs.consumerSessionManager.Initialized() {
				reinitializedChan <- true
				return
			}
			time.Sleep(time.Second)
		}
	}()
	select {
	case <-reinitializedChan:
		break
	case <-time.After(30 * time.Second):
		utils.LavaFormatError("failed initial relays, csm was not initialised after timeout", nil, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}}...)
		return
	}

	ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
	parsing, collectionData, ok := rpccs.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	if !ok {
		utils.LavaFormatWarning("did not send initial relays because the spec does not contain "+spectypes.FUNCTION_TAG_GET_BLOCKNUM.String(), nil, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}}...)
		return
	}
	path := parsing.ApiName
	data := []byte(parsing.FunctionTemplate)
	chainMessage, err := rpccs.chainParser.ParseMsg(path, data, collectionData.Type, nil, 0)
	if err != nil {
		utils.LavaFormatError("failed creating chain message in rpc consumer init relays", err)
		return
	}
	reqBlock, _ := chainMessage.RequestedBlock()
	relayRequestData := lavaprotocol.NewRelayData(ctx, collectionData.Type, path, data, reqBlock, rpccs.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainMessage.GetApiCollection().CollectionData.AddOn, nil)
	unwantedProviders := map[string]struct{}{}
	for iter := 0; iter < count; iter++ {
		relayResult, err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, "-init-", &unwantedProviders)
		if err != nil {
			utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "unwantedProviders", Value: unwantedProviders}}...)
			unwantedProviders[relayResult.ProviderAddress] = struct{}{}
		} else {
			unwantedProviders = map[string]struct{}{}
			utils.LavaFormatInfo("[+] init relay succeeded", []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "latestBlock", Value: relayResult.Reply.LatestBlock}, {Key: "provider address", Value: relayResult.ProviderAddress}}...)
		}
	}
}

func (rpccs *RPCConsumerServer) getLatestBlock() uint64 {
	latestKnownBlock, numProviders := rpccs.finalizationConsensus.ExpectedBlockHeight(rpccs.chainParser)
	if numProviders > 0 && latestKnownBlock > 0 {
		return uint64(latestKnownBlock)
	}
	utils.LavaFormatWarning("no information on latest block", nil, utils.Attribute{Key: "latest block", Value: 0})
	return 0
}

func (rpccs *RPCConsumerServer) SendRelay(
	ctx context.Context,
	url string,
	req string,
	connectionType string,
	dappID string,
	analytics *metrics.RelayMetrics,
	metadata []pairingtypes.Metadata,
) (relayReply *pairingtypes.RelayReply, relayServer *pairingtypes.Relayer_RelaySubscribeClient, errRet error) {
	// gets the relay request data from the ChainListener
	// parses the request into an APIMessage, and validating it corresponds to the spec currently in use
	// construct the common data for a relay message, common data is identical across multiple sends and data reliability
	// sends a relay message to a provider
	// compares the result with other providers if defined so
	// compares the response with other consumer wallets if defined so
	// asynchronously sends data reliability if necessary
	relaySentTime := time.Now()
	chainMessage, err := rpccs.chainParser.ParseMsg(url, []byte(req), connectionType, metadata, rpccs.getLatestBlock())
	if err != nil {
		return nil, nil, err
	}
	if _, ok := rpccs.consumerServices[chainMessage.GetApiCollection().CollectionData.AddOn]; !ok {
		utils.LavaFormatError("unsupported addon usage, consumer policy does not allow", nil,
			utils.Attribute{Key: "addon", Value: chainMessage.GetApiCollection().CollectionData.AddOn},
			utils.Attribute{Key: "allowed", Value: rpccs.consumerServices},
		)
	}
	// Unmarshal request
	unwantedProviders := map[string]struct{}{}
	// do this in a loop with retry attempts, configurable via a flag, limited by the number of providers in CSM
	reqBlock, _ := chainMessage.RequestedBlock()
	relayRequestData := lavaprotocol.NewRelayData(ctx, connectionType, url, []byte(req), reqBlock, rpccs.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainMessage.GetApiCollection().CollectionData.AddOn, common.GetExtensionNames(chainMessage.GetExtensions()))
	relayResults := []*lavaprotocol.RelayResult{}
	relayErrors := []error{}
	blockOnSyncLoss := true
	modifiedOnLatestReq := false
	for retries := 0; retries < MaxRelayRetries; retries++ {
		// TODO: make this async between different providers
		relayResult, err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, &unwantedProviders)
		if relayResult.ProviderAddress != "" {
			if blockOnSyncLoss && lavasession.IsSessionSyncLoss(err) {
				utils.LavaFormatDebug("Identified SyncLoss in provider, not removing it from list for another attempt", utils.Attribute{Key: "address", Value: relayResult.ProviderAddress})
				blockOnSyncLoss = false // on the first sync loss no need to block the provider. give it another chance
			} else {
				unwantedProviders[relayResult.ProviderAddress] = struct{}{}
			}
		}
		if err != nil {
			relayErrors = append(relayErrors, err)
			if lavasession.PairingListEmptyError.Is(err) {
				// if we ran out of pairings because unwantedProviders is too long or validProviders is too short, continue to reply handling code
				break
			}
			// decide if we should break here if its something retry won't solve
			utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err.Error()}, utils.Attribute{Key: "endpoint", Value: rpccs.listenEndpoint})
			continue
		}
		relayResults = append(relayResults, relayResult)
		// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
		// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
		reqBlock, _ := chainMessage.RequestedBlock()
		if reqBlock == spectypes.LATEST_BLOCK {
			modifiedOnLatestReq = chainMessage.UpdateLatestBlockInMessage(relayResult.Request.RelayData.RequestBlock, false)
			if !modifiedOnLatestReq {
				relayResult.Finalized = false // shut down data reliability
			}
		}
		if len(relayResults) >= rpccs.requiredResponses {
			break
		}
	}

	enabled, dataReliabilityThreshold := rpccs.chainParser.DataReliabilityParams()
	if enabled {
		for _, relayResult := range relayResults {
			// new context is needed for data reliability as some clients cancel the context they provide when the relay returns
			// as data reliability happens in a go routine it will continue while the response returns.
			guid, found := utils.GetUniqueIdentifier(ctx)
			dataReliabilityContext := context.Background()
			if found {
				dataReliabilityContext = utils.WithUniqueIdentifier(dataReliabilityContext, guid)
			}
			go rpccs.sendDataReliabilityRelayIfApplicable(dataReliabilityContext, dappID, relayResult, chainMessage, dataReliabilityThreshold, unwantedProviders) // runs asynchronously
		}
	}

	// TODO: secure, go over relay results to find discrepancies and choose majority, or trigger a second wallet relay
	if len(relayResults) == 0 {
		return nil, nil, utils.LavaFormatError("Failed all retries", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors})
	} else if len(relayErrors) > 0 {
		utils.LavaFormatDebug("relay succeeded but had some errors", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors})
	}
	var returnedResult *lavaprotocol.RelayResult
	for _, iteratedResult := range relayResults {
		// TODO: go over rpccs.requiredResponses and get majority
		returnedResult = iteratedResult
	}

	if analytics != nil {
		currentLatency := time.Since(relaySentTime)
		analytics.Latency = currentLatency.Milliseconds()
		analytics.ComputeUnits = returnedResult.Request.RelaySession.CuSum
	}

	return returnedResult.Reply, returnedResult.ReplyServer, nil
}

func (rpccs *RPCConsumerServer) sendRelayToProvider(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayRequestData *pairingtypes.RelayPrivateData,
	dappID string,
	unwantedProviders *map[string]struct{},
) (relayResult *lavaprotocol.RelayResult, errRet error) {
	// get a session for the relay from the ConsumerSessionManager
	// construct a relay message with lavaprotocol package, include QoS and jail providers
	// sign the relay message with the lavaprotocol package
	// send the relay message
	// handle the response verification with the lavaprotocol package
	// handle data reliability provider finalization data with the lavaprotocol package
	// if necessary send detection tx for breach of data reliability provider finalization data
	// handle data reliability hashes consensus checks with the lavaprotocol package
	// if necessary send detection tx for hashes consensus mismatch
	// handle QoS updates
	// in case connection totally fails, update unresponsive providers in ConsumerSessionManager

	isSubscription := chainMessage.GetApi().Category.Subscription
	if isSubscription {
		// temporarily disable subscriptions
		// TODO: fix subscription and disable this case.
		return &lavaprotocol.RelayResult{ProviderAddress: ""}, utils.LavaFormatError("Subscriptions are not supported currently", nil)
	}

	privKey := rpccs.privKey
	chainID := rpccs.listenEndpoint.ChainID
	lavaChainID := rpccs.lavaChainID

	// Calculate extra RelayTimeout
	extraRelayTimeout := time.Duration(0)
	if chainMessage.GetApi().Category.HangingApi {
		_, extraRelayTimeout, _, _ = rpccs.chainParser.ChainBlockStats()
	}

	// Get Session. we get session here so we can use the epoch in the callbacks
	reqBlock, _ := chainMessage.RequestedBlock()
	sessions, err := rpccs.consumerSessionManager.GetSessions(ctx, chainMessage.GetApi().ComputeUnits, *unwantedProviders, reqBlock, chainMessage.GetApiCollection().CollectionData.AddOn, chainMessage.GetExtensions())
	if err != nil {
		return &lavaprotocol.RelayResult{ProviderAddress: ""}, err
	}

	type relayResponse struct {
		relayResult *lavaprotocol.RelayResult
		err         error
	}

	// Make a channel for all providers to send responses
	responses := make(chan *relayResponse, len(sessions))

	// Set relay timout
	relayTimeout := extraRelayTimeout + common.GetTimePerCu(chainMessage.GetApi().ComputeUnits) + common.AverageWorldLatency

	// Iterate over the sessions map
	for providerPublicAddress, sessionInfo := range sessions {
		// Launch a separate goroutine for each session
		go func(providerPublicAddress string, sessionInfo *lavasession.SessionInfo) {
			var localRelayResult *lavaprotocol.RelayResult
			var errResponse error
			goroutineCtx, goroutineCtxCancel := context.WithCancel(context.Background())
			guid, found := utils.GetUniqueIdentifier(ctx)
			if found {
				goroutineCtx = utils.WithUniqueIdentifier(goroutineCtx, guid)
			}
			defer func() {
				// Return response
				responses <- &relayResponse{
					relayResult: localRelayResult,
					err:         errResponse,
				}
				// Close context
				goroutineCtxCancel()
			}()

			localRelayResult = &lavaprotocol.RelayResult{
				ProviderAddress: providerPublicAddress,
				Finalized:       false,
				// setting the single consumer session as the conflict handler.
				//  to be able to validate if we need to report this provider or not.
				// holding the pointer is ok because the session is locked atm,
				// and later after its unlocked we only atomic read / write to it.
				ConflictHandler: sessionInfo.Session.Client,
			}
			localRelayRequestData := *relayRequestData

			// Extract fields from the sessionInfo
			singleConsumerSession := sessionInfo.Session
			epoch := sessionInfo.Epoch
			reportedProviders := sessionInfo.ReportedProviders

			relayRequest, errResponse := lavaprotocol.ConstructRelayRequest(goroutineCtx, privKey, lavaChainID, chainID, &localRelayRequestData, providerPublicAddress, singleConsumerSession, int64(epoch), reportedProviders)
			if errResponse != nil {
				return
			}
			localRelayResult.Request = relayRequest
			endpointClient := *singleConsumerSession.Endpoint.Client

			if isSubscription {
				localRelayResult, errResponse = rpccs.relaySubscriptionInner(goroutineCtx, endpointClient, singleConsumerSession, localRelayResult)
				if errResponse != nil {
					return
				}
			}
			requestedBlock, _ := chainMessage.RequestedBlock()
			if requestedBlock != spectypes.NOT_APPLICABLE {
				// try using cache before sending relay
				var cacheReply *pairingtypes.CacheRelayReply
				cacheReply, errResponse = rpccs.cache.GetEntry(goroutineCtx, localRelayResult.Request.RelayData, nil, chainID, false, localRelayResult.Request.RelaySession.Provider) // caching in the portal doesn't care about hashes, and we don't have data on finalization yet
				reply := cacheReply.GetReply()
				if errResponse == nil && reply != nil {
					// Info was fetched from cache, so we don't need to change the state
					// so we can return here, no need to update anything and calculate as this info was fetched from the cache
					localRelayResult.Reply = reply
					lavaprotocol.UpdateRequestedBlock(localRelayResult.Request.RelayData, reply) // update relay request requestedBlock to the provided one in case it was arbitrary
					errResponse = rpccs.consumerSessionManager.OnSessionUnUsed(singleConsumerSession)

					return
				}
			} else {
				utils.LavaFormatDebug("skipping cache due to requested block being NOT_APPLICABLE", utils.Attribute{Key: "api name", Value: chainMessage.GetApi().Name})
			}

			// cache failed, move on to regular relay
			if performance.NotConnectedError.Is(errResponse) {
				utils.LavaFormatError("cache not connected", errResponse)
			}

			localRelayResult, relayLatency, errResponse, backoff := rpccs.relayInner(goroutineCtx, singleConsumerSession, localRelayResult, relayTimeout, chainMessage)
			if errResponse != nil {
				failRelaySession := func(origErr error, backoff_ bool) {
					backOffDuration := 0 * time.Second
					if backoff_ {
						backOffDuration = lavasession.BACKOFF_TIME_ON_FAILURE
					}
					time.Sleep(backOffDuration) // sleep before releasing this singleConsumerSession
					// relay failed need to fail the session advancement
					errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
					if errReport != nil {
						utils.LavaFormatError("failed relay onSessionFailure errored", errReport, utils.Attribute{Key: "GUID", Value: goroutineCtx}, utils.Attribute{Key: "original error", Value: origErr.Error()})
					}
				}
				go failRelaySession(errResponse, backoff)

				return
			}

			// get here only if performed a regular relay successfully
			expectedBH, numOfProviders := rpccs.finalizationConsensus.ExpectedBlockHeight(rpccs.chainParser)
			pairingAddressesLen := rpccs.consumerSessionManager.GetAtomicPairingAddressesLength()
			latestBlock := localRelayResult.Reply.LatestBlock
			errResponse = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, latestBlock, chainMessage.GetApi().ComputeUnits, relayLatency, singleConsumerSession.CalculateExpectedLatency(relayTimeout), expectedBH, numOfProviders, pairingAddressesLen, chainMessage.GetApi().Category.HangingApi) // session done successfully
			// set cache in a nonblocking call
			go func() {
				requestedBlock, _ := chainMessage.RequestedBlock()
				if requestedBlock == spectypes.NOT_APPLICABLE {
					return
				}
				new_ctx := context.Background()
				new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
				defer cancel()
				err2 := rpccs.cache.SetEntry(new_ctx, localRelayResult.Request.RelayData, nil, chainID, localRelayResult.Reply, localRelayResult.Finalized, localRelayResult.Request.RelaySession.Provider, nil) // caching in the portal doesn't care about hashes
				if err2 != nil && !performance.NotInitialisedError.Is(err2) {
					utils.LavaFormatWarning("error updating cache with new entry", err2)
				}
			}()
		}(providerPublicAddress, sessionInfo)
	}

	result := make(chan *relayResponse)

	go func(timeout time.Duration) {
		responsesReceived := 0
		relayReturned := false
		for {
			select {
			case response := <-responses:
				// increase responses received
				responsesReceived++
				if response.err == nil && !relayReturned {
					// Return the first successful response
					result <- response
					relayReturned = true
				}

				if responsesReceived == len(sessions) {
					// Return the last response if all previous responses were error
					if !relayReturned {
						result <- response
					}

					// if it was returned, just close this go routine
					return
				}
			case <-time.After(relayTimeout + 2*time.Second):
				// Timeout occurred, send an error to result channel
				result <- &relayResponse{nil, NoResponseTimeout}
				return
			}
		}
	}(relayTimeout)

	response := <-result
	return response.relayResult, response.err
}

func (rpccs *RPCConsumerServer) relayInner(ctx context.Context, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *lavaprotocol.RelayResult, relayTimeout time.Duration, chainMessage chainlib.ChainMessage) (relayResultRet *lavaprotocol.RelayResult, relayLatency time.Duration, err error, needsBackoff bool) {
	existingSessionLatestBlock := singleConsumerSession.LatestBlock // we read it now because singleConsumerSession is locked, and later it's not
	endpointClient := *singleConsumerSession.Endpoint.Client
	providerPublicAddress := relayResult.ProviderAddress
	relayRequest := relayResult.Request
	callRelay := func() (reply *pairingtypes.RelayReply, relayLatency time.Duration, err error, backoff bool) {
		relaySentTime := time.Now()
		connectCtx, connectCtxCancel := context.WithTimeout(ctx, relayTimeout)
		defer connectCtxCancel()
		reply, err = endpointClient.Relay(connectCtx, relayRequest)
		relayLatency = time.Since(relaySentTime)
		if err != nil {
			backoff := false
			if errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
				backoff = true
			}
			return reply, 0, err, backoff
		}
		return reply, relayLatency, nil, false
	}
	reply, relayLatency, err, backoff := callRelay()
	if err != nil {
		return relayResult, 0, err, backoff
	}
	relayResult.Reply = reply
	lavaprotocol.UpdateRequestedBlock(relayRequest.RelayData, reply) // update relay request requestedBlock to the provided one in case it was arbitrary
	_, _, blockDistanceForFinalizedData, _ := rpccs.chainParser.ChainBlockStats()
	finalized := spectypes.IsFinalizedBlock(relayRequest.RelayData.RequestBlock, reply.LatestBlock, blockDistanceForFinalizedData)
	filteredHeaders, _, ignoredHeaders := rpccs.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	reply.Metadata = filteredHeaders
	err = lavaprotocol.VerifyRelayReply(reply, relayRequest, providerPublicAddress)
	if err != nil {
		return relayResult, 0, err, false
	}
	reply.Metadata = append(reply.Metadata, ignoredHeaders...)
	// TODO: response data sanity, check its under an expected format add that format to spec
	enabled, _ := rpccs.chainParser.DataReliabilityParams()
	if enabled {
		// TODO: DETECTION instead of existingSessionLatestBlock, we need proof of last reply to send the previous reply and the current reply
		finalizedBlocks, finalizationConflict, err := lavaprotocol.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, rpccs.consumerAddress, existingSessionLatestBlock, blockDistanceForFinalizedData)
		if err != nil {
			if lavaprotocol.ProviderFinzalizationDataAccountabilityError.Is(err) && finalizationConflict != nil {
				go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Client)
			}
			return relayResult, 0, err, false
		}

		finalizationConflict, err = rpccs.finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), providerPublicAddress, finalizedBlocks, relayRequest.RelaySession, reply)
		if err != nil {
			go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Client)
			return relayResult, 0, err, false
		}
	}
	relayResult.Finalized = finalized
	return relayResult, relayLatency, nil, false
}

func (rpccs *RPCConsumerServer) relaySubscriptionInner(ctx context.Context, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *lavaprotocol.RelayResult) (relayResultRet *lavaprotocol.RelayResult, err error) {
	// relaySentTime := time.Now()
	replyServer, err := endpointClient.RelaySubscribe(ctx, relayResult.Request)
	// relayLatency := time.Since(relaySentTime) // TODO: use subscription QoS
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return relayResult, utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "original error", Value: err.Error()})
		}
		return relayResult, err
	}
	// TODO: need to check that if provider fails and returns error, this is reflected here and we run onSessionDone
	// my thoughts are that this fails if the grpc fails not if the provider fails, and if the provider returns an error this is reflected by the Recv function on the chainListener calling us here
	// and this is too late
	relayResult.ReplyServer = &replyServer
	err = rpccs.consumerSessionManager.OnSessionDoneIncreaseCUOnly(singleConsumerSession)
	return relayResult, err
}

func (rpccs *RPCConsumerServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, dappID string, relayResult *lavaprotocol.RelayResult, chainMessage chainlib.ChainMessage, dataReliabilityThreshold uint32, unwantedProviders map[string]struct{}) error {
	// validate relayResult is not nil
	if relayResult == nil || relayResult.Reply == nil || relayResult.Request == nil {
		return utils.LavaFormatError("sendDataReliabilityRelayIfApplicable relayResult nil check", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "relayResult", Value: relayResult})
	}

	specCategory := chainMessage.GetApi().Category
	if !specCategory.Deterministic || !relayResult.Finalized {
		return nil // disabled for this spec and requested block so no data reliability messages
	}

	reqBlock, _ := chainMessage.RequestedBlock()
	if reqBlock <= spectypes.NOT_APPLICABLE {
		if reqBlock <= spectypes.LATEST_BLOCK {
			return utils.LavaFormatError("sendDataReliabilityRelayIfApplicable latest requestBlock", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "RequestBlock", Value: reqBlock})
		}
		// does not support sending data reliability requests on a block that is not specific
		return nil
	}

	if rand.Uint32() > dataReliabilityThreshold {
		// decided not to do data reliability
		return nil
	}

	relayRequestData := lavaprotocol.NewRelayData(ctx, relayResult.Request.RelayData.ConnectionType, relayResult.Request.RelayData.ApiUrl, relayResult.Request.RelayData.Data, reqBlock, relayResult.Request.RelayData.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), relayResult.Request.RelayData.Addon, relayResult.Request.RelayData.Extensions)
	relayResultDataReliability, err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, &unwantedProviders)
	if err != nil {
		errAttributes := []utils.Attribute{}
		// failed to send to a provider
		if relayResultDataReliability.ProviderAddress != "" {
			errAttributes = append(errAttributes, utils.Attribute{Key: "address", Value: relayResultDataReliability.ProviderAddress})
		}
		errAttributes = append(errAttributes, utils.Attribute{Key: "relayRequestData", Value: relayRequestData})
		return utils.LavaFormatWarning("failed data reliability relay to provider", err, errAttributes...)
	}
	if !relayResultDataReliability.Finalized {
		utils.LavaFormatInfo("skipping data reliability check since response from second provider was not finalized", utils.Attribute{Key: "providerAddress", Value: relayResultDataReliability.ProviderAddress})
		return nil
	}
	conflict := lavaprotocol.VerifyReliabilityResults(ctx, relayResult, relayResultDataReliability, chainMessage.GetApiCollection(), rpccs.chainParser)
	if conflict != nil {
		err := rpccs.consumerTxSender.TxConflictDetection(ctx, nil, conflict, nil, relayResultDataReliability.ConflictHandler)
		if err != nil {
			utils.LavaFormatError("could not send detection Transaction", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "conflict", Value: conflict})
		}
	}
	return nil
}
