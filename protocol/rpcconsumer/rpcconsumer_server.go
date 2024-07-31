package rpcconsumer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavaprotocol"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/metrics"
	"github.com/lavanet/lava/v2/protocol/performance"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/protocopy"
	"github.com/lavanet/lava/v2/utils/rand"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// maximum number of retries to send due to the ticker, if we didn't get a response after 10 different attempts then just wait.
	MaximumNumberOfTickerRelayRetries        = 10
	MaxRelayRetries                          = 6
	SendRelayAttempts                        = 3
	numberOfTimesToCheckCurrentlyUsedIsEmpty = 3
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
	ConsumerAddress        sdk.AccAddress
	consumerConsistency    *ConsumerConsistency
	sharedState            bool // using the cache backend to sync the latest seen block with other consumers
	relaysMonitor          *metrics.RelaysMonitor
	reporter               metrics.Reporter
	debugRelays            bool
}

type relayResponse struct {
	relayResult common.RelayResult
	err         error
}

type ConsumerTxSender interface {
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict, conflictHandler common.ConflictHandlerInterface) error
	GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
	GetLatestVirtualEpoch() uint64
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
	consumerConsistency *ConsumerConsistency,
	relaysMonitor *metrics.RelaysMonitor,
	cmdFlags common.ConsumerCmdFlags,
	sharedState bool,
	refererData *chainlib.RefererData,
	reporter metrics.Reporter,
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
	rpccs.ConsumerAddress = consumerAddress
	rpccs.consumerConsistency = consumerConsistency
	rpccs.sharedState = sharedState
	rpccs.reporter = reporter
	rpccs.debugRelays = cmdFlags.DebugRelays
	chainListener, err := chainlib.NewChainListener(ctx, listenEndpoint, rpccs, rpccs, rpcConsumerLogs, chainParser, refererData)
	if err != nil {
		return err
	}

	go chainListener.Serve(ctx, cmdFlags)

	initialRelays := true
	rpccs.relaysMonitor = relaysMonitor

	// we trigger a latest block call to get some more information on our providers, using the relays monitor
	if cmdFlags.RelaysHealthEnableFlag {
		rpccs.relaysMonitor.SetRelaySender(func() (bool, error) {
			success, err := rpccs.sendCraftedRelaysWrapper(initialRelays)
			if success {
				initialRelays = false
			}
			return success, err
		})
		rpccs.relaysMonitor.Start(ctx)
	} else {
		rpccs.sendCraftedRelaysWrapper(true)
	}
	return nil
}

func (rpccs *RPCConsumerServer) sendCraftedRelaysWrapper(initialRelays bool) (bool, error) {
	if initialRelays {
		// Only start after everything is initialized - check consumer session manager
		rpccs.waitForPairing()
	}

	return rpccs.sendCraftedRelays(MaxRelayRetries, initialRelays)
}

func (rpccs *RPCConsumerServer) waitForPairing() {
	reinitializedChan := make(chan bool)

	go func() {
		for {
			if rpccs.consumerSessionManager.Initialized() {
				reinitializedChan <- true
				return
			}
			time.Sleep(time.Second)
		}
	}()

	numberOfTimesChecked := 0
	for {
		select {
		case <-reinitializedChan:
			return
		case <-time.After(30 * time.Second):
			numberOfTimesChecked += 1
			utils.LavaFormatWarning("failed initial relays, csm was not initialized after timeout, or pairing list is empty for that chain", nil,
				utils.LogAttr("times_checked", numberOfTimesChecked),
				utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
				utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface),
			)
		}
	}
}

func (rpccs *RPCConsumerServer) craftRelay(ctx context.Context) (ok bool, relay *pairingtypes.RelayPrivateData, chainMessage chainlib.ChainMessage, err error) {
	parsing, collectionData, ok := rpccs.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	if !ok {
		return false, nil, nil, utils.LavaFormatWarning("did not send initial relays because the spec does not contain "+spectypes.FUNCTION_TAG_GET_BLOCKNUM.String(), nil,
			utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface),
		)
	}

	path := parsing.ApiName
	data := []byte(parsing.FunctionTemplate)
	chainMessage, err = rpccs.chainParser.ParseMsg(path, data, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	if err != nil {
		return false, nil, nil, utils.LavaFormatError("failed creating chain message in rpc consumer init relays", err,
			utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface))
	}

	reqBlock, _ := chainMessage.RequestedBlock()
	seenBlock := int64(0)
	relay = lavaprotocol.NewRelayData(ctx, collectionData.Type, path, data, seenBlock, reqBlock, rpccs.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMessage), nil)
	return
}

func (rpccs *RPCConsumerServer) sendRelayWithRetries(ctx context.Context, retries int, initialRelays bool, relay *pairingtypes.RelayPrivateData, chainMessage chainlib.ChainMessage) (bool, error) {
	success := false
	var err error
	relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(nil), 1, chainMessage, rpccs.consumerConsistency, "-init-", "", rpccs.debugRelays, rpccs.rpcConsumerLogs, rpccs)
	for i := 0; i < retries; i++ {
		err = rpccs.sendRelayToProvider(ctx, chainMessage, relay, "-init-", "", relayProcessor, nil)
		if lavasession.PairingListEmptyError.Is(err) {
			// we don't have pairings anymore, could be related to unwanted providers
			relayProcessor.GetUsedProviders().ClearUnwanted()
			err = rpccs.sendRelayToProvider(ctx, chainMessage, relay, "-init-", "", relayProcessor, nil)
		}
		if err != nil {
			utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
		} else {
			err := relayProcessor.WaitForResults(ctx)
			if err != nil {
				utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
			} else {
				relayResult, err := relayProcessor.ProcessingResult()
				if err == nil {
					utils.LavaFormatInfo("[+] init relay succeeded", []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "latestBlock", Value: relayResult.Reply.LatestBlock}, {Key: "provider address", Value: relayResult.ProviderInfo.ProviderAddress}}...)
					rpccs.relaysMonitor.LogRelay()
					success = true
					// If this is the first time we send relays, we want to send all of them, instead of break on first successful relay
					// That way, we populate the providers with the latest blocks with successful relays
					if !initialRelays {
						break
					}
				} else {
					utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
				}
			}
		}
		time.Sleep(2 * time.Millisecond)
	}

	return success, err
}

