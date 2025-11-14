package rpcsmartrouter

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/finalizationverification"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/performance"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/protocol/statetracker"
	"github.com/lavanet/lava/v5/protocol/upgrade"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/protocopy"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// maximum number of retries to send due to the ticker, if we didn't get a response after 10 different attempts then just wait.
	MaximumNumberOfTickerRelayRetries        = 10
	MaxRelayRetries                          = 6
	SendRelayAttempts                        = 3
	numberOfTimesToCheckCurrentlyUsedIsEmpty = 3
	initRelaysDappId                         = "-init-"
	initRelaysSmartRouterIp                  = ""
)

// NoResponseTimeout is imported from protocolerrors to avoid duplicate error code registration
var NoResponseTimeout = protocolerrors.NoResponseTimeout

type CancelableContextHolder struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

// implements Relay Sender interfaced and uses an ChainListener to get it called
type RPCSmartRouterServer struct {
	smartRouterProcessGuid         string
	chainParser                    chainlib.ChainParser
	sessionManager                 *lavasession.ConsumerSessionManager
	listenEndpoint                 *lavasession.RPCEndpoint
	rpcSmartRouterLogs             *metrics.RPCConsumerLogs
	cache                          *performance.Cache
	privKey                        *btcec.PrivateKey
	requiredResponses              int
	lavaChainID                    string
	SmartRouterAddress             sdk.AccAddress
	smartRouterConsistency         relaycore.Consistency
	sharedState                    bool // using the cache backend to sync the latest seen block
	relaysMonitor                  *metrics.RelaysMonitor
	reporter                       metrics.Reporter
	debugRelays                    bool
	connectedSubscriptionsContexts map[string]*CancelableContextHolder
	chainListener                  chainlib.ChainListener
	connectedSubscriptionsLock     sync.RWMutex
	relayRetriesManager            *lavaprotocol.RelayRetriesManager
	initialized                    atomic.Bool
}

func (rpcss *RPCSmartRouterServer) ServeRPCRequests(
	ctx context.Context,
	listenEndpoint *lavasession.RPCEndpoint,
	chainParser chainlib.ChainParser,
	sessionManager *lavasession.ConsumerSessionManager,
	requiredResponses int,
	privKey *btcec.PrivateKey,
	lavaChainID string,
	cache *performance.Cache,
	rpcSmartRouterLogs *metrics.RPCConsumerLogs,
	smartRouterAddress sdk.AccAddress,
	smartRouterConsistency relaycore.Consistency,
	relaysMonitor *metrics.RelaysMonitor,
	cmdFlags common.ConsumerCmdFlags,
	sharedState bool,
	refererData *chainlib.RefererData,
	reporter metrics.Reporter,
	wsSubscriptionManager *chainlib.ConsumerWSSubscriptionManager,
) (err error) {
	rpcss.sessionManager = sessionManager
	rpcss.listenEndpoint = listenEndpoint
	rpcss.cache = cache
	rpcss.requiredResponses = requiredResponses
	rpcss.lavaChainID = lavaChainID
	rpcss.rpcSmartRouterLogs = rpcSmartRouterLogs
	rpcss.privKey = privKey
	rpcss.chainParser = chainParser
	rpcss.SmartRouterAddress = smartRouterAddress
	rpcss.smartRouterConsistency = smartRouterConsistency
	rpcss.sharedState = sharedState
	rpcss.reporter = reporter
	rpcss.debugRelays = cmdFlags.DebugRelays
	rpcss.connectedSubscriptionsContexts = make(map[string]*CancelableContextHolder)
	rpcss.smartRouterProcessGuid = strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10)
	rpcss.relayRetriesManager = lavaprotocol.NewRelayRetriesManager()
	rpcss.chainListener, err = chainlib.NewChainListener(ctx, listenEndpoint, rpcss, rpcss, rpcSmartRouterLogs, chainParser, refererData, wsSubscriptionManager)
	if err != nil {
		return err
	}

	go rpcss.chainListener.Serve(ctx, cmdFlags)

	initialRelays := true
	rpcss.relaysMonitor = relaysMonitor

	// we trigger a latest block call to get some more information on our providers, using the relays monitor
	if cmdFlags.RelaysHealthEnableFlag {
		rpcss.relaysMonitor.SetRelaySender(func() (bool, error) {
			success, err := rpcss.sendCraftedRelaysWrapper(initialRelays)
			if success {
				initialRelays = false
			}
			return success, err
		})
		rpcss.relaysMonitor.Start(ctx)
	} else {
		rpcss.sendCraftedRelaysWrapper(true)
	}
	return nil
}

func (rpcss *RPCSmartRouterServer) SetConsistencySeenBlock(blockSeen int64, key string) {
	rpcss.smartRouterConsistency.SetSeenBlockFromKey(blockSeen, key)
}

func (rpcss *RPCSmartRouterServer) GetListeningAddress() string {
	return rpcss.chainListener.GetListeningAddress()
}

func (rpcss *RPCSmartRouterServer) sendCraftedRelaysWrapper(initialRelays bool) (bool, error) {
	if initialRelays {
		// Only start after everything is initialized - check consumer session manager
		rpcss.waitForPairing()
	}
	success, err := rpcss.sendCraftedRelays(MaxRelayRetries, initialRelays)
	if success {
		rpcss.initialized.Store(true)
	}
	return success, err
}

