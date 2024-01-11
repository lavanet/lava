package rpcconsumer

import (
	"context"
	"errors"
	"strconv"
	"strings"
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
	"github.com/lavanet/lava/utils/protocopy"
	"github.com/lavanet/lava/utils/rand"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	MaxRelayRetries = 6
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
	consumerConsistency    *ConsumerConsistency
	relaysMonitor          *metrics.RelaysMonitor
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
	rpccs.consumerConsistency = consumerConsistency
	initialRelays := true
	rpccs.relaysMonitor = relaysMonitor
	if RelaysHealthEnable {
		rpccs.relaysMonitor.SetRelaySender(func() (bool, error) {
			if initialRelays {
				// Only start after everything is initialized - check consumer session manager
				ok, err := rpccs.waitForPairing()
				if !ok {
					return false, err
				}
			}

			success, err := rpccs.sendCraftedRelays(MaxRelayRetries, initialRelays)
			initialRelays = false
			return success, err
		})
	}

	chainListener, err := chainlib.NewChainListener(ctx, listenEndpoint, rpccs, rpccs, rpcConsumerLogs, chainParser)
	if err != nil {
		return err
	}

	go chainListener.Serve(ctx, cmdFlags)

	// we trigger a latest block call to get some more information on our providers, using the relays monitor
	if RelaysHealthEnable {
		rpccs.relaysMonitor.Start(ctx)
	}
	return nil
}

func (rpccs *RPCConsumerServer) waitForPairing() (bool, error) {
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

	select {
	case <-reinitializedChan:
		break
	case <-time.After(30 * time.Second):
		return false, utils.LavaFormatError("failed initial relays, csm was not initialized after timeout", nil,
			utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface),
		)
	}

	return true, nil
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
	chainMessage, err = rpccs.chainParser.ParseMsg(path, data, collectionData.Type, nil, 0)
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
	unwantedProviders := map[string]struct{}{}
	timeouts := 0
	success := false
	var err error

	for i := 0; i < retries; i++ {
		var relayResult *common.RelayResult
		relayResult, err = rpccs.sendRelayToProvider(ctx, chainMessage, relay, "-init-", "", &unwantedProviders, timeouts)
		if err != nil {
			utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "unwantedProviders", Value: unwantedProviders}}...)
			if relayResult != nil && relayResult.ProviderInfo.ProviderAddress != "" {
				unwantedProviders[relayResult.ProviderInfo.ProviderAddress] = struct{}{}
			}
			if common.IsTimeout(err) {
				timeouts++
			}
		} else {
			unwantedProviders = map[string]struct{}{}
			utils.LavaFormatInfo("[+] init relay succeeded", []utils.Attribute{{Key: "chainID", Value: rpccs.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpccs.listenEndpoint.ApiInterface}, {Key: "latestBlock", Value: relayResult.Reply.LatestBlock}, {Key: "provider address", Value: relayResult.ProviderInfo.ProviderAddress}}...)

			if RelaysHealthEnable {
				rpccs.relaysMonitor.LogRelay()
			}
			success = true

			// If this is the first time we send relays, we want to send all of them, instead of break on first successful relay
			// That way, we populate the providers with the latest blocks with successful relays
			if !initialRelays {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
	}

	return success, err
}