// sending a few latest blocks relays to providers in order to have some data on the providers when relays start arriving
func (rpccs *RPCConsumerServer) sendCraftedRelays(retries int, initialRelays bool) (success bool, err error) {
	utils.LavaFormatDebug("Sending crafted relays",
		utils.LogAttr("chainId", rpccs.listenEndpoint.ChainID),
		utils.LogAttr("apiInterface", rpccs.listenEndpoint.ApiInterface),
	)

	ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
	ok, relay, chainMessage, err := rpccs.craftRelay(ctx)
	if !ok {
		enabled, _ := rpccs.chainParser.DataReliabilityParams()
		// if DR is disabled it's okay to not have GET_BLOCKNUM
		if !enabled {
			return true, nil
		}
		return false, err
	}

	return rpccs.sendRelayWithRetries(ctx, retries, initialRelays, relay, chainMessage)
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
	consumerIp string,
	analytics *metrics.RelayMetrics,
	metadata []pairingtypes.Metadata,
) (relayResult *common.RelayResult, errRet error) {
	// gets the relay request data from the ChainListener
	// parses the request into an APIMessage, and validating it corresponds to the spec currently in use
	// construct the common data for a relay message, common data is identical across multiple sends and data reliability
	// sends a relay message to a provider
	// compares the result with other providers if defined so
	// compares the response with other consumer wallets if defined so
	// asynchronously sends data reliability if necessary

	// remove lava directive headers
	metadata, directiveHeaders := rpccs.LavaDirectiveHeaders(metadata)
	relaySentTime := time.Now()
	chainMessage, err := rpccs.chainParser.ParseMsg(url, []byte(req), connectionType, metadata, rpccs.getExtensionsFromDirectiveHeaders(directiveHeaders))
	if err != nil {
		return nil, err
	}
	// temporarily disable subscriptions
	isSubscription := chainlib.IsSubscription(chainMessage)
	if isSubscription {
		return &common.RelayResult{ProviderInfo: common.ProviderInfo{ProviderAddress: ""}}, utils.LavaFormatError("Subscriptions are not supported at the moment", nil)
	}

	rpccs.HandleDirectiveHeadersForMessage(chainMessage, directiveHeaders)
	// do this in a loop with retry attempts, configurable via a flag, limited by the number of providers in CSM
	reqBlock, _ := chainMessage.RequestedBlock()
	seenBlock, _ := rpccs.consumerConsistency.GetSeenBlock(dappID, consumerIp)
	if seenBlock < 0 {
		seenBlock = 0
	}
	relayRequestData := lavaprotocol.NewRelayData(ctx, connectionType, url, []byte(req), seenBlock, reqBlock, rpccs.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMessage), common.GetExtensionNames(chainMessage.GetExtensions()))

	relayProcessor, err := rpccs.ProcessRelaySend(ctx, directiveHeaders, chainMessage, relayRequestData, dappID, consumerIp, analytics)
	if err != nil && !relayProcessor.HasResults() {
		// we can't send anymore, and we don't have any responses
		utils.LavaFormatError("failed getting responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.LogAttr("endpoint", rpccs.listenEndpoint.Key()), utils.LogAttr("userIp", consumerIp), utils.LogAttr("relayProcessor", relayProcessor))
		return nil, err
	}
	// Handle Data Reliability
	enabled, dataReliabilityThreshold := rpccs.chainParser.DataReliabilityParams()
	// check if data reliability is enabled and relay processor allows us to perform data reliability
	if enabled && !relayProcessor.getSkipDataReliability() {
		// new context is needed for data reliability as some clients cancel the context they provide when the relay returns
		// as data reliability happens in a go routine it will continue while the response returns.
		guid, found := utils.GetUniqueIdentifier(ctx)
		dataReliabilityContext := context.Background()
		if found {
			dataReliabilityContext = utils.WithUniqueIdentifier(dataReliabilityContext, guid)
		}
		go rpccs.sendDataReliabilityRelayIfApplicable(dataReliabilityContext, dappID, consumerIp, chainMessage, dataReliabilityThreshold, relayProcessor) // runs asynchronously
	}

	returnedResult, err := relayProcessor.ProcessingResult()
	rpccs.appendHeadersToRelayResult(ctx, returnedResult, relayProcessor.ProtocolErrors(), relayProcessor, directiveHeaders)
	if err != nil {
		return returnedResult, utils.LavaFormatError("failed processing responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.LogAttr("endpoint", rpccs.listenEndpoint.Key()))
	}
	if analytics != nil {
		currentLatency := time.Since(relaySentTime)
		analytics.Latency = currentLatency.Milliseconds()
		api := chainMessage.GetApi()
		analytics.ComputeUnits = api.ComputeUnits
		analytics.ApiMethod = api.Name
	}
	rpccs.relaysMonitor.LogRelay()
	return returnedResult, nil
}

func (rpccs *RPCConsumerServer) GetChainIdAndApiInterface() (string, string) {
	return rpccs.listenEndpoint.ChainID, rpccs.listenEndpoint.ApiInterface
}

func (rpccs *RPCConsumerServer) ProcessRelaySend(ctx context.Context, directiveHeaders map[string]string, chainMessage chainlib.ChainMessage, relayRequestData *pairingtypes.RelayPrivateData, dappID string, consumerIp string, analytics *metrics.RelayMetrics) (*RelayProcessor, error) {
	// make sure all of the child contexts are cancelled when we exit
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	relayProcessor := NewRelayProcessor(ctx, lavasession.NewUsedProviders(directiveHeaders), rpccs.requiredResponses, chainMessage, rpccs.consumerConsistency, dappID, consumerIp, rpccs.debugRelays, rpccs.rpcConsumerLogs, rpccs)
	var err error
	// try sending a relay 3 times. if failed return the error
	for retryFirstRelayAttempt := 0; retryFirstRelayAttempt < SendRelayAttempts; retryFirstRelayAttempt++ {
		// record the relay analytics only on the first attempt.
		if analytics != nil && retryFirstRelayAttempt > 0 {
			analytics = nil
		}
		err = rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, consumerIp, relayProcessor, analytics)

		// check if we had an error. if we did, try again.
		if err == nil {
			break
		}
		utils.LavaFormatWarning("Failed retryFirstRelayAttempt, will retry.", err, utils.LogAttr("attempt", retryFirstRelayAttempt))
	}

	if err != nil {
		return relayProcessor, err
	}

	// a channel to be notified processing was done, true means we have results and can return
	gotResults := make(chan bool)
	processingTimeout, relayTimeout := rpccs.getProcessingTimeout(chainMessage)
	if rpccs.debugRelays {
		utils.LavaFormatDebug("Relay initiated with the following timeout schedule", utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("newRelayTimeout", relayTimeout))
	}
	// create the processing timeout prior to entering the method so it wont reset every time
	processingCtx, processingCtxCancel := context.WithTimeout(ctx, processingTimeout)
	defer processingCtxCancel()

	readResultsFromProcessor := func() {
		// ProcessResults is reading responses while blocking until the conditions are met
		relayProcessor.WaitForResults(processingCtx)
		// decide if we need to resend or not
		if relayProcessor.HasRequiredNodeResults() {
			gotResults <- true
		} else {
			gotResults <- false
		}
	}
	go readResultsFromProcessor()

	returnCondition := make(chan error)
	// used for checking whether to return an error to the user or to allow other channels return their result first see detailed description on the switch case below
	validateReturnCondition := func(err error) {
		currentlyUsedIsEmptyCounter := 0
		if err != nil {
			for validateNoProvidersAreUsed := 0; validateNoProvidersAreUsed < numberOfTimesToCheckCurrentlyUsedIsEmpty; validateNoProvidersAreUsed++ {
				if relayProcessor.usedProviders.CurrentlyUsed() == 0 {
					currentlyUsedIsEmptyCounter++
				}
				time.Sleep(5 * time.Millisecond)
			}
			// we failed to send a batch of relays, if there are no active sends we can terminate after validating X amount of times to make sure no racing channels
			if currentlyUsedIsEmptyCounter >= numberOfTimesToCheckCurrentlyUsedIsEmpty {
				returnCondition <- err
			}
		}
	}
	// every relay timeout we send a new batch
	startNewBatchTicker := time.NewTicker(relayTimeout)
	defer startNewBatchTicker.Stop()
	numberOfRetriesLaunched := 0
	for {
		select {
		case success := <-gotResults:
			if success { // check wether we can return the valid results or we need to send another relay
				return relayProcessor, nil
			}
			// if we don't need to retry return what we currently have
			if !relayProcessor.ShouldRetry(numberOfRetriesLaunched) {
				return relayProcessor, nil
			}
			// otherwise continue sending another relay
			err := rpccs.sendRelayToProvider(processingCtx, chainMessage, relayRequestData, dappID, consumerIp, relayProcessor, nil)
			go validateReturnCondition(err)
			go readResultsFromProcessor()
			// increase number of retries launched only if we still have pairing available, if we exhausted the list we don't want to break early
			// so it will just wait for the entire duration of the relay
			if !lavasession.PairingListEmptyError.Is(err) {
				numberOfRetriesLaunched++
			}
		case <-startNewBatchTicker.C:
			// only trigger another batch for non BestResult relays or if we didn't pass the retry limit.
			if relayProcessor.ShouldRetry(numberOfRetriesLaunched) {
				// limit the number of retries called from the new batch ticker flow.
				// if we pass the limit we just wait for the relays we sent to return.
				err := rpccs.sendRelayToProvider(processingCtx, chainMessage, relayRequestData, dappID, consumerIp, relayProcessor, nil)
				go validateReturnCondition(err)
				// add ticker launch metrics
				go rpccs.rpcConsumerLogs.SetRelaySentByNewBatchTickerMetric(rpccs.GetChainIdAndApiInterface())
				// increase number of retries launched only if we still have pairing available, if we exhausted the list we don't want to break early
				// so it will just wait for the entire duration of the relay
				if !lavasession.PairingListEmptyError.Is(err) {
					numberOfRetriesLaunched++
				}
			}
		case returnErr := <-returnCondition:
			// we use this channel because there could be a race condition between us releasing the provider and about to send the return
			// to an error happening on another relay processor's routine. this can cause an error that returns to the user
			// if we don't release the case, it will cause the success case condition to not be executed
			// detailed scenario:
			// sending first relay -> waiting -> sending second relay -> getting an error on the second relay (not returning yet) ->
			// -> (in parallel) first relay finished, removing from CurrentlyUsed providers -> checking currently used (on second failed relay) -> returning error instead of the successful relay.
			// by releasing the case we allow the channel to be chosen again by the successful case.
			return relayProcessor, returnErr
		case <-processingCtx.Done():
			// in case we got a processing timeout we return context deadline exceeded to the user.
			utils.LavaFormatWarning("Relay Got processingCtx timeout", nil,
				utils.LogAttr("processingTimeout", processingTimeout),
				utils.LogAttr("dappId", dappID),
				utils.LogAttr("consumerIp", consumerIp),
				utils.LogAttr("chainMessage.GetApi().Name", chainMessage.GetApi().Name),
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("relayProcessor", relayProcessor),
			)
			return relayProcessor, processingCtx.Err() // returning the context error
		}
	}
}

