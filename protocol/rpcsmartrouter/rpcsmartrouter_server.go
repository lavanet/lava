package rpcsmartrouter

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/performance"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/protocol/upgrade"
	"github.com/lavanet/lava/v5/utils"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"google.golang.org/grpc"
)

const (
	// maximum number of retries to send due to the ticker, if we didn't get a response after 10 different attempts then just wait.
	MaximumNumberOfTickerRelayRetries = 10
	MaxRelayRetries                   = 6
	SendRelayAttempts                 = 3
	initRelaysDappId                  = "-init-"
	initRelaysSmartRouterIp           = ""

	PairingInitializationTimeout = 30 * time.Second
	PairingCheckInterval         = 1 * time.Second
	RelayRetryBackoffDuration    = 2 * time.Millisecond
)

// implements Relay Sender interfaced and uses an ChainListener to get it called
type RPCSmartRouterServer struct {
	chainParser                    chainlib.ChainParser
	chainTracker                   chaintracker.IChainTracker
	sessionManager                 *lavasession.ConsumerSessionManager
	listenEndpoint                 *lavasession.RPCEndpoint
	rpcSmartRouterLogs             *metrics.RPCConsumerLogs
	cache                          *performance.Cache
	smartRouterConsistency         relaycore.Consistency
	sharedState                    bool // using the cache backend to sync the latest seen block
	relaysMonitor                  *metrics.RelaysMonitor
	debugRelays                    bool
	chainListener                  chainlib.ChainListener
	relayRetriesManager            *lavaprotocol.RelayRetriesManager
	initialized                    atomic.Bool
	latestBlockHeight              atomic.Uint64
	latestBlockEstimator           *relaycore.LatestBlockEstimator

	// gRPC streaming subscription manager (nil if not configured)
	grpcSubscriptionManager *DirectGRPCSubscriptionManager
}

func (rpcss *RPCSmartRouterServer) ServeRPCRequests(
	ctx context.Context,
	listenEndpoint *lavasession.RPCEndpoint,
	chainParser chainlib.ChainParser,
	chainTracker chaintracker.IChainTracker,
	sessionManager *lavasession.ConsumerSessionManager,
	cache *performance.Cache,
	rpcSmartRouterLogs *metrics.RPCConsumerLogs,
	smartRouterConsistency relaycore.Consistency,
	relaysMonitor *metrics.RelaysMonitor,
	cmdFlags common.ConsumerCmdFlags,
	sharedState bool,
	wsSubscriptionManager chainlib.WSSubscriptionManager,
) (err error) {
	rpcss.sessionManager = sessionManager
	rpcss.listenEndpoint = listenEndpoint
	rpcss.cache = cache
	rpcss.chainTracker = chainTracker
	rpcss.rpcSmartRouterLogs = rpcSmartRouterLogs
	rpcss.chainParser = chainParser
	rpcss.smartRouterConsistency = smartRouterConsistency
	rpcss.sharedState = sharedState
	rpcss.debugRelays = cmdFlags.DebugRelays
	rpcss.relayRetriesManager = lavaprotocol.NewRelayRetriesManager()
	rpcss.latestBlockEstimator = relaycore.NewLatestBlockEstimator()
	// NewChainListener now accepts WSSubscriptionManager interface, which is implemented
	// by both ConsumerWSSubscriptionManager (provider-relay mode) and
	// DirectWSSubscriptionManager (direct RPC mode for smart router).
	rpcss.chainListener, err = chainlib.NewChainListener(ctx, listenEndpoint, rpcss, rpcss, rpcSmartRouterLogs, chainParser, wsSubscriptionManager)
	if err != nil {
		return err
	}

	go rpcss.chainListener.Serve(ctx, cmdFlags)

	initialRelays := true
	rpcss.relaysMonitor = relaysMonitor

	// we trigger a latest block call to get some more information on our providers, using the relays monitor
	if cmdFlags.RelaysHealthEnableFlag {
		rpcss.relaysMonitor.SetRelaySender(func() (bool, error) {
			success, err := rpcss.sendCraftedRelaysWrapper(ctx, initialRelays)
			if success {
				initialRelays = false
			}
			return success, err
		})
		rpcss.relaysMonitor.Start(ctx)
	} else {
		rpcss.sendCraftedRelaysWrapper(ctx, true)
	}
	return nil
}

func (rpcss *RPCSmartRouterServer) SetConsistencySeenBlock(blockSeen int64, key string) {
	rpcss.smartRouterConsistency.SetSeenBlockFromKey(blockSeen, key)
}

func (rpcss *RPCSmartRouterServer) GetListeningAddress() string {
	return rpcss.chainListener.GetListeningAddress()
}

// GetGRPCReflectionConnection implements chainlib.GRPCReflectionProvider.
// This enables gRPC reflection for tools like grpcurl when using Direct RPC mode.
// Returns a connection to the upstream gRPC server for reflection requests.
func (rpcss *RPCSmartRouterServer) GetGRPCReflectionConnection(ctx context.Context) (*grpc.ClientConn, func(), error) {
	if rpcss.grpcSubscriptionManager == nil {
		return nil, nil, fmt.Errorf("gRPC reflection not available: no gRPC subscription manager configured")
	}

	return rpcss.grpcSubscriptionManager.GetReflectionConnection(ctx)
}

func (rpcss *RPCSmartRouterServer) sendCraftedRelaysWrapper(ctx context.Context, initialRelays bool) (bool, error) {
	if initialRelays {
		// Only start after everything is initialized - check consumer session manager
		rpcss.waitForPairing(ctx)
	}
	success, err := rpcss.sendCraftedRelays(MaxRelayRetries, initialRelays)
	if success {
		rpcss.initialized.Store(true)
	}
	return success, err
}

func (rpcss *RPCSmartRouterServer) waitForPairing(ctx context.Context) {
	reinitializedChan := make(chan bool, 1) // Buffered channel to prevent deadlock

	go func() {
		ticker := time.NewTicker(PairingCheckInterval)
		defer ticker.Stop() // Ensure ticker is cleaned up

		for {
			select {
			case <-ctx.Done():
				// Context cancelled, exit goroutine
				return
			case <-ticker.C:
				if rpcss.sessionManager.Initialized() {
					// Non-blocking send to prevent deadlock
					select {
					case reinitializedChan <- true:
					default:
						// Channel already has value or receiver gone, but we can exit
					}
					return
				}
			}
		}
	}()

	numberOfTimesChecked := 0
	for {
		select {
		case <-reinitializedChan:
			return
		case <-ctx.Done():
			// Context cancelled, exit function
			return
		case <-time.After(PairingInitializationTimeout):
			numberOfTimesChecked += 1
			utils.LavaFormatWarning("failed initial relays, csm was not initialized after timeout, or pairing list is empty for that chain", nil,
				utils.LogAttr("times_checked", numberOfTimesChecked),
				utils.LogAttr("chainID", rpcss.listenEndpoint.ChainID),
				utils.LogAttr("APIInterface", rpcss.listenEndpoint.ApiInterface),
			)
		}
	}
}

func (rpcss *RPCSmartRouterServer) craftRelay(ctx context.Context) (ok bool, relay *pairingtypes.RelayPrivateData, chainMessage chainlib.ChainMessage, err error) {
	parsing, apiCollection, ok := rpcss.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	if !ok {
		return false, nil, nil, utils.LavaFormatWarning("did not send initial relays because the spec does not contain "+spectypes.FUNCTION_TAG_GET_BLOCKNUM.String(), nil,
			utils.LogAttr("chainID", rpcss.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpcss.listenEndpoint.ApiInterface),
		)
	}
	collectionData := apiCollection.CollectionData

	path := parsing.ApiName
	data := []byte(parsing.FunctionTemplate)
	chainMessage, err = rpcss.chainParser.ParseMsg(path, data, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	if err != nil {
		return false, nil, nil, utils.LavaFormatError("failed creating chain message in rpc consumer init relays", err,
			utils.LogAttr("chainID", rpcss.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpcss.listenEndpoint.ApiInterface))
	}

	reqBlock, _ := chainMessage.RequestedBlock()
	seenBlock := int64(0)
	relay = lavaprotocol.NewRelayData(ctx, collectionData.Type, path, data, seenBlock, reqBlock, rpcss.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMessage), nil)
	return true, relay, chainMessage, nil
}