// sending a few latest blocks relays to providers in order to have some data on the providers when relays start arriving
func (rpccs *RPCConsumerServer) sendCraftedRelays(retries int, initialRelays bool) (success bool, err error) {
	ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
	ok, relay, chainMessage, err := rpccs.craftRelay(ctx)
	if !ok {
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
	chainMessage, err := rpccs.chainParser.ParseMsg(url, []byte(req), connectionType, metadata, rpccs.getLatestBlock())
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
	relayResults := []*common.RelayResult{}
	relayErrors := &RelayErrors{}
	blockOnSyncLoss := map[string]struct{}{}
	modifiedOnLatestReq := false
	errorRelayResult := &common.RelayResult{} // returned on error
	retries := uint64(0)
	timeouts := 0
	unwantedProviders := rpccs.GetInitialUnwantedProviders(directiveHeaders)
	for ; retries < MaxRelayRetries; retries++ {
		// TODO: make this async between different providers
		relayResult, err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, consumerIp, &unwantedProviders, timeouts)
		if relayResult.ProviderInfo.ProviderAddress != "" {
			if err != nil {
				// add this provider to the erroring providers
				if errorRelayResult.ProviderInfo.ProviderAddress != "" {
					errorRelayResult.ProviderInfo.ProviderAddress += ","
				}
				errorRelayResult.ProviderInfo.ProviderAddress += relayResult.ProviderInfo.ProviderAddress
				_, ok := blockOnSyncLoss[relayResult.ProviderInfo.ProviderAddress]
				if !ok && lavasession.IsSessionSyncLoss(err) {
					// allow this provider to be wantedProvider on a retry, if it didn't fail once on syncLoss
					blockOnSyncLoss[relayResult.ProviderInfo.ProviderAddress] = struct{}{}
					utils.LavaFormatWarning("Identified SyncLoss in provider, not removing it from list for another attempt", err, utils.Attribute{Key: "address", Value: relayResult.ProviderInfo.ProviderAddress})
				} else {
					unwantedProviders[relayResult.ProviderInfo.ProviderAddress] = struct{}{}
				}
				if common.IsTimeout(err) {
					timeouts++
				}
			}
		}
		if err != nil {
			if relayResult.GetStatusCode() != 0 {
				// keep the error status code
				errorRelayResult.StatusCode = relayResult.GetStatusCode()
			}
			relayErrors.relayErrors = append(relayErrors.relayErrors, RelayError{err: err, ProviderInfo: relayResult.ProviderInfo})
			if lavasession.PairingListEmptyError.Is(err) {
				// if we ran out of pairings because unwantedProviders is too long or validProviders is too short, continue to reply handling code
				break
			}
			// decide if we should break here if its something retry won't solve
			utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err.Error()}, utils.Attribute{Key: "endpoint", Value: rpccs.listenEndpoint})
			continue
		}
		relayResults = append(relayResults, relayResult)
		unwantedProviders[relayResult.ProviderInfo.ProviderAddress] = struct{}{}
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
			go rpccs.sendDataReliabilityRelayIfApplicable(dataReliabilityContext, dappID, consumerIp, relayResult, chainMessage, dataReliabilityThreshold, unwantedProviders) // runs asynchronously
		}
	}

	if len(relayResults) == 0 {
		rpccs.appendHeadersToRelayResult(ctx, errorRelayResult, retries)
		// suggest the user to add the timeout flag
		if uint64(timeouts) == retries && retries > 0 {
			utils.LavaFormatDebug("all relays timeout", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors.relayErrors})
			return errorRelayResult, utils.LavaFormatError("Failed all relay retries due to timeout consider adding 'lava-relay-timeout' header to extend the allowed timeout duration", nil, utils.Attribute{Key: "GUID", Value: ctx})
		}
		return errorRelayResult, utils.LavaFormatError("Failed all retries", nil, utils.Attribute{Key: "GUID", Value: ctx}, relayErrors.GetBestErrorMessageForUser())
	} else if len(relayErrors.relayErrors) > 0 {
		utils.LavaFormatDebug("relay succeeded but had some errors", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors})
	}
	var returnedResult *common.RelayResult
	for _, iteratedResult := range relayResults {
		// TODO: go over rpccs.requiredResponses and get majority
		returnedResult = iteratedResult
	}

	if analytics != nil {
		currentLatency := time.Since(relaySentTime)
		analytics.Latency = currentLatency.Milliseconds()
		analytics.ComputeUnits = chainMessage.GetApi().ComputeUnits
	}
	if retries > 0 {
		utils.LavaFormatDebug("relay succeeded after retries", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "retries", Value: retries})
	}
	rpccs.appendHeadersToRelayResult(ctx, returnedResult, retries)

	if RelaysHealthEnable {
		rpccs.relaysMonitor.LogRelay()
	}

	return returnedResult, nil
}