func (rpccs *RPCConsumerServer) sendRelayToProvider(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayRequestData *pairingtypes.RelayPrivateData,
	dappID string,
	consumerIp string,
	relayProcessor *RelayProcessor,
	analytics *metrics.RelayMetrics,
) (errRet error) {
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
	isSubscription := chainlib.IsSubscription(chainMessage)
	var sharedStateId string // defaults to "", if shared state is disabled then no shared state will be used.
	if rpccs.sharedState {
		sharedStateId = rpccs.consumerConsistency.Key(dappID, consumerIp) // use same key as we use for consistency, (for better consistency :-D)
	}

	privKey := rpccs.privKey
	chainId, apiInterface := rpccs.GetChainIdAndApiInterface()
	lavaChainID := rpccs.lavaChainID

	// Get Session. we get session here so we can use the epoch in the callbacks
	reqBlock, _ := chainMessage.RequestedBlock()

	// try using cache before sending relay
	var cacheError error
	if rpccs.cache.CacheActive() { // use cache only if its defined.
		if !chainMessage.GetForceCacheRefresh() { // don't use cache if user specified
			if reqBlock != spectypes.NOT_APPLICABLE { // don't use cache if requested block is not applicable
				var cacheReply *pairingtypes.CacheRelayReply
				hashKey, outputFormatter, err := chainlib.HashCacheRequest(relayRequestData, chainId)
				if err != nil {
					utils.LavaFormatError("sendRelayToProvider Failed getting Hash for cache request", err)
				} else {
					cacheCtx, cancel := context.WithTimeout(ctx, common.CacheTimeout)
					cacheReply, cacheError = rpccs.cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
						RequestHash:    hashKey,
						RequestedBlock: relayRequestData.RequestBlock,
						ChainId:        chainId,
						BlockHash:      nil,
						Finalized:      false,
						SharedStateId:  sharedStateId,
						SeenBlock:      relayRequestData.SeenBlock,
					}) // caching in the portal doesn't care about hashes, and we don't have data on finalization yet
					cancel()
					reply := cacheReply.GetReply()

					// read seen block from cache even if we had a miss we still want to get the seen block so we can use it to get the right provider.
					cacheSeenBlock := cacheReply.GetSeenBlock()
					// check if the cache seen block is greater than my local seen block, this means the user requested this
					// request spoke with another consumer instance and use that block for inter consumer consistency.
					if rpccs.sharedState && cacheSeenBlock > relayRequestData.SeenBlock {
						utils.LavaFormatDebug("shared state seen block is newer", utils.LogAttr("cache_seen_block", cacheSeenBlock), utils.LogAttr("local_seen_block", relayRequestData.SeenBlock))
						relayRequestData.SeenBlock = cacheSeenBlock
						// setting the fetched seen block from the cache server to our local cache as well.
						rpccs.consumerConsistency.SetSeenBlock(cacheSeenBlock, dappID, consumerIp)
					}

					// handle cache reply
					if cacheError == nil && reply != nil {
						// Info was fetched from cache, so we don't need to change the state
						// so we can return here, no need to update anything and calculate as this info was fetched from the cache
						reply.Data = outputFormatter(reply.Data)
						relayResult := common.RelayResult{
							Reply: reply,
							Request: &pairingtypes.RelayRequest{
								RelayData: relayRequestData,
							},
							Finalized:    false, // set false to skip data reliability
							StatusCode:   200,
							ProviderInfo: common.ProviderInfo{ProviderAddress: ""},
						}
						relayProcessor.SetResponse(&relayResponse{
							relayResult: relayResult,
							err:         nil,
						})
						return nil
					}
					// cache failed, move on to regular relay
					if performance.NotConnectedError.Is(cacheError) {
						utils.LavaFormatDebug("cache not connected", utils.LogAttr("error", cacheError))
					}
				}
			} else {
				utils.LavaFormatDebug("skipping cache due to requested block being NOT_APPLICABLE", utils.Attribute{Key: "api name", Value: chainMessage.GetApi().Name})
			}
		}
	}

	if reqBlock == spectypes.LATEST_BLOCK && relayRequestData.SeenBlock != 0 {
		// make optimizer select a provider that is likely to have the latest seen block
		reqBlock = relayRequestData.SeenBlock
	}
	// consumerEmergencyTracker always use latest virtual epoch
	virtualEpoch := rpccs.consumerTxSender.GetLatestVirtualEpoch()
	addon := chainlib.GetAddon(chainMessage)
	extensions := chainMessage.GetExtensions()
	usedProviders := relayProcessor.GetUsedProviders()
	sessions, err := rpccs.consumerSessionManager.GetSessions(ctx, chainlib.GetComputeUnits(chainMessage), usedProviders, reqBlock, addon, extensions, chainlib.GetStateful(chainMessage), virtualEpoch)
	if err != nil {
		if lavasession.PairingListEmptyError.Is(err) {
			if addon != "" {
				return utils.LavaFormatError("No Providers For Addon", err, utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("userIp", consumerIp))
			} else if len(extensions) > 0 && relayProcessor.GetAllowSessionDegradation() { // if we have no providers for that extension, use a regular provider, otherwise return the extension results
				sessions, err = rpccs.consumerSessionManager.GetSessions(ctx, chainlib.GetComputeUnits(chainMessage), usedProviders, reqBlock, addon, []*spectypes.Extension{}, chainlib.GetStateful(chainMessage), virtualEpoch)
				if err != nil {
					return err
				}
				relayProcessor.setSkipDataReliability(true) // disabling data reliability when disabling extensions.
				relayRequestData.Extensions = []string{}    // reset request data extensions
				extensions = []*spectypes.Extension{}       // reset extensions too so we wont hit SetDisallowDegradation
			} else {
				return err
			}
		} else {
			return err
		}
	}

	// making sure next get sessions wont use regular providers
	if len(extensions) > 0 {
		relayProcessor.SetDisallowDegradation()
	}

	if rpccs.debugRelays {
		utils.LavaFormatDebug("[Before Send] returned the following sessions",
			utils.LogAttr("sessions", sessions),
			utils.LogAttr("usedProviders.GetUnwantedProvidersToSend", usedProviders.GetUnwantedProvidersToSend()),
			utils.LogAttr("usedProviders.GetErroredProviders", usedProviders.GetErroredProviders()),
			utils.LogAttr("addons", addon),
			utils.LogAttr("extensions", extensions),
			utils.LogAttr("AllowSessionDegradation", relayProcessor.GetAllowSessionDegradation()),
		)
	}

	// Iterate over the sessions map
	for providerPublicAddress, sessionInfo := range sessions {
		// Launch a separate goroutine for each session
		go func(providerPublicAddress string, sessionInfo *lavasession.SessionInfo) {
			// add ticker launch metrics
			localRelayResult := &common.RelayResult{
				ProviderInfo: common.ProviderInfo{ProviderAddress: providerPublicAddress, ProviderStake: sessionInfo.StakeSize, ProviderQoSExcellenceSummery: sessionInfo.QoSSummeryResult},
				Finalized:    false,
				// setting the single consumer session as the conflict handler.
				//  to be able to validate if we need to report this provider or not.
				// holding the pointer is ok because the session is locked atm,
				// and later after its unlocked we only atomic read / write to it.
				ConflictHandler: sessionInfo.Session.Parent,
			}
			var errResponse error
			goroutineCtx, goroutineCtxCancel := context.WithCancel(context.Background())
			guid, found := utils.GetUniqueIdentifier(ctx)
			if found {
				goroutineCtx = utils.WithUniqueIdentifier(goroutineCtx, guid)
			}

			defer func() {
				// Return response
				relayProcessor.SetResponse(&relayResponse{
					relayResult: *localRelayResult,
					err:         errResponse,
				})

				// Close context
				goroutineCtxCancel()
			}()

			localRelayRequestData := *relayRequestData

			// Extract fields from the sessionInfo
			singleConsumerSession := sessionInfo.Session
			epoch := sessionInfo.Epoch
			reportedProviders := sessionInfo.ReportedProviders

			relayRequest, errResponse := lavaprotocol.ConstructRelayRequest(goroutineCtx, privKey, lavaChainID, chainId, &localRelayRequestData, providerPublicAddress, singleConsumerSession, int64(epoch), reportedProviders)
			if errResponse != nil {
				utils.LavaFormatError("Failed ConstructRelayRequest", errResponse, utils.LogAttr("Request data", localRelayRequestData))
				return
			}
			localRelayResult.Request = relayRequest
			endpointClient := *singleConsumerSession.EndpointConnection.Client

			// set relay sent metric
			go rpccs.rpcConsumerLogs.SetRelaySentToProviderMetric(chainId, apiInterface)

			if isSubscription {
				errResponse = rpccs.relaySubscriptionInner(goroutineCtx, endpointClient, singleConsumerSession, localRelayResult)
				if errResponse != nil {
					utils.LavaFormatError("Failed relaySubscriptionInner", errResponse, utils.LogAttr("Request data", localRelayRequestData))
					return
				}
			}

			// unique per dappId and ip
			consumerToken := common.GetUniqueToken(dappID, consumerIp)
			processingTimeout, expectedRelayTimeoutForQOS := rpccs.getProcessingTimeout(chainMessage)
			deadline, ok := ctx.Deadline()
			if ok { // we have ctx deadline. we cant go past it.
				processingTimeout = time.Until(deadline)
				if processingTimeout <= 0 {
					// no need to send we are out of time
					utils.LavaFormatWarning("Creating context deadline for relay attempt ran out of time, processingTimeout <= 0 ", nil, utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("ApiUrl", localRelayRequestData.ApiUrl))
					return
				}
				// to prevent absurdly short context timeout set the shortest timeout to be the expected latency for qos time.
				if processingTimeout < expectedRelayTimeoutForQOS {
					processingTimeout = expectedRelayTimeoutForQOS
				}
			}
			// send relay
			relayLatency, errResponse, backoff := rpccs.relayInner(goroutineCtx, singleConsumerSession, localRelayResult, processingTimeout, chainMessage, consumerToken, analytics)
			if errResponse != nil {
				failRelaySession := func(origErr error, backoff_ bool) {
					backOffDuration := 0 * time.Second
					if backoff_ {
						backOffDuration = lavasession.BACKOFF_TIME_ON_FAILURE
					}
					time.Sleep(backOffDuration) // sleep before releasing this singleConsumerSession
					// relay failed need to fail the session advancement
					errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, origErr)
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
			if expectedBH-latestBlock > 1000 {
				utils.LavaFormatWarning("identified block gap", nil,
					utils.Attribute{Key: "expectedBH", Value: expectedBH},
					utils.Attribute{Key: "latestServicedBlock", Value: latestBlock},
					utils.Attribute{Key: "session_id", Value: singleConsumerSession.SessionId},
					utils.Attribute{Key: "provider_address", Value: singleConsumerSession.Parent.PublicLavaAddress},
					utils.Attribute{Key: "providersCount", Value: pairingAddressesLen},
					utils.Attribute{Key: "finalizationConsensus", Value: rpccs.finalizationConsensus.String()},
				)
			}
			if rpccs.debugRelays && singleConsumerSession.QoSInfo.LastQoSReport != nil &&
				singleConsumerSession.QoSInfo.LastQoSReport.Sync.BigInt() != nil &&
				singleConsumerSession.QoSInfo.LastQoSReport.Sync.LT(sdk.MustNewDecFromStr("0.9")) {
				utils.LavaFormatDebug("identified QoS mismatch",
					utils.Attribute{Key: "expectedBH", Value: expectedBH},
					utils.Attribute{Key: "latestServicedBlock", Value: latestBlock},
					utils.Attribute{Key: "session_id", Value: singleConsumerSession.SessionId},
					utils.Attribute{Key: "provider_address", Value: singleConsumerSession.Parent.PublicLavaAddress},
					utils.Attribute{Key: "providersCount", Value: pairingAddressesLen},
					utils.Attribute{Key: "singleConsumerSession.QoSInfo", Value: singleConsumerSession.QoSInfo},
					utils.Attribute{Key: "finalizationConsensus", Value: rpccs.finalizationConsensus.String()},
				)
			}

			errResponse = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, latestBlock, chainlib.GetComputeUnits(chainMessage), relayLatency, singleConsumerSession.CalculateExpectedLatency(expectedRelayTimeoutForQOS), expectedBH, numOfProviders, pairingAddressesLen, chainMessage.GetApi().Category.HangingApi) // session done successfully

			if rpccs.cache.CacheActive() && rpcclient.ValidateStatusCodes(localRelayResult.StatusCode, true) == nil {
				isNodeError, _ := chainMessage.CheckResponseError(localRelayResult.Reply.Data, localRelayResult.StatusCode)
				// in case the error is a node error we don't want to cache
				if !isNodeError {
					// copy reply data so if it changes it doesn't panic mid async send
					copyReply := &pairingtypes.RelayReply{}
					copyReplyErr := protocopy.DeepCopyProtoObject(localRelayResult.Reply, copyReply)
					// set cache in a non blocking call
					requestedBlock := localRelayResult.Request.RelayData.RequestBlock                             // get requested block before removing it from the data
					seenBlock := localRelayResult.Request.RelayData.SeenBlock                                     // get seen block before removing it from the data
					hashKey, _, hashErr := chainlib.HashCacheRequest(localRelayResult.Request.RelayData, chainId) // get the hash (this changes the data)

					go func() {
						// deal with copying error.
						if copyReplyErr != nil || hashErr != nil {
							utils.LavaFormatError("Failed copying relay private data sendRelayToProvider", nil,
								utils.LogAttr("copyReplyErr", copyReplyErr),
								utils.LogAttr("hashErr", hashErr),
							)
							return
						}
						chainMessageRequestedBlock, _ := chainMessage.RequestedBlock()
						if chainMessageRequestedBlock == spectypes.NOT_APPLICABLE {
							return
						}

						new_ctx := context.Background()
						new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
						defer cancel()
						_, averageBlockTime, _, _ := rpccs.chainParser.ChainBlockStats()
						err2 := rpccs.cache.SetEntry(new_ctx, &pairingtypes.RelayCacheSet{
							RequestHash:      hashKey,
							ChainId:          chainId,
							RequestedBlock:   requestedBlock,
							SeenBlock:        seenBlock,
							BlockHash:        nil, // consumer cache doesn't care about block hashes
							Response:         copyReply,
							Finalized:        localRelayResult.Finalized,
							OptionalMetadata: nil,
							SharedStateId:    sharedStateId,
							AverageBlockTime: int64(averageBlockTime), // by using average block time we can set longer TTL
							IsNodeError:      isNodeError,
						})
						if err2 != nil {
							utils.LavaFormatWarning("error updating cache with new entry", err2)
						}
					}()
				}
			}
			// localRelayResult is being sent on the relayProcessor by a deferred function
		}(providerPublicAddress, sessionInfo)
	}
	// finished setting up go routines, can return and wait for responses
	return nil
}