func (rpcss *RPCSmartRouterServer) sendRelayWithRetries(ctx context.Context, retries int, initialRelays bool, protocolMessage chainlib.ProtocolMessage) (bool, error) {
	success := false
	var err error
	usedProviders := lavasession.NewUsedProviders(nil)

	// Get quorum parameters from protocol message
	quorumParams, err := protocolMessage.GetQuorumParameters()
	if err != nil {
		return false, err
	}

	// Validate that quorum min doesn't exceed available providers
	if quorumParams.Enabled() && quorumParams.Min > rpcss.sessionManager.GetNumberOfValidProviders() {
		return false, utils.LavaFormatError("requested quorum min exceeds available providers",
			lavasession.PairingListEmptyError,
			utils.LogAttr("quorumMin", quorumParams.Min),
			utils.LogAttr("availableProviders", rpcss.sessionManager.GetNumberOfValidProviders()),
			utils.LogAttr("GUID", ctx))
	}

	relayProcessor := relaycore.NewRelayProcessor(
		ctx,
		quorumParams,
		rpcss.smartRouterConsistency,
		rpcss.rpcSmartRouterLogs,
		rpcss,
		rpcss.relayRetriesManager,
		NewSmartRouterRelayStateMachine(ctx, usedProviders, rpcss, protocolMessage, nil, rpcss.debugRelays, rpcss.rpcSmartRouterLogs),
		rpcss.sessionManager.GetQoSManager(),
	)
	usedProvidersResets := 1
	for i := 0; i < retries; i++ {
		// Check if we even have enough providers to communicate with them all.
		// If we have 1 provider we will reset the used providers always.
		// Instead of spamming no pairing available on bootstrap
		if ((i + 1) * usedProvidersResets) > rpcss.sessionManager.GetNumberOfValidProviders() {
			usedProvidersResets++
			relayProcessor.GetUsedProviders().ClearUnwanted()
		}
		err = rpcss.sendRelayToProvider(ctx, 1, relaycore.GetEmptyRelayState(ctx, protocolMessage), relayProcessor, nil)
		if lavasession.PairingListEmptyError.Is(err) {
			// we don't have pairings anymore, could be related to unwanted providers
			relayProcessor.GetUsedProviders().ClearUnwanted()
			err = rpcss.sendRelayToProvider(ctx, 1, relaycore.GetEmptyRelayState(ctx, protocolMessage), relayProcessor, nil)
		}
		if err != nil {
			utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpcss.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpcss.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
		} else {
			err := relayProcessor.WaitForResults(ctx)
			if err != nil {
				utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpcss.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpcss.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
			} else {
				relayResult, err := relayProcessor.ProcessingResult()
				if err == nil && relayResult != nil && relayResult.Reply != nil {
					utils.LavaFormatInfo("[+] init relay succeeded",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("chainID", rpcss.listenEndpoint.ChainID),
						utils.LogAttr("APIInterface", rpcss.listenEndpoint.ApiInterface),
						utils.LogAttr("latestBlock", relayResult.Reply.LatestBlock),
						utils.LogAttr("provider address", relayResult.ProviderInfo.ProviderAddress),
					)
					if relayResult.Reply.LatestBlock > 0 {
						rpcss.updateLatestBlockHeight(uint64(relayResult.Reply.LatestBlock), relayResult.ProviderInfo.ProviderAddress)
					}
					rpcss.relaysMonitor.LogRelay()
					success = true
					// If this is the first time we send relays, we want to send all of them, instead of break on first successful relay
					// That way, we populate the providers with the latest blocks with successful relays
					if !initialRelays {
						break
					}
				} else if err != nil {
					utils.LavaFormatError("[-] failed sending init relay", err, []utils.Attribute{{Key: "chainID", Value: rpcss.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpcss.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
				} else {
					utils.LavaFormatError("[-] failed sending init relay - nil result", nil, []utils.Attribute{{Key: "chainID", Value: rpcss.listenEndpoint.ChainID}, {Key: "APIInterface", Value: rpcss.listenEndpoint.ApiInterface}, {Key: "relayProcessor", Value: relayProcessor}}...)
				}
			}
		}
		time.Sleep(RelayRetryBackoffDuration)
	}

	return success, err
}

// sending a few latest blocks relays to providers in order to have some data on the providers when relays start arriving
func (rpcss *RPCSmartRouterServer) sendCraftedRelays(retries int, initialRelays bool) (bool, error) {
	utils.LavaFormatDebug("Sending crafted relays",
		utils.LogAttr("chainId", rpcss.listenEndpoint.ChainID),
		utils.LogAttr("apiInterface", rpcss.listenEndpoint.ApiInterface),
	)

	ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
	ok, relay, chainMessage, _ := rpcss.craftRelay(ctx)
	if !ok {
		return true, nil
	}
	protocolMessage := chainlib.NewProtocolMessage(chainMessage, nil, relay, initRelaysDappId, initRelaysSmartRouterIp)
	return rpcss.sendRelayWithRetries(ctx, retries, initialRelays, protocolMessage)
}

func (rpcss *RPCSmartRouterServer) getLatestBlock() uint64 {
	// Primary: Use chain tracker if available (for archive extension routing)
	if rpcss.chainTracker != nil && !rpcss.chainTracker.IsDummy() {
		if block, _ := rpcss.chainTracker.GetLatestBlockNum(); block > 0 {
			return uint64(block)
		}
	}

	// Secondary: Use latest block estimator
	if rpcss.latestBlockEstimator != nil {
		latestKnownBlock, numProviders := rpcss.latestBlockEstimator.Estimate(rpcss.chainParser)
		if numProviders > 0 && latestKnownBlock > 0 {
			return uint64(latestKnownBlock)
		}
	}

	// Tertiary: Use stored latest block height
	if latest := rpcss.latestBlockHeight.Load(); latest > 0 {
		return latest
	}

	utils.LavaFormatWarning("smart router has no information on latest block", nil, utils.Attribute{Key: "latest block", Value: 0})
	return 0
}

func (rpcss *RPCSmartRouterServer) updateLatestBlockHeight(blockHeight uint64, providerAddress string) {
	for {
		current := rpcss.latestBlockHeight.Load()
		if blockHeight <= current {
			break
		}
		if rpcss.latestBlockHeight.CompareAndSwap(current, blockHeight) {
			break
		}
	}

	if providerAddress != "" && rpcss.latestBlockEstimator != nil {
		rpcss.latestBlockEstimator.Record(providerAddress, int64(blockHeight))
	}
}

func (rpcss *RPCSmartRouterServer) SendRelay(
	ctx context.Context,
	url string,
	req string,
	connectionType string,
	dappID string,
	consumerIp string,
	analytics *metrics.RelayMetrics,
	metadata []pairingtypes.Metadata,
) (relayResult *common.RelayResult, errRet error) {
	protocolMessage, err := rpcss.ParseRelay(ctx, url, req, connectionType, dappID, consumerIp, metadata)
	if err != nil {
		return nil, err
	}

	return rpcss.SendParsedRelay(ctx, analytics, protocolMessage)
}

func (rpcss *RPCSmartRouterServer) ParseRelay(
	ctx context.Context,
	url string,
	req string,
	connectionType string,
	dappID string,
	consumerIp string,
	metadata []pairingtypes.Metadata,
) (protocolMessage chainlib.ProtocolMessage, err error) {
	// gets the relay request data from the ChainListener
	// parses the request into an APIMessage, and validating it corresponds to the spec currently in use
	// construct the common data for a relay message, common data is identical across multiple sends and data reliability

	// remove lava directive headers
	metadata, directiveHeaders := rpcss.LavaDirectiveHeaders(metadata)
	extensions := rpcss.getExtensionsFromDirectiveHeaders(directiveHeaders)
	utils.LavaFormatTrace("[Archive Debug] ParseRelay extensions",
		utils.LogAttr("extensions", extensions),
		utils.LogAttr("GUID", ctx))
	utils.LavaFormatTrace("[Archive Debug] Calling chainParser.ParseMsg", utils.LogAttr("url", url), utils.LogAttr("req", req), utils.LogAttr("extensions", extensions), utils.LogAttr("chainParserType", fmt.Sprintf("%T", rpcss.chainParser)), utils.LogAttr("GUID", ctx))
	chainMessage, err := rpcss.chainParser.ParseMsg(url, []byte(req), connectionType, metadata, extensions)
	if err != nil {
		return nil, err
	}

	rpcss.HandleDirectiveHeadersForMessage(chainMessage, directiveHeaders)

	// do this in a loop with retry attempts, configurable via a flag, limited by the number of providers in CSM
	reqBlock, _ := chainMessage.RequestedBlock()
	seenBlock, _ := rpcss.smartRouterConsistency.GetSeenBlock(common.UserData{DappId: dappID, ConsumerIp: consumerIp})
	if seenBlock < 0 {
		seenBlock = 0
	}

	relayRequestData := lavaprotocol.NewRelayData(ctx, connectionType, url, []byte(req), seenBlock, reqBlock, rpcss.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMessage), common.GetExtensionNames(chainMessage.GetExtensions()))
	protocolMessage = chainlib.NewProtocolMessage(chainMessage, directiveHeaders, relayRequestData, dappID, consumerIp)
	return protocolMessage, nil
}

func (rpcss *RPCSmartRouterServer) SendParsedRelay(
	ctx context.Context,
	analytics *metrics.RelayMetrics,
	protocolMessage chainlib.ProtocolMessage,
) (relayResult *common.RelayResult, errRet error) {
	// sends a relay message to a provider
	// compares the result with other providers if defined so
	// compares the response with other consumer wallets if defined so
	// asynchronously sends data reliability if necessary

	relaySentTime := time.Now()
	relayProcessor, err := rpcss.ProcessRelaySend(ctx, protocolMessage, analytics)
	if err != nil && (relayProcessor == nil || !relayProcessor.HasResults()) {
		userData := protocolMessage.GetUserData()
		// we can't send anymore, and we don't have any responses
		utils.LavaFormatError("failed getting responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: utils.KEY_REQUEST_ID, Value: ctx}, utils.Attribute{Key: utils.KEY_TASK_ID, Value: ctx}, utils.Attribute{Key: utils.KEY_TRANSACTION_ID, Value: ctx}, utils.LogAttr("endpoint", rpcss.listenEndpoint.Key()), utils.LogAttr("userIp", userData.ConsumerIp), utils.LogAttr("relayProcessor", relayProcessor))

		return nil, err
	}

	returnedResult, err := relayProcessor.ProcessingResult()

	utils.LavaFormatInfo("ProcessingResult RETURNED",
		utils.LogAttr("has_result", returnedResult != nil),
		utils.LogAttr("has_reply", returnedResult != nil && returnedResult.Reply != nil),
		utils.LogAttr("reply_size", func() int {
			if returnedResult != nil && returnedResult.Reply != nil {
				return len(returnedResult.Reply.Data)
			}
			return 0
		}()),
		utils.LogAttr("error", err),
		utils.LogAttr("GUID", ctx),
	)

	rpcss.appendHeadersToRelayResult(ctx, returnedResult, relayProcessor.ProtocolErrors(), relayProcessor, protocolMessage, protocolMessage.GetApi().GetName())
	if err != nil {
		// Check if this is an unsupported method error from the error message
		// This catches cases where the provider returns a gRPC error instead of response data
		if rpcss.cache.CacheActive() && chainlib.IsUnsupportedMethodError(err) {
			utils.LavaFormatDebug("ProcessingResult returned unsupported method error - caching",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("error", err.Error()),
			)
			// Cache the unsupported method error for future requests
			go rpcss.cacheUnsupportedMethodErrorResponse(ctx, protocolMessage, returnedResult)
		}
		return returnedResult, utils.LavaFormatError("failed processing responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: utils.KEY_REQUEST_ID, Value: ctx}, utils.Attribute{Key: utils.KEY_TASK_ID, Value: ctx}, utils.Attribute{Key: utils.KEY_TRANSACTION_ID, Value: ctx}, utils.LogAttr("endpoint", rpcss.listenEndpoint.Key()))
	}

	if analytics != nil {
		currentLatency := time.Since(relaySentTime)
		analytics.Latency = currentLatency.Milliseconds()
		api := protocolMessage.GetApi()
		analytics.ComputeUnits = api.ComputeUnits
		analytics.ApiMethod = api.Name
	}
	rpcss.relaysMonitor.LogRelay()
	return returnedResult, nil
}

func (rpcss *RPCSmartRouterServer) GetChainIdAndApiInterface() (string, string) {
	return rpcss.listenEndpoint.ChainID, rpcss.listenEndpoint.ApiInterface
}

// cacheUnsupportedMethodErrorResponse caches an unsupported method error like a regular response
// This is called when the provider returns a gRPC error instead of response data
func (rpcss *RPCSmartRouterServer) cacheUnsupportedMethodErrorResponse(ctx context.Context, protocolMessage chainlib.ProtocolMessage, relayResult *common.RelayResult) {
	if relayResult == nil {
		return
	}

	chainId, _ := rpcss.GetChainIdAndApiInterface()
	relayData := protocolMessage.RelayPrivateData()
	if relayData == nil {
		return
	}

	// Create error response with placeholder GUID
	// Note: When retrieved from cache, the actual request GUID will be different
	// We use "CACHED_ERROR" as a marker to indicate this is a cached unsupported method error
	errorData := `{"Error_GUID":"CACHED_ERROR","error":{"code":-32601,"message":"Method not found"}}`
	errorReply := &pairingtypes.RelayReply{
		Data:        []byte(errorData),
		LatestBlock: relayResult.Reply.GetLatestBlock(),
	}

	// Get the resolved block for caching
	// relayData.RequestBlock may still be negative (-2 for LATEST_BLOCK)
	// Use seenBlock as the resolved block for caching
	requestedBlock := relayData.SeenBlock
	if requestedBlock <= 0 {
		// Fallback to 0 if seenBlock is also not set
		requestedBlock = 0
	}
	seenBlock := relayData.SeenBlock

	hashKey, _, hashErr := chainlib.HashCacheRequest(relayData, chainId)
	if hashErr != nil {
		return
	}

	// Determine if finalized based on block age (like regular responses)
	_, _, blockDistanceForFinalizedData, _ := rpcss.chainParser.ChainBlockStats()
	var latestBlock int64
	if relayResult.Reply != nil {
		latestBlock = relayResult.Reply.LatestBlock
	}
	finalized := spectypes.IsFinalizedBlock(requestedBlock, latestBlock, int64(blockDistanceForFinalizedData))

	cacheCtx, cancel := context.WithTimeout(context.Background(), common.CacheWriteTimeout)
	defer cancel()
	_, averageBlockTime, _, _ := rpcss.chainParser.ChainBlockStats()

	err := rpcss.cache.SetEntry(cacheCtx, &pairingtypes.RelayCacheSet{
		RequestHash:           hashKey,
		ChainId:               chainId,
		RequestedBlock:        requestedBlock,
		SeenBlock:             seenBlock,
		BlockHash:             nil,
		Response:              errorReply,
		Finalized:             finalized,
		OptionalMetadata:      nil,
		SharedStateId:         "",
		AverageBlockTime:      int64(averageBlockTime),
		IsNodeError:           false,
		BlocksHashesToHeights: nil,
	})

	if err != nil {
		utils.LavaFormatWarning("error caching unsupported method error response", err, utils.LogAttr("GUID", ctx))
	} else {
		utils.LavaFormatDebug("Successfully cached unsupported method error",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("requestedBlock", requestedBlock),
			utils.LogAttr("finalized", finalized),
		)
	}
}

func (rpcss *RPCSmartRouterServer) ProcessRelaySend(ctx context.Context, protocolMessage chainlib.ProtocolMessage, analytics *metrics.RelayMetrics) (*relaycore.RelayProcessor, error) {
	// make sure all of the child contexts are cancelled when we exit
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	usedProviders := lavasession.NewUsedProviders(protocolMessage)

	quorumParams, err := protocolMessage.GetQuorumParameters()
	if err != nil {
		return nil, err
	}

	// Validate that quorum min doesn't exceed available providers
	if quorumParams.Enabled() && quorumParams.Min > rpcss.sessionManager.GetNumberOfValidProviders() {
		return nil, utils.LavaFormatError("requested quorum min exceeds available providers",
			lavasession.PairingListEmptyError,
			utils.LogAttr("quorumMin", quorumParams.Min),
			utils.LogAttr("availableProviders", rpcss.sessionManager.GetNumberOfValidProviders()),
			utils.LogAttr("GUID", ctx))
	}

	relayProcessor := relaycore.NewRelayProcessor(
		ctx,
		quorumParams,
		rpcss.smartRouterConsistency,
		rpcss.rpcSmartRouterLogs,
		rpcss,
		rpcss.relayRetriesManager,
		NewSmartRouterRelayStateMachine(ctx, usedProviders, rpcss, protocolMessage, analytics, rpcss.debugRelays, rpcss.rpcSmartRouterLogs),
		rpcss.sessionManager.GetQoSManager(),
	)

	relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
	if err != nil {
		return relayProcessor, err
	}

	utils.LavaFormatInfo("🎬 STARTING TASK CHANNEL LOOP",
		utils.LogAttr("GUID", ctx),
	)

	for task := range relayTaskChannel {
		utils.LavaFormatInfo("📨 RECEIVED TASK FROM CHANNEL",
			utils.LogAttr("is_done", task.IsDone()),
			utils.LogAttr("has_error", task.Err != nil),
			utils.LogAttr("num_providers", task.NumOfProviders),
			utils.LogAttr("GUID", ctx),
		)

		if task.IsDone() {
			utils.LavaFormatInfo("🏁 TASK IS DONE - RETURNING FROM ProcessRelaySend",
				utils.LogAttr("error", task.Err),
				utils.LogAttr("GUID", ctx),
			)
			return relayProcessor, task.Err
		}
		utils.LavaFormatTrace("[RPCSmartRouterServer] ProcessRelaySend - task", utils.LogAttr("GUID", ctx), utils.LogAttr("numOfProviders", task.NumOfProviders))
		err := rpcss.sendRelayToProvider(ctx, task.NumOfProviders, task.RelayState, relayProcessor, task.Analytics)

		utils.LavaFormatInfo("UPDATING BATCH",
			utils.LogAttr("error", err),
			utils.LogAttr("GUID", ctx),
		)

		relayProcessor.UpdateBatch(err)

		utils.LavaFormatInfo("LOOPING BACK TO RECEIVE NEXT TASK",
			utils.LogAttr("GUID", ctx),
		)
	}

	// shouldn't happen.
	utils.LavaFormatError("CHANNEL CLOSED UNEXPECTEDLY", nil,
		utils.LogAttr("GUID", ctx),
	)
	return relayProcessor, utils.LavaFormatError("ProcessRelaySend channel closed unexpectedly", nil)
}

func (rpcss *RPCSmartRouterServer) CreateDappKey(userData common.UserData) string {
	return rpcss.smartRouterConsistency.Key(userData)
}

func (rpcss *RPCSmartRouterServer) CancelSubscriptionContext(subscriptionKey string) {
	// Direct RPC subscription managers handle their own lifecycle.
}

func (rpcss *RPCSmartRouterServer) getEarliestBlockHashRequestedFromCacheReply(cacheReply *pairingtypes.CacheRelayReply) (int64, int64) {
	blocksHashesToHeights := cacheReply.GetBlocksHashesToHeights()
	earliestRequestedBlock := spectypes.NOT_APPLICABLE
	latestRequestedBlock := spectypes.NOT_APPLICABLE

	for _, blockHashToHeight := range blocksHashesToHeights {
		if blockHashToHeight.Height >= 0 && (earliestRequestedBlock == spectypes.NOT_APPLICABLE || blockHashToHeight.Height < earliestRequestedBlock) {
			earliestRequestedBlock = blockHashToHeight.Height
		}
		if blockHashToHeight.Height >= 0 && (latestRequestedBlock == spectypes.NOT_APPLICABLE || blockHashToHeight.Height > latestRequestedBlock) {
			latestRequestedBlock = blockHashToHeight.Height
		}
	}
	return latestRequestedBlock, earliestRequestedBlock
}

func (rpcss *RPCSmartRouterServer) resolveRequestedBlock(reqBlock int64, seenBlock int64, latestBlockHashRequested int64, protocolMessage chainlib.ProtocolMessage) int64 {
	if reqBlock == spectypes.LATEST_BLOCK && seenBlock != 0 {
		// make optimizer select a provider that is likely to have the latest seen block
		reqBlock = seenBlock
	}

	// Following logic to set the requested block as a new value fetched from the cache reply.
	// 1. We managed to get a value from the cache reply. (latestBlockHashRequested >= 0)
	// 2. We didn't manage to parse the block and used the default value meaning we didnt have knowledge of the requested block (reqBlock == spectypes.LATEST_BLOCK && protocolMessage.GetUsedDefaultValue())
	// 3. The requested block is smaller than the latest block hash requested from the cache reply (reqBlock >= 0 && reqBlock < latestBlockHashRequested)
	// 4. The requested block is not applicable meaning block parsing failed completely (reqBlock == spectypes.NOT_APPLICABLE)
	if latestBlockHashRequested >= 0 &&
		((reqBlock == spectypes.LATEST_BLOCK && protocolMessage.GetUsedDefaultValue()) ||
			reqBlock >= 0 && reqBlock < latestBlockHashRequested) {
		reqBlock = latestBlockHashRequested
	}
	return reqBlock
}

func (rpcss *RPCSmartRouterServer) newBlocksHashesToHeightsSliceFromRequestedBlockHashes(requestedBlockHashes []string) []*pairingtypes.BlockHashToHeight {
	var blocksHashesToHeights []*pairingtypes.BlockHashToHeight
	for _, blockHash := range requestedBlockHashes {
		blocksHashesToHeights = append(blocksHashesToHeights, &pairingtypes.BlockHashToHeight{Hash: blockHash, Height: spectypes.NOT_APPLICABLE})
	}
	return blocksHashesToHeights
}

func deepCopyRelayPrivateData(original *pairingtypes.RelayPrivateData) *pairingtypes.RelayPrivateData {
	if original == nil {
		return nil
	}

	// Deep copy all byte slices and string slices
	dataCopy := make([]byte, len(original.Data))
	copy(dataCopy, original.Data)

	saltCopy := make([]byte, len(original.Salt))
	copy(saltCopy, original.Salt)

	metadataCopy := make([]pairingtypes.Metadata, len(original.Metadata))
	copy(metadataCopy, original.Metadata)

	extensionsCopy := make([]string, len(original.Extensions))
	copy(extensionsCopy, original.Extensions)

	return &pairingtypes.RelayPrivateData{
		ConnectionType: original.ConnectionType,
		ApiUrl:         original.ApiUrl,
		Data:           dataCopy,
		RequestBlock:   original.RequestBlock,
		ApiInterface:   original.ApiInterface,
		Salt:           saltCopy,
		Metadata:       metadataCopy,
		Addon:          original.Addon,
		Extensions:     extensionsCopy,
		SeenBlock:      original.SeenBlock,
	}
}

// sendRelayToDirectEndpoints handles relay for direct RPC sessions (smart router direct mode)
func (rpcss *RPCSmartRouterServer) sendRelayToDirectEndpoints(
	ctx context.Context,
	sessions lavasession.ConsumerSessionsMap,
	protocolMessage chainlib.ProtocolMessage,
	relayProcessor *relaycore.RelayProcessor,
	analytics *metrics.RelayMetrics,
) error {
	chainMessage := protocolMessage

	// Get relay timeout
	_, averageBlockTime, _, _ := rpcss.chainParser.ChainBlockStats()
	relayTimeout := chainlib.GetRelayTimeout(protocolMessage, averageBlockTime)

	utils.LavaFormatDebug("sending direct RPC relay",
		utils.LogAttr("num_endpoints", len(sessions)),
		utils.LogAttr("timeout", relayTimeout),
		utils.LogAttr("method", chainMessage.GetApi().Name),
		utils.LogAttr("GUID", ctx),
	)

	// Launch goroutines for each direct RPC endpoint (parallel relay pattern)
	for endpointAddress, sessionInfo := range sessions {
		go func(endpointAddress string, sessionInfo *lavasession.SessionInfo) {
			goroutineCtx, goroutineCtxCancel := context.WithCancel(context.Background())

			guid, found := utils.GetUniqueIdentifier(ctx)
			if found {
				goroutineCtx = utils.WithUniqueIdentifier(goroutineCtx, guid)
			}

			singleConsumerSession := sessionInfo.Session

			localRelayResult := &common.RelayResult{
				ProviderInfo: common.ProviderInfo{ProviderAddress: endpointAddress},
				Finalized:    true, // Direct responses don't need consensus
			}

			var errResponse error

			// CRITICAL: Use defer to set response (same as provider-relay pattern)
			// This ensures all work completes before response is sent
			defer func() {
				// Set response for relay processor (MUST be in defer!)
				relayProcessor.SetResponse(&relaycore.RelayResponse{
					RelayResult: *localRelayResult,
					Err:         errResponse,
				})

				// Close context
				goroutineCtxCancel()
			}()

			relayLatency, err, _ := rpcss.relayInnerDirect(
				goroutineCtx,
				singleConsumerSession,
				localRelayResult,
				relayTimeout,
				chainMessage,
				analytics,
			)

			// Handle response
			if err != nil {
				utils.LavaFormatDebug("direct RPC relay failed in goroutine",
					utils.LogAttr("endpoint", endpointAddress),
					utils.LogAttr("error", err.Error()),
					utils.LogAttr("latency", relayLatency),
					utils.LogAttr("GUID", goroutineCtx),
				)
				errResponse = err
			} else {
				utils.LavaFormatDebug("direct RPC relay succeeded in goroutine",
					utils.LogAttr("endpoint", endpointAddress),
					utils.LogAttr("latency", relayLatency),
					utils.LogAttr("GUID", goroutineCtx),
				)
			}

			// Update session manager with result
			// Check status code to determine if session should fail
			// For REST 5xx/429, err == nil but we still want to fail the session for QoS/retry
			statusCode := localRelayResult.StatusCode
			shouldFailSession := err != nil || statusCode >= 500 || statusCode == 429

			if !shouldFailSession {
				// Success or client error (4xx except 429) - update session as success
				// Get endpoint reference for per-endpoint tracking
				var targetEndpoint *lavasession.Endpoint
				if drsc, ok := singleConsumerSession.Connection.(*lavasession.DirectRPCSessionConnection); ok {
					targetEndpoint = drsc.Endpoint // Use stored reference (robust)
				}

				// Get latest block: prefer response extraction, fallback to ChainTracker
				latestBlock := int64(0)
				if localRelayResult.Reply != nil && localRelayResult.Reply.LatestBlock > 0 {
					// Block extracted from response (eth_blockNumber, eth_getBlockByNumber, etc.)
					latestBlock = localRelayResult.Reply.LatestBlock

					// Update endpoint's latest block (per-endpoint tracking)
					if targetEndpoint != nil {
						targetEndpoint.LatestBlock.Store(latestBlock)
						targetEndpoint.LastBlockUpdate = time.Now()
						utils.LavaFormatTrace("updated endpoint latest block",
							utils.LogAttr("endpoint", endpointAddress),
							utils.LogAttr("latest_block", latestBlock),
							utils.LogAttr("GUID", goroutineCtx),
						)
					}

					// Update global latest block height and estimator (for getLatestBlock fallback)
					rpcss.updateLatestBlockHeight(uint64(latestBlock), endpointAddress)
				} else if rpcss.chainTracker != nil && !rpcss.chainTracker.IsDummy() {
					// Fallback to ChainTracker (global)
					latestBlock, _ = rpcss.chainTracker.GetLatestBlockNum()
					utils.LavaFormatTrace("using latest block from chain tracker",
						utils.LogAttr("latest_block", latestBlock),
						utils.LogAttr("GUID", goroutineCtx),
					)
				} else {
					utils.LavaFormatDebug("no latest block available",
						utils.LogAttr("GUID", goroutineCtx),
					)
				}

				// Calculate syncGap (detect lagging endpoints)
				syncGap := int64(0)
				if targetEndpoint != nil && rpcss.chainTracker != nil && !rpcss.chainTracker.IsDummy() {
					globalLatest, _ := rpcss.chainTracker.GetLatestBlockNum()
					endpointLatest := targetEndpoint.LatestBlock.Load()
					if globalLatest > 0 && endpointLatest > 0 {
						syncGap = globalLatest - endpointLatest
						if syncGap < 0 {
							syncGap = 0 // Endpoint ahead is fine
						}
						utils.LavaFormatDebug("calculated sync gap",
							utils.LogAttr("endpoint", endpointAddress),
							utils.LogAttr("global_latest", globalLatest),
							utils.LogAttr("endpoint_latest", endpointLatest),
							utils.LogAttr("sync_gap", syncGap),
							utils.LogAttr("GUID", goroutineCtx),
						)
					}
				}

				// Call OnSessionDone with correct signature
				numSessions := len(sessions)
				errSession := rpcss.sessionManager.OnSessionDone(
					singleConsumerSession,
					latestBlock, // latestServicedBlock
					chainlib.GetComputeUnits(protocolMessage), // specComputeUnits
					relayLatency, // currentLatency
					singleConsumerSession.CalculateExpectedLatency(relayTimeout), // expectedLatency
					syncGap,             // Real syncGap (detect lagging endpoints)
					numSessions,         // numOfProviders (int)
					uint64(numSessions), // providersCount (uint64)
					protocolMessage.GetApi().Category.HangingApi, // hangingApi
					protocolMessage.GetExtensions(),              // extensions
				)
				if errSession != nil {
					utils.LavaFormatWarning("OnSessionDone failed for direct RPC", errSession,
						utils.LogAttr("GUID", goroutineCtx),
					)
				}
			} else {
				// Failure case: err != nil OR status >= 500 OR status == 429
				failureErr := err
				if failureErr == nil {
					// REST 5xx/429 with err == nil - create descriptive error
					failureErr = fmt.Errorf("upstream returned HTTP %d", statusCode)
				}
				rpcss.sessionManager.OnSessionFailure(singleConsumerSession, failureErr)
			}

			// NOTE: Don't call Free() here - OnSessionDone/OnSessionFailure already do it!
		}(endpointAddress, sessionInfo)
	}

	// NOTE: Don't call WaitForResults here!
	// The state machine already calls it via readResultsFromProcessor
	// Calling it twice causes a deadlock

	utils.LavaFormatInfo("GOROUTINES LAUNCHED - RETURNING TO LET STATE MACHINE WAIT",
		utils.LogAttr("num_endpoints", len(sessions)),
		utils.LogAttr("GUID", ctx),
	)

	return nil
}

func (rpcss *RPCSmartRouterServer) sendRelayToProvider(
	ctx context.Context,
	numOfProviders int,
	relayState *relaycore.RelayState,
	relayProcessor *relaycore.RelayProcessor,
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
	// Use the latest protocol message from the relay state machine to ensure we have any archive upgrades
	protocolMessage := relayProcessor.GetProtocolMessage()
	// IMPORTANT: Create an isolated copy of RelayPrivateData at function entry to prevent race conditions.
	// This ensures that modifications in this call don't affect goroutines from previous calls,
	// and goroutines launched in this call aren't affected by future calls.
	localRelayData := deepCopyRelayPrivateData(protocolMessage.RelayPrivateData())
	if localRelayData == nil {
		return utils.LavaFormatError("RelayPrivateData is nil", nil, utils.LogAttr("GUID", ctx))
	}

	userData := protocolMessage.GetUserData()
	var sharedStateId string // defaults to "", if shared state is disabled then no shared state will be used.
	if rpcss.sharedState {
		sharedStateId = rpcss.smartRouterConsistency.Key(userData) // use same key as we use for consistency, (for better consistency :-D)
	}

	chainId, _ := rpcss.GetChainIdAndApiInterface()

	// Get Session. we get session here so we can use the epoch in the callbacks
	reqBlock, _ := protocolMessage.RequestedBlock()

	// try using cache before sending relay
	earliestBlockHashRequested := spectypes.NOT_APPLICABLE
	latestBlockHashRequested := spectypes.NOT_APPLICABLE
	var cacheError error
	quorumParams := relayProcessor.GetQuorumParams()
	if rpcss.cache.CacheActive() && !quorumParams.Enabled() { // use cache only if its defined and quorum is disabled.
		utils.LavaFormatDebug("Cache lookup attempt",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("cacheActive", true),
			utils.LogAttr("reqBlock", reqBlock),
			utils.LogAttr("forceCacheRefresh", protocolMessage.GetForceCacheRefresh()),
			utils.LogAttr("quorumEnabled", false),
		)
	} else if rpcss.cache.CacheActive() && quorumParams.Enabled() {
		utils.LavaFormatDebug("Cache bypassed due to quorum validation requirements",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("cacheActive", true),
			utils.LogAttr("quorumEnabled", true),
			utils.LogAttr("quorumMin", quorumParams.Min),
			utils.LogAttr("reason", "quorum requires fresh provider validation, cache would defeat consensus verification"),
		)
	}
	if rpcss.cache.CacheActive() && !quorumParams.Enabled() { // use cache only if its defined and quorum is disabled.
		if !protocolMessage.GetForceCacheRefresh() {
			allowCacheLookup := reqBlock != spectypes.NOT_APPLICABLE

			if allowCacheLookup {
				var cacheReply *pairingtypes.CacheRelayReply
				hashKey, outputFormatter, err := protocolMessage.HashCacheRequest(chainId)
				if err != nil {
					utils.LavaFormatError("sendRelayToProvider Failed getting Hash for cache request", err, utils.LogAttr("GUID", ctx))
				} else {
					utils.LavaFormatDebug("Cache lookup hash generated",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("hashKey", fmt.Sprintf("%x", hashKey)),
						utils.LogAttr("apiUrl", localRelayData.ApiUrl),
					)

					// Resolve the requested block for cache lookup
					// The cache server doesn't accept negative blocks
					requestedBlockForCache := reqBlock
					if reqBlock == spectypes.LATEST_BLOCK {
						// For LATEST_BLOCK queries, use the latest known block from smartRouterConsistency
						// This ensures methods like eth_blockNumber use the actual current block for caching,
						// not the potentially stale seenBlock from when this request started.
						// The consistency cache is updated immediately after each successful response,
						// so it reflects the most recent block across all requests for this user.
						latestKnownBlock, found := rpcss.smartRouterConsistency.GetSeenBlock(userData)
						if found && latestKnownBlock > 0 {
							requestedBlockForCache = latestKnownBlock
						} else if localRelayData.SeenBlock != 0 {
							// Fallback to seen block from the protocol message
							requestedBlockForCache = localRelayData.SeenBlock
						} else {
							requestedBlockForCache = 0 // Final fallback
						}
					}

					// Always use finalized=false for lookups
					// The cache will search both tempCache and finalizedCache, finding data in either
					lookupFinalized := false

					cacheCtx, cancel := context.WithTimeout(ctx, common.CacheTimeout)

					utils.LavaFormatDebug("Cache lookup configuration",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("reqBlock", reqBlock),
						utils.LogAttr("requestedBlockForCache", requestedBlockForCache),
						utils.LogAttr("seenBlock", localRelayData.SeenBlock),
						utils.LogAttr("lookupFinalized", lookupFinalized),
					)

					cacheReply, cacheError = rpcss.cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
						RequestHash:           hashKey,
						RequestedBlock:        requestedBlockForCache,
						ChainId:               chainId,
						BlockHash:             nil,
						Finalized:             lookupFinalized,
						SharedStateId:         sharedStateId,
						SeenBlock:             localRelayData.SeenBlock,
						BlocksHashesToHeights: rpcss.newBlocksHashesToHeightsSliceFromRequestedBlockHashes(protocolMessage.GetRequestedBlocksHashes()),
					}) // caching in the consumer doesn't care about hashes, and we don't have data on finalization yet
					cancel()

					// Generate the actual cache key that will be used for lookup
					actualLookupCacheKey := make([]byte, len(hashKey))
					copy(actualLookupCacheKey, hashKey)
					actualLookupCacheKey = binary.LittleEndian.AppendUint64(actualLookupCacheKey, uint64(requestedBlockForCache))

					utils.LavaFormatDebug("Cache lookup result",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("hashKeyHex", fmt.Sprintf("%x", hashKey)),
						utils.LogAttr("actualLookupCacheKeyHex", fmt.Sprintf("%x", actualLookupCacheKey)),
						utils.LogAttr("reqBlock", reqBlock),
						utils.LogAttr("requestedBlockForCache", requestedBlockForCache),
						utils.LogAttr("seenBlock", localRelayData.SeenBlock),
						utils.LogAttr("cacheError", cacheError),
						utils.LogAttr("replyFound", cacheReply != nil && cacheReply.GetReply() != nil),
					)
					reply := cacheReply.GetReply()

					// read seen block from cache even if we had a miss we still want to get the seen block so we can use it to get the right provider.
					cacheSeenBlock := cacheReply.GetSeenBlock()
					// check if the cache seen block is greater than my local seen block, this means the user requested this
					// request spoke with another consumer instance and use that block for inter consumer consistency.
					if rpcss.sharedState && cacheSeenBlock > localRelayData.SeenBlock {
						utils.LavaFormatDebug("shared state seen block is newer", utils.LogAttr("cache_seen_block", cacheSeenBlock), utils.LogAttr("local_seen_block", localRelayData.SeenBlock), utils.LogAttr("GUID", ctx))
						localRelayData.SeenBlock = cacheSeenBlock
						// setting the fetched seen block from the cache server to our local cache as well.
						rpcss.smartRouterConsistency.SetSeenBlock(cacheSeenBlock, userData)
					}

					// handle cache reply
					if cacheError == nil && reply != nil {
						// Info was fetched from cache, so we don't need to change the state
						// so we can return here, no need to update anything and calculate as this info was fetched from the cache
						reply.Data = outputFormatter(reply.Data)

						// If this is a cached error response with placeholder GUID, replace it with current request GUID
						replyDataStr := string(reply.Data)
						if strings.Contains(replyDataStr, `"Error_GUID":"CACHED_ERROR"`) {
							guid, guidOk := utils.GetUniqueIdentifier(ctx)
							if guidOk {
								guidStr := strconv.FormatUint(guid, 10)
								// Replace the placeholder GUID with the actual request GUID
								replyDataStr = strings.Replace(replyDataStr, `"Error_GUID":"CACHED_ERROR"`, `"Error_GUID":"`+guidStr+`"`, 1)
								reply.Data = []byte(replyDataStr)
							}
						}

						relayResult := common.RelayResult{
							Reply: reply,
							Request: &pairingtypes.RelayRequest{
								RelayData: localRelayData,
							},
							Finalized:    false, // set false to skip data reliability
							StatusCode:   200,
							ProviderInfo: common.ProviderInfo{ProviderAddress: ""},
						}
						if reply.LatestBlock > 0 {
							rpcss.updateLatestBlockHeight(uint64(reply.LatestBlock), "")
						}
						relayProcessor.SetResponse(&relaycore.RelayResponse{
							RelayResult: relayResult,
							Err:         nil,
						})
						return nil
					}
					latestBlockHashRequested, earliestBlockHashRequested = rpcss.getEarliestBlockHashRequestedFromCacheReply(cacheReply)
					utils.LavaFormatTrace("[Archive Debug] Reading block hashes from cache", utils.LogAttr("latestBlockHashRequested", latestBlockHashRequested), utils.LogAttr("earliestBlockHashRequested", earliestBlockHashRequested), utils.LogAttr("GUID", ctx))
					// cache failed, move on to regular relay
					if performance.NotConnectedError.Is(cacheError) {
						utils.LavaFormatDebug("cache not connected", utils.LogAttr("error", cacheError), utils.LogAttr("GUID", ctx))
					}
				}
			} else {
				utils.LavaFormatDebug("skipping cache due to requested block being NOT_APPLICABLE",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("apiName", protocolMessage.GetApi().Name),
					utils.LogAttr("reqBlock", reqBlock),
				)
			}
		}
	}

	addon := chainlib.GetAddon(protocolMessage)
	reqBlock = rpcss.resolveRequestedBlock(reqBlock, localRelayData.SeenBlock, latestBlockHashRequested, protocolMessage)
	// check whether we need a new protocol message with the new earliest block hash requested
	protocolMessage = rpcss.updateProtocolMessageIfNeededWithNewEarliestData(ctx, relayState, protocolMessage, earliestBlockHashRequested, addon)

	// Smart router doesn't track epochs, use fixed value
	virtualEpoch := uint64(0)
	extensions := protocolMessage.GetExtensions()
	utils.LavaFormatTrace("[Archive Debug] Extensions to send", utils.LogAttr("extensions", extensions), utils.LogAttr("GUID", ctx))

	// Debug: Check if the protocol message has the archive extension in its internal state
	utils.LavaFormatTrace("[Archive Debug] RelayPrivateData extensions", utils.LogAttr("relayPrivateDataExtensions", localRelayData.Extensions), utils.LogAttr("GUID", ctx))
	usedProviders := relayProcessor.GetUsedProviders()
	directiveHeaders := protocolMessage.GetDirectiveHeaders()

	// stickines id for future use
	stickiness, ok := directiveHeaders[common.STICKINESS_HEADER_NAME]
	if ok {
		utils.LavaFormatTrace("found stickiness header", utils.LogAttr("id", stickiness), utils.LogAttr("GUID", ctx))
	}

	sessions, err := rpcss.sessionManager.GetSessions(ctx, numOfProviders, chainlib.GetComputeUnits(protocolMessage), usedProviders, reqBlock, addon, extensions, chainlib.GetStateful(protocolMessage), virtualEpoch, stickiness)
	if err != nil {
		if lavasession.PairingListEmptyError.Is(err) {
			if addon != "" {
				return utils.LavaFormatError("No Providers For Addon", err, utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("userIp", userData.ConsumerIp), utils.LogAttr("GUID", ctx))
			} else if len(extensions) > 0 && relayProcessor.GetAllowSessionDegradation() { // if we have no providers for that extension, use a regular provider, otherwise return the extension results
				sessions, err = rpcss.sessionManager.GetSessions(ctx, numOfProviders, chainlib.GetComputeUnits(protocolMessage), usedProviders, reqBlock, addon, []*spectypes.Extension{}, chainlib.GetStateful(protocolMessage), virtualEpoch, stickiness)
				if err != nil {
					return err
				}
				localRelayData.Extensions = []string{} // reset request data extensions in our local copy
				extensions = []*spectypes.Extension{}  // reset extensions too so we wont hit SetDisallowDegradation
			} else {
				return err
			}
		} else {
			return err
		}
	}

	// For stateful APIs, capture all providers that we're sending the relay to
	// This must be done immediately after GetSessions while all providers are still in the sessions map
	if chainlib.GetStateful(protocolMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		statefulRelayTargets := make([]string, 0, len(sessions))
		for providerPublicAddress := range sessions {
			statefulRelayTargets = append(statefulRelayTargets, providerPublicAddress)
		}
		relayProcessor.SetStatefulRelayTargets(statefulRelayTargets)
	}

	// making sure next get sessions wont use regular providers
	if len(extensions) > 0 {
		relayProcessor.SetDisallowDegradation()
	}

	if rpcss.debugRelays {
		routerKey := lavasession.NewRouterKeyFromExtensions(extensions)
		utils.LavaFormatDebug("[Before Send] returned the following sessions",
			utils.LogAttr("sessions", sessions),
			utils.LogAttr("usedProviders.GetUnwantedProvidersToSend", usedProviders.GetUnwantedProvidersToSend(routerKey)),
			utils.LogAttr("usedProviders.GetErroredProviders", usedProviders.GetErroredProviders(routerKey)),
			utils.LogAttr("addons", addon),
			utils.LogAttr("extensions", extensions),
			utils.LogAttr("AllowSessionDegradation", relayProcessor.GetAllowSessionDegradation()),
			utils.LogAttr("GUID", ctx),
		)
	}

	// Smart router supports direct RPC sessions only.
	if len(sessions) == 0 {
		return utils.LavaFormatError("no sessions available for direct RPC", nil, utils.LogAttr("GUID", ctx))
	}
	for providerPublicAddress, sessionInfo := range sessions {
		if sessionInfo == nil || sessionInfo.Session == nil || !sessionInfo.Session.IsDirectRPC() {
			return utils.LavaFormatError("rpcsmartrouter only supports direct RPC sessions", nil,
				utils.LogAttr("provider", providerPublicAddress),
				utils.LogAttr("GUID", ctx),
			)
		}
	}

	utils.LavaFormatDebug("routing to direct RPC flow (direct-only)",
		utils.LogAttr("num_sessions", len(sessions)),
		utils.LogAttr("GUID", ctx),
	)
	return rpcss.sendRelayToDirectEndpoints(ctx, sessions, protocolMessage, relayProcessor, analytics)
}

// relayInnerDirect handles relay requests using direct RPC connections (smart router mode)
func (rpcss *RPCSmartRouterServer) relayInnerDirect(
	ctx context.Context,
	singleConsumerSession *lavasession.SingleConsumerSession,
	relayResult *common.RelayResult,
	relayTimeout time.Duration,
	chainMessage chainlib.ChainMessage,
	analytics *metrics.RelayMetrics,
) (relayLatency time.Duration, err error, needsBackoff bool) {
	// Get direct connection from session
	directConnection, ok := singleConsumerSession.GetDirectConnection()
	if !ok {
		return 0, fmt.Errorf("session does not have direct RPC connection"), false
	}

	if rpcss.debugRelays {
		utils.LavaFormatDebug("Sending direct RPC relay",
			utils.LogAttr("timeout", relayTimeout),
			utils.LogAttr("method", chainMessage.GetApi().Name),
			utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
			utils.LogAttr("protocol", directConnection.GetProtocol()),
			utils.LogAttr("GUID", ctx),
		)
	}

	// Check for gRPC streaming method - currently not supported in Direct RPC mode
	// TODO: Full streaming support requires ChainListener changes to maintain client connections
	// and route repliesChan messages back to the client. For now, we refuse streaming RPCs
	// to avoid resource leaks (upstream subscriptions left running without consumers).
	if rpcss.grpcSubscriptionManager != nil && directConnection.GetProtocol() == "grpc" {
		methodPath := chainMessage.GetApi().Name
		isStreaming, _, streamErr := rpcss.grpcSubscriptionManager.IsStreamingMethod(ctx, methodPath)
		if streamErr == nil && isStreaming {
			utils.LavaFormatWarning("gRPC streaming methods not yet supported in Direct RPC mode",
				nil,
				utils.LogAttr("method", methodPath),
				utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
			)
			return 0, fmt.Errorf("gRPC streaming method %q not supported in Direct RPC mode; use provider-based relay for streaming", methodPath), false
		}
	}

	// Create direct RPC relay sender
	// Use provider name (configured name) instead of raw URL to avoid leaking API keys
	endpointName := singleConsumerSession.Parent.PublicLavaAddress
	directSender := &DirectRPCRelaySender{
		directConnection: directConnection,
		endpointName:     endpointName,
	}

	// Add metric for processing latency (compatible with existing metrics)
	rpcss.rpcSmartRouterLogs.AddMetricForProcessingLatencyBeforeProvider(
		analytics,
		rpcss.listenEndpoint.ChainID,
		rpcss.listenEndpoint.ApiInterface,
	)

	// Send relay directly to RPC endpoint
	startTime := time.Now()
	result, err := directSender.SendDirectRelay(ctx, chainMessage, relayTimeout)
	relayLatency = time.Since(startTime)

	// Get endpoint for health tracking (use stored reference, not string lookup)
	var targetEndpoint *lavasession.Endpoint
	if drsc, ok := singleConsumerSession.Connection.(*lavasession.DirectRPCSessionConnection); ok {
		targetEndpoint = drsc.Endpoint // Robust: use stored reference
	}

	if err != nil {
		utils.LavaFormatDebug("direct RPC relay failed",
			utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
			utils.LogAttr("error", err.Error()),
			utils.LogAttr("latency", relayLatency),
			utils.LogAttr("GUID", ctx),
		)

		// Classify error and decide on health tracking
		shouldMarkUnhealthy := false
		needsBackoff = false

		// Check if this is an HTTP status error
		if httpErr, ok := err.(*lavasession.HTTPStatusError); ok {
			statusCode := httpErr.StatusCode

			switch {
			case statusCode >= 500:
				// 5xx errors indicate server/node issues - mark unhealthy and backoff
				shouldMarkUnhealthy = true
				needsBackoff = true
				utils.LavaFormatDebug("endpoint returned server error",
					utils.LogAttr("status", statusCode),
					utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
				)
			case statusCode == 429:
				// Rate limit - backoff but DON'T mark unhealthy (endpoint is healthy, just busy)
				needsBackoff = true
				utils.LavaFormatDebug("endpoint rate limited",
					utils.LogAttr("status", statusCode),
					utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
				)
			case statusCode >= 400:
				// 4xx errors are client errors - don't mark unhealthy, don't backoff
				utils.LavaFormatDebug("client error",
					utils.LogAttr("status", statusCode),
					utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
				)
			}
		} else {
			// Non-HTTP errors (timeout, connection refused, network errors)
			// These indicate endpoint/network issues - mark unhealthy and backoff
			shouldMarkUnhealthy = true
			needsBackoff = true
		}

		// Apply health tracking based on error classification
		if shouldMarkUnhealthy && targetEndpoint != nil {
			targetEndpoint.MarkUnhealthy()
		}

		return relayLatency, err, needsBackoff
	}

	// Check status code even when err == nil (for REST 5xx/429)
	statusCode := result.StatusCode
	if statusCode >= 500 || statusCode == 429 {
		// REST returned 5xx or 429 (no transport error, but node issue)
		shouldMarkUnhealthy := (statusCode >= 500) // Mark unhealthy for 5xx, not 429
		needsBackoff = true                        // Both should backoff/retry

		if shouldMarkUnhealthy && targetEndpoint != nil {
			targetEndpoint.MarkUnhealthy()
			utils.LavaFormatDebug("endpoint returned error status",
				utils.LogAttr("status", statusCode),
				utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
			)
		} else if statusCode == 429 {
			utils.LavaFormatDebug("endpoint rate limited",
				utils.LogAttr("status", statusCode),
				utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
			)
		}

		// Return error to trigger backoff (but preserve result for client)
		return relayLatency, fmt.Errorf("HTTP %d", statusCode), needsBackoff
	}

	// Success - reset endpoint health
	if targetEndpoint != nil && targetEndpoint.ConnectionRefusals > 0 {
		targetEndpoint.ResetHealth()
	}

	// Update relayResult with the response
	relayResult.Reply = result.Reply
	relayResult.Finalized = result.Finalized
	relayResult.StatusCode = result.StatusCode
	relayResult.IsNodeError = result.IsNodeError
	relayResult.ProviderInfo = result.ProviderInfo
	if relayResult.Reply != nil {
		relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, pairingtypes.Metadata{
			Name:  common.LAVA_RELAY_PROTOCOL_HEADER_NAME,
			Value: string(directConnection.GetProtocol()),
		})
	}

	// Update analytics
	if analytics != nil {
		analytics.Success = true
	}

	utils.LavaFormatTrace("direct RPC relay succeeded",
		utils.LogAttr("endpoint", singleConsumerSession.Parent.PublicLavaAddress),
		utils.LogAttr("latency", relayLatency),
		utils.LogAttr("status_code", result.StatusCode),
		utils.LogAttr("response_size", len(result.Reply.Data)),
		utils.LogAttr("GUID", ctx),
	)

	return relayLatency, nil, false
}


func (rpcss *RPCSmartRouterServer) getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration) {
	_, averageBlockTime, _, _ := rpcss.chainParser.ChainBlockStats()
	relayTimeout = chainlib.GetRelayTimeout(chainMessage, averageBlockTime)
	processingTimeout = common.GetTimeoutForProcessing(relayTimeout, chainlib.GetTimeoutInfo(chainMessage))
	return processingTimeout, relayTimeout
}

