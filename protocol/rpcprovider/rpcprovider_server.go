package rpcprovider

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol/protocolerrors"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/performance"
	"github.com/lavanet/lava/v4/protocol/provideroptimizer"
	rewardserver "github.com/lavanet/lava/v4/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/v4/protocol/upgrade"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	"github.com/lavanet/lava/v4/utils/protocopy"
	"github.com/lavanet/lava/v4/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	debugConsistency = false
	debugLatency     = false
)

var (
	RPCProviderStickinessHeaderName    = "X-Node-Sticky"
	numberOfRetriesAllowedOnNodeErrors = 2
)

const (
	RPCProviderAddressHeader = "Lava-Provider-Address"
)

type RPCProviderServer struct {
	cache                           *performance.Cache
	chainRouter                     chainlib.ChainRouter
	privKey                         *btcec.PrivateKey
	reliabilityManager              ReliabilityManagerInf
	providerSessionManager          *lavasession.ProviderSessionManager
	rewardServer                    RewardServerInf
	chainParser                     chainlib.ChainParser
	rpcProviderEndpoint             *lavasession.RPCProviderEndpoint
	stateTracker                    StateTrackerInf
	providerAddress                 sdk.AccAddress
	lavaChainID                     string
	allowedMissingCUThreshold       float64
	metrics                         *metrics.ProviderMetrics
	relaysMonitor                   *metrics.RelaysMonitor
	providerNodeSubscriptionManager *chainlib.ProviderNodeSubscriptionManager
	providerUniqueId                string
	StaticProvider                  bool
	providerStateMachine            *ProviderStateMachine
	providerLoadManager             *ProviderLoadManager
}

type ReliabilityManagerInf interface {
	GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error)
	GetLatestBlockNum() (int64, time.Time)
}

type RewardServerInf interface {
	SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr, apiInterface string) (existingCU uint64, updatedWithProof bool)
	SubscribeStarted(consumer string, epoch uint64, subscribeID string)
	SubscribeEnded(consumer string, epoch uint64, subscribeID string)
}

type StateTrackerInf interface {
	LatestBlock() int64
	GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epocu uint64) (maxCu uint64, err error)
	VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error)
	GetVirtualEpoch(epoch uint64) uint64
}

func (rpcps *RPCProviderServer) SetProviderUniqueId(uniqueId string) {
	rpcps.providerUniqueId = uniqueId
}

func (rpcps *RPCProviderServer) ServeRPCRequests(
	ctx context.Context, rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	chainParser chainlib.ChainParser,
	rewardServer RewardServerInf,
	providerSessionManager *lavasession.ProviderSessionManager,
	reliabilityManager ReliabilityManagerInf,
	privKey *btcec.PrivateKey,
	cache *performance.Cache,
	chainRouter chainlib.ChainRouter,
	stateTracker StateTrackerInf,
	providerAddress sdk.AccAddress,
	lavaChainID string,
	allowedMissingCUThreshold float64,
	providerMetrics *metrics.ProviderMetrics,
	relaysMonitor *metrics.RelaysMonitor,
	providerNodeSubscriptionManager *chainlib.ProviderNodeSubscriptionManager,
	staticProvider bool,
	providerLoadManager *ProviderLoadManager,
	numberOfRetries int,
) {
	rpcps.cache = cache
	rpcps.chainRouter = chainRouter
	rpcps.privKey = privKey
	rpcps.providerSessionManager = providerSessionManager
	rpcps.reliabilityManager = reliabilityManager
	if rewardServer == nil {
		utils.LavaFormatError("disabled rewards for provider, reward server not defined", nil)
		rewardServer = &rewardserver.DisabledRewardServer{}
	}
	rpcps.StaticProvider = staticProvider
	rpcps.rewardServer = rewardServer
	rpcps.chainParser = chainParser
	rpcps.rpcProviderEndpoint = rpcProviderEndpoint
	rpcps.stateTracker = stateTracker
	rpcps.providerAddress = providerAddress
	rpcps.lavaChainID = lavaChainID
	rpcps.allowedMissingCUThreshold = allowedMissingCUThreshold
	rpcps.metrics = providerMetrics
	rpcps.relaysMonitor = relaysMonitor
	rpcps.providerNodeSubscriptionManager = providerNodeSubscriptionManager
	rpcps.providerStateMachine = NewProviderStateMachine(rpcProviderEndpoint.ChainID, lavaprotocol.NewRelayRetriesManager(), chainRouter, numberOfRetries)
	rpcps.providerLoadManager = providerLoadManager

	rpcps.initRelaysMonitor(ctx)
}

func (rpcps *RPCProviderServer) initRelaysMonitor(ctx context.Context) {
	if rpcps.relaysMonitor == nil {
		return
	}

	rpcps.relaysMonitor.SetRelaySender(func() (bool, error) {
		chainMessage, err := rpcps.craftChainMessage()
		if err != nil {
			return false, err
		}

		_, _, _, _, _, err = rpcps.chainRouter.SendNodeMsg(ctx, nil, chainMessage, nil)
		return err == nil, err
	})

	rpcps.relaysMonitor.Start(ctx)
}

func (rpcps *RPCProviderServer) craftChainMessage() (chainMessage chainlib.ChainMessage, err error) {
	parsing, apiCollection, ok := rpcps.chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_GET_BLOCKNUM)
	if !ok {
		return nil, utils.LavaFormatWarning("did not send initial relays because the spec does not contain "+spectypes.FUNCTION_TAG_GET_BLOCKNUM.String(), nil,
			utils.LogAttr("chainID", rpcps.rpcProviderEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpcps.rpcProviderEndpoint.ApiInterface),
		)
	}
	collectionData := apiCollection.CollectionData

	path := parsing.ApiName
	data := []byte(parsing.FunctionTemplate)
	chainMessage, err = rpcps.chainParser.ParseMsg(path, data, collectionData.Type, nil, extensionslib.ExtensionInfo{LatestBlock: 0})
	if err != nil {
		return nil, utils.LavaFormatError("failed creating chain message in rpc consumer init relays", err,
			utils.LogAttr("chainID", rpcps.rpcProviderEndpoint.ChainID),
			utils.LogAttr("APIInterface", rpcps.rpcProviderEndpoint.ApiInterface))
	}

	return chainMessage, nil
}