func (rpccs *RPCConsumerServer) sendRelayToProvider(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	relayRequestData *pairingtypes.RelayPrivateData,
	dappID string,
	consumerIp string,
	unwantedProviders *map[string]struct{},
	timeouts int,
) (relayResult *common.RelayResult, errRet error) {
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
	if isSubscription {
		// temporarily disable subscriptions
		// TODO: fix subscription and disable this case.
		return &common.RelayResult{ProviderInfo: common.ProviderInfo{ProviderAddress: ""}}, utils.LavaFormatError("Subscriptions are disabled currently", nil)
	}

	privKey := rpccs.privKey
	chainID := rpccs.listenEndpoint.ChainID
	lavaChainID := rpccs.lavaChainID

	// Get Session. we get session here so we can use the epoch in the callbacks
	reqBlock, _ := chainMessage.RequestedBlock()
	if reqBlock == spectypes.LATEST_BLOCK && relayRequestData.SeenBlock != 0 {
		// make optimizer select a provider that is likely to have the latest seen block
		reqBlock = relayRequestData.SeenBlock
	}
	// consumerEmergencyTracker always use latest virtual epoch
	virtualEpoch := rpccs.consumerTxSender.GetLatestVirtualEpoch()
	addon := chainlib.GetAddon(chainMessage)
	extensions := chainMessage.GetExtensions()
	sessions, err := rpccs.consumerSessionManager.GetSessions(ctx, chainlib.GetComputeUnits(chainMessage), *unwantedProviders, reqBlock, addon, extensions, chainlib.GetStateful(chainMessage), virtualEpoch)
	if err != nil {
		if lavasession.PairingListEmptyError.Is(err) && (addon != "" || len(extensions) > 0) {
			// if we have no providers for a specific addon or extension, return an indicative error
			err = utils.LavaFormatError("No Providers For Addon Or Extension", err, utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions))
		}
		return &common.RelayResult{ProviderInfo: common.ProviderInfo{ProviderAddress: ""}}, err
	}

	type relayResponse struct {
		relayResult *common.RelayResult
		err         error
	}

	// Make a channel for all providers to send responses
	responses := make(chan *relayResponse, len(sessions))

	relayTimeout := chainlib.GetRelayTimeout(chainMessage, rpccs.chainParser, timeouts)
	// Iterate over the sessions map
	for providerPublicAddress, sessionInfo := range sessions {
		// Launch a separate goroutine for each session
		go func(providerPublicAddress string, sessionInfo *lavasession.SessionInfo) {
			var localRelayResult *common.RelayResult
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
			localRelayResult = &common.RelayResult{
				ProviderInfo: common.ProviderInfo{ProviderAddress: providerPublicAddress, ProviderStake: sessionInfo.StakeSize, ProviderQoSExcellenceSummery: sessionInfo.QoSSummeryResult},
				Finalized:    false,
				// setting the single consumer session as the conflict handler.
				//  to be able to validate if we need to report this provider or not.
				// holding the pointer is ok because the session is locked atm,
				// and later after its unlocked we only atomic read / write to it.
				ConflictHandler: sessionInfo.Session.Parent,
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
				utils.LavaFormatDebug("cache not connected", utils.LogAttr("error", errResponse))
			}

			// unique per dappId and ip
			consumerToken := common.GetUniqueToken(dappID, consumerIp)

			localRelayResult, relayLatency, errResponse, backoff := rpccs.relayInner(goroutineCtx, singleConsumerSession, localRelayResult, relayTimeout, chainMessage, consumerToken)
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
			if DebugRelaysFlag && singleConsumerSession.QoSInfo.LastQoSReport != nil &&
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
			errResponse = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, latestBlock, chainlib.GetComputeUnits(chainMessage), relayLatency, singleConsumerSession.CalculateExpectedLatency(relayTimeout), expectedBH, numOfProviders, pairingAddressesLen, chainMessage.GetApi().Category.HangingApi) // session done successfully

			if rpccs.cache.CacheActive() {
				// copy private data so if it changes it doesn't panic mid async send
				copyPrivateData := &pairingtypes.RelayPrivateData{}
				copyRequestErr := protocopy.DeepCopyProtoObject(localRelayResult.Request.RelayData, copyPrivateData)
				copyReply := &pairingtypes.RelayReply{}
				copyReplyErr := protocopy.DeepCopyProtoObject(localRelayResult.Reply, copyReply)
				// set cache in a non blocking call
				go func() {
					// deal with copying error.
					if copyRequestErr != nil || copyReplyErr != nil {
						utils.LavaFormatError("Failed copying relay private data sendRelayToProvider", nil, utils.LogAttr("copyReplyErr", copyReplyErr), utils.LogAttr("copyRequestErr", copyRequestErr))
						return
					}
					requestedBlock, _ := chainMessage.RequestedBlock()
					if requestedBlock == spectypes.NOT_APPLICABLE {
						return
					}
					new_ctx := context.Background()
					new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
					defer cancel()
					err2 := rpccs.cache.SetEntry(new_ctx, copyPrivateData, nil, chainID, copyReply, localRelayResult.Finalized, localRelayResult.Request.RelaySession.Provider, nil) // caching in the portal doesn't care about hashes
					if err2 != nil {
						utils.LavaFormatWarning("error updating cache with new entry", err2)
					}
				}()
			}
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

	if response.err == nil && response.relayResult != nil && response.relayResult.Reply != nil {
		// no error, update the seen block
		blockSeen := response.relayResult.Reply.LatestBlock
		rpccs.consumerConsistency.SetSeenBlock(blockSeen, dappID, consumerIp)
	}

	return response.relayResult, response.err
}