func (rpccs *RPCConsumerServer) relayInner(ctx context.Context, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult, relayTimeout time.Duration, chainMessage chainlib.ChainMessage, consumerToken string, analytics *metrics.RelayMetrics) (relayLatency time.Duration, err error, needsBackoff bool) {
	existingSessionLatestBlock := singleConsumerSession.LatestBlock // we read it now because singleConsumerSession is locked, and later it's not
	endpointClient := *singleConsumerSession.EndpointConnection.Client
	providerPublicAddress := relayResult.ProviderInfo.ProviderAddress
	relayRequest := relayResult.Request
	if rpccs.debugRelays {
		utils.LavaFormatDebug("Sending relay", utils.LogAttr("timeout", relayTimeout), utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock), utils.LogAttr("GUID", ctx), utils.LogAttr("provider", relayRequest.RelaySession.Provider))
	}
	callRelay := func() (reply *pairingtypes.RelayReply, relayLatency time.Duration, err error, backoff bool) {
		relaySentTime := time.Now()
		connectCtx, connectCtxCancel := context.WithTimeout(ctx, relayTimeout)
		metadataAdd := metadata.New(map[string]string{common.IP_FORWARDING_HEADER_NAME: consumerToken})
		connectCtx = metadata.NewOutgoingContext(connectCtx, metadataAdd)
		defer connectCtxCancel()

		// add consumer processing timestamp before provider metric and start measuring time after the provider replied
		rpccs.rpcConsumerLogs.AddMetricForProcessingLatencyBeforeProvider(analytics, rpccs.listenEndpoint.ChainID, rpccs.listenEndpoint.ApiInterface)

		reply, err = endpointClient.Relay(connectCtx, relayRequest, grpc.Trailer(&relayResult.ProviderTrailer))
		statuses := relayResult.ProviderTrailer.Get(common.StatusCodeMetadataKey)
		if len(statuses) > 0 {
			codeNum, errStatus := strconv.Atoi(statuses[0])
			if errStatus != nil {
				utils.LavaFormatWarning("failed converting status code", errStatus)
			}
			relayResult.StatusCode = codeNum
		}
		relayLatency = time.Since(relaySentTime)
		if rpccs.debugRelays {
			providerNodeHashes := relayResult.ProviderTrailer.Get(chainlib.RPCProviderNodeAddressHash)
			attributes := []utils.Attribute{
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("addon", relayRequest.RelayData.Addon),
				utils.LogAttr("extensions", relayRequest.RelayData.Extensions),
				utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock),
				utils.LogAttr("seenBlock", relayRequest.RelayData.SeenBlock),
				utils.LogAttr("provider", relayRequest.RelaySession.Provider),
				utils.LogAttr("cuSum", relayRequest.RelaySession.CuSum),
				utils.LogAttr("QosReport", relayRequest.RelaySession.QosReport),
				utils.LogAttr("QosReportExcellence", relayRequest.RelaySession.QosExcellenceReport),
				utils.LogAttr("relayNum", relayRequest.RelaySession.RelayNum),
				utils.LogAttr("sessionId", relayRequest.RelaySession.SessionId),
				utils.LogAttr("latency", relayLatency),
				utils.LogAttr("replyErred", err != nil),
				utils.LogAttr("replyLatestBlock", reply.GetLatestBlock()),
				utils.LogAttr("method", chainMessage.GetApi().Name),
				utils.LogAttr("providerNodeHashes", providerNodeHashes),
			}
			internalPath := chainMessage.GetApiCollection().CollectionData.InternalPath
			if internalPath != "" {
				attributes = append(attributes, utils.LogAttr("internal_path", internalPath),
					utils.LogAttr("apiUrl", relayRequest.RelayData.ApiUrl))
			}
			utils.LavaFormatDebug("sending relay to provider", attributes...)
		}
		if err != nil {
			backoff := false
			if errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
				backoff = true
			}
			return reply, 0, err, backoff
		}
		analytics.SetProcessingTimestampAfterRelay(time.Now())

		return reply, relayLatency, nil, false
	}
	reply, relayLatency, err, backoff := callRelay()
	if err != nil {
		// adding some error information for future debug
		if relayRequest.RelayData.RequestBlock == 0 {
			reqBlock, _ := chainMessage.RequestedBlock()
			utils.LavaFormatWarning("Got Error, with requested block 0", err,
				utils.LogAttr("relayRequest.RelayData.RequestBlock", relayRequest.RelayData.RequestBlock),
				utils.LogAttr("chainMessage.RequestedBlock", reqBlock),
				utils.LogAttr("existingSessionLatestBlock", existingSessionLatestBlock),
				utils.LogAttr("SeenBlock", relayRequest.RelayData.SeenBlock),
				utils.LogAttr("msg_api", relayRequest.RelayData.ApiUrl),
				utils.LogAttr("msg_data", string(relayRequest.RelayData.Data)),
			)
		}
		return 0, err, backoff
	}
	relayResult.Reply = reply
	lavaprotocol.UpdateRequestedBlock(relayRequest.RelayData, reply) // update relay request requestedBlock to the provided one in case it was arbitrary
	_, _, blockDistanceForFinalizedData, _ := rpccs.chainParser.ChainBlockStats()
	finalized := spectypes.IsFinalizedBlock(relayRequest.RelayData.RequestBlock, reply.LatestBlock, blockDistanceForFinalizedData)
	filteredHeaders, _, ignoredHeaders := rpccs.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	reply.Metadata = filteredHeaders
	err = lavaprotocol.VerifyRelayReply(ctx, reply, relayRequest, providerPublicAddress)
	if err != nil {
		return 0, err, false
	}
	reply.Metadata = append(reply.Metadata, ignoredHeaders...)
	// TODO: response data sanity, check its under an expected format add that format to spec
	enabled, _ := rpccs.chainParser.DataReliabilityParams()
	if enabled {
		// TODO: DETECTION instead of existingSessionLatestBlock, we need proof of last reply to send the previous reply and the current reply
		finalizedBlocks, finalizationConflict, err := lavaprotocol.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, rpccs.ConsumerAddress, existingSessionLatestBlock, blockDistanceForFinalizedData)
		if err != nil {
			if sdkerrors.IsOf(err, lavaprotocol.ProviderFinzalizationDataAccountabilityError) && finalizationConflict != nil {
				go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Parent)
			}
			return 0, err, false
		}

		finalizationConflict, err = rpccs.finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), providerPublicAddress, finalizedBlocks, relayRequest.RelaySession, reply)
		if err != nil {
			go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Parent)
			return 0, err, false
		}
	}
	relayResult.Finalized = finalized
	return relayLatency, nil, false
}