// function used to handle relay requests from a consumer, it is called by a provider_listener by calling RegisterReceiver
func (rpcps *RPCProviderServer) Relay(ctx context.Context, request *pairingtypes.RelayRequest) (*pairingtypes.RelayReply, error) {
	// get the number of simultaneous relay calls
	currentLoad := rpcps.providerLoadManager.addAndSetRelayLoadToContextTrailer(ctx)
	defer func() {
		// add load metric and subtract the load at the end of the relay using a routine.
		go func() {
			rpcps.providerLoadManager.subtractRelayCall()
			rpcps.metrics.SetLoadRate(currentLoad)
		}()
	}()
	trailerMd := metadata.Pairs(chainlib.RpcProviderUniqueIdHeader, rpcps.providerUniqueId)
	grpc.SetTrailer(ctx, trailerMd)
	if request.RelayData == nil || request.RelaySession == nil {
		return nil, utils.LavaFormatWarning("invalid relay request, internal fields are nil", nil)
	}
	ctx = utils.AppendUniqueIdentifier(ctx, lavaprotocol.GetSalt(request.RelayData))
	startTime := time.Now()
	// This is for the SDK, since the timeout is not automatically added to the request like in Go
	timeout, timeoutFound, err := rpcps.tryGetTimeoutFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	if timeoutFound {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	utils.LavaFormatDebug("Provider got relay request",
		utils.Attribute{Key: "GUID", Value: ctx},
		utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
		utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
		utils.Attribute{Key: "request.cu", Value: request.RelaySession.CuSum},
		utils.Attribute{Key: "relay_timeout", Value: common.GetRemainingTimeoutFromContext(ctx)},
		utils.Attribute{Key: "relay addon", Value: request.RelayData.Addon},
		utils.Attribute{Key: "relay extensions", Value: request.RelayData.GetExtensions()},
		utils.Attribute{Key: "seenBlock", Value: request.RelayData.GetSeenBlock()},
		utils.Attribute{Key: "requestBlock", Value: request.RelayData.GetRequestBlock()},
	)
	// Init relay
	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		utils.LavaFormatDebug("got error from init relay", utils.LogAttr("error", err))
		return nil, rpcps.handleRelayErrorStatus(err)
	}

	// Check that this is not subscription related messages
	if chainlib.IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
		return nil, errors.New("subscribe method is not supported through Relay")
	}

	if chainlib.IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE_ALL) {
		return nil, errors.New("unsubscribe_all method  is not supported through Relay")
	}

	var reply *pairingtypes.RelayReply
	if chainlib.IsFunctionTagOfType(chainMessage, spectypes.FUNCTION_TAG_UNSUBSCRIBE) {
		reply, err = rpcps.TryRelayUnsubscribe(ctx, request, consumerAddress, chainMessage)
	} else {
		// Try sending relay
		reply, err = rpcps.TryRelay(ctx, request, consumerAddress, chainMessage)
	}

	// static provider doesnt handle sessions, so just return the response
	if rpcps.StaticProvider {
		return reply, rpcps.handleRelayErrorStatus(err)
	}

	if err != nil || common.ContextOutOfTime(ctx) {
		// failed to send relay. we need to adjust session state. cuSum and relayNumber.
		relayFailureError := rpcps.providerSessionManager.OnSessionFailure(relaySession, request.RelaySession.RelayNum)
		if relayFailureError != nil {
			var extraInfo string
			if err != nil {
				extraInfo = err.Error()
			}
			err = sdkerrors.Wrapf(relayFailureError, "On relay failure: %s", extraInfo)
		}
		err = utils.LavaFormatError("TryRelay Failed", err,
			utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
			utils.Attribute{Key: "request.userAddr", Value: consumerAddress},
			utils.Attribute{Key: "GUID", Value: ctx},
			utils.Attribute{Key: "timed_out", Value: common.ContextOutOfTime(ctx)},
		)
		go rpcps.metrics.AddError()
	} else {
		// On successful relay
		pairingEpoch := relaySession.PairingEpoch
		sendRewards := relaySession.IsPayingRelay() // when consumer mismatch causes this relay not to provide cu
		replyBlock := reply.LatestBlock
		go rpcps.metrics.AddRelay(consumerAddress.String(), relaySession.LatestRelayCu, request.RelaySession.QosReport)
		relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, request.RelaySession.RelayNum)
		if relayError != nil {
			utils.LavaFormatError("OnSession Done failure: ", relayError)
		} else if sendRewards {
			// SendProof gets the request copy, as in the case of data reliability enabled the request.blockNumber is changed.
			// Therefore the signature changes, so we need the original copy to extract the address from it.
			// we want this code to run in parallel so it doesn't stop the flow

			go rpcps.SendProof(ctx, pairingEpoch, request, consumerAddress, chainMessage.GetApiCollection().CollectionData.ApiInterface)
			utils.LavaFormatDebug("Provider Finished Relay Successfully",
				utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
				utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
				utils.Attribute{Key: "GUID", Value: ctx},
				utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock},
				utils.Attribute{Key: "replyBlock", Value: replyBlock},
				utils.Attribute{Key: "method", Value: chainMessage.GetApi().Name},
			)
		}
	}
	utils.LavaFormatDebug("Provider returned a relay response",
		utils.Attribute{Key: "GUID", Value: ctx},
		utils.Attribute{Key: "request.SessionId", Value: request.RelaySession.SessionId},
		utils.Attribute{Key: "request.relayNumber", Value: request.RelaySession.RelayNum},
		utils.Attribute{Key: "request.cu", Value: request.RelaySession.CuSum},
		utils.Attribute{Key: "relay_timeout", Value: common.GetRemainingTimeoutFromContext(ctx)},
		utils.Attribute{Key: "timeTaken", Value: time.Since(startTime)},
	)
	return reply, rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) initRelay(ctx context.Context, request *pairingtypes.RelayRequest) (relaySession *lavasession.SingleProviderSession, consumerAddress sdk.AccAddress, chainMessage chainlib.ChainMessage, err error) {
	if !rpcps.StaticProvider {
		relaySession, consumerAddress, err = rpcps.verifyRelaySession(ctx, request)
		if err != nil {
			return nil, nil, nil, err
		}
		defer func(relaySession *lavasession.SingleProviderSession) {
			// if we error in here until PrepareSessionForUsage was called successfully we can't call OnSessionFailure
			if err != nil {
				relaySession.DisbandSession()
			}
		}(relaySession) // lock in the session address
	}
	extensionInfo := extensionslib.ExtensionInfo{LatestBlock: 0, ExtensionOverride: request.RelayData.Extensions}
	if extensionInfo.ExtensionOverride == nil { // in case consumer did not set an extension, we skip the extension parsing and we are sending it to the regular url
		extensionInfo.ExtensionOverride = []string{}
	}
	// parse the message to extract the cu and chainMessage for sending it
	chainMessage, err = rpcps.chainParser.ParseMsg(request.RelayData.ApiUrl, request.RelayData.Data, request.RelayData.ConnectionType, request.RelayData.GetMetadata(), extensionInfo)
	if err != nil {
		return nil, nil, nil, err
	}
	// we only need the chainMessage for a static provider
	if rpcps.StaticProvider {
		// extract consumer address from signature
		extractedConsumerAddress, err := rpcps.ExtractConsumerAddress(ctx, request.RelaySession)
		if err != nil {
			return nil, nil, nil, err
		}

		return nil, extractedConsumerAddress, chainMessage, nil
	}
	relayCU := chainMessage.GetApi().ComputeUnits
	virtualEpoch := rpcps.stateTracker.GetVirtualEpoch(uint64(request.RelaySession.Epoch))
	err = relaySession.PrepareSessionForUsage(ctx, relayCU, request.RelaySession.CuSum, rpcps.allowedMissingCUThreshold, virtualEpoch)
	if err != nil {
		// If PrepareSessionForUsage, session lose sync.
		// We then wrap the error with the SessionOutOfSyncError that has a unique error code.
		// The consumer knows the session lost sync using the code and will create a new session.
		return nil, nil, nil, utils.LavaFormatError("Session Out of sync", lavasession.SessionOutOfSyncError, utils.Attribute{Key: "PrepareSessionForUsage_Error", Value: err.Error()}, utils.Attribute{Key: "GUID", Value: ctx})
	}
	return relaySession, consumerAddress, chainMessage, nil
}

func (rpcps *RPCProviderServer) ValidateAddonsExtensions(addon string, extensions []string, chainMessage chainlib.ChainMessage) error {
	// this validates all of the values are handled by chainParser
	_, _, err := rpcps.chainParser.SeparateAddonsExtensions(append(extensions, addon))
	if err != nil {
		return err
	}
	apiCollection := chainMessage.GetApiCollection()
	if apiCollection.CollectionData.AddOn != addon {
		return utils.LavaFormatWarning("invalid addon in relay, parsed addon is not the same as requested", nil, utils.Attribute{Key: "requested addon", Value: addon[0]}, utils.Attribute{Key: "parsed addon", Value: chainMessage.GetApiCollection().CollectionData.AddOn})
	}
	if !rpcps.chainRouter.ExtensionsSupported(apiCollection.CollectionData.InternalPath, extensions) {
		return utils.LavaFormatWarning("requested extensions are unsupported in chainRouter", nil, utils.Attribute{Key: "requested extensions", Value: extensions})
	}
	return nil
}

