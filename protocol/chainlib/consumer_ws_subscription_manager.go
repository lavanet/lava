package chainlib

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	gojson "github.com/goccy/go-json"
	rpcclient "github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/protocopy"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

type unsubscribeRelayData struct {
	protocolMessage ProtocolMessage
}

type activeSubscriptionHolder struct {
	firstSubscriptionReply                  *pairingtypes.RelayReply
	subscriptionOriginalRequest             *pairingtypes.RelayRequest
	subscriptionOriginalRequestChainMessage ChainMessage
	firstSubscriptionReplyAsJsonrpcMessage  *rpcclient.JsonrpcMessage
	replyServer                             pairingtypes.Relayer_RelaySubscribeClient
	closeSubscriptionChan                   chan *unsubscribeRelayData
	connectedDappKeys                       map[string]struct{} // key is dapp key
	subscriptionId                          string
}

// by using the broadcast manager, we make sure we don't have a race between read and writes and make sure we don't hang for ever
type pendingSubscriptionsBroadcastManager struct {
	broadcastChannelList []chan bool
}

func (psbm *pendingSubscriptionsBroadcastManager) broadcastToChannelList(value bool) {
	for _, ch := range psbm.broadcastChannelList {
		utils.LavaFormatTrace("broadcastToChannelList Notified pending subscriptions", utils.LogAttr("success", value))
		ch <- value
	}
}

type ConsumerWSSubscriptionManager struct {
	connectedDapps                     map[string]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply] // first key is dapp key, second key is hashed params
	activeSubscriptions                map[string]*activeSubscriptionHolder                                      // key is params hash
	relaySender                        RelaySender
	consumerSessionManager             *lavasession.ConsumerSessionManager
	chainParser                        ChainParser
	refererData                        *RefererData
	connectionType                     string
	activeSubscriptionProvidersStorage *lavasession.ActiveSubscriptionProvidersStorage
	currentlyPendingSubscriptions      map[string]*pendingSubscriptionsBroadcastManager
	lock                               sync.RWMutex
	consumerMetricsManager             *metrics.ConsumerMetricsManager
}