func (rpccs *RPCConsumerServer) relayInner(ctx context.Context, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult, relayTimeout time.Duration, chainMessage chainlib.ChainMessage, consumerToken string) (relayResultRet *common.RelayResult, relayLatency time.Duration, err error, needsBackoff bool) {
	existingSessionLatestBlock := singleConsumerSession.LatestBlock // we read it now because singleConsumerSession is locked, and later it's not
	endpointClient := *singleConsumerSession.Endpoint.Client
	providerPublicAddress := relayResult.ProviderInfo.ProviderAddress
	relayRequest := relayResult.Request
	callRelay := func() (reply *pairingtypes.RelayReply, relayLatency time.Duration, err error, backoff bool) {
		relaySentTime := time.Now()
		connectCtx, connectCtxCancel := context.WithTimeout(ctx, relayTimeout)
		metadataAdd := metadata.New(map[string]string{common.IP_FORWARDING_HEADER_NAME: consumerToken})
		connectCtx = metadata.NewOutgoingContext(connectCtx, metadataAdd)
		defer connectCtxCancel()
		var trailer metadata.MD
		reply, err = endpointClient.Relay(connectCtx, relayRequest, grpc.Trailer(&trailer))
		statuses := trailer.Get(common.StatusCodeMetadataKey)
		if len(statuses) > 0 {
			codeNum, errStatus := strconv.Atoi(statuses[0])
			if errStatus != nil {
				utils.LavaFormatWarning("failed converting status code", errStatus)
			}
			relayResult.StatusCode = codeNum
		}
		relayLatency = time.Since(relaySentTime)
		if DebugRelaysFlag {
			utils.LavaFormatDebug("sending relay to provider",
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
			)
		}
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
	err = lavaprotocol.VerifyRelayReply(ctx, reply, relayRequest, providerPublicAddress)
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
				go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Parent)
			}
			return relayResult, 0, err, false
		}

		finalizationConflict, err = rpccs.finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), providerPublicAddress, finalizedBlocks, relayRequest.RelaySession, reply)
		if err != nil {
			go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationConflict, nil, nil, singleConsumerSession.Parent)
			return relayResult, 0, err, false
		}
	}
	relayResult.Finalized = finalized
	return relayResult, relayLatency, nil, false
}