func (rpcps *RPCProviderServer) ValidateRequest(chainMessage chainlib.ChainMessage, request *pairingtypes.RelayRequest, ctx context.Context) error {
	// TODO: remove this if case, the reason its here is because lava-sdk does't have data reliability + block parsing.
	// this is a temporary solution until we have a working block parsing in lava-sdk
	if request.RelayData.RequestBlock == spectypes.NOT_APPLICABLE {
		return nil
	}
	seenBlock := request.RelayData.GetSeenBlock()
	if seenBlock < 0 {
		return utils.LavaFormatError("invalid seen block", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seenBlock", Value: seenBlock})
	}
	reqBlock, _ := chainMessage.RequestedBlock()
	if reqBlock != request.RelayData.RequestBlock {
		// the consumer either configured an invalid value or is modifying the requested block as part of a data reliability message
		// see if this modification is supported
		providerRequestedBlockPreUpdate := reqBlock
		chainMessage.UpdateLatestBlockInMessage(request.RelayData.RequestBlock, true)
		// if after UpdateLatestBlockInMessage it's not aligned we have a problem
		reqBlock, _ = chainMessage.RequestedBlock()
		if reqBlock != request.RelayData.RequestBlock {
			utils.LavaFormatDebug("requested block mismatch between consumer and provider",
				utils.LogAttr("request data", request.RelayData.Data),
				utils.LogAttr("request path", request.RelayData.ApiUrl),
				utils.LogAttr("method", chainMessage.GetApi().Name),
				utils.Attribute{Key: "provider_parsed_block_pre_update", Value: providerRequestedBlockPreUpdate},
				utils.Attribute{Key: "provider_requested_block", Value: reqBlock},
				utils.Attribute{Key: "consumer_requested_block", Value: request.RelayData.RequestBlock},
				utils.Attribute{Key: "GUID", Value: ctx})
			// TODO, we need to return an error here, this was disabled so relays will pass, but it will cause data reliability issues.
			// once we understand the issue return the error.
			utils.LavaFormatError("requested block mismatch between consumer and provider", nil,
				utils.LogAttr("request data", string(request.RelayData.Data)),
				utils.LogAttr("request path", request.RelayData.ApiUrl),
				utils.LogAttr("method", chainMessage.GetApi().Name),
				utils.Attribute{Key: "provider_parsed_block_pre_update", Value: providerRequestedBlockPreUpdate},
				utils.Attribute{Key: "provider_requested_block", Value: reqBlock},
				utils.Attribute{Key: "consumer_requested_block", Value: request.RelayData.RequestBlock},
				utils.Attribute{Key: "GUID", Value: ctx},
				utils.Attribute{Key: "metadata", Value: request.RelayData.Metadata},
			)
		}
	}
	return nil
}

func (rpcps *RPCProviderServer) RelaySubscribe(request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer) error {
	if request.RelayData == nil || request.RelaySession == nil {
		return utils.LavaFormatError("invalid relay subscribe request, internal fields are nil", nil)
	}

	ctx := utils.AppendUniqueIdentifier(context.Background(), lavaprotocol.GetSalt(request.RelayData))
	utils.LavaFormatDebug("Provider got relay subscribe request",
		utils.LogAttr("request.SessionId", request.RelaySession.SessionId),
		utils.LogAttr("request.relayNumber", request.RelaySession.RelayNum),
		utils.LogAttr("request.cu", request.RelaySession.CuSum),
		utils.LogAttr("GUID", ctx),
	)

	relaySession, consumerAddress, chainMessage, err := rpcps.initRelay(ctx, request)
	if err != nil {
		utils.LavaFormatDebug("got error from init relay", utils.LogAttr("error", err))
		return rpcps.handleRelayErrorStatus(err)
	}

	// TryRelaySubscribe is blocking until subscription ends
	subscribed, err := rpcps.TryRelaySubscribe(ctx, uint64(request.RelaySession.Epoch), request, srv, chainMessage, consumerAddress, relaySession, request.RelaySession.RelayNum)
	if subscribed {
		utils.LavaFormatDebug("Provider Finished Relay Successfully",
			utils.LogAttr("request.SessionId", request.RelaySession.SessionId),
			utils.LogAttr("request.relayNumber", request.RelaySession.RelayNum),
			utils.LogAttr("GUID", ctx),
		)
		err = nil // we don't want to return an error here
	} else {
		// we didn't even manage to subscribe
		err = utils.LavaFormatError("failed subscribing", lavasession.SubscriptionInitiationError,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("error", err),
		)
	}

	return rpcps.handleRelayErrorStatus(err)
}

func (rpcps *RPCProviderServer) SendProof(ctx context.Context, epoch uint64, request *pairingtypes.RelayRequest, consumerAddress sdk.AccAddress, apiInterface string) error {
	storedCU, updatedWithProof := rpcps.rewardServer.SendNewProof(ctx, request.RelaySession, epoch, consumerAddress.String(), apiInterface)
	if !updatedWithProof && storedCU > request.RelaySession.CuSum {
		rpcps.providerSessionManager.UpdateSessionCU(consumerAddress.String(), epoch, request.RelaySession.SessionId, storedCU)
		err := utils.LavaFormatError("Cu in relay smaller than existing proof", lavasession.ProviderConsumerCuMisMatch, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "session_cu_sum", Value: request.RelaySession.CuSum}, utils.Attribute{Key: "existing_proof_cu", Value: storedCU}, utils.Attribute{Key: "sessionId", Value: request.RelaySession.SessionId}, utils.Attribute{Key: "chainID", Value: request.RelaySession.SpecId})
		return rpcps.handleRelayErrorStatus(err)
	}
	return nil
}

func (rpcps *RPCProviderServer) TryRelaySubscribe(ctx context.Context, requestBlockHeight uint64, request *pairingtypes.RelayRequest, srv pairingtypes.Relayer_RelaySubscribeServer, chainMessage chainlib.ChainMessage, consumerAddress sdk.AccAddress, relaySession *lavasession.SingleProviderSession, relayNumber uint64) (subscribedSuccessfully bool, errRet error) {
	subscribeRepliesChan := make(chan *pairingtypes.RelayReply)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	consumerProcessGuid, found := rpcps.fetchConsumerProcessGuidFromContext(srv.Context())
	if !found {
		return false, utils.LavaFormatWarning("Could not find consumer process GUID in context, which is required for subscription relays", nil)
	}

	// The reasons that we have a wait group here, and we pass it to the go routine is because we want to start the channel read before calling AddConsumer,
	// because it might stuck on writing to the channel if we don't do that, which will create a deadlock.
	// But, we still want to wait the go routine to finish before we return (because the gRPC stream will close on return), so we use a wait group to wait for the go routine to finish.
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Process subscription messages
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
			case <-srv.Context().Done():
				utils.LavaFormatTrace("ctx or relay server context closed",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("consumerAddr", consumerAddress),
				)

				err := rpcps.providerNodeSubscriptionManager.RemoveConsumer(ctx, chainMessage, consumerAddress, true, consumerProcessGuid)
				if err != nil {
					errRet = utils.LavaFormatError("Error RemoveConsumer", err, utils.LogAttr("GUID", ctx))
				}
				return
			case subscribeReply, ok := <-subscribeRepliesChan:
				if !ok { // channel is closed
					errRet = utils.LavaFormatTrace("subscribeRepliesChan closed",
						utils.LogAttr("GUID", ctx),
						utils.LogAttr("consumerAddr", consumerAddress),
					)
					err := rpcps.providerNodeSubscriptionManager.RemoveConsumer(ctx, chainMessage, consumerAddress, false, consumerProcessGuid) // false because the channel is already closed
					if err != nil {
						errRet = utils.LavaFormatError("Error RemoveConsumer", err, utils.LogAttr("GUID", ctx))
					}
					return
				}

				errRet = srv.Send(subscribeReply)

				if errRet != nil {
					// usually triggered when client closes connection
					if strings.Contains(errRet.Error(), "Canceled desc = context canceled") {
						errRet = utils.LavaFormatWarning("Client closed connection", errRet, utils.Attribute{Key: "GUID", Value: ctx})
					} else {
						errRet = utils.LavaFormatError("Got error from srv.Send()", errRet, utils.Attribute{Key: "GUID", Value: ctx})
					}

					return
				}

				subscribedSuccessfully = true
				utils.LavaFormatTrace("Sending data to consumer",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("data", subscribeReply.Data),
					utils.LogAttr("consumerAddr", consumerAddress),
				)
			}
		}
	}()

	subscriptionId, err := rpcps.providerNodeSubscriptionManager.AddConsumer(ctx, request, chainMessage, consumerAddress, subscribeRepliesChan, consumerProcessGuid)
	if err != nil {
		// Subscription failed due to node error mark session as done and return
		relayError := rpcps.providerSessionManager.OnSessionFailure(relaySession, relayNumber)
		if relayError != nil {
			utils.LavaFormatError("Error OnSessionDone", relayError)
		}

		return false, utils.LavaFormatWarning("RPCProviderServer: Subscription failed", err,
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddress),
		)
	}

	relayError := rpcps.providerSessionManager.OnSessionDone(relaySession, relayNumber)
	if relayError != nil {
		utils.LavaFormatError("Error OnSessionDone", relayError)
	}

	go rpcps.SendProof(ctx, relaySession.PairingEpoch, request, consumerAddress, chainMessage.GetApiCollection().CollectionData.ApiInterface)

	rpcps.rewardServer.SubscribeStarted(consumerAddress.String(), requestBlockHeight, subscriptionId)
	wg.Wait() // Block until subscription is done

	rpcps.rewardServer.SubscribeEnded(consumerAddress.String(), requestBlockHeight, subscriptionId)
	return subscribedSuccessfully, errRet
}