func NewConsumerWSSubscriptionManager(
	consumerSessionManager *lavasession.ConsumerSessionManager,
	relaySender RelaySender,
	refererData *RefererData,
	connectionType string,
	chainParser ChainParser,
	activeSubscriptionProvidersStorage *lavasession.ActiveSubscriptionProvidersStorage,
	consumerMetricsManager *metrics.ConsumerMetricsManager,
) *ConsumerWSSubscriptionManager {
	return &ConsumerWSSubscriptionManager{
		connectedDapps:                     make(map[string]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		activeSubscriptions:                make(map[string]*activeSubscriptionHolder),
		currentlyPendingSubscriptions:      make(map[string]*pendingSubscriptionsBroadcastManager),
		consumerSessionManager:             consumerSessionManager,
		chainParser:                        chainParser,
		refererData:                        refererData,
		relaySender:                        relaySender,
		connectionType:                     connectionType,
		activeSubscriptionProvidersStorage: activeSubscriptionProvidersStorage,
		consumerMetricsManager:             consumerMetricsManager,
	}
}

// must be called while locked!
// checking whether hashed params exist in storage, if it does return the subscription stream and indicate it was found.
// otherwise return false
func (cwsm *ConsumerWSSubscriptionManager) checkForActiveSubscriptionAndConnect(webSocketCtx context.Context, hashedParams string, chainMessage ChainMessage, dappKey string, websocketRepliesSafeChannelSender *common.SafeChannelSender[*pairingtypes.RelayReply]) (*pairingtypes.RelayReply, bool) {
	activeSubscription, found := cwsm.activeSubscriptions[hashedParams]
	if found {
		utils.LavaFormatTrace("found active subscription for given params",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)

		if _, ok := activeSubscription.connectedDappKeys[dappKey]; ok {
			utils.LavaFormatTrace("found active subscription for given params and dappKey",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("dappKey", dappKey),
			)

			return activeSubscription.firstSubscriptionReply, true // found and already active
		}

		// Add to existing subscription
		cwsm.connectDappWithSubscription(dappKey, websocketRepliesSafeChannelSender, hashedParams)

		return activeSubscription.firstSubscriptionReply, false // found and not active, register new.
	}
	// not found, need to apply new message
	return nil, false
}

func (cwsm *ConsumerWSSubscriptionManager) failedPendingSubscription(hashedParams string) {
	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()
	pendingSubscriptionChannel, ok := cwsm.currentlyPendingSubscriptions[hashedParams]
	if !ok {
		utils.LavaFormatError("failed fetching hashed params in failedPendingSubscriptions", nil, utils.LogAttr("hash", hashedParams), utils.LogAttr("cwsm.currentlyPendingSubscriptions", cwsm.currentlyPendingSubscriptions))
	} else {
		pendingSubscriptionChannel.broadcastToChannelList(false)
		delete(cwsm.currentlyPendingSubscriptions, hashedParams) // removed pending
	}
}

// must be called under a lock.
func (cwsm *ConsumerWSSubscriptionManager) successfulPendingSubscription(hashedParams string) {
	pendingSubscriptionChannel, ok := cwsm.currentlyPendingSubscriptions[hashedParams]
	if !ok {
		utils.LavaFormatError("failed fetching hashed params in successfulPendingSubscription", nil, utils.LogAttr("hash", hashedParams), utils.LogAttr("cwsm.currentlyPendingSubscriptions", cwsm.currentlyPendingSubscriptions))
	} else {
		pendingSubscriptionChannel.broadcastToChannelList(true)
		delete(cwsm.currentlyPendingSubscriptions, hashedParams) // removed pending
	}
}

func (cwsm *ConsumerWSSubscriptionManager) checkAndAddPendingSubscriptionsWithLock(hashedParams string) (chan bool, bool) {
	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()
	pendingSubscriptionBroadcastManager, ok := cwsm.currentlyPendingSubscriptions[hashedParams]
	if !ok {
		// we didn't find hashed params for pending subscriptions, we can create a new subscription
		utils.LavaFormatTrace("No pending subscription for incoming hashed params found", utils.LogAttr("params", hashedParams))
		// create pending subscription broadcast manager for other users to sync on the same relay.
		cwsm.currentlyPendingSubscriptions[hashedParams] = &pendingSubscriptionsBroadcastManager{}
		return nil, ok
	}
	utils.LavaFormatTrace("found subscription for incoming hashed params, registering our channel", utils.LogAttr("params", hashedParams))
	// by creating a buffered channel we make sure that we wont miss out on the update between the time we register and listen to the channel
	listenChan := make(chan bool, 1)
	pendingSubscriptionBroadcastManager.broadcastChannelList = append(pendingSubscriptionBroadcastManager.broadcastChannelList, listenChan)
	return listenChan, ok
}

func (cwsm *ConsumerWSSubscriptionManager) checkForActiveSubscriptionWithLock(
	webSocketCtx context.Context,
	hashedParams string,
	chainMessage ChainMessage,
	dappKey string,
	websocketRepliesSafeChannelSender *common.SafeChannelSender[*pairingtypes.RelayReply],
	closeWebsocketRepliesChannel func(),
) (*pairingtypes.RelayReply, bool) {
	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	firstSubscriptionReply, alreadyActiveSubscription := cwsm.checkForActiveSubscriptionAndConnect(
		webSocketCtx, hashedParams, chainMessage, dappKey, websocketRepliesSafeChannelSender,
	)

	if firstSubscriptionReply != nil {
		if alreadyActiveSubscription { // same dapp Id, no need for new channel
			closeWebsocketRepliesChannel()
			return firstSubscriptionReply, false
		}
		// Added to existing subscriptions with a new dappId.
		return firstSubscriptionReply, true
	}

	// if we reached here, the subscription is currently not registered, we will need to check again later when we apply the subscription, and
	// handle the case where two identical subscriptions were launched at the same time.
	return nil, false
}

func (cwsm *ConsumerWSSubscriptionManager) StartSubscription(
	webSocketCtx context.Context,
	protocolMessage ProtocolMessage,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) (firstReply *pairingtypes.RelayReply, repliesChan <-chan *pairingtypes.RelayReply, err error) {
	hashedParams, _, err := cwsm.getHashedParams(protocolMessage)
	if err != nil {
		return nil, nil, utils.LavaFormatError("could not marshal params", err)
	}

	dappKey := cwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	utils.LavaFormatTrace("request to start subscription",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
	)

	websocketRepliesChan := make(chan *pairingtypes.RelayReply)
	websocketRepliesSafeChannelSender := common.NewSafeChannelSender(webSocketCtx, websocketRepliesChan)

	closeWebsocketRepliesChan := make(chan struct{})
	closeWebsocketRepliesChannel := func() {
		select {
		case closeWebsocketRepliesChan <- struct{}{}:
		default:
		}
	}

	// called after send relay failure or parsing failure afterwards
	onSubscriptionFailure := func() {
		go cwsm.consumerMetricsManager.SetFailedWsSubscriptionRequestMetric(metricsData.ChainID, metricsData.APIType)
		cwsm.failedPendingSubscription(hashedParams)
		closeWebsocketRepliesChannel()
	}

	go func() {
		<-closeWebsocketRepliesChan
		utils.LavaFormatTrace("requested to close websocketRepliesChan",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)

		websocketRepliesSafeChannelSender.Close()
	}()

	// Remove the websocket from the active subscriptions, when the websocket is closed
	go func() {
		<-webSocketCtx.Done()
		utils.LavaFormatTrace("websocket context is done, removing websocket from active subscriptions",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		if _, ok := cwsm.connectedDapps[dappKey]; ok {
			// The websocket can be closed before the first reply is received, so we need to check if the dapp was even added to the connectedDapps map
			cwsm.verifyAndDisconnectDappFromSubscription(webSocketCtx, dappKey, hashedParams, nil)
		}
		closeWebsocketRepliesChannel()
	}()

	// Validated there are no active subscriptions that we can use.
	firstSubscriptionReply, returnWebsocketRepliesChan := cwsm.checkForActiveSubscriptionWithLock(webSocketCtx, hashedParams, protocolMessage, dappKey, websocketRepliesSafeChannelSender, closeWebsocketRepliesChannel)
	if firstSubscriptionReply != nil {
		go cwsm.consumerMetricsManager.SetDuplicatedWsSubscriptionRequestMetric(metricsData.ChainID, metricsData.APIType)
		if returnWebsocketRepliesChan {
			return firstSubscriptionReply, websocketRepliesChan, nil
		}
		return firstSubscriptionReply, nil, nil
	}

	// This for loop will break when there is a successful queue lock, allowing us to avoid racing new subscription creation when
	// there is a failed subscription. the loop will break for the first routine the manages to lock and create the pendingSubscriptionsBroadcastManager
	for {
		// Incase there are no active subscriptions, check for pending subscriptions with the same hashed params.
		// avoiding having the same subscription twice.
		pendingSubscriptionChannel, foundPendingSubscription := cwsm.checkAndAddPendingSubscriptionsWithLock(hashedParams)
		if foundPendingSubscription {
			utils.LavaFormatTrace("Found pending subscription, waiting for it to complete")
			// this is a buffered channel, it wont get stuck even if it is written to before the time we listen
			res := <-pendingSubscriptionChannel
			utils.LavaFormatTrace("Finished pending for subscription, have results", utils.LogAttr("success", res))
			// Check res is valid, if not fall through logs and try again with a new client.
			if res {
				firstSubscriptionReply, returnWebsocketRepliesChan := cwsm.checkForActiveSubscriptionWithLock(webSocketCtx, hashedParams, protocolMessage, dappKey, websocketRepliesSafeChannelSender, closeWebsocketRepliesChannel)
				if firstSubscriptionReply != nil {
					if returnWebsocketRepliesChan {
						return firstSubscriptionReply, websocketRepliesChan, nil
					}
					return firstSubscriptionReply, nil, nil
				}
				// In case we expected a subscription to return as res != nil we should find an active subscription.
				// If we fail to find it, it might have suddenly stopped. we will log a warning and try with a new client.
				utils.LavaFormatWarning("failed getting a result when channel indicated we got a successful relay", nil)
			}
			// Failed the subscription attempt, will retry using current relay.
			utils.LavaFormatDebug("Failed the subscription attempt, retrying with the incoming message", utils.LogAttr("hash", hashedParams))
		} else {
			utils.LavaFormatDebug("No Pending subscriptions, creating a new one", utils.LogAttr("hash", hashedParams))
			break
		}
	}

	utils.LavaFormatTrace("could not find active subscription for given params, creating new one",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
	)

	relayResult, err := cwsm.relaySender.SendParsedRelay(webSocketCtx, metricsData, protocolMessage)
	if err != nil {
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("could not send subscription relay", err)
	}

	utils.LavaFormatTrace("got relay result from SendParsedRelay",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("relayResult", relayResult),
	)

	replyServer := relayResult.GetReplyServer()
	if replyServer == nil {
		// This code should never be reached, but just in case
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("reply server is nil, probably an error with the subscription initiation", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	reply := *relayResult.Reply
	if reply.Data == nil {
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("Reply data is nil", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	copiedRequest := &pairingtypes.RelayRequest{}
	err = protocopy.DeepCopyProtoObject(relayResult.Request, copiedRequest)
	if err != nil {
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("could not copy relay request", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	err = cwsm.verifySubscriptionMessage(hashedParams, protocolMessage, relayResult.Request, &reply, relayResult.ProviderInfo.ProviderAddress)
	if err != nil {
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("reply", string(reply.Data)),
		)
	}

	// Parse the reply
	var replyJsonrpcMessage rpcclient.JsonrpcMessage
	err = gojson.Unmarshal(reply.Data, &replyJsonrpcMessage)
	if err != nil {
		onSubscriptionFailure()
		return nil, nil, utils.LavaFormatError("could not parse reply into json", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("reply", reply.Data),
		)
	}

	utils.LavaFormatTrace("Adding new subscription",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
	)

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	subscriptionId := protocolMessage.SubscriptionIdExtractor(&replyJsonrpcMessage)
	subscriptionId = common.UnSquareBracket(subscriptionId)
	if common.IsQuoted(subscriptionId) {
		subscriptionId, _ = strconv.Unquote(subscriptionId)
	}

	// we don't have a subscription of this hashedParams stored, create a new one.
	closeSubscriptionChan := make(chan *unsubscribeRelayData)
	cwsm.activeSubscriptions[hashedParams] = &activeSubscriptionHolder{
		firstSubscriptionReply:                  &reply,
		firstSubscriptionReplyAsJsonrpcMessage:  &replyJsonrpcMessage,
		replyServer:                             replyServer,
		subscriptionOriginalRequest:             copiedRequest,
		subscriptionOriginalRequestChainMessage: protocolMessage,
		closeSubscriptionChan:                   closeSubscriptionChan,
		connectedDappKeys:                       map[string]struct{}{dappKey: {}},
		subscriptionId:                          subscriptionId,
	}

	providerAddr := relayResult.ProviderInfo.ProviderAddress
	cwsm.activeSubscriptionProvidersStorage.AddProvider(providerAddr)
	cwsm.connectDappWithSubscription(dappKey, websocketRepliesSafeChannelSender, hashedParams)
	// trigger success for other pending subscriptions
	cwsm.successfulPendingSubscription(hashedParams)
	// Need to be run once for subscription
	go cwsm.listenForSubscriptionMessages(webSocketCtx, dappID, consumerIp, replyServer, hashedParams, providerAddr, metricsData, closeSubscriptionChan)
	go cwsm.consumerMetricsManager.SetWsSubscriptionRequestMetric(metricsData.ChainID, metricsData.APIType)
	return &reply, websocketRepliesChan, nil
}

func (cwsm *ConsumerWSSubscriptionManager) listenForSubscriptionMessages(
	webSocketCtx context.Context,
	dappID string,
	userIp string,
	replyServer pairingtypes.Relayer_RelaySubscribeClient,
	hashedParams string,
	providerAddr string,
	metricsData *metrics.RelayMetrics,
	closeSubscriptionChan chan *unsubscribeRelayData,
) {
	var unsubscribeData *unsubscribeRelayData

	defer func() {
		// Only gets here when there is an issue with the connection to the provider or the connection's context is canceled
		// Then, we close all active connections with dapps

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		utils.LavaFormatTrace("closing all connected dapps for closed subscription connection",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		cwsm.activeSubscriptions[hashedParams].connectedDappKeys = make(map[string]struct{}) // disconnect all dapps at once from active subscription

		// Close all remaining active connections
		for _, connectedDapp := range cwsm.connectedDapps {
			if _, ok := connectedDapp[hashedParams]; ok {
				connectedDapp[hashedParams].Close()
				delete(connectedDapp, hashedParams)
			}
		}

		// we run the unsubscribe flow in an inner function so it wont prevent us from removing the activeSubscriptions at the end.
		func() {
			var err error
			var protocolMessage ProtocolMessage
			if unsubscribeData != nil {
				// This unsubscribe request was initiated by the user
				utils.LavaFormatTrace("unsubscribe request was made by the user",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)

				protocolMessage = unsubscribeData.protocolMessage
			} else {
				// This unsubscribe request was initiated by us
				utils.LavaFormatTrace("unsubscribe request was made automatically",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)

				protocolMessage, err = cwsm.craftUnsubscribeMessage(hashedParams, dappID, userIp, metricsData)
				if err != nil {
					utils.LavaFormatError("could not craft unsubscribe message", err, utils.LogAttr("GUID", webSocketCtx))
					return
				}

				stringJson, err := gojson.Marshal(protocolMessage.GetRPCMessage())
				if err != nil {
					utils.LavaFormatError("could not marshal chain message", err, utils.LogAttr("GUID", webSocketCtx))
					return
				}

				utils.LavaFormatTrace("crafted unsubscribe message to send to the provider",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					utils.LogAttr("chainMessage", string(stringJson)),
				)
			}

			unsubscribeRelayCtx := utils.WithUniqueIdentifier(context.Background(), utils.GenerateUniqueIdentifier())
			err = cwsm.sendUnsubscribeMessage(unsubscribeRelayCtx, protocolMessage, metricsData)
			if err != nil {
				utils.LavaFormatError("could not send unsubscribe message due to a relay error",
					err,
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("relayRequestData", protocolMessage.RelayPrivateData()),
					utils.LogAttr("dappID", dappID),
					utils.LogAttr("userIp", userIp),
					utils.LogAttr("api", protocolMessage.GetApi().Name),
					utils.LogAttr("params", protocolMessage.GetRPCMessage().GetParams()),
				)
			} else {
				utils.LavaFormatTrace("success sending unsubscribe message, deleting hashed params from activeSubscriptions",
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					utils.LogAttr("chainMessage", cwsm.activeSubscriptions),
				)
			}
		}()

		delete(cwsm.activeSubscriptions, hashedParams)
		utils.LavaFormatTrace("after delete")

		cwsm.activeSubscriptionProvidersStorage.RemoveProvider(providerAddr)
		utils.LavaFormatTrace("after remove")
		cwsm.relaySender.CancelSubscriptionContext(hashedParams)
		utils.LavaFormatTrace("after cancel")
	}()

	for {
		select {
		case unsubscribeData = <-closeSubscriptionChan:
			utils.LavaFormatTrace("requested to close subscription connection",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			go cwsm.consumerMetricsManager.SetWsSubscriptioDisconnectRequestMetric(metricsData.ChainID, metricsData.APIType, metrics.WsDisconnectionReasonUser)
			return
		case <-replyServer.Context().Done():
			utils.LavaFormatTrace("reply server context canceled",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			go cwsm.consumerMetricsManager.SetWsSubscriptioDisconnectRequestMetric(metricsData.ChainID, metricsData.APIType, metrics.WsDisconnectionReasonConsumer)
			return
		default:
			var reply pairingtypes.RelayReply
			err := replyServer.RecvMsg(&reply)
			if err != nil {
				// The connection was closed by the provider
				utils.LavaFormatTrace("error reading from subscription stream", utils.LogAttr("original error", err.Error()))
				go cwsm.consumerMetricsManager.SetWsSubscriptioDisconnectRequestMetric(metricsData.ChainID, metricsData.APIType, metrics.WsDisconnectionReasonProvider)
				return
			}
			err = cwsm.handleIncomingSubscriptionNodeMessage(hashedParams, &reply, providerAddr)
			if err != nil {
				utils.LavaFormatError("failed handling subscription message", err,
					utils.LogAttr("hashedParams", hashedParams),
					utils.LogAttr("reply", reply),
				)
				go cwsm.consumerMetricsManager.SetFailedWsSubscriptionRequestMetric(metricsData.ChainID, metricsData.APIType)
				return
			}
		}
	}
}

func (cwsm *ConsumerWSSubscriptionManager) verifySubscriptionMessage(hashedParams string, chainMessage ChainMessage, request *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply, providerAddr string) error {
	lavaprotocol.UpdateRequestedBlock(request.RelayData, reply) // update relay request requestedBlock to the provided one in case it was arbitrary
	filteredHeaders, _, ignoredHeaders := cwsm.chainParser.HandleHeaders(reply.Metadata, chainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	reply.Metadata = filteredHeaders
	err := lavaprotocol.VerifyRelayReply(context.Background(), reply, request, providerAddr)
	if err != nil {
		return utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("subscriptionMsg", reply.Data),
			utils.LogAttr("hashedParams", hashedParams),
			utils.LogAttr("originalRequest", request),
		)
	}

	reply.Metadata = append(reply.Metadata, ignoredHeaders...)
	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) handleIncomingSubscriptionNodeMessage(hashedParams string, subscriptionRelayReplyMsg *pairingtypes.RelayReply, providerAddr string) error {
	cwsm.lock.RLock()
	defer cwsm.lock.RUnlock()

	activeSubscription := cwsm.activeSubscriptions[hashedParams]
	// we need to copy the original message because the verify changes the requested block every time.
	copiedRequest := &pairingtypes.RelayRequest{}
	err := protocopy.DeepCopyProtoObject(activeSubscription.subscriptionOriginalRequest, copiedRequest)
	if err != nil {
		return utils.LavaFormatError("could not copy relay request", err,
			utils.LogAttr("hashedParams", hashedParams),
			utils.LogAttr("subscriptionMsg", subscriptionRelayReplyMsg.Data),
			utils.LogAttr("providerAddr", providerAddr),
		)
	}

	chainMessage := activeSubscription.subscriptionOriginalRequestChainMessage
	err = cwsm.verifySubscriptionMessage(hashedParams, chainMessage, copiedRequest, subscriptionRelayReplyMsg, providerAddr)
	if err != nil {
		// Critical error, we need to close the connection
		return utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("hashedParams", hashedParams),
			utils.LogAttr("subscriptionMsg", subscriptionRelayReplyMsg.Data),
			utils.LogAttr("providerAddr", providerAddr),
		)
	}

	// message is valid, we can now distribute the message to all active listening users.
	for connectedDappKey := range activeSubscription.connectedDappKeys {
		if _, ok := cwsm.connectedDapps[connectedDappKey]; !ok {
			utils.LavaFormatError("connected dapp not found", nil,
				utils.LogAttr("connectedDappKey", connectedDappKey),
				utils.LogAttr("hashedParams", hashedParams),
				utils.LogAttr("activeSubscriptions[hashedParams].connectedDapps", activeSubscription.connectedDappKeys),
				utils.LogAttr("connectedDapps", cwsm.connectedDapps),
			)
			continue
		}

		if _, ok := cwsm.connectedDapps[connectedDappKey][hashedParams]; !ok {
			utils.LavaFormatError("dapp is not connected to subscription", nil,
				utils.LogAttr("connectedDappKey", connectedDappKey),
				utils.LogAttr("hashedParams", hashedParams),
				utils.LogAttr("activeSubscriptions[hashedParams].connectedDapps", activeSubscription.connectedDappKeys),
				utils.LogAttr("connectedDapps[connectedDappKey]", cwsm.connectedDapps[connectedDappKey]),
			)
			continue
		}
		// set consistency seen block
		cwsm.relaySender.SetConsistencySeenBlock(subscriptionRelayReplyMsg.LatestBlock, connectedDappKey)
		// send the reply to the user
		cwsm.connectedDapps[connectedDappKey][hashedParams].Send(subscriptionRelayReplyMsg)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) getHashedParams(chainMessage ChainMessageForSend) (hashedParams string, params []byte, err error) {
	params, err = gojson.Marshal(chainMessage.GetRPCMessage().GetParams())
	if err != nil {
		return "", nil, utils.LavaFormatError("could not marshal params", err)
	}

	hashedParams = rpcclient.CreateHashFromParams(params)

	return hashedParams, params, nil
}

func (cwsm *ConsumerWSSubscriptionManager) Unsubscribe(webSocketCtx context.Context, protocolMessage ProtocolMessage, dappID, consumerIp string, webSocketConnectionUniqueId string, metricsData *metrics.RelayMetrics) error {
	utils.LavaFormatTrace("want to unsubscribe",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
		utils.LogAttr("webSocketConnectionUniqueId", webSocketConnectionUniqueId),
	)

	dappKey := cwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()
	hashedParams, err := cwsm.findActiveSubscriptionHashedParamsFromChainMessage(protocolMessage)
	if err != nil {
		return err
	}
	return cwsm.verifyAndDisconnectDappFromSubscription(webSocketCtx, dappKey, hashedParams, func() (*unsubscribeRelayData, error) {
		return &unsubscribeRelayData{protocolMessage}, nil
	})
}

func (cwsm *ConsumerWSSubscriptionManager) craftUnsubscribeMessage(hashedParams, dappID, consumerIp string, metricsData *metrics.RelayMetrics) (ProtocolMessage, error) {
	request := cwsm.activeSubscriptions[hashedParams].subscriptionOriginalRequestChainMessage
	subscriptionId := cwsm.activeSubscriptions[hashedParams].subscriptionId

	// Craft the message data from function template
	var unsubscribeRequestData string
	var found bool
	for _, currParseDirective := range request.GetApiCollection().ParseDirectives {
		if currParseDirective.FunctionTag == spectypes.FUNCTION_TAG_UNSUBSCRIBE {
			unsubscribeRequestData = fmt.Sprintf(currParseDirective.FunctionTemplate, subscriptionId)
			found = true
			break
		}
	}

	if !found {
		return nil, utils.LavaFormatError("could not find unsubscribe parse directive for given chain message", nil,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("subscriptionId", subscriptionId),
		)
	}

	if unsubscribeRequestData == "" {
		return nil, utils.LavaFormatError("unsubscribe request data is empty", nil,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("subscriptionId", subscriptionId),
		)
	}

	// Craft the unsubscribe chain message
	ctx := context.Background()
	protocolMessage, err := cwsm.relaySender.ParseRelay(ctx, "", unsubscribeRequestData, cwsm.connectionType, dappID, consumerIp, nil)
	if err != nil {
		return nil, utils.LavaFormatError("could not craft unsubscribe chain message", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("subscriptionId", subscriptionId),
			utils.LogAttr("unsubscribeRequestData", unsubscribeRequestData),
			utils.LogAttr("cwsm.connectionType", cwsm.connectionType),
		)
	}

	return protocolMessage, nil
}

func (cwsm *ConsumerWSSubscriptionManager) sendUnsubscribeMessage(ctx context.Context, protocolMessage ProtocolMessage, metricsData *metrics.RelayMetrics) error {
	// Send the crafted unsubscribe relay
	userData := protocolMessage.GetUserData()
	utils.LavaFormatTrace("sending unsubscribe relay",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("dappID", userData.DappId),
		utils.LogAttr("consumerIp", userData.ConsumerIp),
	)

	_, err := cwsm.relaySender.SendParsedRelay(ctx, metricsData, protocolMessage)
	if err != nil {
		return utils.LavaFormatError("could not send unsubscribe relay", err)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) connectDappWithSubscription(dappKey string, webSocketChan *common.SafeChannelSender[*pairingtypes.RelayReply], hashedParams string) {
	// Must be called under a lock

	// Validate hashedParams is in active subscriptions.
	if _, ok := cwsm.activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatError("Failed finding hashed params in connectDappWithSubscription, should never happen", nil, utils.LogAttr("hashedParams", hashedParams), utils.LogAttr("cwsm.activeSubscriptions", cwsm.activeSubscriptions))
		return
	}
	cwsm.activeSubscriptions[hashedParams].connectedDappKeys[dappKey] = struct{}{}
	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		cwsm.connectedDapps[dappKey] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	}
	cwsm.connectedDapps[dappKey][hashedParams] = webSocketChan
}

func (cwsm *ConsumerWSSubscriptionManager) CreateWebSocketConnectionUniqueKey(dappID, consumerIp string, webSocketConnectionUniqueId string) string {
	return cwsm.relaySender.CreateDappKey(common.UserData{DappId: dappID, ConsumerIp: consumerIp}) + "__" + webSocketConnectionUniqueId
}

func (cwsm *ConsumerWSSubscriptionManager) UnsubscribeAll(webSocketCtx context.Context, dappID, consumerIp string, webSocketConnectionUniqueId string, metricsData *metrics.RelayMetrics) error {
	utils.LavaFormatTrace("want to unsubscribe all",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
	)

	dappKey := cwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	// Look for active connection
	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		return utils.LavaFormatDebug("webSocket has no active subscriptions",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("consumerIp", consumerIp),
		)
	}

	for hashedParams := range cwsm.connectedDapps[dappKey] {
		utils.LavaFormatTrace("disconnecting dapp from subscription",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("consumerIp", consumerIp),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		unsubscribeRelayGetter := func() (*unsubscribeRelayData, error) {
			protocolMessage, err := cwsm.craftUnsubscribeMessage(hashedParams, dappID, consumerIp, metricsData)
			if err != nil {
				return nil, err
			}

			return &unsubscribeRelayData{protocolMessage}, nil
		}

		cwsm.verifyAndDisconnectDappFromSubscription(webSocketCtx, dappKey, hashedParams, unsubscribeRelayGetter)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) findActiveSubscriptionHashedParamsFromChainMessage(chainMessage ChainMessage) (string, error) {
	// Must be called under lock

	// Extract the subscription id from the chain message
	unsubscribeRequestParams, err := gojson.Marshal(chainMessage.GetRPCMessage().GetParams())
	if err != nil {
		return "", utils.LavaFormatError("could not marshal params", err, utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()))
	}

	unsubscribeRequestParamsString := string(unsubscribeRequestParams)

	// In JsonRPC, the subscription id is a string, but it is sent in an array
	// In Tendermint, the subscription id is the query params, and sent as an object, so skipped
	unsubscribeRequestParamsString = common.UnSquareBracket(unsubscribeRequestParamsString)

	if common.IsQuoted(unsubscribeRequestParamsString) {
		unsubscribeRequestParamsString, err = strconv.Unquote(unsubscribeRequestParamsString)
		if err != nil {
			return "", utils.LavaFormatError("could not unquote params", err)
		}
	}

	for hashesParams, activeSubscription := range cwsm.activeSubscriptions {
		if activeSubscription.subscriptionId == unsubscribeRequestParamsString {
			return hashesParams, nil
		}
	}

	utils.LavaFormatDebug("could not find active subscription for given params", utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()))

	return "", common.SubscriptionNotFoundError
}

func (cwsm *ConsumerWSSubscriptionManager) verifyAndDisconnectDappFromSubscription(
	webSocketCtx context.Context,
	dappKey string,
	hashedParams string,
	unsubscribeRelayDataGetter func() (*unsubscribeRelayData, error),
) error {
	// Must be called under lock
	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		utils.LavaFormatDebug("webSocket has no active subscriptions",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		return nil
	}

	if _, ok := cwsm.connectedDapps[dappKey][hashedParams]; !ok {
		utils.LavaFormatDebug("no active subscription found for given dapp",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("cwsm.connectedDapps", cwsm.connectedDapps),
		)

		return nil
	}

	if _, ok := cwsm.activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatError("no active subscription found, but the subscription is found in connectedDapps, this should never happen", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		return common.SubscriptionNotFoundError
	}

	if _, ok := cwsm.activeSubscriptions[hashedParams].connectedDappKeys[dappKey]; !ok {
		utils.LavaFormatError("active subscription found, but the dappKey is not found in it's connectedDapps, this should never happen", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		return common.SubscriptionNotFoundError
	}

	cwsm.connectedDapps[dappKey][hashedParams].Close() // close the subscription msgs channel

	delete(cwsm.connectedDapps[dappKey], hashedParams)
	utils.LavaFormatTrace("deleted hashedParams from connected dapp's active subscriptions",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("connectedDappActiveSubs", cwsm.connectedDapps[dappKey]),
	)

	if len(cwsm.connectedDapps[dappKey]) == 0 {
		delete(cwsm.connectedDapps, dappKey)
		utils.LavaFormatTrace("deleted dappKey from connected dapps",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	delete(cwsm.activeSubscriptions[hashedParams].connectedDappKeys, dappKey)
	utils.LavaFormatTrace("deleted dappKey from active subscriptions",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("activeSubConnectedDapps", cwsm.activeSubscriptions[hashedParams].connectedDappKeys),
	)

	if len(cwsm.activeSubscriptions[hashedParams].connectedDappKeys) == 0 {
		// No more dapps are connected, close the subscription with provider
		utils.LavaFormatTrace("no more dapps are connected to subscription, closing subscription",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		var unsubscribeData *unsubscribeRelayData

		if unsubscribeRelayDataGetter != nil {
			var err error
			unsubscribeData, err = unsubscribeRelayDataGetter()
			if err != nil {
				return utils.LavaFormatError("got error from getUnsubscribeRelay function", err,
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("dappKey", dappKey),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)
			}
		}

		// Close subscription with provider
		go func() {
			// In a go routine because the reading routine is also locking on new messages from the node
			// So we need to release the lock here, and let the last message be sent, and then the channel will be released
			cwsm.activeSubscriptions[hashedParams].closeSubscriptionChan <- unsubscribeData
		}()
	}

	return nil
}