func (rpccs *RPCConsumerServer) relaySubscriptionInner(ctx context.Context, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult) (relayResultRet *common.RelayResult, err error) {
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

func (rpccs *RPCConsumerServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, dappID string, consumerIp string, relayResult *common.RelayResult, chainMessage chainlib.ChainMessage, dataReliabilityThreshold uint32, unwantedProviders map[string]struct{}) error {
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
	relayRequestData := lavaprotocol.NewRelayData(ctx, relayResult.Request.RelayData.ConnectionType, relayResult.Request.RelayData.ApiUrl, relayResult.Request.RelayData.Data, relayResult.Request.RelayData.SeenBlock, reqBlock, relayResult.Request.RelayData.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), relayResult.Request.RelayData.Addon, relayResult.Request.RelayData.Extensions)
	// TODO: give the same timeout the original provider got by setting the same retry
	relayResultDataReliability, err := rpccs.sendRelayToProvider(ctx, chainMessage, relayRequestData, dappID, consumerIp, &unwantedProviders, 0)
	if err != nil {
		errAttributes := []utils.Attribute{}
		// failed to send to a provider
		if relayResultDataReliability.ProviderInfo.ProviderAddress != "" {
			errAttributes = append(errAttributes, utils.Attribute{Key: "address", Value: relayResultDataReliability.ProviderInfo.ProviderAddress})
		}
		errAttributes = append(errAttributes, utils.Attribute{Key: "relayRequestData", Value: relayRequestData})
		return utils.LavaFormatWarning("failed data reliability relay to provider", err, errAttributes...)
	}
	if !relayResultDataReliability.Finalized {
		utils.LavaFormatInfo("skipping data reliability check since response from second provider was not finalized", utils.Attribute{Key: "providerAddress", Value: relayResultDataReliability.ProviderInfo.ProviderAddress})
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

func (rpccs *RPCConsumerServer) LavaDirectiveHeaders(metadata []pairingtypes.Metadata) ([]pairingtypes.Metadata, map[string]string) {
	metadataRet := []pairingtypes.Metadata{}
	headerDirectives := map[string]string{}
	for _, metaElement := range metadata {
		name := strings.ToLower(metaElement.Name)
		switch name {
		case common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME:
			headerDirectives[name] = metaElement.Value
		case common.RELAY_TIMEOUT_HEADER_NAME:
			headerDirectives[name] = metaElement.Value
		case common.EXTENSION_OVERRIDE_HEADER_NAME:
			headerDirectives[name] = metaElement.Value
		default:
			metadataRet = append(metadataRet, metaElement)
		}
	}
	return metadataRet, headerDirectives
}

func (rpccs *RPCConsumerServer) GetInitialUnwantedProviders(directiveHeaders map[string]string) map[string]struct{} {
	unwantedProviders := map[string]struct{}{}
	blockedProviders, ok := directiveHeaders[common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME]
	if ok {
		providerAddressesToBlock := strings.Split(blockedProviders, ",")
		for _, providerAddress := range providerAddressesToBlock {
			unwantedProviders[providerAddress] = struct{}{}
		}
	}
	return unwantedProviders
}

func (rpccs *RPCConsumerServer) HandleDirectiveHeadersForMessage(chainMessage chainlib.ChainMessage, directiveHeaders map[string]string) {
	timeoutStr, ok := directiveHeaders[common.RELAY_TIMEOUT_HEADER_NAME]
	if ok {
		timeout, err := time.ParseDuration(timeoutStr)
		if err == nil {
			// set an override timeout
			chainMessage.TimeoutOverride(timeout)
		}
	}
	extensionsStr, ok := directiveHeaders[common.EXTENSION_OVERRIDE_HEADER_NAME]
	if ok {
		extensions := strings.Split(extensionsStr, ",")
		_, extensions, _ = rpccs.chainParser.SeparateAddonsExtensions(extensions)
		if len(extensions) == 1 && extensions[0] == "none" {
			// none eliminates existing extensions
			chainMessage.OverrideExtensions([]string{}, rpccs.chainParser.ExtensionsParser())
		} else if len(extensions) > 0 {
			chainMessage.OverrideExtensions(extensions, rpccs.chainParser.ExtensionsParser())
		}
	}
}

func (rpccs *RPCConsumerServer) appendHeadersToRelayResult(ctx context.Context, relayResult *common.RelayResult, retries uint64) {
	if relayResult == nil {
		return
	}
	metadataReply := []pairingtypes.Metadata{}
	// add the provider that responded
	if relayResult.GetProvider() != "" {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.PROVIDER_ADDRESS_HEADER_NAME,
				Value: relayResult.GetProvider(),
			})
	}
	// add the relay retried count
	if retries > 0 {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.RETRY_COUNT_HEADER_NAME,
				Value: strconv.FormatUint(retries, 10),
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
	if relayResult.Reply == nil {
		relayResult.Reply = &pairingtypes.RelayReply{}
	}
	relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, metadataReply...)
}

func (rpccs *RPCConsumerServer) IsHealthy() bool {
	if RelaysHealthEnable {
		return rpccs.relaysMonitor.IsHealthy()
	} else {
		return false
	}
}