func (rpcss *RPCSmartRouterServer) LavaDirectiveHeaders(metadata []pairingtypes.Metadata) ([]pairingtypes.Metadata, map[string]string) {
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

func (rpcss *RPCSmartRouterServer) getExtensionsFromDirectiveHeaders(directiveHeaders map[string]string) extensionslib.ExtensionInfo {
	extensionsStr, ok := directiveHeaders[common.EXTENSION_OVERRIDE_HEADER_NAME]
	if ok {
		utils.LavaFormatTrace("[Archive Debug] Found extension override header", utils.LogAttr("extensionsStr", extensionsStr))
		extensions := strings.Split(extensionsStr, ",")
		_, extensions, _ = rpcss.chainParser.SeparateAddonsExtensions(context.Background(), extensions)
		utils.LavaFormatTrace("[Archive Debug] Processed extensions", utils.LogAttr("extensions", extensions))
		if len(extensions) == 1 && extensions[0] == "none" {
			// none eliminates existing extensions
			return extensionslib.ExtensionInfo{LatestBlock: rpcss.getLatestBlock(), ExtensionOverride: []string{}}
		} else if len(extensions) > 0 {
			// All extensions from headers use AdditionalExtensions (consistent behavior)
			utils.LavaFormatTrace("[Archive Debug] Using AdditionalExtensions for all header extensions", utils.LogAttr("extensions", extensions))
			return extensionslib.ExtensionInfo{LatestBlock: rpcss.getLatestBlock(), AdditionalExtensions: extensions}
		}
	}
	utils.LavaFormatTrace("[Archive Debug] No extension override header found")
	return extensionslib.ExtensionInfo{LatestBlock: rpcss.getLatestBlock()}
}

func (rpcss *RPCSmartRouterServer) HandleDirectiveHeadersForMessage(chainMessage chainlib.ChainMessage, directiveHeaders map[string]string) {
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

// Iterating over metadataHeaders adding each trailer that fits the header if found to relayResult.Relay.Metadata
func (rpcss *RPCSmartRouterServer) getMetadataFromRelayTrailer(metadataHeaders []string, relayResult *common.RelayResult) {
	for _, metadataHeader := range metadataHeaders {
		trailerValue := relayResult.ProviderTrailer.Get(metadataHeader)
		if len(trailerValue) > 0 {
			extensionMD := pairingtypes.Metadata{
				Name:  metadataHeader,
				Value: trailerValue[0],
			}
			relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, extensionMD)
		}
	}
}

// RelayProcessorForHeaders interface for methods used by appendHeadersToRelayResult
type RelayProcessorForHeaders interface {
	GetQuorumParams() common.QuorumParams
	GetResultsData() ([]common.RelayResult, []common.RelayResult, []relaycore.RelayError)
	GetStatefulRelayTargets() []string
	GetUsedProviders() *lavasession.UsedProviders
	NodeErrors() (ret []common.RelayResult)
}

func (rpcss *RPCSmartRouterServer) appendHeadersToRelayResult(ctx context.Context, relayResult *common.RelayResult, protocolErrors uint64, relayProcessor RelayProcessorForHeaders, protocolMessage chainlib.ProtocolMessage, apiName string) {
	if relayResult == nil {
		return
	}

	metadataReply := []pairingtypes.Metadata{}

	// Check if quorum feature is enabled
	quorumParams := relayProcessor.GetQuorumParams()

	if quorumParams.Enabled() {
		// For quorum mode: show all participating providers instead of single provider
		successResults, nodeErrors, _ := relayProcessor.GetResultsData()

		allProvidersMap := make(map[string]bool)

		// Add successful providers
		for _, result := range successResults {
			if result.ProviderInfo.ProviderAddress != "" {
				allProvidersMap[result.ProviderInfo.ProviderAddress] = true
			}
		}

		// Add providers that had node errors (they still participated)
		for _, result := range nodeErrors {
			if result.ProviderInfo.ProviderAddress != "" {
				allProvidersMap[result.ProviderInfo.ProviderAddress] = true
			}
		}

		// Convert to slice
		allProvidersList := make([]string, 0, len(allProvidersMap))
		for provider := range allProvidersMap {
			allProvidersList = append(allProvidersList, provider)
		}

		if len(allProvidersList) > 0 {
			allProvidersString := fmt.Sprintf("%v", allProvidersList)
			metadataReply = append(metadataReply, pairingtypes.Metadata{
				Name:  common.QUORUM_ALL_PROVIDERS_HEADER_NAME,
				Value: allProvidersString,
			})
		}
	} else {
		// For non-quorum mode: keep existing single provider behavior
		providerAddress := relayResult.GetProvider()
		if providerAddress == "" {
			providerAddress = "Cached"
		}
		metadataReply = append(metadataReply, pairingtypes.Metadata{
			Name:  common.PROVIDER_ADDRESS_HEADER_NAME,
			Value: providerAddress,
		})

		// add the relay retried count (including both node errors and protocol errors)
		successResults, nodeErrorResults, protocolErrorResults := relayProcessor.GetResultsData()
		totalRetries := protocolErrors + uint64(len(nodeErrorResults))
		if totalRetries > 0 {
			metadataReply = append(metadataReply, pairingtypes.Metadata{
				Name:  common.RETRY_COUNT_HEADER_NAME,
				Value: strconv.FormatUint(totalRetries, 10),
			})

			// When there are retries, show all attempted providers (similar to REST behavior)
			allProvidersMap := make(map[string]bool)

			// Add the current provider (might be from successful result or last error)
			if providerAddress != "Cached" && providerAddress != "" {
				allProvidersMap[providerAddress] = true
			}

			// Add providers from node errors
			for _, result := range nodeErrorResults {
				if result.ProviderInfo.ProviderAddress != "" {
					allProvidersMap[result.ProviderInfo.ProviderAddress] = true
				}
			}

			// Add providers from protocol errors
			for _, result := range protocolErrorResults {
				if result.ProviderInfo.ProviderAddress != "" {
					allProvidersMap[result.ProviderInfo.ProviderAddress] = true
				}
			}

			// Add providers from successful results (in case of partial success)
			for _, result := range successResults {
				if result.ProviderInfo.ProviderAddress != "" {
					allProvidersMap[result.ProviderInfo.ProviderAddress] = true
				}
			}

			// Convert to slice and update provider address header with all participating providers
			if len(allProvidersMap) > 0 {
				allProvidersList := make([]string, 0, len(allProvidersMap))
				for provider := range allProvidersMap {
					allProvidersList = append(allProvidersList, provider)
				}
				allProvidersString := strings.Join(allProvidersList, ",")

				// Update the existing PROVIDER_ADDRESS_HEADER_NAME with all providers
				for i, metadata := range metadataReply {
					if metadata.Name == common.PROVIDER_ADDRESS_HEADER_NAME {
						metadataReply[i].Value = allProvidersString
						break
					}
				}
			}
		}
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

	// add stateful API (hanging, transactions)
	if protocolMessage.GetApi().Category.Stateful == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.STATEFUL_API_HEADER,
				Value: "true",
			})

		// add all providers that received the stateful relay
		statefulRelayTargets := relayProcessor.GetStatefulRelayTargets()
		if len(statefulRelayTargets) > 0 {
			allProvidersString := fmt.Sprintf("%v", statefulRelayTargets)
			metadataReply = append(metadataReply,
				pairingtypes.Metadata{
					Name:  common.STATEFUL_ALL_PROVIDERS_HEADER_NAME,
					Value: allProvidersString,
				})
		}
	}

	// add user requested API
	metadataReply = append(metadataReply,
		pairingtypes.Metadata{
			Name:  common.USER_REQUEST_TYPE,
			Value: apiName,
		})

	// add is node error flag
	if relayResult.IsNodeError {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.LAVA_IDENTIFIED_NODE_ERROR_HEADER,
				Value: "true",
			})
	}

	// fetch trailer information from the provider by using the provider trailer field.
	rpcss.getMetadataFromRelayTrailer(chainlib.TrailersToAddToHeaderResponse, relayResult)

	directiveHeaders := protocolMessage.GetDirectiveHeaders()
	_, debugRelays := directiveHeaders[common.LAVA_DEBUG_RELAY]
	if debugRelays {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.REQUESTED_BLOCK_HEADER_NAME,
				Value: strconv.FormatInt(protocolMessage.RelayPrivateData().GetRequestBlock(), 10),
			})

		routerKey := lavasession.NewRouterKeyFromExtensions(protocolMessage.GetExtensions())
		erroredProviders := relayProcessor.GetUsedProviders().GetErroredProviders(routerKey)
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

		nodeErrors := relayProcessor.NodeErrors()
		if len(nodeErrors) > 0 {
			nodeErrorHeaderString := ""
			for _, nodeError := range nodeErrors {
				nodeErrorHeaderString += fmt.Sprintf("%s: %s,", nodeError.GetProvider(), string(nodeError.Reply.Data))
			}
			relayResult.Reply.Metadata = append(relayResult.Reply.Metadata,
				pairingtypes.Metadata{
					Name:  common.NODE_ERRORS_PROVIDERS_HEADER_NAME,
					Value: nodeErrorHeaderString,
				})
		}

		if relayResult.Request != nil && relayResult.Request.RelaySession != nil {
			currentReportedProviders := rpcss.sessionManager.GetReportedProviders(uint64(relayResult.Request.RelaySession.Epoch))
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

		version := pairingtypes.Metadata{
			Name:  common.LAVAP_VERSION_HEADER_NAME,
			Value: upgrade.GetCurrentVersion().ConsumerVersion,
		}
		relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, version)
	}

	relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, metadataReply...)
}