// verifies basic relay fields, and gets a provider session
func (rpcps *RPCProviderServer) verifyRelaySession(ctx context.Context, request *pairingtypes.RelayRequest) (singleProviderSession *lavasession.SingleProviderSession, extractedConsumerAddress sdk.AccAddress, err error) {
	valid := rpcps.providerSessionManager.IsValidEpoch(uint64(request.RelaySession.Epoch))
	if !valid {
		latestBlock := rpcps.stateTracker.LatestBlock()
		errorMessage := "user reported invalid lava block height"
		if request.RelaySession.Epoch > latestBlock {
			errorMessage = "provider is behind user's block height"
		} else if request.RelaySession.Epoch == 0 {
			errorMessage = "user reported lava block 0, either it's test rpcprovider or a consumer that has no node access"
		}
		utils.LavaFormatInfo(errorMessage,
			utils.Attribute{Key: "Info Type", Value: lavasession.EpochMismatchError},
			utils.Attribute{Key: "current lava block", Value: latestBlock},
			utils.Attribute{Key: "requested lava block", Value: request.RelaySession.Epoch},
			utils.Attribute{Key: "threshold", Value: rpcps.providerSessionManager.GetBlockedEpochHeight()},
			utils.Attribute{Key: "GUID", Value: ctx},
		)
		return nil, nil, lavasession.EpochMismatchError
	}

	// Check data
	err = rpcps.verifyRelayRequestMetaData(ctx, request.RelaySession, request.RelayData)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("did not pass relay validation", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	// check signature
	extractedConsumerAddress, err = rpcps.ExtractConsumerAddress(ctx, request.RelaySession)
	if err != nil {
		return nil, nil, err
	}
	consumerAddressString := extractedConsumerAddress.String()

	// validate & fetch badge to send into provider session manager
	err = rpcps.validateBadgeSession(ctx, request.RelaySession)
	if err != nil {
		return nil, nil, utils.LavaFormatWarning("badge validation err", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	singleProviderSession, err = rpcps.getSingleProviderSession(ctx, request.RelaySession, consumerAddressString)
	return singleProviderSession, extractedConsumerAddress, err
}

func (rpcps *RPCProviderServer) ExtractConsumerAddress(ctx context.Context, relaySession *pairingtypes.RelaySession) (extractedConsumerAddress sdk.AccAddress, err error) {
	if relaySession.Badge != nil {
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(*relaySession.Badge)
		if err != nil {
			return nil, err
		}
	} else {
		extractedConsumerAddress, err = sigs.ExtractSignerAddress(relaySession)
		if err != nil {
			return nil, utils.LavaFormatWarning("failed to extract signer address from relay session", err, utils.LogAttr("GUID", ctx))
		}
	}
	return extractedConsumerAddress, nil
}

func (rpcps *RPCProviderServer) validateBadgeSession(ctx context.Context, relaySession *pairingtypes.RelaySession) error {
	if relaySession.Badge == nil { // not a badge session
		return nil
	}

	// validating badge signer
	badgeUserSigner, err := sigs.ExtractSignerAddress(relaySession)
	if err != nil {
		return utils.LavaFormatWarning("cannot extract badge user from relay", err, utils.LogAttr("GUID", ctx))
	}

	// validating badge signer
	if badgeUserSigner.String() != relaySession.Badge.Address {
		return utils.LavaFormatWarning("did not pass badge signer validation", nil, utils.LogAttr("GUID", ctx))
	}

	// validating badge lavaChainId
	if relaySession.LavaChainId != relaySession.Badge.LavaChainId {
		return utils.LavaFormatWarning("mismatch in badge lavaChainId", nil, utils.LogAttr("GUID", ctx))
	}

	// validating badge epoch
	if int64(relaySession.Badge.Epoch) != relaySession.Epoch {
		return utils.LavaFormatWarning("Badge epoch validation failed", nil,
			utils.LogAttr("badgeEpoch", relaySession.Badge.Epoch),
			utils.LogAttr("relayEpoch", relaySession.Epoch),
		)
	}

	if int64(relaySession.Badge.Epoch) != relaySession.Epoch {
		return utils.LavaFormatWarning("Badge epoch validation failed", nil, utils.LogAttr("badge_epoch", relaySession.Badge.Epoch), utils.LogAttr("relay_epoch", relaySession.Epoch))
	}
	return nil
}

func (rpcps *RPCProviderServer) getSingleProviderSession(ctx context.Context, request *pairingtypes.RelaySession, consumerAddressString string) (*lavasession.SingleProviderSession, error) {
	// regular session, verifies pairing epoch and relay number
	singleProviderSession, err := rpcps.providerSessionManager.GetSession(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, request.Badge)
	if err != nil {
		if lavasession.ConsumerNotRegisteredYet.Is(err) {
			valid, pairedProviders, projectId, verifyPairingError := rpcps.stateTracker.VerifyPairing(ctx, consumerAddressString, rpcps.providerAddress.String(), uint64(request.Epoch), request.SpecId)
			if verifyPairingError != nil {
				return nil, utils.LavaFormatInfo("Failed to VerifyPairing for new consumer",
					utils.Attribute{Key: "Error", Value: verifyPairingError},
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
					utils.Attribute{Key: "Providers block", Value: rpcps.stateTracker.LatestBlock()},
				)
			}
			if !valid {
				return nil, utils.LavaFormatError("VerifyPairing, this consumer address is not valid with this provider", nil,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "epoch", Value: request.Epoch},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
			maxCuForConsumer, getMaxCuError := rpcps.stateTracker.GetMaxCuForUser(ctx, consumerAddressString, request.SpecId, uint64(request.Epoch))
			if getMaxCuError != nil {
				return nil, utils.LavaFormatError("ConsumerNotRegisteredYet: GetMaxCuForUser failed", getMaxCuError,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "epoch", Value: request.Epoch},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "provider", Value: rpcps.providerAddress},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
			// After validating the consumer we can register it with provider session manager.
			singleProviderSession, err = rpcps.providerSessionManager.RegisterProviderSessionWithConsumer(ctx, consumerAddressString, uint64(request.Epoch), request.SessionId, request.RelayNum, maxCuForConsumer, pairedProviders, projectId, request.Badge)
			if err != nil {
				return nil, utils.LavaFormatError("Failed to RegisterProviderSessionWithConsumer", err,
					utils.Attribute{Key: "GUID", Value: ctx},
					utils.Attribute{Key: "sessionID", Value: request.SessionId},
					utils.Attribute{Key: "consumer", Value: consumerAddressString},
					utils.Attribute{Key: "relayNum", Value: request.RelayNum},
				)
			}
		} else {
			return nil, utils.LavaFormatError("Failed to get a provider session", err,
				utils.Attribute{Key: "GUID", Value: ctx},
				utils.Attribute{Key: "sessionID", Value: request.SessionId},
				utils.Attribute{Key: "consumer", Value: consumerAddressString},
				utils.Attribute{Key: "relayNum", Value: request.RelayNum},
			)
		}
	}
	return singleProviderSession, nil
}

func (rpcps *RPCProviderServer) verifyRelayRequestMetaData(ctx context.Context, requestSession *pairingtypes.RelaySession, relayData *pairingtypes.RelayPrivateData) error {
	providerAddress := rpcps.providerAddress.String()
	if requestSession.Provider != providerAddress {
		return utils.LavaFormatError("request had the wrong provider", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "providerAddress", Value: providerAddress}, utils.Attribute{Key: "request_provider", Value: requestSession.Provider})
	}
	if requestSession.SpecId != rpcps.rpcProviderEndpoint.ChainID {
		return utils.LavaFormatError("request had the wrong specID", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "request_specID", Value: requestSession.SpecId}, utils.Attribute{Key: "chainID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	if requestSession.LavaChainId != rpcps.lavaChainID {
		return utils.LavaFormatError("request had the wrong lava chain ID", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "request_lavaChainID", Value: requestSession.LavaChainId}, utils.Attribute{Key: "lava chain id", Value: rpcps.lavaChainID})
	}

	if !bytes.Equal(requestSession.ContentHash, sigs.HashMsg(relayData.GetContentHashData())) {
		return utils.LavaFormatError("content hash mismatch between consumer and provider", nil,
			utils.Attribute{Key: "ApiInterface", Value: relayData.ApiInterface},
			utils.Attribute{Key: "ApiUrl", Value: relayData.ApiUrl},
			utils.Attribute{Key: "RequestBlock", Value: relayData.RequestBlock},
			utils.Attribute{Key: "ConnectionType", Value: relayData.ConnectionType},
			utils.Attribute{Key: "Metadata", Value: relayData.Metadata},
			utils.Attribute{Key: "GUID", Value: ctx})
	}

	return nil
}

func (rpcps *RPCProviderServer) handleRelayErrorStatus(err error) error {
	if err == nil {
		return nil
	}
	if lavasession.SessionOutOfSyncError.Is(err) {
		err = status.Error(codes.Code(lavasession.SessionOutOfSyncError.ABCICode()), err.Error())
	} else if lavasession.EpochMismatchError.Is(err) {
		err = status.Error(codes.Code(lavasession.EpochMismatchError.ABCICode()), err.Error())
	}
	return err
}

func (rpcps *RPCProviderServer) TryRelay(ctx context.Context, request *pairingtypes.RelayRequest, consumerAddr sdk.AccAddress, chainMsg chainlib.ChainMessage) (*pairingtypes.RelayReply, error) {
	errV := rpcps.ValidateRequest(chainMsg, request, ctx)
	if errV != nil {
		return nil, errV
	}

	errV = rpcps.ValidateAddonsExtensions(request.RelayData.Addon, request.RelayData.Extensions, chainMsg)
	if errV != nil {
		return nil, errV
	}

	var latestBlock int64
	var requestedBlockHash []byte
	var requestedHashes []*chaintracker.BlockStore
	var modifiedReqBlock int64

	finalized := false
	updatedChainMessage := false

	dataReliabilityEnabled, _ := rpcps.chainParser.DataReliabilityParams()
	blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData := rpcps.chainParser.ChainBlockStats()
	relayTimeout := chainlib.GetRelayTimeout(chainMsg, averageBlockTime)

	if dataReliabilityEnabled {
		var err error
		latestBlock, requestedBlockHash, requestedHashes, modifiedReqBlock, finalized, updatedChainMessage, err = rpcps.GetParametersForRelayDataReliability(ctx, request, chainMsg, relayTimeout, blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData)
		if err != nil {
			return nil, err
		}
	}

	// TODO: handle cache on fork for dataReliability = false
	var reply *pairingtypes.RelayReply
	var ignoredMetadata []pairingtypes.Metadata
	var err error
	if requestedBlockHash != nil || finalized { // try get reply from cache
		reply, ignoredMetadata, err = rpcps.tryGetRelayReplyFromCache(ctx, request, requestedBlockHash, finalized)
	}

	if err != nil || reply == nil {
		// we need to send relay, cache miss or invalid
		var replyWrapper *chainlib.RelayReplyWrapper
		replyWrapper, err = rpcps.sendRelayMessageToNode(ctx, request, chainMsg, consumerAddr)
		if err != nil {
			return nil, err
		}

		reply = replyWrapper.RelayReply

		reply.Metadata, _, ignoredMetadata = rpcps.chainParser.HandleHeaders(reply.Metadata, chainMsg.GetApiCollection(), spectypes.Header_pass_reply)
		// TODO: use overwriteReqBlock on the reply metadata to set the correct latest block
		if rpcps.cache.CacheActive() && (requestedBlockHash != nil || finalized) {
			rpcps.trySetRelayReplyInCache(ctx, request, chainMsg, replyWrapper, latestBlock, averageBlockTime, requestedBlockHash, finalized, ignoredMetadata)
		}
	} else if len(request.RelayData.Extensions) > 0 {
		// if cached, Add Archive trailer if requested by the consumer.
		grpc.SetTrailer(ctx, metadata.Pairs(chainlib.RPCProviderNodeExtension, lavasession.NewRouterKey(request.RelayData.Extensions).String()))
	}

	if dataReliabilityEnabled {
		err := rpcps.BuildRelayFinalizedBlockHashes(ctx, request, reply, latestBlock, requestedHashes, updatedChainMessage, relayTimeout, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData, modifiedReqBlock)
		if err != nil {
			return nil, err
		}
	}

	// utils.LavaFormatDebug("response signing", utils.LogAttr("request block", request.RelayData.RequestBlock), utils.LogAttr("GUID", ctx), utils.LogAttr("latestBlock", reply.LatestBlock))
	reply, err = lavaprotocol.SignRelayResponse(consumerAddr, *request, rpcps.privKey, reply, dataReliabilityEnabled)
	if err != nil {
		return nil, err
	}

	reply.Metadata = append(reply.Metadata, ignoredMetadata...) // appended here only after signing

	// return reply to user
	return reply, nil
}

func (rpcps *RPCProviderServer) tryGetRelayReplyFromCache(ctx context.Context, request *pairingtypes.RelayRequest, requestedBlockHash []byte, finalized bool) (*pairingtypes.RelayReply, []pairingtypes.Metadata, error) {
	cache := rpcps.cache
	hashKey, outPutFormatter, hashErr := chainlib.HashCacheRequest(request.RelayData, rpcps.rpcProviderEndpoint.ChainID)
	if hashErr != nil {
		utils.LavaFormatError("TryRelay Failed computing hash for cache request", hashErr)
		return nil, nil, nil
	}
	cacheCtx, cancel := context.WithTimeout(ctx, common.CacheTimeout)
	cacheReply, err := cache.GetEntry(cacheCtx, &pairingtypes.RelayCacheGet{
		RequestHash:    hashKey,
		RequestedBlock: request.RelayData.RequestBlock,
		ChainId:        rpcps.rpcProviderEndpoint.ChainID,
		BlockHash:      requestedBlockHash,
		Finalized:      finalized,
		SeenBlock:      request.RelayData.SeenBlock,
	})
	cancel()

	if err != nil && performance.NotConnectedError.Is(err) {
		utils.LavaFormatDebug("cache not connected", utils.LogAttr("err", err), utils.Attribute{Key: "GUID", Value: ctx})
		return nil, nil, err
	}

	reply := cacheReply.GetReply()
	if reply != nil {
		reply.Data = outPutFormatter(reply.Data) // setting request id back to reply.
	}

	ignoredMetadata := cacheReply.GetOptionalMetadata()

	return reply, ignoredMetadata, err
}

func (rpcps *RPCProviderServer) trySetRelayReplyInCache(ctx context.Context, request *pairingtypes.RelayRequest, chainMsg chainlib.ChainMessage, replyWrapper *chainlib.RelayReplyWrapper, latestBlock int64, averageBlockTime time.Duration, requestedBlockHash []byte, finalized bool, ignoredMetadata []pairingtypes.Metadata) {
	cache := rpcps.cache
	reply := replyWrapper.RelayReply

	isNodeError, _ := chainMsg.CheckResponseError(reply.Data, replyWrapper.StatusCode)
	// in case the error is a node error we don't want to cache
	if !isNodeError {
		// copy request and reply as they change later on and we call SetEntry in a routine.
		requestedBlock := request.RelayData.RequestBlock                                                       // get requested block before removing it from the data
		hashKey, _, hashErr := chainlib.HashCacheRequest(request.RelayData, rpcps.rpcProviderEndpoint.ChainID) // get the hash (this changes the data)
		copyReply := &pairingtypes.RelayReply{}
		copyReplyErr := protocopy.DeepCopyProtoObject(reply, copyReply)
		go func() {
			if hashErr != nil || copyReplyErr != nil {
				utils.LavaFormatError("Failed copying relay private data on TryRelay", nil, utils.LogAttr("copyReplyErr", copyReplyErr), utils.LogAttr("hashErr", hashErr))
				return
			}
			new_ctx := context.Background()
			new_ctx, cancel := context.WithTimeout(new_ctx, common.DataReliabilityTimeoutIncrease)
			defer cancel()
			err := cache.SetEntry(new_ctx, &pairingtypes.RelayCacheSet{
				RequestHash:      hashKey,
				RequestedBlock:   requestedBlock,
				BlockHash:        requestedBlockHash,
				ChainId:          rpcps.rpcProviderEndpoint.ChainID,
				Response:         copyReply,
				Finalized:        finalized,
				OptionalMetadata: ignoredMetadata,
				AverageBlockTime: int64(averageBlockTime),
				SeenBlock:        latestBlock,
				IsNodeError:      isNodeError,
			})
			if err != nil && request.RelaySession.Epoch != spectypes.NOT_APPLICABLE {
				utils.LavaFormatWarning("error updating cache with new entry", err, utils.Attribute{Key: "GUID", Value: ctx})
			}
		}()
	}
}

func (rpcps *RPCProviderServer) sendRelayMessageToNode(ctx context.Context, request *pairingtypes.RelayRequest, chainMsg chainlib.ChainMessage, consumerAddr sdk.AccAddress) (*chainlib.RelayReplyWrapper, error) {
	if debugLatency {
		utils.LavaFormatDebug("sending relay to node", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "specID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	// add stickiness header
	chainMsg.AppendHeader([]pairingtypes.Metadata{{Name: RPCProviderStickinessHeaderName, Value: common.GetUniqueToken(common.UserData{DappId: consumerAddr.String(), ConsumerIp: common.GetTokenFromGrpcContext(ctx)})}})
	chainMsg.AppendHeader([]pairingtypes.Metadata{{Name: RPCProviderAddressHeader, Value: rpcps.providerAddress.String()}})
	if debugConsistency {
		utils.LavaFormatDebug("adding stickiness header", utils.LogAttr("tokenFromContext", common.GetTokenFromGrpcContext(ctx)), utils.LogAttr("unique_token", common.GetUniqueToken(common.UserData{DappId: consumerAddr.String(), ConsumerIp: common.GetIpFromGrpcContext(ctx)})))
	}
	// use the provider state machine to send the messages
	return rpcps.providerStateMachine.SendNodeMessage(ctx, chainMsg, request)
}

func (rpcps *RPCProviderServer) TryRelayUnsubscribe(ctx context.Context, request *pairingtypes.RelayRequest, consumerAddress sdk.AccAddress, chainMessage chainlib.ChainMessage) (*pairingtypes.RelayReply, error) {
	errV := rpcps.ValidateRequest(chainMessage, request, ctx)
	if errV != nil {
		return nil, errV
	}

	utils.LavaFormatDebug("Provider got unsubscribe request", utils.LogAttr("GUID", ctx))

	consumerProcessGuid, found := rpcps.fetchConsumerProcessGuidFromContext(ctx)
	if !found {
		return nil, utils.LavaFormatWarning("Could not find consumer process GUID in context, which is required for unsubscribe relays", nil)
	}

	// Remove the consumer from the connected consumers list of the subscription
	err := rpcps.providerNodeSubscriptionManager.RemoveConsumer(ctx, chainMessage, consumerAddress, true, consumerProcessGuid)
	if err != nil {
		return nil, err
	}

	rpcResponse, err := lavaprotocol.CraftEmptyRPCResponseFromGenericMessage(chainMessage.GetRPCMessage())
	if err != nil {
		return nil, utils.LavaFormatError("failed crafting empty rpc response", err)
	}

	dataToSend, err := json.Marshal(rpcResponse)
	if err != nil {
		return nil, utils.LavaFormatError("failed marshaling json response", err)
	}

	reply := &pairingtypes.RelayReply{
		Data: dataToSend,
	}

	dataReliabilityEnabled, _ := rpcps.chainParser.DataReliabilityParams()
	if dataReliabilityEnabled {
		blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData := rpcps.chainParser.ChainBlockStats()
		relayTimeout := chainlib.GetRelayTimeout(chainMessage, averageBlockTime)
		latestBlock, _, requestedHashes, modifiedReqBlock, _, updatedChainMessage, err := rpcps.GetParametersForRelayDataReliability(ctx, request, chainMessage, relayTimeout, blockLagForQosSync, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData)
		if err != nil {
			return nil, err
		}

		err = rpcps.BuildRelayFinalizedBlockHashes(ctx, request, reply, latestBlock, requestedHashes, updatedChainMessage, relayTimeout, averageBlockTime, blockDistanceToFinalization, blocksInFinalizationData, modifiedReqBlock)
		if err != nil {
			return nil, err
		}
	}

	var ignoredMetadata []pairingtypes.Metadata
	reply.Metadata, _, ignoredMetadata = rpcps.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	reply, err = lavaprotocol.SignRelayResponse(consumerAddress, *request, rpcps.privKey, reply, dataReliabilityEnabled)
	if err != nil {
		return nil, err
	}
	reply.Metadata = append(reply.Metadata, ignoredMetadata...) // appended here only after signing

	return reply, nil
}

func (rpcps *RPCProviderServer) GetParametersForRelayDataReliability(
	ctx context.Context,
	request *pairingtypes.RelayRequest,
	chainMsg chainlib.ChainMessage,
	relayTimeout time.Duration,
	blockLagForQosSync int64,
	averageBlockTime time.Duration,
	blockDistanceToFinalization,
	blocksInFinalizationData uint32,
) (latestBlock int64, requestedBlockHash []byte, requestedHashes []*chaintracker.BlockStore, modifiedReqBlock int64, finalized, updatedChainMessage bool, err error) {
	specificBlock := request.RelayData.RequestBlock
	if specificBlock < spectypes.LATEST_BLOCK {
		// cases of EARLIEST, FINALIZED, SAFE
		// GetLatestBlockData only supports latest relative queries or specific block numbers
		specificBlock = spectypes.NOT_APPLICABLE
	}

	// handle consistency, if the consumer requested information we do not have in the state tracker

	latestBlock, requestedHashes, _, err = rpcps.handleConsistency(ctx, relayTimeout, request.RelayData.GetSeenBlock(), request.RelayData.GetRequestBlock(), averageBlockTime, blockLagForQosSync, blockDistanceToFinalization, blocksInFinalizationData)
	if err != nil {
		return 0, nil, nil, 0, false, false, err
	}

	// get specific block data for caching
	_, specificRequestedHashes, _, getLatestBlockErr := rpcps.reliabilityManager.GetLatestBlockData(spectypes.NOT_APPLICABLE, spectypes.NOT_APPLICABLE, specificBlock)
	if getLatestBlockErr == nil && len(specificRequestedHashes) == 1 {
		requestedBlockHash = []byte(specificRequestedHashes[0].Hash)
	}

	// TODO: take latestBlock and lastSeenBlock and put the greater one of them
	updatedChainMessage = chainMsg.UpdateLatestBlockInMessage(latestBlock, true)

	modifiedReqBlock = lavaprotocol.ReplaceRequestedBlock(request.RelayData.RequestBlock, latestBlock)
	if modifiedReqBlock != request.RelayData.RequestBlock {
		request.RelayData.RequestBlock = modifiedReqBlock
		updatedChainMessage = true // meaning we can't bring a newer proof
	}
	// requestedBlockHash, finalizedBlockHashes = chaintracker.FindRequestedBlockHash(requestedHashes, request.RelayData.RequestBlock, toBlock, fromBlock, finalizedBlockHashes)
	finalized = spectypes.IsFinalizedBlock(modifiedReqBlock, latestBlock, int64(blockDistanceToFinalization))
	if !finalized && requestedBlockHash == nil && modifiedReqBlock != spectypes.NOT_APPLICABLE {
		// avoid using cache, but can still service
		utils.LavaFormatWarning("no hash data for requested block", nil, utils.Attribute{Key: "specID", Value: rpcps.rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: request.RelayData.RequestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "modifiedReqBlock", Value: modifiedReqBlock}, utils.Attribute{Key: "specificBlock", Value: specificBlock})
	}

	return latestBlock, requestedBlockHash, requestedHashes, modifiedReqBlock, finalized, updatedChainMessage, nil
}

func (rpcps *RPCProviderServer) BuildRelayFinalizedBlockHashes(
	ctx context.Context,
	request *pairingtypes.RelayRequest,
	reply *pairingtypes.RelayReply,
	latestBlock int64,
	requestedHashes []*chaintracker.BlockStore,
	updatedChainMessage bool,
	relayTimeout time.Duration,
	averageBlockTime time.Duration,
	blockDistanceToFinalization uint32,
	blocksInFinalizationData uint32,
	modifiedReqBlock int64,
) (err error) {
	// now we need to provide the proof for the response
	proofBlock := latestBlock
	if !updatedChainMessage || len(requestedHashes) == 0 {
		// we can fetch a more advanced finalization proof, than we fetched previously
		proofBlock, requestedHashes, _, err = rpcps.GetLatestBlockData(ctx, blockDistanceToFinalization, blocksInFinalizationData)
		if err != nil {
			return err
		}
	} // else: we updated the chain message to request the specific latestBlock we fetched earlier, so use the previously fetched latest block and hashes
	if proofBlock < modifiedReqBlock && proofBlock < request.RelayData.SeenBlock {
		// we requested with a newer block, but don't necessarily have the finaliziation proof, chaintracker might be behind
		proofBlock = lavaslices.Min([]int64{modifiedReqBlock, request.RelayData.SeenBlock})

		proofBlock, requestedHashes, err = rpcps.GetBlockDataForOptimisticFetch(ctx, relayTimeout, proofBlock, blockDistanceToFinalization, blocksInFinalizationData, averageBlockTime)
		if err != nil {
			return utils.LavaFormatError("error getting block range for finalization proof", err)
		}
	}

	finalizedBlockHashes := chaintracker.BuildProofFromBlocks(requestedHashes)
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	if err != nil {
		return utils.LavaFormatError("failed unmarshaling finalizedBlockHashes", err, utils.Attribute{Key: "GUID", Value: ctx},
			utils.Attribute{Key: "finalizedBlockHashes", Value: finalizedBlockHashes}, utils.Attribute{Key: "specID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = proofBlock
	return nil
}

func (rpcps *RPCProviderServer) GetBlockDataForOptimisticFetch(ctx context.Context, relayBaseTimeout time.Duration, requiredProofBlock int64, blockDistanceToFinalization uint32, blocksInFinalizationData uint32, averageBlockTime time.Duration) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error) {
	utils.LavaFormatDebug("getting new blockData for optimistic fetch", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requiredProofBlock", Value: requiredProofBlock})
	proofBlock := requiredProofBlock
	toBlock := proofBlock - int64(blockDistanceToFinalization)
	fromBlock := toBlock - int64(blocksInFinalizationData) + 1
	deadline, ok := ctx.Deadline()
	oneSideTravel := common.AverageWorldLatency / 2
	timeCanWait := time.Until(deadline) - oneSideTravel
	if !ok {
		timeCanWait = 0
	}
	timeSlept := 0 * time.Millisecond
	refreshTime := (averageBlockTime / chaintracker.MostFrequentPollingMultiplier) / 2
	sleepTime := lavaslices.Min([]time.Duration{10 * refreshTime, timeCanWait, relayBaseTimeout / 2})
	sleepContext, cancel := context.WithTimeout(context.Background(), sleepTime)
	fetchedWithoutError := func() bool {
		timeSlept += refreshTime
		proofBlock, requestedHashes, _, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, spectypes.NOT_APPLICABLE)
		return err != nil
	}
	rpcps.SleepUntilTimeOrConditionReached(sleepContext, refreshTime, fetchedWithoutError)
	cancel()

	for err != nil && ok && timeCanWait > refreshTime && timeSlept < 5*refreshTime {
		time.Sleep(refreshTime)

		proofBlock, requestedHashes, _, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, spectypes.NOT_APPLICABLE)
		deadline, ok = ctx.Deadline()
		timeCanWait = time.Until(deadline) - oneSideTravel
	}
	if err != nil {
		return 0, nil, utils.LavaFormatError("error getting block range for optimistic finalization proof", err, utils.Attribute{Key: "refreshTime", Value: refreshTime}, utils.Attribute{Key: "timeCanWait", Value: timeCanWait}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "fromBlock", Value: fromBlock}, utils.Attribute{Key: "requiredProofBlock", Value: requiredProofBlock}, utils.Attribute{Key: "timeWaited", Value: timeSlept}, utils.Attribute{Key: "proofBlock", Value: proofBlock}, utils.Attribute{Key: "toBlock", Value: toBlock}, utils.Attribute{Key: "specID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	return proofBlock, requestedHashes, err
}

func (rpcps *RPCProviderServer) handleConsistency(ctx context.Context, baseRelayTimeout time.Duration, seenBlock int64, requestBlock int64, averageBlockTime time.Duration, blockLagForQosSync int64, blockDistanceToFinalization uint32, blocksInFinalizationData uint32) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, timeSlept time.Duration, err error) {
	latestBlock, requestedHashes, changeTime, err := rpcps.GetLatestBlockData(ctx, blockDistanceToFinalization, blocksInFinalizationData)
	if err != nil {
		return 0, nil, 0, err
	}
	if requestBlock == spectypes.LATEST_BLOCK && seenBlock > latestBlock {
		// we can't just replace requested block here with what we have, it must be with at least seen block
		requestBlock = seenBlock
	}
	if requestBlock <= latestBlock || seenBlock <= latestBlock {
		// requested block is older than our information, or the consumer is asking a future block he has no information about
		return latestBlock, requestedHashes, 0, nil
	}
	// consumer asked for a block that is newer than our state tracker, we cant sign this for DR, calculate wether we should wait and try to update
	blockGap := requestBlock - latestBlock
	if seenBlock < requestBlock {
		// we don't have to wait until we reach requested block for consistency here, we just need to reach the seen block height
		blockGap = seenBlock - latestBlock
	}
	deadline, ok := ctx.Deadline()
	probabilityBlockError := 0.0
	halfTimeLeft := time.Until(deadline) / 2 // giving the node at least half the timeout time to process
	if baseRelayTimeout/2 < halfTimeLeft {
		// do not allow waiting the full timeout since now it's absurdly high
		halfTimeLeft = baseRelayTimeout / 2
	}
	if ok {
		timeProviderHasS := (time.Since(changeTime) + halfTimeLeft).Seconds() // add waiting half the timeout time
		if changeTime.IsZero() {
			// we don't have information on block changes
			timeProviderHasS = halfTimeLeft.Seconds()
		}
		averageBlockTimeS := averageBlockTime.Seconds()
		eventRate := timeProviderHasS / averageBlockTimeS // a new block every average block time, numerator is time we have, gamma=rt
		if eventRate < 0 {
			utils.LavaFormatError("invalid rate params", nil, utils.Attribute{Key: "changeTime", Value: changeTime}, utils.Attribute{Key: "averageBlockTime", Value: averageBlockTime}, utils.Attribute{Key: "eventRate", Value: eventRate}, utils.Attribute{Key: "time", Value: time.Until(deadline)}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: requestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "blockGap", Value: blockGap})
		} else {
			probabilityBlockError = provideroptimizer.CumulativeProbabilityFunctionForPoissonDist(uint64(blockGap-1), eventRate) // this calculates the probability we received insufficient blocks. too few when we don't wait
			if debugConsistency {
				utils.LavaFormatDebug("consistency calculations breakdown", utils.Attribute{Key: "averageBlockTime", Value: averageBlockTime}, utils.Attribute{Key: "eventRate", Value: eventRate}, utils.Attribute{Key: "probabilityBlockError", Value: probabilityBlockError}, utils.Attribute{Key: "time", Value: time.Until(deadline)}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: requestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "blockGap", Value: blockGap})
			}
		}
	}
	// we only bail if there is no chance for the provider to get to the requested block and the consumer has already got a response from a different provider with that block
	if (blockGap > blockLagForQosSync*2 || (blockGap > 1 && probabilityBlockError > 0.4)) && (seenBlock >= latestBlock) {
		return latestBlock, requestedHashes, 0, utils.LavaFormatWarning("Requested a block that is too new", protocolerrors.ConsistencyError, utils.Attribute{Key: "blockGap", Value: blockGap}, utils.Attribute{Key: "probabilityBlockError", Value: probabilityBlockError}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seenBlock", Value: seenBlock}, utils.Attribute{Key: "requestedBlock", Value: requestBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "chainID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}

	if !ok {
		// we didn't get any timeout so we are using a default waiting time
		deadline = time.Now().Add(500 * time.Millisecond)
	}
	// we are waiting for the state tracker to catch up with the requested block
	utils.LavaFormatDebug("waiting for state tracker to update", utils.Attribute{Key: "probabilityBlockError", Value: probabilityBlockError}, utils.Attribute{Key: "time", Value: time.Until(deadline)}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "requestedBlock", Value: requestBlock}, utils.Attribute{Key: "seenBlock", Value: seenBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "blockGap", Value: blockGap})
	sleepContext, cancel := context.WithTimeout(context.Background(), halfTimeLeft)
	getLatestBlock := func() bool {
		ret, _ := rpcps.reliabilityManager.GetLatestBlockNum()
		// if we hit either seen or requested we can return
		return ret >= requestBlock || ret >= seenBlock
	}
	sleptTime := rpcps.SleepUntilTimeOrConditionReached(sleepContext, 50*time.Millisecond, getLatestBlock)
	cancel()
	// see if there is an updated info
	latestBlock, requestedHashes, _, err = rpcps.GetLatestBlockData(ctx, blockDistanceToFinalization, blocksInFinalizationData)
	if err != nil {
		return 0, nil, sleptTime, utils.LavaFormatWarning("delayed fetch failed", err, utils.Attribute{Key: "chainID", Value: rpcps.rpcProviderEndpoint.ChainID})
	}
	if requestBlock > latestBlock && seenBlock > latestBlock {
		// meaning we can't guarantee it will work since chainTracker didn't see this requested block yet
		return 0, nil, sleptTime, utils.LavaFormatWarning("rquested block is too new", nil, utils.Attribute{Key: "sleptTime", Value: sleptTime}, utils.Attribute{Key: "requested", Value: requestBlock}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "chainID", Value: rpcps.rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "seenBlock", Value: seenBlock})
	}
	if debugConsistency {
		utils.LavaFormatDebug("consistency sleep done", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "sleptTime", Value: sleptTime})
	}
	return latestBlock, requestedHashes, sleptTime, nil
}

func (rpcps *RPCProviderServer) SleepUntilTimeOrConditionReached(ctx context.Context, queryTime time.Duration, condition func() bool) (sleepTime time.Duration) {
	sleepTime = 0
	blockReached := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return // Context canceled, exit goroutine
			default:
				var sleeping time.Duration
				deadline, ok := ctx.Deadline()
				if ok {
					sleeping = lavaslices.Min([]time.Duration{queryTime, time.Until(deadline) / 4})
				} else {
					sleeping = queryTime
				}
				sleepTime += sleeping
				time.Sleep(sleeping)
				if condition() {
					close(blockReached) // Signal that the block is reached
					return
				}
			}
		}
	}()

	select {
	case <-blockReached:
		return sleepTime
	case <-ctx.Done():
		return sleepTime
	}
}