func (rpcss *RPCSmartRouterServer) waitForPairing() {
	reinitializedChan := make(chan bool)

	go func() {
		for {
			if rpcss.sessionManager.Initialized() {
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
		time.Sleep(2 * time.Millisecond)
	}

	return success, err
}

// sending a few latest blocks relays to providers in order to have some data on the providers when relays start arriving
func (rpcss *RPCSmartRouterServer) sendCraftedRelays(retries int, initialRelays bool) (success bool, err error) {
	utils.LavaFormatDebug("Sending crafted relays",
		utils.LogAttr("chainId", rpcss.listenEndpoint.ChainID),
		utils.LogAttr("apiInterface", rpcss.listenEndpoint.ApiInterface),
	)

	ctx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
	ok, relay, chainMessage, err := rpcss.craftRelay(ctx)
	if !ok {
		enabled, _ := rpcss.chainParser.DataReliabilityParams()
		// if DR is disabled it's okay to not have GET_BLOCKNUM
		if !enabled {
			return true, nil
		}
		return false, err
	}
	protocolMessage := chainlib.NewProtocolMessage(chainMessage, nil, relay, initRelaysDappId, initRelaysSmartRouterIp)
	return rpcss.sendRelayWithRetries(ctx, retries, initialRelays, protocolMessage)
}

func (rpcss *RPCSmartRouterServer) getLatestBlock() uint64 {
	// Smart router doesn't track expected block height
	// Return 0 to use default behavior
	return 0
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

	// Handle Data Reliability
	enabled, dataReliabilityThreshold := rpcss.chainParser.DataReliabilityParams()
	// check if data reliability is enabled and relay processor allows us to perform data reliability
	if enabled && !relayProcessor.GetSkipDataReliability() {
		// new context is needed for data reliability as some clients cancel the context they provide when the relay returns
		// as data reliability happens in a go routine it will continue while the response returns.
		guid, found := utils.GetUniqueIdentifier(ctx)
		dataReliabilityContext := context.Background()
		if found {
			dataReliabilityContext = utils.WithUniqueIdentifier(dataReliabilityContext, guid)
		}
		go rpcss.sendDataReliabilityRelayIfApplicable(dataReliabilityContext, protocolMessage, dataReliabilityThreshold, relayProcessor) // runs asynchronously
	}

	returnedResult, err := relayProcessor.ProcessingResult()
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

	cacheCtx, cancel := context.WithTimeout(context.Background(), common.DataReliabilityTimeoutIncrease)
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
	for task := range relayTaskChannel {
		if task.IsDone() {
			return relayProcessor, task.Err
		}
		utils.LavaFormatTrace("[RPCSmartRouterServer] ProcessRelaySend - task", utils.LogAttr("GUID", ctx), utils.LogAttr("numOfProviders", task.NumOfProviders))
		err := rpcss.sendRelayToProvider(ctx, task.NumOfProviders, task.RelayState, relayProcessor, task.Analytics)
		relayProcessor.UpdateBatch(err)
	}

	// shouldn't happen.
	return relayProcessor, utils.LavaFormatError("ProcessRelaySend channel closed unexpectedly", nil)
}

func (rpcss *RPCSmartRouterServer) CreateDappKey(userData common.UserData) string {
	return rpcss.smartRouterConsistency.Key(userData)
}

func (rpcss *RPCSmartRouterServer) CancelSubscriptionContext(subscriptionKey string) {
	rpcss.connectedSubscriptionsLock.Lock()
	defer rpcss.connectedSubscriptionsLock.Unlock()

	ctxHolder, ok := rpcss.connectedSubscriptionsContexts[subscriptionKey]
	if ok {
		utils.LavaFormatTrace("cancelling subscription context", utils.LogAttr("subscriptionID", subscriptionKey))
		ctxHolder.CancelFunc()
		delete(rpcss.connectedSubscriptionsContexts, subscriptionKey)
	} else {
		utils.LavaFormatWarning("tried to cancel context for subscription ID that does not exist", nil, utils.LogAttr("subscriptionID", subscriptionKey))
	}
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

func (rpcss *RPCSmartRouterServer) updateBlocksHashesToHeightsIfNeeded(extensions []*spectypes.Extension, chainMessage chainlib.ChainMessage, blockHashesToHeights []*pairingtypes.BlockHashToHeight, latestBlock int64, finalized bool, relayState *relaycore.RelayState) ([]*pairingtypes.BlockHashToHeight, bool) {
	// This function will add the requested block hash with the height of the block that will force it to be archive on the following conditions:
	// 1. The current extension is archive.
	// 2. The user requested a single block hash.
	// 3. The archive extension rule is set

	// Adding to cache only if we upgraded to archive, meaning normal relay didn't work and archive did.
	// It is safe to assume in most cases this hash should be used with archive.
	// And if it is not, it will only increase archive load but wont result in user errors.
	// After the finalized cache duration it will be reset until next time.
	if relayState == nil {
		return blockHashesToHeights, finalized
	}
	if !relayState.GetIsUpgraded() {
		if relayState.GetIsEarliestUsed() && relayState.GetIsArchive() && chainMessage.GetUsedDefaultValue() {
			finalized = true
		}
		return blockHashesToHeights, finalized
	}

	isArchiveRelay := false
	var rule *spectypes.Rule
	for _, extension := range extensions {
		if extension.Name == extensionslib.ArchiveExtension {
			isArchiveRelay = true
			rule = extension.Rule
			break
		}
	}
	requestedBlocksHashes := chainMessage.GetRequestedBlocksHashes()
	isUserRequestedSingleBlocksHashes := len(requestedBlocksHashes) == 1

	if isArchiveRelay && isUserRequestedSingleBlocksHashes && rule != nil {
		ruleBlock := int64(rule.Block)
		if ruleBlock >= 0 {
			height := latestBlock - ruleBlock - 1
			if height < 0 {
				height = 0
			}
			blockHashesToHeights = append(blockHashesToHeights, &pairingtypes.BlockHashToHeight{
				Hash:   requestedBlocksHashes[0],
				Height: height,
			})
			// we can assume this result is finalized.
			finalized = true
		}
	}

	return blockHashesToHeights, finalized
}

func (rpcss *RPCSmartRouterServer) newBlocksHashesToHeightsSliceFromRequestedBlockHashes(requestedBlockHashes []string) []*pairingtypes.BlockHashToHeight {
	var blocksHashesToHeights []*pairingtypes.BlockHashToHeight
	for _, blockHash := range requestedBlockHashes {
		blocksHashesToHeights = append(blocksHashesToHeights, &pairingtypes.BlockHashToHeight{Hash: blockHash, Height: spectypes.NOT_APPLICABLE})
	}
	return blocksHashesToHeights
}

func (rpcss *RPCSmartRouterServer) newBlocksHashesToHeightsSliceFromFinalizationConsensus(finalizedBlockHashes map[int64]string) []*pairingtypes.BlockHashToHeight {
	var blocksHashesToHeights []*pairingtypes.BlockHashToHeight
	for height, blockHash := range finalizedBlockHashes {
		blocksHashesToHeights = append(blocksHashesToHeights, &pairingtypes.BlockHashToHeight{Hash: blockHash, Height: height})
	}
	return blocksHashesToHeights
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
	userData := protocolMessage.GetUserData()
	var sharedStateId string // defaults to "", if shared state is disabled then no shared state will be used.
	if rpcss.sharedState {
		sharedStateId = rpcss.smartRouterConsistency.Key(userData) // use same key as we use for consistency, (for better consistency :-D)
	}

	privKey := rpcss.privKey
	chainId, apiInterface := rpcss.GetChainIdAndApiInterface()
	lavaChainID := rpcss.lavaChainID

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
		if !protocolMessage.GetForceCacheRefresh() { // don't use cache if user specified
			// Allow cache lookup for all requests (including method-based caching for unsupported method errors)
			allowCacheLookup := true

			if allowCacheLookup {
				var cacheReply *pairingtypes.CacheRelayReply
				hashKey, outputFormatter, err := protocolMessage.HashCacheRequest(chainId)
				if err != nil {
					utils.LavaFormatError("sendRelayToProvider Failed getting Hash for cache request", err, utils.LogAttr("GUID", ctx))
				} else {
					utils.LavaFormatDebug("Cache lookup hash generated",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("hashKey", fmt.Sprintf("%x", hashKey)),
						utils.LogAttr("apiUrl", protocolMessage.RelayPrivateData().ApiUrl),
					)

					// Resolve the requested block for cache lookup
					// The cache server doesn't accept negative blocks
					requestedBlockForCache := reqBlock
					if reqBlock == spectypes.LATEST_BLOCK {
						// For LATEST_BLOCK, use seen block (will be resolved to actual block number)
						if protocolMessage.RelayPrivateData().SeenBlock != 0 {
							requestedBlockForCache = protocolMessage.RelayPrivateData().SeenBlock
						} else {
							requestedBlockForCache = 0 // Fallback to 0 if no seen block
						}
					} else if reqBlock == spectypes.NOT_APPLICABLE {
						requestedBlockForCache = 0 // NOT_APPLICABLE maps to 0
					}

					// Always use finalized=false for lookups
					// The cache will search both tempCache and finalizedCache, finding data in either
					lookupFinalized := false

					cacheCtx, cancel := context.WithTimeout(ctx, common.CacheTimeout)

					utils.LavaFormatDebug("Cache lookup configuration",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("reqBlock", reqBlock),
						utils.LogAttr("requestedBlockForCache", requestedBlockForCache),
						utils.LogAttr("seenBlock", protocolMessage.RelayPrivateData().SeenBlock),
						utils.LogAttr("lookupFinalized", lookupFinalized),
					)

					cacheReply, cacheError = rpcss.cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
						RequestHash:           hashKey,
						RequestedBlock:        requestedBlockForCache,
						ChainId:               chainId,
						BlockHash:             nil,
						Finalized:             lookupFinalized,
						SharedStateId:         sharedStateId,
						SeenBlock:             protocolMessage.RelayPrivateData().SeenBlock,
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
						utils.LogAttr("seenBlock", protocolMessage.RelayPrivateData().SeenBlock),
						utils.LogAttr("cacheError", cacheError),
						utils.LogAttr("replyFound", cacheReply != nil && cacheReply.GetReply() != nil),
					)
					reply := cacheReply.GetReply()

					// read seen block from cache even if we had a miss we still want to get the seen block so we can use it to get the right provider.
					cacheSeenBlock := cacheReply.GetSeenBlock()
					// check if the cache seen block is greater than my local seen block, this means the user requested this
					// request spoke with another consumer instance and use that block for inter consumer consistency.
					if rpcss.sharedState && cacheSeenBlock > protocolMessage.RelayPrivateData().SeenBlock {
						utils.LavaFormatDebug("shared state seen block is newer", utils.LogAttr("cache_seen_block", cacheSeenBlock), utils.LogAttr("local_seen_block", protocolMessage.RelayPrivateData().SeenBlock), utils.LogAttr("GUID", ctx))
						protocolMessage.RelayPrivateData().SeenBlock = cacheSeenBlock
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
								RelayData: protocolMessage.RelayPrivateData(),
							},
							Finalized:    false, // set false to skip data reliability
							StatusCode:   200,
							ProviderInfo: common.ProviderInfo{ProviderAddress: ""},
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
	reqBlock = rpcss.resolveRequestedBlock(reqBlock, protocolMessage.RelayPrivateData().SeenBlock, latestBlockHashRequested, protocolMessage)
	// check whether we need a new protocol message with the new earliest block hash requested
	protocolMessage = rpcss.updateProtocolMessageIfNeededWithNewEarliestData(ctx, relayState, protocolMessage, earliestBlockHashRequested, addon)

	// Smart router doesn't track epochs, use fixed value
	virtualEpoch := uint64(0)
	extensions := protocolMessage.GetExtensions()
	utils.LavaFormatTrace("[Archive Debug] Extensions to send", utils.LogAttr("extensions", extensions), utils.LogAttr("GUID", ctx))
	utils.LavaFormatTrace("[Archive Debug] ProtocolMessage details", utils.LogAttr("relayPrivateData", protocolMessage.RelayPrivateData()), utils.LogAttr("GUID", ctx))

	// Debug: Check if the protocol message has the archive extension in its internal state
	if relayPrivateData := protocolMessage.RelayPrivateData(); relayPrivateData != nil {
		utils.LavaFormatTrace("[Archive Debug] RelayPrivateData extensions", utils.LogAttr("relayPrivateDataExtensions", relayPrivateData.Extensions), utils.LogAttr("GUID", ctx))
	}
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
				relayProcessor.SetSkipDataReliability(true)                // disabling data reliability when disabling extensions.
				protocolMessage.RelayPrivateData().Extensions = []string{} // reset request data extensions
				extensions = []*spectypes.Extension{}                      // reset extensions too so we wont hit SetDisallowDegradation
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

	// Iterate over the sessions map
	for providerPublicAddress, sessionInfo := range sessions {
		// Launch a separate goroutine for each session
		go func(providerPublicAddress string, sessionInfo *lavasession.SessionInfo) {
			// add ticker launch metrics
			localRelayResult := &common.RelayResult{
				ProviderInfo: common.ProviderInfo{ProviderAddress: providerPublicAddress, ProviderStake: sessionInfo.StakeSize, ProviderReputationSummary: sessionInfo.QoSSummaryResult},
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
				relayProcessor.SetResponse(&relaycore.RelayResponse{
					RelayResult: *localRelayResult,
					Err:         errResponse,
				})

				// Close context
				goroutineCtxCancel()
			}()

			localRelayRequestData := *protocolMessage.RelayPrivateData()

			// Extract fields from the sessionInfo
			singleConsumerSession := sessionInfo.Session
			epoch := sessionInfo.Epoch
			reportedProviders := sessionInfo.ReportedProviders

			relayRequest, errResponse := lavaprotocol.ConstructRelayRequest(goroutineCtx, privKey, lavaChainID, chainId, &localRelayRequestData, providerPublicAddress, singleConsumerSession, int64(epoch), reportedProviders)
			if errResponse != nil {
				utils.LavaFormatError("Failed ConstructRelayRequest", errResponse, utils.LogAttr("Request data", localRelayRequestData), utils.LogAttr("GUID", ctx))
				return
			}
			localRelayResult.Request = relayRequest
			endpointClient := singleConsumerSession.EndpointConnection.Client

			// set relay sent metric
			go rpcss.rpcSmartRouterLogs.SetRelaySentToProviderMetric(providerPublicAddress, chainId, apiInterface)

			if chainlib.IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
				utils.LavaFormatTrace("inside sendRelayToProvider, relay is subscription", utils.LogAttr("requestData", localRelayRequestData.Data), utils.LogAttr("GUID", ctx))

				params, err := json.Marshal(protocolMessage.GetRPCMessage().GetParams())
				if err != nil {
					utils.LavaFormatError("could not marshal params", err, utils.LogAttr("GUID", ctx))
					return
				}

				hashedParams := rpcclient.CreateHashFromParams(params)
				cancellableCtx, cancelFunc := context.WithCancel(utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier()))

				ctxHolder := func() *CancelableContextHolder {
					rpcss.connectedSubscriptionsLock.Lock()
					defer rpcss.connectedSubscriptionsLock.Unlock()

					ctxHolder := &CancelableContextHolder{
						Ctx:        cancellableCtx,
						CancelFunc: cancelFunc,
					}
					rpcss.connectedSubscriptionsContexts[hashedParams] = ctxHolder
					return ctxHolder
				}()

				errResponse = rpcss.relaySubscriptionInner(ctxHolder.Ctx, hashedParams, endpointClient, singleConsumerSession, localRelayResult)
				if errResponse != nil {
					utils.LavaFormatError("Failed relaySubscriptionInner", errResponse,
						utils.LogAttr("Request", localRelayRequestData),
						utils.LogAttr("Request data", string(localRelayRequestData.Data)),
						utils.LogAttr("Provider", providerPublicAddress),
						utils.LogAttr("GUID", ctx),
					)
				}
				return
			}

			// unique per dappId and ip
			consumerToken := common.GetUniqueToken(userData)
			processingTimeout, expectedRelayTimeoutForQOS := rpcss.getProcessingTimeout(protocolMessage)
			deadline, ok := ctx.Deadline()
			if ok { // we have ctx deadline. we cant go past it.
				processingTimeout = time.Until(deadline)
				if processingTimeout <= 0 {
					// no need to send we are out of time
					utils.LavaFormatWarning("Creating context deadline for relay attempt ran out of time, processingTimeout <= 0 ", nil, utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("ApiUrl", localRelayRequestData.ApiUrl), utils.LogAttr("GUID", ctx))
					return
				}
				// to prevent absurdly short context timeout set the shortest timeout to be the expected latency for qos time.
				if processingTimeout < expectedRelayTimeoutForQOS {
					processingTimeout = expectedRelayTimeoutForQOS
				}
			}
			// send relay
			relayLatency, errResponse, backoff := rpcss.relayInner(goroutineCtx, singleConsumerSession, localRelayResult, processingTimeout, protocolMessage, consumerToken, analytics)
			if errResponse != nil {
				failRelaySession := func(origErr error, backoff_ bool) {
					backOffDuration := 0 * time.Second
					if backoff_ {
						backOffDuration = lavasession.BACKOFF_TIME_ON_FAILURE
					}
					time.Sleep(backOffDuration) // sleep before releasing this singleConsumerSession
					// relay failed need to fail the session advancement
					errReport := rpcss.sessionManager.OnSessionFailure(singleConsumerSession, origErr)
					if errReport != nil {
						utils.LavaFormatError("failed relay onSessionFailure errored", errReport, utils.Attribute{Key: "GUID", Value: goroutineCtx}, utils.Attribute{Key: "original error", Value: origErr.Error()})
					}
				}
				go failRelaySession(errResponse, backoff)
				go rpcss.rpcSmartRouterLogs.SetProtocolError(chainId, providerPublicAddress)
				return
			}

			// get here only if performed a regular relay successfully
			// Smart router doesn't track expected block height
			expectedBH := int64(math.MaxInt64)
			numOfProviders := 1 // Smart router always has static providers
			pairingAddressesLen := rpcss.sessionManager.GetAtomicPairingAddressesLength()
			latestBlock := localRelayResult.Reply.LatestBlock
			if expectedBH-latestBlock > 1000 {
				utils.LavaFormatWarning("identified block gap", nil,
					utils.Attribute{Key: "expectedBH", Value: expectedBH},
					utils.Attribute{Key: "latestServicedBlock", Value: latestBlock},
					utils.Attribute{Key: "session_id", Value: singleConsumerSession.SessionId},
					utils.Attribute{Key: "provider_address", Value: singleConsumerSession.Parent.PublicLavaAddress},
					utils.Attribute{Key: "providersCount", Value: pairingAddressesLen},
					utils.LogAttr("GUID", ctx),
				)
			}

			if rpcss.debugRelays {
				lastQoSReport := singleConsumerSession.QoSManager.GetLastQoSReport(epoch, singleConsumerSession.SessionId)
				if lastQoSReport != nil && lastQoSReport.Sync.BigInt() != nil &&
					lastQoSReport.Sync.LT(sdk.MustNewDecFromStr("0.9")) {
					utils.LavaFormatDebug("identified QoS mismatch",
						utils.Attribute{Key: "expectedBH", Value: expectedBH},
						utils.Attribute{Key: "latestServicedBlock", Value: latestBlock},
						utils.Attribute{Key: "session_id", Value: singleConsumerSession.SessionId},
						utils.Attribute{Key: "provider_address", Value: singleConsumerSession.Parent.PublicLavaAddress},
						utils.Attribute{Key: "providersCount", Value: pairingAddressesLen},
						utils.Attribute{Key: "singleConsumerSession.QoSInfo", Value: singleConsumerSession.QoSManager},
						utils.LogAttr("GUID", ctx),
					)
				}
			}

			errResponse = rpcss.sessionManager.OnSessionDone(singleConsumerSession, latestBlock, chainlib.GetComputeUnits(protocolMessage), relayLatency, singleConsumerSession.CalculateExpectedLatency(expectedRelayTimeoutForQOS), expectedBH, numOfProviders, pairingAddressesLen, protocolMessage.GetApi().Category.HangingApi, extensions) // session done successfully
			isNodeError, _ := protocolMessage.CheckResponseError(localRelayResult.Reply.Data, localRelayResult.StatusCode)
			localRelayResult.IsNodeError = isNodeError
			if rpcss.debugRelays {
				utils.LavaFormatDebug("Result Code", utils.LogAttr("isNodeError", isNodeError), utils.LogAttr("StatusCode", localRelayResult.StatusCode), utils.LogAttr("GUID", ctx))
			}
			if rpcss.cache.CacheActive() && rpcclient.ValidateStatusCodes(localRelayResult.StatusCode, true) == nil {
				// Check if this is an unsupported method error
				replyDataStr := string(localRelayResult.Reply.Data)
				isUnsupportedMethodError := chainlib.IsUnsupportedMethodErrorMessage(replyDataStr)

				// Determine if we should cache this response
				// - Always cache unsupported method errors (treat like regular API responses based on block)
				// - Only cache successful responses when quorum is disabled
				shouldCache := false
				if isUnsupportedMethodError {
					// Cache unsupported method errors like regular responses
					// This allows latest block errors to use tempCache and historical to use finalizedCache
					shouldCache = true
				} else if !quorumParams.Enabled() {
					shouldCache = !isNodeError // Cache successful responses only when quorum is disabled
				} else {
					// Quorum is enabled and this is not an unsupported method error
					utils.LavaFormatDebug("Skipping cache for successful response due to quorum validation",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("quorumEnabled", true),
						utils.LogAttr("isNodeError", isNodeError),
						utils.LogAttr("reason", "quorum requires fresh provider validation on each request"),
					)
				}

				if shouldCache {
					// copy reply data so if it changes it doesn't panic mid async send
					copyReply := &pairingtypes.RelayReply{}
					copyReplyErr := protocopy.DeepCopyProtoObject(localRelayResult.Reply, copyReply)
					// set cache in a non blocking call
					requestedBlock := localRelayResult.Request.RelayData.RequestBlock                             // get requested block before removing it from the data
					seenBlock := localRelayResult.Request.RelayData.SeenBlock                                     // get seen block before removing it from the data
					hashKey, _, hashErr := chainlib.HashCacheRequest(localRelayResult.Request.RelayData, chainId) // get the hash (this changes the data)
					finalizedBlockHashes := localRelayResult.Reply.FinalizedBlocksHashes

					go func() {
						// deal with copying error.
						if copyReplyErr != nil || hashErr != nil {
							utils.LavaFormatError("Failed copying relay private data sendRelayToProvider", nil,
								utils.LogAttr("copyReplyErr", copyReplyErr),
								utils.LogAttr("hashErr", hashErr),
								utils.LogAttr("GUID", ctx),
							)
							return
						}
						chainMessageRequestedBlock, _ := protocolMessage.RequestedBlock()
						if chainMessageRequestedBlock == spectypes.NOT_APPLICABLE {
							return
						}

						blockHashesToHeights := make([]*pairingtypes.BlockHashToHeight, 0)

						var finalizedBlockHashesObj map[int64]string
						err := json.Unmarshal(finalizedBlockHashes, &finalizedBlockHashesObj)
						if err != nil {
							utils.LavaFormatError("failed unmarshalling finalizedBlockHashes", err,
								utils.LogAttr("GUID", ctx),
								utils.LogAttr("finalizedBlockHashes", finalizedBlockHashes),
								utils.LogAttr("providerAddr", providerPublicAddress),
							)
						} else {
							blockHashesToHeights = rpcss.newBlocksHashesToHeightsSliceFromFinalizationConsensus(finalizedBlockHashesObj)
						}
						var finalized bool
						blockHashesToHeights, finalized = rpcss.updateBlocksHashesToHeightsIfNeeded(extensions, protocolMessage, blockHashesToHeights, latestBlock, localRelayResult.Finalized, relayState)
						utils.LavaFormatTrace("[Archive Debug] Adding HASH TO CACHE", utils.LogAttr("blockHashesToHeights", blockHashesToHeights), utils.LogAttr("GUID", ctx))

						new_ctx := context.Background()
						new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
						defer cancel()
						_, averageBlockTime, _, _ := rpcss.chainParser.ChainBlockStats()

						err2 := rpcss.cache.SetEntry(new_ctx, &pairingtypes.RelayCacheSet{
							RequestHash:           hashKey,
							ChainId:               chainId,
							RequestedBlock:        requestedBlock,
							SeenBlock:             seenBlock,
							BlockHash:             nil, // consumer cache doesn't care about block hashes
							Response:              copyReply,
							Finalized:             finalized,
							OptionalMetadata:      nil,
							SharedStateId:         sharedStateId,
							AverageBlockTime:      int64(averageBlockTime),
							IsNodeError:           isNodeError,
							BlocksHashesToHeights: blockHashesToHeights,
						})
						if err2 != nil {
							utils.LavaFormatWarning("error updating cache with new entry", err2, utils.LogAttr("GUID", ctx))
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

func (rpcss *RPCSmartRouterServer) relayInner(ctx context.Context, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult, relayTimeout time.Duration, chainMessage chainlib.ChainMessage, consumerToken string, analytics *metrics.RelayMetrics) (relayLatency time.Duration, err error, needsBackoff bool) {
	existingSessionLatestBlock := singleConsumerSession.LatestBlock // we read it now because singleConsumerSession is locked, and later it's not
	endpointClient := singleConsumerSession.EndpointConnection.Client
	providerPublicAddress := relayResult.ProviderInfo.ProviderAddress
	relayRequest := relayResult.Request
	if rpcss.debugRelays {
		utils.LavaFormatDebug("Sending relay", utils.LogAttr("timeout", relayTimeout), utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock), utils.LogAttr("GUID", ctx), utils.LogAttr("provider", relayRequest.RelaySession.Provider))
	}
	callRelay := func() (reply *pairingtypes.RelayReply, relayLatency time.Duration, err error, backoff bool) {
		connectCtx, connectCtxCancel := context.WithTimeout(ctx, relayTimeout)
		metadataAdd := metadata.New(map[string]string{
			common.IP_FORWARDING_HEADER_NAME:  consumerToken,
			common.LAVA_CONSUMER_PROCESS_GUID: rpcss.smartRouterProcessGuid,
			common.LAVA_LB_UNIQUE_ID_HEADER:   singleConsumerSession.EndpointConnection.GetLbUniqueId(),
		})

		utils.LavaFormatTrace("Sending relay to provider",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("lbUniqueId", singleConsumerSession.EndpointConnection.GetLbUniqueId()),
			utils.LogAttr("providerAddress", providerPublicAddress),
			utils.LogAttr("requestBlock", relayResult.Request.RelayData.RequestBlock),
			utils.LogAttr("seenBlock", relayResult.Request.RelayData.SeenBlock),
			utils.LogAttr("extensions", relayResult.Request.RelayData.Extensions),
		)
		connectCtx = metadata.NewOutgoingContext(connectCtx, metadataAdd)
		defer connectCtxCancel()

		// add consumer processing timestamp before provider metric and start measuring time after the provider replied
		rpcss.rpcSmartRouterLogs.AddMetricForProcessingLatencyBeforeProvider(analytics, rpcss.listenEndpoint.ChainID, rpcss.listenEndpoint.ApiInterface)

		if relayResult.ProviderTrailer == nil {
			// if the provider trailer is nil, we need to initialize it
			relayResult.ProviderTrailer = metadata.MD{}
		}

		relaySentTime := time.Now()
		reply, err = endpointClient.Relay(connectCtx, relayRequest, grpc.Trailer(&relayResult.ProviderTrailer))
		relayLatency = time.Since(relaySentTime)

		providerUniqueId := relayResult.ProviderTrailer.Get(chainlib.RpcProviderUniqueIdHeader)
		if len(providerUniqueId) > 0 {
			if len(providerUniqueId) > 1 {
				utils.LavaFormatInfo("Received more than one provider unique id in header, skipping",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("provider", relayRequest.RelaySession.Provider),
					utils.LogAttr("providerUniqueId", providerUniqueId),
				)
			} else if providerUniqueId[0] != "" { // Otherwise, the header is "" which is fine - it means the header is not set
				utils.LavaFormatTrace("Received provider unique id",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("provider", relayRequest.RelaySession.Provider),
					utils.LogAttr("providerUniqueId", providerUniqueId),
				)

				if !singleConsumerSession.VerifyProviderUniqueIdAndStoreIfFirstTime(providerUniqueId[0]) {
					return reply, 0, utils.LavaFormatError("provider unique id mismatch",
						lavasession.SessionOutOfSyncError,
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("sessionId", relayRequest.RelaySession.SessionId),
						utils.LogAttr("provider", relayRequest.RelaySession.Provider),
						utils.LogAttr("providedProviderUniqueId", providerUniqueId),
						utils.LogAttr("providerUniqueId", singleConsumerSession.GetProviderUniqueId()),
					), false
				} else {
					utils.LavaFormatTrace("Provider unique id match",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("sessionId", relayRequest.RelaySession.SessionId),
						utils.LogAttr("provider", relayRequest.RelaySession.Provider),
						utils.LogAttr("providerUniqueId", providerUniqueId),
					)
				}
			}
		}

		statuses := relayResult.ProviderTrailer.Get(common.StatusCodeMetadataKey)

		if len(statuses) > 0 {
			codeNum, errStatus := strconv.Atoi(statuses[0])
			if errStatus != nil {
				utils.LavaFormatWarning("failed converting status code", errStatus, utils.LogAttr("statuses", statuses), utils.LogAttr("GUID", ctx))
			}

			relayResult.StatusCode = codeNum
		}

		if rpcss.debugRelays {
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
				utils.LogAttr("ReputationReport", relayRequest.RelaySession.QosExcellenceReport),
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
			backoff := errors.Is(connectCtx.Err(), context.DeadlineExceeded)
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
				utils.LogAttr("GUID", ctx),
			)
		}
		return 0, err, backoff
	}

	utils.LavaFormatTrace("Relay succeeded",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("provider", relayRequest.RelaySession.Provider),
		utils.LogAttr("latestBlock", reply.LatestBlock),
		utils.LogAttr("latency", relayLatency),
		utils.LogAttr("method", chainMessage.GetApi().Name),
	)
	relayResult.Reply = reply

	// Update relay request requestedBlock to the provided one in case it was arbitrary
	lavaprotocol.UpdateRequestedBlock(relayRequest.RelayData, reply)

	_, _, blockDistanceForFinalizedData, blocksInFinalizationProof := rpcss.chainParser.ChainBlockStats()
	isFinalized := spectypes.IsFinalizedBlock(relayRequest.RelayData.RequestBlock, reply.LatestBlock, int64(blockDistanceForFinalizedData))
	if !rpcss.chainParser.ParseDirectiveEnabled() {
		isFinalized = false
	}

	filteredHeaders, _, ignoredHeaders := rpcss.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	reply.Metadata = filteredHeaders

	// check the signature on the reply
	if !singleConsumerSession.StaticProvider {
		err = lavaprotocol.VerifyRelayReply(ctx, reply, relayRequest, providerPublicAddress)
		if err != nil {
			return 0, err, false
		}
	}

	reply.Metadata = append(reply.Metadata, ignoredHeaders...)

	// TODO: response data sanity, check its under an expected format add that format to spec
	enabled, _ := rpcss.chainParser.DataReliabilityParams()
	if enabled && !singleConsumerSession.StaticProvider && rpcss.chainParser.ParseDirectiveEnabled() {
		// TODO: allow static providers to detect hash mismatches,
		// triggering conflict with them is impossible so we skip this for now, but this can be used to block malicious providers
		_, err := finalizationverification.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, rpcss.SmartRouterAddress, existingSessionLatestBlock, int64(blockDistanceForFinalizedData), int64(blocksInFinalizationProof))
		if err != nil {
			if sdkerrors.IsOf(err, protocolerrors.ProviderFinalizationDataAccountabilityError) {
				utils.LavaFormatInfo("provider finalization data accountability error", utils.LogAttr("provider", relayRequest.RelaySession.Provider), utils.LogAttr("GUID", ctx))
			}
			return 0, err, false
		}

		// Smart router doesn't track finalization consensus - no special handling needed
	}
	relayResult.Finalized = isFinalized
	return relayLatency, nil, false
}

func (rpcss *RPCSmartRouterServer) relaySubscriptionInner(ctx context.Context, hashedParams string, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult) (err error) {
	// add consumer guid to relay request.
	ctx = metadata.AppendToOutgoingContext(ctx,
		common.LAVA_LB_UNIQUE_ID_HEADER, singleConsumerSession.EndpointConnection.GetLbUniqueId(),
		common.LAVA_CONSUMER_PROCESS_GUID, rpcss.smartRouterProcessGuid,
	)

	replyServer, err := endpointClient.RelaySubscribe(ctx, relayResult.Request)
	if err != nil {
		errReport := rpcss.sessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("originalError", err.Error()),
			)
		}

		return err
	}

	reply, err := rpcss.getFirstSubscriptionReply(ctx, hashedParams, replyServer)
	if err != nil {
		errReport := rpcss.sessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("originalError", err.Error()),
			)
		}
		return err
	}

	utils.LavaFormatTrace("subscribe relay succeeded",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	relayResult.ReplyServer = replyServer
	relayResult.Reply = reply
	latestBlock := relayResult.Reply.LatestBlock
	err = rpcss.sessionManager.OnSessionDoneIncreaseCUOnly(singleConsumerSession, latestBlock)
	return err
}

func (rpcss *RPCSmartRouterServer) getFirstSubscriptionReply(ctx context.Context, hashedParams string, replyServer pairingtypes.Relayer_RelaySubscribeClient) (*pairingtypes.RelayReply, error) {
	var reply pairingtypes.RelayReply
	gotFirstReplyChanOrErr := make(chan struct{})

	// Cancel the context after SubscriptionFirstReplyTimeout duration, so we won't hang forever
	go func() {
		for {
			select {
			case <-time.After(common.SubscriptionFirstReplyTimeout):
				if reply.Data == nil {
					utils.LavaFormatError("Timeout exceeded when waiting for first reply message from subscription, cancelling the context with the provider", nil,
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					)
					rpcss.CancelSubscriptionContext(hashedParams) // Cancel the context with the provider, which will trigger the replyServer's context to be cancelled
				}
			case <-gotFirstReplyChanOrErr:
				return
			}
		}
	}()

	select {
	case <-replyServer.Context().Done(): // Make sure the reply server is open
		return nil, utils.LavaFormatError("reply server context canceled before first time read", nil,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
	default:
		err := replyServer.RecvMsg(&reply)
		gotFirstReplyChanOrErr <- struct{}{}
		if err != nil {
			return nil, utils.LavaFormatError("Could not read reply from reply server", err,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
		}
	}

	utils.LavaFormatTrace("successfully got first reply",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("reply", string(reply.Data)),
	)

	// Make sure we can parse the reply
	var replyJson rpcclient.JsonrpcMessage
	err := json.Unmarshal(reply.Data, &replyJson)
	if err != nil {
		return nil, utils.LavaFormatError("could not parse reply into json", err,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("reply", reply.Data),
		)
	}

	if replyJson.Error != nil {
		// Node error, subscription was not initialized, triggering OnSessionFailure
		return nil, utils.LavaFormatError("error in reply from subscription", nil,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("reply", replyJson),
		)
	}

	return &reply, nil
}

func (rpcss *RPCSmartRouterServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, protocolMessage chainlib.ProtocolMessage, dataReliabilityThreshold uint32, relayProcessor *relaycore.RelayProcessor) error {
	if statetracker.DisableDR {
		return nil
	}
	processingTimeout, expectedRelayTimeout := rpcss.getProcessingTimeout(protocolMessage)
	// Wait another relayTimeout duration to maybe get additional relay results
	if relayProcessor.GetUsedProviders().CurrentlyUsed() > 0 {
		time.Sleep(expectedRelayTimeout)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	specCategory := protocolMessage.GetApi().Category
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

	reqBlock, _ := protocolMessage.RequestedBlock()
	if reqBlock <= spectypes.NOT_APPLICABLE {
		if reqBlock <= spectypes.LATEST_BLOCK {
			return utils.LavaFormatError("sendDataReliabilityRelayIfApplicable latest requestBlock", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "RequestBlock", Value: reqBlock})
		}
		// does not support sending data reliability requests on a block that is not specific
		return nil
	}
	relayResult := results[0]
	if len(results) < 2 {
		relayRequestData := lavaprotocol.NewRelayData(ctx, relayResult.Request.RelayData.ConnectionType, relayResult.Request.RelayData.ApiUrl, relayResult.Request.RelayData.Data, relayResult.Request.RelayData.SeenBlock, reqBlock, relayResult.Request.RelayData.ApiInterface, protocolMessage.GetRPCMessage().GetHeaders(), relayResult.Request.RelayData.Addon, relayResult.Request.RelayData.Extensions)
		userData := protocolMessage.GetUserData()
		//  We create new protocol message from the old one, but with a new instance of relay request data.
		dataReliabilityProtocolMessage := chainlib.NewProtocolMessage(protocolMessage, nil, relayRequestData, userData.DappId, userData.ConsumerIp)

		// Get quorum parameters from the data reliability protocol message
		quorumParams, err := dataReliabilityProtocolMessage.GetQuorumParameters()
		if err != nil {
			return utils.LavaFormatError("failed to get quorum parameters from data reliability protocol message", err, utils.LogAttr("dataReliabilityProtocolMessage", dataReliabilityProtocolMessage))
		}

		// Validate that quorum min doesn't exceed available providers
		if quorumParams.Enabled() && quorumParams.Min > rpcss.sessionManager.GetNumberOfValidProviders() {
			return utils.LavaFormatError("requested quorum min exceeds available providers for data reliability",
				lavasession.PairingListEmptyError,
				utils.LogAttr("quorumMin", quorumParams.Min),
				utils.LogAttr("availableProviders", rpcss.sessionManager.GetNumberOfValidProviders()),
				utils.LogAttr("GUID", ctx))
		}

		relayProcessorDataReliability := relaycore.NewRelayProcessor(
			ctx,
			quorumParams,
			rpcss.smartRouterConsistency,
			rpcss.rpcSmartRouterLogs,
			rpcss,
			rpcss.relayRetriesManager,
			NewSmartRouterRelayStateMachine(ctx, relayProcessor.GetUsedProviders(), rpcss, dataReliabilityProtocolMessage, nil, rpcss.debugRelays, rpcss.rpcSmartRouterLogs),
			rpcss.sessionManager.GetQoSManager(),
		)
		err = rpcss.sendRelayToProvider(ctx, 1, relaycore.GetEmptyRelayState(ctx, dataReliabilityProtocolMessage), relayProcessorDataReliability, nil)
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
		conflict := lavaprotocol.VerifyReliabilityResults(ctx, &relayResult, &relayResultDataReliability, protocolMessage.GetApiCollection(), rpcss.chainParser)
		if conflict != nil {
			// TODO: remove this check when we fix the missing extensions information on conflict detection transaction
			if len(protocolMessage.GetExtensions()) == 0 {
				// Smart router doesn't report conflicts to blockchain
				utils.LavaFormatWarning("Data reliability conflict detected in smart router mode", nil,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "conflict", Value: conflict})
				if rpcss.reporter != nil {
					utils.LavaFormatDebug("sending conflict report to BE", utils.LogAttr("conflicting api", protocolMessage.GetApi().Name))
					rpcss.reporter.AppendConflict(metrics.NewConflictRequest(relayResult.Request, relayResult.Reply, relayResultDataReliability.Request, relayResultDataReliability.Reply))
				}
			}
		} else {
			utils.LavaFormatDebug("[+] verified relay successfully with data reliability", utils.LogAttr("api", protocolMessage.GetApi().Name))
		}
	}
	return nil
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