func (rpcss *RPCSmartRouterServer) IsHealthy() bool {
	return rpcss.relaysMonitor.IsHealthy()
}

func (rpcss *RPCSmartRouterServer) IsInitialized() bool {
	if rpcss == nil {
		return false
	}

	return rpcss.initialized.Load()
}

func (rpcss *RPCSmartRouterServer) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guid := utils.GenerateUniqueIdentifier()
	ctx = utils.WithUniqueIdentifier(ctx, guid)
	url, data, connectionType, metadata, err := rpcss.chainParser.ExtractDataFromRequest(req)
	if err != nil {
		return nil, err
	}
	relayResult, err := rpcss.SendRelay(ctx, url, data, connectionType, "", "", nil, metadata)
	if err != nil {
		return nil, err
	}
	resp, err := rpcss.chainParser.SetResponseFromRelayResult(relayResult)
	rpcss.rpcSmartRouterLogs.SetLoLResponse(err == nil)
	return resp, err
}

func (rpcss *RPCSmartRouterServer) updateProtocolMessageIfNeededWithNewEarliestData(
	ctx context.Context,
	relayState *relaycore.RelayState,
	protocolMessage chainlib.ProtocolMessage,
	earliestBlockHashRequested int64,
	addon string,
) chainlib.ProtocolMessage {
	if !relayState.GetIsEarliestUsed() && earliestBlockHashRequested != spectypes.NOT_APPLICABLE {
		// We got a earliest block data from cache, we need to create a new protocol message with the new earliest block hash parsed
		// and update the extension rules with the new earliest block data as it might be archive.
		// Setting earliest used to attempt this only once.
		relayState.SetIsEarliestUsed()
		relayRequestData := protocolMessage.RelayPrivateData()
		userData := protocolMessage.GetUserData()
		newProtocolMessage, err := rpcss.ParseRelay(ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, nil)
		if err != nil {
			utils.LavaFormatError("Failed copying protocol message in sendRelayToProvider", err)
			return protocolMessage
		}

		extensionAdded := newProtocolMessage.UpdateEarliestAndValidateExtensionRules(rpcss.chainParser.ExtensionsParser(), earliestBlockHashRequested, addon, relayRequestData.SeenBlock)
		if extensionAdded && relayState.CheckIsArchive(newProtocolMessage.RelayPrivateData()) {
			relayState.SetIsArchive(true)
		}
		relayState.SetProtocolMessage(newProtocolMessage)
		return newProtocolMessage
	}
	return protocolMessage
}