func (rpccs *RPCConsumerServer) relaySubscriptionInner(ctx context.Context, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult) (err error) {
	// relaySentTime := time.Now()
	replyServer, err := endpointClient.RelaySubscribe(ctx, relayResult.Request)
	// relayLatency := time.Since(relaySentTime) // TODO: use subscription QoS
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "original error", Value: err.Error()})
		}
		return err
	}
	// TODO: need to check that if provider fails and returns error, this is reflected here and we run onSessionDone
	// my thoughts are that this fails if the grpc fails not if the provider fails, and if the provider returns an error this is reflected by the Recv function on the chainListener calling us here
	// and this is too late
	relayResult.ReplyServer = &replyServer
	err = rpccs.consumerSessionManager.OnSessionDoneIncreaseCUOnly(singleConsumerSession)
	return err
}

func (rpccs *RPCConsumerServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, dappID string, consumerIp string, chainMessage chainlib.ChainMessage, dataReliabilityThreshold uint32, relayProcessor *RelayProcessor) error {
	processingTimeout, expectedRelayTimeout := rpccs.getProcessingTimeout(chainMessage)
	// Wait another relayTimeout duration to maybe get additional relay results
	if relayProcessor.usedProviders.CurrentlyUsed() > 0 {
		time.Sleep(expectedRelayTimeout)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	specCategory := chainMessage.GetApi().Category
	if !specCategory.Deterministic {
		return nil // disabled for this spec and requested block so no data reliability messages
	}

	if rand.Uint32() > dataReliabilityThreshold {
		// decided not to do data reliability
		return nil
	}
	// only need to send another relay if we don't have enough replies
	results := []common.RelayResult{}
	for _, result := range relayProcessor.NodeResults() {
		if result.Finalized {
			results = append(results, result)
		}
	}
	if len(results) == 0 {
		// nothing to check
		return nil
	}

	reqBlock, _ := chainMessage.RequestedBlock()
	if reqBlock <= spectypes.NOT_APPLICABLE {
		if reqBlock <= spectypes.LATEST_BLOCK {
			return utils.LavaFormatError("sendDataReliabilityRelayIfApplicable latest requestBlock", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "RequestBlock", Value: reqBlock})
		}
		// does not support sending data reliability requests on a block that is not specific
		return nil
	}
	relayResult := results[0]
	if len(results) < 2 {
		relayRequestData := lavaprotocol.NewRelayData(ctx, relayResult.Request.RelayData.ConnectionType, relayResult.Request.RelayData.ApiUrl, relayResult.Request.RelayData.Data, relayResult.Request.RelayData.SeenBlock, reqBlock, relayResult.Request.RelayData.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), relayResult.Request.RelayData.Addon, relayResult.Request.RelayData.Extensions)
		relayProcessorDataReliability := NewRelayProcessor(ctx, relayProcessor.usedProviders, 1, chainMessage, rpccs.consumerConsistency, dappID, consumerIp, rpccs.debugRelays, rpccs.rpcConsumerLogs, rpccs)
		err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, consumerIp, relayProcessorDataReliability, nil)
		if err != nil {
			return utils.LavaFormatWarning("failed data reliability relay to provider", err, utils.LogAttr("relayProcessorDataReliability", relayProcessorDataReliability))
		}

		processingCtx, cancel := context.WithTimeout(ctx, processingTimeout)
		defer cancel()
		err = relayProcessorDataReliability.WaitForResults(processingCtx)
		if err != nil {
			return utils.LavaFormatWarning("failed sending data reliability relays", err, utils.Attribute{Key: "relayProcessorDataReliability", Value: relayProcessorDataReliability})
		}
		relayResultsDataReliability := relayProcessorDataReliability.NodeResults()
		resultsDataReliability := []common.RelayResult{}
		for _, result := range relayResultsDataReliability {
			if result.Finalized {
				resultsDataReliability = append(resultsDataReliability, result)
			}
		}
		if len(resultsDataReliability) == 0 {
			utils.LavaFormatDebug("skipping data reliability check since responses from second batch was not finalized", utils.Attribute{Key: "results", Value: relayResultsDataReliability})
			return nil
		}
		results = append(results, resultsDataReliability...)
	}
	for i := 0; i < len(results)-1; i++ {
		relayResult := results[i]
		relayResultDataReliability := results[i+1]
		conflict := lavaprotocol.VerifyReliabilityResults(ctx, &relayResult, &relayResultDataReliability, chainMessage.GetApiCollection(), rpccs.chainParser)
		if conflict != nil {
			// TODO: remove this check when we fix the missing extensions information on conflict detection transaction
			if len(chainMessage.GetExtensions()) == 0 {
				err := rpccs.consumerTxSender.TxConflictDetection(ctx, nil, conflict, nil, relayResultDataReliability.ConflictHandler)
				if err != nil {
					utils.LavaFormatError("could not send detection Transaction", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "conflict", Value: conflict})
				}
				if rpccs.reporter != nil {
					utils.LavaFormatDebug("sending conflict report to BE", utils.LogAttr("conflicting api", chainMessage.GetApi().Name))
					rpccs.reporter.AppendConflict(metrics.NewConflictRequest(relayResult.Request, relayResult.Reply, relayResultDataReliability.Request, relayResultDataReliability.Reply))
				}
			}
		} else {
			utils.LavaFormatDebug("[+] verified relay successfully with data reliability", utils.LogAttr("api", chainMessage.GetApi().Name))
		}
	}
	return nil
}