func (rpcps *RPCProviderServer) GetLatestBlockData(ctx context.Context, blockDistanceToFinalization uint32, blocksInFinalizationData uint32) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	toBlock := spectypes.LATEST_BLOCK - int64(blockDistanceToFinalization)
	fromBlock := toBlock - int64(blocksInFinalizationData) + 1
	latestBlock, requestedHashes, changeTime, err = rpcps.reliabilityManager.GetLatestBlockData(fromBlock, toBlock, spectypes.NOT_APPLICABLE)
	if err != nil {
		err = utils.LavaFormatError("failed fetching finalization block data", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "fromBlock", Value: fromBlock}, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "toBlock", Value: toBlock})
	}
	return
}

func (rpcps *RPCProviderServer) Probe(ctx context.Context, probeReq *pairingtypes.ProbeRequest) (*pairingtypes.ProbeReply, error) {
	latestB, _ := rpcps.reliabilityManager.GetLatestBlockNum()
	probeReply := &pairingtypes.ProbeReply{
		Guid:                  probeReq.GetGuid(),
		LatestBlock:           latestB,
		FinalizedBlocksHashes: []byte{},
		LavaEpoch:             rpcps.providerSessionManager.GetCurrentEpochAtomic(),
		LavaLatestBlock:       uint64(rpcps.stateTracker.LatestBlock()),
	}
	trailer := metadata.Pairs(common.VersionMetadataKey, upgrade.GetCurrentVersion().ProviderVersion)
	trailer.Append(chainlib.RpcProviderUniqueIdHeader, rpcps.providerUniqueId)
	trailer.Append(common.LavaChainIdMetadataKey, rpcps.lavaChainID)
	grpc.SetTrailer(ctx, trailer) // we ignore this error here since this code can be triggered not from grpc
	return probeReply, nil
}

