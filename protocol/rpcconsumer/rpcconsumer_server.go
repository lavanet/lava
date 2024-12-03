package rpcconsumer

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/finalizationconsensus"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/finalizationverification"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/performance"
	"github.com/lavanet/lava/v4/protocol/upgrade"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/protocopy"
	"github.com/lavanet/lava/v4/utils/rand"
	conflicttypes "github.com/lavanet/lava/v4/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	plantypes "github.com/lavanet/lava/v4/x/plans/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
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
	initRelaysConsumerIp                     = ""
)

var NoResponseTimeout = sdkerrors.New("NoResponseTimeout Error", 685, "timeout occurred while waiting for providers responses")

type CancelableContextHolder struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

// implements Relay Sender interfaced and uses an ChainListener to get it called
type RPCConsumerServer struct {
	consumerProcessGuid            string
	chainParser                    chainlib.ChainParser
	consumerSessionManager         *lavasession.ConsumerSessionManager
	listenEndpoint                 *lavasession.RPCEndpoint
	rpcConsumerLogs                *metrics.RPCConsumerLogs
	cache                          *performance.Cache
	privKey                        *btcec.PrivateKey
	consumerTxSender               ConsumerTxSender
	requiredResponses              int
	finalizationConsensus          finalizationconsensus.FinalizationConsensusInf
	lavaChainID                    string
	ConsumerAddress                sdk.AccAddress
	consumerConsistency            *ConsumerConsistency
	sharedState                    bool // using the cache backend to sync the latest seen block with other consumers
	relaysMonitor                  *metrics.RelaysMonitor
	reporter                       metrics.Reporter
	debugRelays                    bool
	connectedSubscriptionsContexts map[string]*CancelableContextHolder
	chainListener                  chainlib.ChainListener
	connectedSubscriptionsLock     sync.RWMutex
	relayRetriesManager            *lavaprotocol.RelayRetriesManager
	initialized                    atomic.Bool
}

type relayResponse struct {
	relayResult common.RelayResult
	err         error
}

type ConsumerTxSender interface {
	TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error
	GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error)
	GetLatestVirtualEpoch() uint64
}

func (rpccs *RPCConsumerServer) ServeRPCRequests(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	consumerStateTracker ConsumerStateTrackerInf,
	chainParser chainlib.ChainParser,
	finalizationConsensus finalizationconsensus.FinalizationConsensusInf,
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
	consumerWsSubscriptionManager *chainlib.ConsumerWSSubscriptionManager,
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
	rpccs.connectedSubscriptionsContexts = make(map[string]*CancelableContextHolder)
	rpccs.consumerProcessGuid = strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10)
	rpccs.relayRetriesManager = lavaprotocol.NewRelayRetriesManager()
	rpccs.chainListener, err = chainlib.NewChainListener(ctx, listenEndpoint, rpccs, rpccs, rpcConsumerLogs, chainParser, refererData, consumerWsSubscriptionManager)
	if err != nil {
		return err
	}

	go rpccs.chainListener.Serve(ctx, cmdFlags)

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

func (rpccs *RPCConsumerServer) SetConsistencySeenBlock(blockSeen int64, key string) {
	rpccs.consumerConsistency.SetSeenBlockFromKey(blockSeen, key)
}

func (rpccs *RPCConsumerServer) GetListeningAddress() string {
	return rpccs.chainListener.GetListeningAddress()
}

func (rpccs *RPCConsumerServer) sendCraftedRelaysWrapper(initialRelays bool) (bool, error) {
	if initialRelays {
		// Only start after everything is initialized - check consumer session manager
		rpccs.waitForPairing()
	}
	success, err := rpccs.sendCraftedRelays(MaxRelayRetries, initialRelays)
	if success {
		rpccs.initialized.Store(true)
	}
	return success, err
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
	parsing, apiCollection, ok := rpccs.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	if !ok {
		return false, nil, nil, utils.LavaFormatWarning("did not send initial relays because the spec does not contain "+spectypes.FUNCTION_TAG_GET_BLOCKNUM.String(), nil,
			utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface),
		)
	}
	collectionData := apiCollection.CollectionData

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