func (rpccs *RPCConsumerServer) getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration) {
	_, averageBlockTime, _, _ := rpccs.chainParser.ChainBlockStats()
	relayTimeout = chainlib.GetRelayTimeout(chainMessage, averageBlockTime)
	processingTimeout = common.GetTimeoutForProcessing(relayTimeout, chainlib.GetTimeoutInfo(chainMessage))
	return processingTimeout, relayTimeout
}

func (rpccs *RPCConsumerServer) LavaDirectiveHeaders(metadata []pairingtypes.Metadata) ([]pairingtypes.Metadata, map[string]string) {
	metadataRet := []pairingtypes.Metadata{}
	headerDirectives := map[string]string{}
	for _, metaElement := range metadata {
		name := strings.ToLower(metaElement.Name)
		if _, found := common.SPECIAL_LAVA_DIRECTIVE_HEADERS[name]; found {
			headerDirectives[name] = metaElement.Value
		} else {
			metadataRet = append(metadataRet, metaElement)
		}
	}
	return metadataRet, headerDirectives
}

func (rpccs *RPCConsumerServer) getExtensionsFromDirectiveHeaders(directiveHeaders map[string]string) extensionslib.ExtensionInfo {
	extensionsStr, ok := directiveHeaders[common.EXTENSION_OVERRIDE_HEADER_NAME]
	if ok {
		extensions := strings.Split(extensionsStr, ",")
		_, extensions, _ = rpccs.chainParser.SeparateAddonsExtensions(extensions)
		if len(extensions) == 1 && extensions[0] == "none" {
			// none eliminates existing extensions
			return extensionslib.ExtensionInfo{LatestBlock: rpccs.getLatestBlock(), ExtensionOverride: []string{}}
		} else if len(extensions) > 0 {
			return extensionslib.ExtensionInfo{LatestBlock: rpccs.getLatestBlock(), AdditionalExtensions: extensions}
		}
	}
	return extensionslib.ExtensionInfo{LatestBlock: rpccs.getLatestBlock()}
}