func (rpcps *RPCProviderServer) fetchConsumerProcessGuidFromContext(ctx context.Context) (string, bool) {
	incomingMetaData, found := metadata.FromIncomingContext(ctx)
	if !found {
		utils.LavaFormatDebug("fetchConsumerProcessGuidFromContext: no incoming meta found in context")
		return "", false
	}
	for key, value := range incomingMetaData {
		if key == common.LAVA_CONSUMER_PROCESS_GUID {
			for _, metaDataValue := range value {
				return metaDataValue, true
			}
		}
	}
	utils.LavaFormatDebug("incoming meta data does not contain process guid", utils.LogAttr("incoming_meta_data", incomingMetaData))
	return "", false
}

func (rpcps *RPCProviderServer) tryGetTimeoutFromRequest(ctx context.Context) (time.Duration, bool, error) {
	incomingMetaData, found := metadata.FromIncomingContext(ctx)
	if !found {
		return 0, false, nil
	}
	for key, listOfMetaDataValues := range incomingMetaData {
		if key == "lava-sdk-relay-timeout" {
			var timeout int64
			var err error
			for _, metaDataValue := range listOfMetaDataValues {
				timeout, err = strconv.ParseInt(metaDataValue, 10, 64)
			}
			if err != nil {
				return 0, false, utils.LavaFormatInfo("invalid relay request, timeout is not a number", utils.Attribute{Key: "error", Value: err})
			}
			if timeout < 0 {
				return 0, false, utils.LavaFormatInfo("invalid relay request, timeout is negative", utils.Attribute{Key: "error", Value: err})
			}
			return time.Duration(timeout) * time.Millisecond, true, nil
		}
	}
	return 0, false, nil
}

func (rpcps *RPCProviderServer) IsHealthy() bool {
	return rpcps.relaysMonitor.IsHealthy()
}