func (rpccs *RPCConsumerServer) sendRelayWithRetries(ctx context.Context, retries int, initialRelays bool, protocolMessage chainlib.ProtocolMessage) (bool, error) {
	success := false
	var err error
	usedProviders := lavasession.NewUsedProviders(nil)
	relayProcessor := NewRelayProcessor(
		ctx,
		1,
		rpccs.consumerConsistency,
		rpccs.rpcConsumerLogs,
		rpccs,
		rpccs.relayRetriesManager,
		NewRelayStateMachine(ctx, usedProviders, rpccs, protocolMessage, nil, rpccs.debugRelays, rpccs.rpcConsumerLogs),
	)
	usedProvidersResets := 1
	for i := 0; i < retries; i++ {
		// Check if we even have enough providers to communicate with them all.
		// If we have 1 provider we will reset the used providers always.
		// Instead of spamming no pairing available on bootstrap
		if ((i + 1) * usedProvidersResets) > rpccs.consumerSessionManager.GetNumberOfValidProviders() {
			usedProvidersResets++
			relayProcessor.GetUsedProviders().ClearUnwanted()
		}
		err = rpccs.sendRelayToProvider(ctx, GetEmptyRelayState(ctx, protocolMessage), relayProcessor, nil)
		if lavasession.PairingListEmptyError.Is(err) {
			// we don't have pairings anymore, could be related to unwanted providers
			relayProcessor.GetUsedProviders().ClearUnwanted()
			err = rpccs.sendRelayToProvider(ctx, GetEmptyRelayState(ctx, protocolMessage), relayProcessor, nil)
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
					utils.LavaFormatInfo("[+] init relay succeeded",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("chainID", rpccs.listenEndpoint.ChainID),
						utils.LogAttr("APIInterface", rpccs.listenEndpoint.ApiInterface),
						utils.LogAttr("latestBlock", relayResult.Reply.LatestBlock),
						utils.LogAttr("provider address", relayResult.ProviderInfo.ProviderAddress),
					)
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
	protocolMessage := chainlib.NewProtocolMessage(chainMessage, nil, relay, initRelaysDappId, initRelaysConsumerIp)
	return rpccs.sendRelayWithRetries(ctx, retries, initialRelays, protocolMessage)
}

func (rpccs *RPCConsumerServer) getLatestBlock() uint64 {
	latestKnownBlock, numProviders := rpccs.finalizationConsensus.GetExpectedBlockHeight(rpccs.chainParser)
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
	protocolMessage, err := rpccs.ParseRelay(ctx, url, req, connectionType, dappID, consumerIp, metadata)
	if err != nil {
		return nil, err
	}

	return rpccs.SendParsedRelay(ctx, analytics, protocolMessage)
}

func (rpccs *RPCConsumerServer) ParseRelay(
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
	metadata, directiveHeaders := rpccs.LavaDirectiveHeaders(metadata)
	chainMessage, err := rpccs.chainParser.ParseMsg(url, []byte(req), connectionType, metadata, rpccs.getExtensionsFromDirectiveHeaders(directiveHeaders))
	if err != nil {
		return nil, err
	}

	rpccs.HandleDirectiveHeadersForMessage(chainMessage, directiveHeaders)

	// do this in a loop with retry attempts, configurable via a flag, limited by the number of providers in CSM
	reqBlock, _ := chainMessage.RequestedBlock()
	seenBlock, _ := rpccs.consumerConsistency.GetSeenBlock(common.UserData{DappId: dappID, ConsumerIp: consumerIp})
	if seenBlock < 0 {
		seenBlock = 0
	}

	relayRequestData := lavaprotocol.NewRelayData(ctx, connectionType, url, []byte(req), seenBlock, reqBlock, rpccs.listenEndpoint.ApiInterface, chainMessage.GetRPCMessage().GetHeaders(), chainlib.GetAddon(chainMessage), common.GetExtensionNames(chainMessage.GetExtensions()))
	protocolMessage = chainlib.NewProtocolMessage(chainMessage, directiveHeaders, relayRequestData, dappID, consumerIp)
	return protocolMessage, nil
}

func (rpccs *RPCConsumerServer) SendParsedRelay(
	ctx context.Context,
	analytics *metrics.RelayMetrics,
	protocolMessage chainlib.ProtocolMessage,
) (relayResult *common.RelayResult, errRet error) {
	// sends a relay message to a provider
	// compares the result with other providers if defined so
	// compares the response with other consumer wallets if defined so
	// asynchronously sends data reliability if necessary

	relaySentTime := time.Now()
	relayProcessor, err := rpccs.ProcessRelaySend(ctx, protocolMessage, analytics)
	if err != nil && !relayProcessor.HasResults() {
		userData := protocolMessage.GetUserData()
		// we can't send anymore, and we don't have any responses
		utils.LavaFormatError("failed getting responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.LogAttr("endpoint", rpccs.listenEndpoint.Key()), utils.LogAttr("userIp", userData.ConsumerIp), utils.LogAttr("relayProcessor", relayProcessor))
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
		go rpccs.sendDataReliabilityRelayIfApplicable(dataReliabilityContext, protocolMessage, dataReliabilityThreshold, relayProcessor) // runs asynchronously
	}

	returnedResult, err := relayProcessor.ProcessingResult()
	rpccs.appendHeadersToRelayResult(ctx, returnedResult, relayProcessor.ProtocolErrors(), relayProcessor, protocolMessage, protocolMessage.GetApi().GetName())
	if err != nil {
		return returnedResult, utils.LavaFormatError("failed processing responses from providers", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.LogAttr("endpoint", rpccs.listenEndpoint.Key()))
	}

	if analytics != nil {
		currentLatency := time.Since(relaySentTime)
		analytics.Latency = currentLatency.Milliseconds()
		api := protocolMessage.GetApi()
		analytics.ComputeUnits = api.ComputeUnits
		analytics.ApiMethod = api.Name
	}
	rpccs.relaysMonitor.LogRelay()
	return returnedResult, nil
}

func (rpccs *RPCConsumerServer) GetChainIdAndApiInterface() (string, string) {
	return rpccs.listenEndpoint.ChainID, rpccs.listenEndpoint.ApiInterface
}

func (rpccs *RPCConsumerServer) ProcessRelaySend(ctx context.Context, protocolMessage chainlib.ProtocolMessage, analytics *metrics.RelayMetrics) (*RelayProcessor, error) {
	// make sure all of the child contexts are cancelled when we exit
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	usedProviders := lavasession.NewUsedProviders(protocolMessage)
	relayProcessor := NewRelayProcessor(
		ctx,
		rpccs.requiredResponses,
		rpccs.consumerConsistency,
		rpccs.rpcConsumerLogs,
		rpccs,
		rpccs.relayRetriesManager,
		NewRelayStateMachine(ctx, usedProviders, rpccs, protocolMessage, analytics, rpccs.debugRelays, rpccs.rpcConsumerLogs),
	)

	relayTaskChannel, err := relayProcessor.GetRelayTaskChannel()
	if err != nil {
		return relayProcessor, err
	}
	for task := range relayTaskChannel {
		if task.IsDone() {
			return relayProcessor, task.err
		}
		err := rpccs.sendRelayToProvider(ctx, task.relayState, relayProcessor, task.analytics)
		relayProcessor.UpdateBatch(err)
	}

	// shouldn't happen.
	return relayProcessor, utils.LavaFormatError("ProcessRelaySend channel closed unexpectedly", nil)
}

func (rpccs *RPCConsumerServer) CreateDappKey(userData common.UserData) string {
	return rpccs.consumerConsistency.Key(userData)
}

func (rpccs *RPCConsumerServer) CancelSubscriptionContext(subscriptionKey string) {
	rpccs.connectedSubscriptionsLock.Lock()
	defer rpccs.connectedSubscriptionsLock.Unlock()

	ctxHolder, ok := rpccs.connectedSubscriptionsContexts[subscriptionKey]
	if ok {
		utils.LavaFormatTrace("cancelling subscription context", utils.LogAttr("subscriptionID", subscriptionKey))
		ctxHolder.CancelFunc()
		delete(rpccs.connectedSubscriptionsContexts, subscriptionKey)
	} else {
		utils.LavaFormatWarning("tried to cancel context for subscription ID that does not exist", nil, utils.LogAttr("subscriptionID", subscriptionKey))
	}
}

func (rpccs *RPCConsumerServer) getEarliestBlockHashRequestedFromCacheReply(cacheReply *pairingtypes.CacheRelayReply) (int64, int64) {
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

func (rpccs *RPCConsumerServer) resolveRequestedBlock(reqBlock int64, seenBlock int64, latestBlockHashRequested int64, protocolMessage chainlib.ProtocolMessage) int64 {
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

func (rpccs *RPCConsumerServer) updateBlocksHashesToHeightsIfNeeded(extensions []*spectypes.Extension, chainMessage chainlib.ChainMessage, blockHashesToHeights []*pairingtypes.BlockHashToHeight, latestBlock int64, finalized bool, relayState *RelayState) ([]*pairingtypes.BlockHashToHeight, bool) {
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

func (rpccs *RPCConsumerServer) newBlocksHashesToHeightsSliceFromRequestedBlockHashes(requestedBlockHashes []string) []*pairingtypes.BlockHashToHeight {
	var blocksHashesToHeights []*pairingtypes.BlockHashToHeight
	for _, blockHash := range requestedBlockHashes {
		blocksHashesToHeights = append(blocksHashesToHeights, &pairingtypes.BlockHashToHeight{Hash: blockHash, Height: spectypes.NOT_APPLICABLE})
	}
	return blocksHashesToHeights
}

func (rpccs *RPCConsumerServer) newBlocksHashesToHeightsSliceFromFinalizationConsensus(finalizedBlockHashes map[int64]string) []*pairingtypes.BlockHashToHeight {
	var blocksHashesToHeights []*pairingtypes.BlockHashToHeight
	for height, blockHash := range finalizedBlockHashes {
		blocksHashesToHeights = append(blocksHashesToHeights, &pairingtypes.BlockHashToHeight{Hash: blockHash, Height: height})
	}
	return blocksHashesToHeights
}

func (rpccs *RPCConsumerServer) sendRelayToProvider(
	ctx context.Context,
	relayState *RelayState,
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
	protocolMessage := relayState.GetProtocolMessage()
	userData := protocolMessage.GetUserData()
	var sharedStateId string // defaults to "", if shared state is disabled then no shared state will be used.
	if rpccs.sharedState {
		sharedStateId = rpccs.consumerConsistency.Key(userData) // use same key as we use for consistency, (for better consistency :-D)
	}

	privKey := rpccs.privKey
	chainId, apiInterface := rpccs.GetChainIdAndApiInterface()
	lavaChainID := rpccs.lavaChainID

	// Get Session. we get session here so we can use the epoch in the callbacks
	reqBlock, _ := protocolMessage.RequestedBlock()

	// try using cache before sending relay
	earliestBlockHashRequested := spectypes.NOT_APPLICABLE
	latestBlockHashRequested := spectypes.NOT_APPLICABLE
	var cacheError error
	if rpccs.cache.CacheActive() { // use cache only if its defined.
		if !protocolMessage.GetForceCacheRefresh() { // don't use cache if user specified
			if reqBlock != spectypes.NOT_APPLICABLE { // don't use cache if requested block is not applicable
				var cacheReply *pairingtypes.CacheRelayReply
				hashKey, outputFormatter, err := protocolMessage.HashCacheRequest(chainId)
				if err != nil {
					utils.LavaFormatError("sendRelayToProvider Failed getting Hash for cache request", err)
				} else {
					cacheCtx, cancel := context.WithTimeout(ctx, common.CacheTimeout)
					cacheReply, cacheError = rpccs.cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
						RequestHash:           hashKey,
						RequestedBlock:        reqBlock,
						ChainId:               chainId,
						BlockHash:             nil,
						Finalized:             false,
						SharedStateId:         sharedStateId,
						SeenBlock:             protocolMessage.RelayPrivateData().SeenBlock,
						BlocksHashesToHeights: rpccs.newBlocksHashesToHeightsSliceFromRequestedBlockHashes(protocolMessage.GetRequestedBlocksHashes()),
					}) // caching in the consumer doesn't care about hashes, and we don't have data on finalization yet
					cancel()
					reply := cacheReply.GetReply()

					// read seen block from cache even if we had a miss we still want to get the seen block so we can use it to get the right provider.
					cacheSeenBlock := cacheReply.GetSeenBlock()
					// check if the cache seen block is greater than my local seen block, this means the user requested this
					// request spoke with another consumer instance and use that block for inter consumer consistency.
					if rpccs.sharedState && cacheSeenBlock > protocolMessage.RelayPrivateData().SeenBlock {
						utils.LavaFormatDebug("shared state seen block is newer", utils.LogAttr("cache_seen_block", cacheSeenBlock), utils.LogAttr("local_seen_block", protocolMessage.RelayPrivateData().SeenBlock))
						protocolMessage.RelayPrivateData().SeenBlock = cacheSeenBlock
						// setting the fetched seen block from the cache server to our local cache as well.
						rpccs.consumerConsistency.SetSeenBlock(cacheSeenBlock, userData)
					}

					// handle cache reply
					if cacheError == nil && reply != nil {
						// Info was fetched from cache, so we don't need to change the state
						// so we can return here, no need to update anything and calculate as this info was fetched from the cache
						reply.Data = outputFormatter(reply.Data)
						relayResult := common.RelayResult{
							Reply: reply,
							Request: &pairingtypes.RelayRequest{
								RelayData: protocolMessage.RelayPrivateData(),
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
					latestBlockHashRequested, earliestBlockHashRequested = rpccs.getEarliestBlockHashRequestedFromCacheReply(cacheReply)
					utils.LavaFormatTrace("[Archive Debug] Reading block hashes from cache", utils.LogAttr("latestBlockHashRequested", latestBlockHashRequested), utils.LogAttr("earliestBlockHashRequested", earliestBlockHashRequested))
					// cache failed, move on to regular relay
					if performance.NotConnectedError.Is(cacheError) {
						utils.LavaFormatDebug("cache not connected", utils.LogAttr("error", cacheError))
					}
				}
			} else {
				utils.LavaFormatDebug("skipping cache due to requested block being NOT_APPLICABLE", utils.Attribute{Key: "api name", Value: protocolMessage.GetApi().Name})
			}
		}
	}

	addon := chainlib.GetAddon(protocolMessage)
	reqBlock = rpccs.resolveRequestedBlock(reqBlock, protocolMessage.RelayPrivateData().SeenBlock, latestBlockHashRequested, protocolMessage)
	// check whether we need a new protocol message with the new earliest block hash requested
	protocolMessage = rpccs.updateProtocolMessageIfNeededWithNewEarliestData(ctx, relayState, protocolMessage, earliestBlockHashRequested, addon)

	// consumerEmergencyTracker always use latest virtual epoch
	virtualEpoch := rpccs.consumerTxSender.GetLatestVirtualEpoch()
	extensions := protocolMessage.GetExtensions()
	utils.LavaFormatTrace("[Archive Debug] Extensions to send", utils.LogAttr("extensions", extensions))
	usedProviders := relayProcessor.GetUsedProviders()
	sessions, err := rpccs.consumerSessionManager.GetSessions(ctx, chainlib.GetComputeUnits(protocolMessage), usedProviders, reqBlock, addon, extensions, chainlib.GetStateful(protocolMessage), virtualEpoch)
	if err != nil {
		if lavasession.PairingListEmptyError.Is(err) {
			if addon != "" {
				return utils.LavaFormatError("No Providers For Addon", err, utils.LogAttr("addon", addon), utils.LogAttr("extensions", extensions), utils.LogAttr("userIp", userData.ConsumerIp))
			} else if len(extensions) > 0 && relayProcessor.GetAllowSessionDegradation() { // if we have no providers for that extension, use a regular provider, otherwise return the extension results
				sessions, err = rpccs.consumerSessionManager.GetSessions(ctx, chainlib.GetComputeUnits(protocolMessage), usedProviders, reqBlock, addon, []*spectypes.Extension{}, chainlib.GetStateful(protocolMessage), virtualEpoch)
				if err != nil {
					return err
				}
				relayProcessor.setSkipDataReliability(true)                // disabling data reliability when disabling extensions.
				protocolMessage.RelayPrivateData().Extensions = []string{} // reset request data extensions
				extensions = []*spectypes.Extension{}                      // reset extensions too so we wont hit SetDisallowDegradation
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
		routerKey := lavasession.NewRouterKeyFromExtensions(extensions)
		utils.LavaFormatDebug("[Before Send] returned the following sessions",
			utils.LogAttr("sessions", sessions),
			utils.LogAttr("usedProviders.GetUnwantedProvidersToSend", usedProviders.GetUnwantedProvidersToSend(routerKey)),
			utils.LogAttr("usedProviders.GetErroredProviders", usedProviders.GetErroredProviders(routerKey)),
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

			localRelayRequestData := *protocolMessage.RelayPrivateData()

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
			endpointClient := singleConsumerSession.EndpointConnection.Client

			// set relay sent metric
			go rpccs.rpcConsumerLogs.SetRelaySentToProviderMetric(providerPublicAddress, chainId, apiInterface)

			if chainlib.IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
				utils.LavaFormatTrace("inside sendRelayToProvider, relay is subscription", utils.LogAttr("requestData", localRelayRequestData.Data))

				params, err := json.Marshal(protocolMessage.GetRPCMessage().GetParams())
				if err != nil {
					utils.LavaFormatError("could not marshal params", err)
					return
				}

				hashedParams := rpcclient.CreateHashFromParams(params)
				cancellableCtx, cancelFunc := context.WithCancel(utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier()))

				ctxHolder := func() *CancelableContextHolder {
					rpccs.connectedSubscriptionsLock.Lock()
					defer rpccs.connectedSubscriptionsLock.Unlock()

					ctxHolder := &CancelableContextHolder{
						Ctx:        cancellableCtx,
						CancelFunc: cancelFunc,
					}
					rpccs.connectedSubscriptionsContexts[hashedParams] = ctxHolder
					return ctxHolder
				}()

				errResponse = rpccs.relaySubscriptionInner(ctxHolder.Ctx, hashedParams, endpointClient, singleConsumerSession, localRelayResult)
				if errResponse != nil {
					utils.LavaFormatError("Failed relaySubscriptionInner", errResponse,
						utils.LogAttr("Request", localRelayRequestData),
						utils.LogAttr("Request data", string(localRelayRequestData.Data)),
						utils.LogAttr("Provider", providerPublicAddress),
					)
				}
				return
			}

			// unique per dappId and ip
			consumerToken := common.GetUniqueToken(userData)
			processingTimeout, expectedRelayTimeoutForQOS := rpccs.getProcessingTimeout(protocolMessage)
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
			relayLatency, errResponse, backoff := rpccs.relayInner(goroutineCtx, singleConsumerSession, localRelayResult, processingTimeout, protocolMessage, consumerToken, analytics)
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
			expectedBH, numOfProviders := rpccs.finalizationConsensus.GetExpectedBlockHeight(rpccs.chainParser)
			pairingAddressesLen := rpccs.consumerSessionManager.GetAtomicPairingAddressesLength()
			latestBlock := localRelayResult.Reply.LatestBlock
			if expectedBH-latestBlock > 1000 {
				utils.LavaFormatWarning("identified block gap", nil,
					utils.Attribute{Key: "expectedBH", Value: expectedBH},
					utils.Attribute{Key: "latestServicedBlock", Value: latestBlock},
					utils.Attribute{Key: "session_id", Value: singleConsumerSession.SessionId},
					utils.Attribute{Key: "provider_address", Value: singleConsumerSession.Parent.PublicLavaAddress},
					utils.Attribute{Key: "providersCount", Value: pairingAddressesLen},
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
				)
			}

			errResponse = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, latestBlock, chainlib.GetComputeUnits(protocolMessage), relayLatency, singleConsumerSession.CalculateExpectedLatency(expectedRelayTimeoutForQOS), expectedBH, numOfProviders, pairingAddressesLen, protocolMessage.GetApi().Category.HangingApi, extensions) // session done successfully
			isNodeError, _ := protocolMessage.CheckResponseError(localRelayResult.Reply.Data, localRelayResult.StatusCode)
			localRelayResult.IsNodeError = isNodeError
			if rpccs.debugRelays {
				utils.LavaFormatDebug("Result Code", utils.LogAttr("isNodeError", isNodeError), utils.LogAttr("StatusCode", localRelayResult.StatusCode))
			}
			if rpccs.cache.CacheActive() && rpcclient.ValidateStatusCodes(localRelayResult.StatusCode, true) == nil {
				// in case the error is a node error we don't want to cache
				if !isNodeError {
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
							blockHashesToHeights = rpccs.newBlocksHashesToHeightsSliceFromFinalizationConsensus(finalizedBlockHashesObj)
						}
						var finalized bool
						blockHashesToHeights, finalized = rpccs.updateBlocksHashesToHeightsIfNeeded(extensions, protocolMessage, blockHashesToHeights, latestBlock, localRelayResult.Finalized, relayState)
						utils.LavaFormatTrace("[Archive Debug] Adding HASH TO CACHE", utils.LogAttr("blockHashesToHeights", blockHashesToHeights))

						new_ctx := context.Background()
						new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
						defer cancel()
						_, averageBlockTime, _, _ := rpccs.chainParser.ChainBlockStats()
						err2 := rpccs.cache.SetEntry(new_ctx, &pairingtypes.RelayCacheSet{
							RequestHash:           hashKey,
							ChainId:               chainId,
							RequestedBlock:        requestedBlock,
							SeenBlock:             seenBlock,
							BlockHash:             nil, // consumer cache doesn't care about block hashes
							Response:              copyReply,
							Finalized:             finalized,
							OptionalMetadata:      nil,
							SharedStateId:         sharedStateId,
							AverageBlockTime:      int64(averageBlockTime), // by using average block time we can set longer TTL
							IsNodeError:           isNodeError,
							BlocksHashesToHeights: blockHashesToHeights,
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
	endpointClient := singleConsumerSession.EndpointConnection.Client
	providerPublicAddress := relayResult.ProviderInfo.ProviderAddress
	relayRequest := relayResult.Request
	if rpccs.debugRelays {
		utils.LavaFormatDebug("Sending relay", utils.LogAttr("timeout", relayTimeout), utils.LogAttr("requestedBlock", relayRequest.RelayData.RequestBlock), utils.LogAttr("GUID", ctx), utils.LogAttr("provider", relayRequest.RelaySession.Provider))
	}
	callRelay := func() (reply *pairingtypes.RelayReply, relayLatency time.Duration, err error, backoff bool) {
		connectCtx, connectCtxCancel := context.WithTimeout(ctx, relayTimeout)
		metadataAdd := metadata.New(map[string]string{
			common.IP_FORWARDING_HEADER_NAME:  consumerToken,
			common.LAVA_CONSUMER_PROCESS_GUID: rpccs.consumerProcessGuid,
			common.LAVA_LB_UNIQUE_ID_HEADER:   singleConsumerSession.EndpointConnection.GetLbUniqueId(),
		})

		utils.LavaFormatTrace("Sending relay to provider",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("lbUniqueId", singleConsumerSession.EndpointConnection.GetLbUniqueId()),
		)
		connectCtx = metadata.NewOutgoingContext(connectCtx, metadataAdd)
		defer connectCtxCancel()

		// add consumer processing timestamp before provider metric and start measuring time after the provider replied
		rpccs.rpcConsumerLogs.AddMetricForProcessingLatencyBeforeProvider(analytics, rpccs.listenEndpoint.ChainID, rpccs.listenEndpoint.ApiInterface)

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
						errors.Join(lavasession.SessionOutOfSyncError, lavasession.BlockEndpointError),
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
				utils.LavaFormatWarning("failed converting status code", errStatus, utils.LogAttr("statuses", statuses))
			}

			relayResult.StatusCode = codeNum
		}

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
			)
		}
		return 0, err, backoff
	}

	relayResult.Reply = reply

	// Update relay request requestedBlock to the provided one in case it was arbitrary
	lavaprotocol.UpdateRequestedBlock(relayRequest.RelayData, reply)

	_, _, blockDistanceForFinalizedData, blocksInFinalizationProof := rpccs.chainParser.ChainBlockStats()
	isFinalized := spectypes.IsFinalizedBlock(relayRequest.RelayData.RequestBlock, reply.LatestBlock, int64(blockDistanceForFinalizedData))

	filteredHeaders, _, ignoredHeaders := rpccs.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
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
	enabled, _ := rpccs.chainParser.DataReliabilityParams()
	if enabled && !singleConsumerSession.StaticProvider {
		// TODO: allow static providers to detect hash mismatches,
		// triggering conflict with them is impossible so we skip this for now, but this can be used to block malicious providers
		finalizedBlocks, err := finalizationverification.VerifyFinalizationData(reply, relayRequest, providerPublicAddress, rpccs.ConsumerAddress, existingSessionLatestBlock, int64(blockDistanceForFinalizedData), int64(blocksInFinalizationProof))
		if err != nil {
			if sdkerrors.IsOf(err, protocolerrors.ProviderFinalizationDataAccountabilityError) {
				utils.LavaFormatInfo("provider finalization data accountability error", utils.LogAttr("provider", relayRequest.RelaySession.Provider))
			}
			return 0, err, false
		}

		finalizationAccountabilityError, err := rpccs.finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), rpccs.ConsumerAddress, providerPublicAddress, finalizedBlocks, relayRequest.RelaySession, reply)
		if err != nil {
			if finalizationAccountabilityError != nil {
				go rpccs.consumerTxSender.TxConflictDetection(ctx, finalizationAccountabilityError, nil, singleConsumerSession.Parent)
			}
			return 0, err, false
		}
	}
	relayResult.Finalized = isFinalized
	return relayLatency, nil, false
}

func (rpccs *RPCConsumerServer) relaySubscriptionInner(ctx context.Context, hashedParams string, endpointClient pairingtypes.RelayerClient, singleConsumerSession *lavasession.SingleConsumerSession, relayResult *common.RelayResult) (err error) {
	// add consumer guid to relay request.
	ctx = metadata.AppendToOutgoingContext(ctx,
		common.LAVA_LB_UNIQUE_ID_HEADER, singleConsumerSession.EndpointConnection.GetLbUniqueId(),
		common.LAVA_CONSUMER_PROCESS_GUID, rpccs.consumerProcessGuid,
	)

	replyServer, err := endpointClient.RelaySubscribe(ctx, relayResult.Request)
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
		if errReport != nil {
			return utils.LavaFormatError("subscribe relay failed onSessionFailure errored", errReport,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("originalError", err.Error()),
			)
		}

		return err
	}

	reply, err := rpccs.getFirstSubscriptionReply(ctx, hashedParams, replyServer)
	if err != nil {
		errReport := rpccs.consumerSessionManager.OnSessionFailure(singleConsumerSession, err)
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
	err = rpccs.consumerSessionManager.OnSessionDoneIncreaseCUOnly(singleConsumerSession, latestBlock)
	return err
}

func (rpccs *RPCConsumerServer) getFirstSubscriptionReply(ctx context.Context, hashedParams string, replyServer pairingtypes.Relayer_RelaySubscribeClient) (*pairingtypes.RelayReply, error) {
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
					rpccs.CancelSubscriptionContext(hashedParams) // Cancel the context with the provider, which will trigger the replyServer's context to be cancelled
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

func (rpccs *RPCConsumerServer) sendDataReliabilityRelayIfApplicable(ctx context.Context, protocolMessage chainlib.ProtocolMessage, dataReliabilityThreshold uint32, relayProcessor *RelayProcessor) error {
	processingTimeout, expectedRelayTimeout := rpccs.getProcessingTimeout(protocolMessage)
	// Wait another relayTimeout duration to maybe get additional relay results
	if relayProcessor.usedProviders.CurrentlyUsed() > 0 {
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
		relayProcessorDataReliability := NewRelayProcessor(
			ctx,
			1,
			rpccs.consumerConsistency,
			rpccs.rpcConsumerLogs,
			rpccs,
			rpccs.relayRetriesManager,
			NewRelayStateMachine(ctx, relayProcessor.usedProviders, rpccs, dataReliabilityProtocolMessage, nil, rpccs.debugRelays, rpccs.rpcConsumerLogs),
		)
		err := rpccs.sendRelayToProvider(ctx, GetEmptyRelayState(ctx, dataReliabilityProtocolMessage), relayProcessorDataReliability, nil)
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
		conflict := lavaprotocol.VerifyReliabilityResults(ctx, &relayResult, &relayResultDataReliability, protocolMessage.GetApiCollection(), rpccs.chainParser)
		if conflict != nil {
			// TODO: remove this check when we fix the missing extensions information on conflict detection transaction
			if len(protocolMessage.GetExtensions()) == 0 {
				err := rpccs.consumerTxSender.TxConflictDetection(ctx, nil, conflict, relayResultDataReliability.ConflictHandler)
				if err != nil {
					utils.LavaFormatError("could not send detection Transaction", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "conflict", Value: conflict})
				}
				if rpccs.reporter != nil {
					utils.LavaFormatDebug("sending conflict report to BE", utils.LogAttr("conflicting api", protocolMessage.GetApi().Name))
					rpccs.reporter.AppendConflict(metrics.NewConflictRequest(relayResult.Request, relayResult.Reply, relayResultDataReliability.Request, relayResultDataReliability.Reply))
				}
			}
		} else {
			utils.LavaFormatDebug("[+] verified relay successfully with data reliability", utils.LogAttr("api", protocolMessage.GetApi().Name))
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

// Iterating over metadataHeaders adding each trailer that fits the header if found to relayResult.Relay.Metadata
func (rpccs *RPCConsumerServer) getMetadataFromRelayTrailer(metadataHeaders []string, relayResult *common.RelayResult) {
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

func (rpccs *RPCConsumerServer) appendHeadersToRelayResult(ctx context.Context, relayResult *common.RelayResult, protocolErrors uint64, relayProcessor *RelayProcessor, protocolMessage chainlib.ProtocolMessage, apiName string) {
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

	// add stateful API (hanging, transactions)
	if protocolMessage.GetApi().Category.Stateful == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		metadataReply = append(metadataReply,
			pairingtypes.Metadata{
				Name:  common.STATEFUL_API_HEADER,
				Value: "true",
			})
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
	rpccs.getMetadataFromRelayTrailer(chainlib.TrailersToAddToHeaderResponse, relayResult)

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

		nodeErrors := relayProcessor.nodeErrors()
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

		version := pairingtypes.Metadata{
			Name:  common.LAVAP_VERSION_HEADER_NAME,
			Value: upgrade.GetCurrentVersion().ConsumerVersion,
		}
		relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, version)
	}

	relayResult.Reply.Metadata = append(relayResult.Reply.Metadata, metadataReply...)
}

func (rpccs *RPCConsumerServer) IsHealthy() bool {
	return rpccs.relaysMonitor.IsHealthy()
}

func (rpccs *RPCConsumerServer) IsInitialized() bool {
	if rpccs == nil {
		return false
	}

	return rpccs.initialized.Load()
}

func (rpccs *RPCConsumerServer) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guid := utils.GenerateUniqueIdentifier()
	ctx = utils.WithUniqueIdentifier(ctx, guid)
	url, data, connectionType, metadata, err := rpccs.chainParser.ExtractDataFromRequest(req)
	if err != nil {
		return nil, err
	}
	relayResult, err := rpccs.SendRelay(ctx, url, data, connectionType, "", "", nil, metadata)
	if err != nil {
		return nil, err
	}
	resp, err := rpccs.chainParser.SetResponseFromRelayResult(relayResult)
	rpccs.rpcConsumerLogs.SetLoLResponse(err == nil)
	return resp, err
}

func (rpccs *RPCConsumerServer) updateProtocolMessageIfNeededWithNewEarliestData(
	ctx context.Context,
	relayState *RelayState,
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
		newProtocolMessage, err := rpccs.ParseRelay(ctx, relayRequestData.ApiUrl, string(relayRequestData.Data), relayRequestData.ConnectionType, userData.DappId, userData.ConsumerIp, nil)
		if err != nil {
			utils.LavaFormatError("Failed copying protocol message in sendRelayToProvider", err)
			return protocolMessage
		}

		extensionAdded := newProtocolMessage.UpdateEarliestAndValidateExtensionRules(rpccs.chainParser.ExtensionsParser(), earliestBlockHashRequested, addon, relayRequestData.SeenBlock)
		if extensionAdded && relayState.CheckIsArchive(newProtocolMessage.RelayPrivateData()) {
			relayState.SetIsArchive(true)
		}
		relayState.SetProtocolMessage(newProtocolMessage)
		return newProtocolMessage
	}
	return protocolMessage
}