func (rpccs *RPCConsumerServer) HandleDirectiveHeadersForMessage(chainMessage chainlib.ChainMessage, directiveHeaders map[string]string) {
	timeoutStr, ok := directiveHeaders[common.RELAY_TIMEOUT_HEADER_NAME]
	if ok {
		timeout, err := time.ParseDuration(timeoutStr)
		if err == nil {
			// set an override timeout
			utils.LavaFormatDebug("User indicated to set the timeout using flag", utils.LogAttr("timeout", timeoutStr))
			chainMessage.TimeoutOverride(timeout)
		}
	}

	_, ok = directiveHeaders[common.FORCE_CACHE_REFRESH_HEADER_NAME]
	chainMessage.SetForceCacheRefresh(ok)
}

func (rpccs *RPCConsumerServer) appendHeadersToRelayResult(ctx context.Context, relayResult *common.RelayResult, protocolErrors uint64, relayProcessor *RelayProcessor, directiveHeaders map[string]string) {
	if relayResult == nil {
		return
	}
	metadataReply := []pairingtypes.Metadata{}
	// add the provider that responded

	providerAddress := relayResult.GetProvider()
	if providerAddress == "" {
		providerAddress = "Cached"
	}
	metadataReply = append(metadataReply,
		pairingtypes.Metadata{
			Name:  common.PROVIDER_ADDRESS_HEADER_NAME,
			Value: providerAddress,
		})

	// add the relay retried count
	if protocolErrors > 0 {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.RETRY_COUNT_HEADER_NAME,
				Value: strconv.FormatUint(protocolErrors, 10),
			})
	}
	if relayResult.Reply == nil {
		relayResult.Reply = &pairingtypes.RelayReply{}
	}
	if relayResult.Reply.LatestBlock > 0 {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.PROVIDER_LATEST_BLOCK_HEADER_NAME,
				Value: strconv.FormatInt(relayResult.Reply.LatestBlock, 10),
			})
	}
	guid, found := utils.GetUniqueIdentifier(ctx)
	if found && guid != 0 {
		guidStr := strconv.FormatUint(guid, 10)
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.GUID_HEADER_NAME,
				Value: guidStr,
			})
	}

	// fetch trailer information from the provider by using the provider trailer field.
	providerNodeExtensions := relayResult.ProviderTrailer.Get(chainlib.RPCProviderNodeExtension)
	if len(providerNodeExtensions) > 0 {
		extensionMD := pairingtypes.Metadata{
			Name:  chainlib.RPCProviderNodeExtension,
			Value: providerNodeExtensions[0],
		}
		relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, extensionMD)
	}

	_, debugRelays := directiveHeaders[common.LAVA_DEBUG_RELAY]
	if debugRelays {
		erroredProviders := relayProcessor.GetUsedProviders().GetErroredProviders()
		if len(erroredProviders) > 0 {
			erroredProvidersArray := make([]string, len(erroredProviders))
			idx := 0
			for providerAddress := range erroredProviders {
				erroredProvidersArray[idx] = providerAddress
				idx++
			}
			erroredProvidersString := fmt.Sprintf("%v", erroredProvidersArray)
			erroredProvidersMD := pairingtypes.Metadata{
				Name:  common.ERRORED_PROVIDERS_HEADER_NAME,
				Value: erroredProvidersString,
			}
			relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, erroredProvidersMD)
		}

		currentReportedProviders := rpccs.consumerSessionManager.GetReportedProviders(uint64(relayResult.Request.RelaySession.Epoch))
		if len(currentReportedProviders) > 0 {
			reportedProvidersArray := make([]string, len(currentReportedProviders))
			for idx, providerAddress := range currentReportedProviders {
				reportedProvidersArray[idx] = providerAddress.Address
			}
			reportedProvidersString := fmt.Sprintf("%v", reportedProvidersArray)
			reportedProvidersMD := pairingtypes.Metadata{
				Name:  common.REPORTED_PROVIDERS_HEADER_NAME,
				Value: reportedProvidersString,
			}
			relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, reportedProvidersMD)
		}
	}

	relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, metadataReply...)
}

func (rpccs *RPCConsumerServer) IsHealthy() bool {
	return rpccs.relaysMonitor.IsHealthy()
}
