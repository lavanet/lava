package chainlib

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	rpcclient "github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/protocopy"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type unsubscribeRelayData struct {
	chainMessage     ChainMessage
	directiveHeaders map[string]string
	relayRequestData *pairingtypes.RelayPrivateData
}

type activeSubscriptionHolder struct {
	firstSubscriptionReply              *pairingtypes.RelayReply
	subscriptionOrigRequest             *pairingtypes.RelayRequest
	subscriptionOrigRequestChainMessage ChainMessage
	subscriptionFirstReply              *rpcclient.JsonrpcMessage
	replyServer                         pairingtypes.Relayer_RelaySubscribeClient
	closeSubscriptionChan               chan *unsubscribeRelayData
	connectedDapps                      map[string]struct{} // key is dapp key
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
	unsubscribeParamsExtractor         func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string
	lock                               sync.RWMutex
}

func NewConsumerWSSubscriptionManager(
	consumerSessionManager *lavasession.ConsumerSessionManager,
	relaySender RelaySender,
	refererData *RefererData,
	connectionType string,
	chainParser ChainParser,
	activeSubscriptionProvidersStorage *lavasession.ActiveSubscriptionProvidersStorage,
	unsubscribeParamsExtractor func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string,
) *ConsumerWSSubscriptionManager {
	return &ConsumerWSSubscriptionManager{
		connectedDapps:                     make(map[string]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		activeSubscriptions:                make(map[string]*activeSubscriptionHolder),
		consumerSessionManager:             consumerSessionManager,
		chainParser:                        chainParser,
		refererData:                        refererData,
		relaySender:                        relaySender,
		connectionType:                     connectionType,
		activeSubscriptionProvidersStorage: activeSubscriptionProvidersStorage,
		unsubscribeParamsExtractor:         unsubscribeParamsExtractor,
	}
}

func (cwsm *ConsumerWSSubscriptionManager) StartSubscription(
	webSocketCtx context.Context,
	chainMessage ChainMessage,
	directiveHeaders map[string]string,
	relayRequestData *pairingtypes.RelayPrivateData,
	dappID string,
	consumerIp string,
	metricsData *metrics.RelayMetrics,
) (firstReply *pairingtypes.RelayReply, repliesChan <-chan *pairingtypes.RelayReply, err error) {
	hashedParams, _, err := cwsm.getHashedParams(chainMessage)
	if err != nil {
		return nil, nil, utils.LavaFormatError("could not marshal params", err)
	}

	dappKey := cwsm.relaySender.CreateDappKey(dappID, consumerIp)

	utils.LavaFormatTrace("request to start subscription",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("connectedDapps", cwsm.connectedDapps),
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

	go func() {
		<-closeWebsocketRepliesChan
		utils.LavaFormatTrace("requested to close websocketRepliesChan",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
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
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
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

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	activeSubscription, found := cwsm.activeSubscriptions[hashedParams]
	if found {
		utils.LavaFormatTrace("found active subscription for given params",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)

		if _, ok := activeSubscription.connectedDapps[dappKey]; ok {
			utils.LavaFormatTrace("found active subscription for given params and dappKey",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("dappKey", dappKey),
			)

			closeWebsocketRepliesChannel()
			return activeSubscription.firstSubscriptionReply, nil, nil
		}

		// Add to existing subscription
		cwsm.connectDappWithSubscription(dappKey, websocketRepliesSafeChannelSender, hashedParams)

		return activeSubscription.firstSubscriptionReply, websocketRepliesChan, nil
	}

	utils.LavaFormatTrace("could not find active subscription for given params, creating new one",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
	)

	relayResult, err := cwsm.relaySender.SendParsedRelay(webSocketCtx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
	if err != nil {
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("could not send subscription relay", err)
	}

	utils.LavaFormatTrace("got relay result from SendParsedRelay",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("relayResult", relayResult),
	)

	replyServer := relayResult.GetReplyServer()
	if replyServer == nil {
		// This code should never be reached, but just in case
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("reply server is nil, probably an error with the subscription initiation", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	reply := *relayResult.Reply
	if reply.Data == nil {
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("Reply data is nil", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	copiedRequest := &pairingtypes.RelayRequest{}
	err = protocopy.DeepCopyProtoObject(relayResult.Request, copiedRequest)
	if err != nil {
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("could not copy relay request", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
		)
	}

	err = cwsm.verifySubscriptionMessage(hashedParams, chainMessage, relayResult.Request, &reply, relayResult.ProviderInfo.ProviderAddress)
	if err != nil {
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("reply", string(reply.Data)),
		)
	}

	// Parse the reply
	var replyJson rpcclient.JsonrpcMessage
	err = json.Unmarshal(reply.Data, &replyJson)
	if err != nil {
		closeWebsocketRepliesChannel()
		return nil, nil, utils.LavaFormatError("could not parse reply into json", err,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
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

	closeSubscriptionChan := make(chan *unsubscribeRelayData)
	cwsm.activeSubscriptions[hashedParams] = &activeSubscriptionHolder{
		firstSubscriptionReply:              &reply,
		replyServer:                         replyServer,
		subscriptionOrigRequest:             copiedRequest,
		subscriptionOrigRequestChainMessage: chainMessage,
		subscriptionFirstReply:              &replyJson,
		closeSubscriptionChan:               closeSubscriptionChan,
		connectedDapps:                      map[string]struct{}{dappKey: {}},
	}

	providerAddr := relayResult.ProviderInfo.ProviderAddress
	cwsm.activeSubscriptionProvidersStorage.AddProvider(providerAddr)
	cwsm.connectDappWithSubscription(dappKey, websocketRepliesSafeChannelSender, hashedParams)

	// Need to be run once for subscription
	go cwsm.listenForSubscriptionMessages(webSocketCtx, dappID, consumerIp, replyServer, hashedParams, providerAddr, metricsData, closeSubscriptionChan)

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

		cwsm.activeSubscriptions[hashedParams].connectedDapps = make(map[string]struct{}) // disconnect all dapps at once from active subscription

		// Close all remaining active connections
		for _, connectedDapp := range cwsm.connectedDapps {
			if _, ok := connectedDapp[hashedParams]; ok {
				connectedDapp[hashedParams].Close()
				delete(connectedDapp, hashedParams)
			}
		}

		var err error
		var chainMessage ChainMessage
		var directiveHeaders map[string]string
		var relayRequestData *pairingtypes.RelayPrivateData

		if unsubscribeData != nil {
			// This unsubscribe request was initiated by the user
			utils.LavaFormatTrace("unsubscribe request was made by the user",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)

			chainMessage = unsubscribeData.chainMessage
			directiveHeaders = unsubscribeData.directiveHeaders
			relayRequestData = unsubscribeData.relayRequestData
		} else {
			// This unsubscribe request was initiated by us
			utils.LavaFormatTrace("unsubscribe request was made automatically",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)

			chainMessage, directiveHeaders, relayRequestData, err = cwsm.craftUnsubscribeMessage(hashedParams, dappID, userIp, metricsData)
			if err != nil {
				utils.LavaFormatError("could not craft unsubscribe message", err, utils.LogAttr("GUID", webSocketCtx))
				return
			}

			stringJson, err := json.Marshal(chainMessage.GetRPCMessage())
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
		err = cwsm.sendUnsubscribeMessage(unsubscribeRelayCtx, dappID, userIp, chainMessage, directiveHeaders, relayRequestData, metricsData)
		if err != nil {
			utils.LavaFormatError("could not send unsubscribe message", err, utils.LogAttr("GUID", webSocketCtx))
		}

		delete(cwsm.activeSubscriptions, hashedParams)

		cwsm.activeSubscriptionProvidersStorage.RemoveProvider(providerAddr)
		cwsm.relaySender.CancelSubscriptionContext(hashedParams)
	}()

	for {
		select {
		case unsubscribeData = <-closeSubscriptionChan:
			utils.LavaFormatTrace("requested to close subscription connection",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			return
		case <-replyServer.Context().Done():
			utils.LavaFormatTrace("reply server context canceled",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			return
		default:
			var reply pairingtypes.RelayReply
			err := replyServer.RecvMsg(&reply)
			if err != nil {
				// The connection was closed by the provider
				utils.LavaFormatTrace("error reading from subscription stream", utils.LogAttr("original error", err.Error()))
				return
			}

			// TODO: Elad - Test this with 2 consumers and 1 provider

			err = cwsm.handleSubscriptionNodeMessage(hashedParams, &reply, providerAddr)
			if err != nil {
				utils.LavaFormatError("failed handling subscription message", err,
					utils.LogAttr("hashedParams", hashedParams),
					utils.LogAttr("reply", reply),
				)
				return
			}
		}
	}
}

func (cwsm *ConsumerWSSubscriptionManager) verifySubscriptionMessage(hashedParams string, chainMessage ChainMessage, request *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply, providerAddr string) error {
	// Must be called under lock
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

func (cwsm *ConsumerWSSubscriptionManager) handleSubscriptionNodeMessage(hashedParams string, subMsg *pairingtypes.RelayReply, providerAddr string) error {
	cwsm.lock.RLock()
	defer cwsm.lock.RUnlock()

	activeSubscription := cwsm.activeSubscriptions[hashedParams]
	copiedRequest := &pairingtypes.RelayRequest{}
	err := protocopy.DeepCopyProtoObject(activeSubscription.subscriptionOrigRequest, copiedRequest)
	if err != nil {
		return utils.LavaFormatError("could not copy relay request", err,
			utils.LogAttr("hashedParams", hashedParams),
			utils.LogAttr("subscriptionMsg", subMsg.Data),
			utils.LogAttr("providerAddr", providerAddr),
		)
	}

	chainMessage := activeSubscription.subscriptionOrigRequestChainMessage
	err = cwsm.verifySubscriptionMessage(hashedParams, chainMessage, copiedRequest, subMsg, providerAddr)
	if err != nil {
		// Critical error, we need to close the connection
		return utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("hashedParams", hashedParams),
			utils.LogAttr("subscriptionMsg", subMsg.Data),
			utils.LogAttr("providerAddr", providerAddr),
		)
	}

	for connectedDappKey := range activeSubscription.connectedDapps {
		if _, ok := cwsm.connectedDapps[connectedDappKey]; !ok {
			utils.LavaFormatError("connected dapp not found", nil,
				utils.LogAttr("connectedDappKey", connectedDappKey),
				utils.LogAttr("hashedParams", hashedParams),
				utils.LogAttr("activeSubscriptions[hashedParams].connectedDapps", activeSubscription.connectedDapps),
				utils.LogAttr("connectedDapps", cwsm.connectedDapps),
			)
			continue
		}

		if _, ok := cwsm.connectedDapps[connectedDappKey][hashedParams]; !ok {
			utils.LavaFormatError("dapp is not connected to subscription", nil,
				utils.LogAttr("connectedDappKey", connectedDappKey),
				utils.LogAttr("hashedParams", hashedParams),
				utils.LogAttr("activeSubscriptions[hashedParams].connectedDapps", activeSubscription.connectedDapps),
				utils.LogAttr("connectedDapps[connectedDappKey]", cwsm.connectedDapps[connectedDappKey]),
			)
			continue
		}

		cwsm.connectedDapps[connectedDappKey][hashedParams].Send(subMsg)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) getHashedParams(chainMessage ChainMessageForSend) (hashedParams string, params []byte, err error) {
	params, err = json.Marshal(chainMessage.GetRPCMessage().GetParams())
	if err != nil {
		return "", nil, utils.LavaFormatError("could not marshal params", err)
	}

	hashedParams = rpcclient.CreateHashFromParams(params)

	return hashedParams, params, nil
}

func (cwsm *ConsumerWSSubscriptionManager) Unsubscribe(webSocketCtx context.Context, chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData, dappID, consumerIp string, metricsData *metrics.RelayMetrics) error {
	utils.LavaFormatTrace("want to unsubscribe",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
	)

	hashedParams, _, err := cwsm.getHashedParams(chainMessage)
	if err != nil {
		return utils.LavaFormatError("could not marshal params", err)
	}

	dappKey := cwsm.relaySender.CreateDappKey(dappID, consumerIp)

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	return cwsm.verifyAndDisconnectDappFromSubscription(webSocketCtx, dappKey, hashedParams, func() (*unsubscribeRelayData, error) {
		return &unsubscribeRelayData{chainMessage, directiveHeaders, relayRequestData}, nil
	})
}

func (cwsm *ConsumerWSSubscriptionManager) craftUnsubscribeMessage(hashedParams, dappID, consumerIp string, metricsData *metrics.RelayMetrics) (ChainMessage, map[string]string, *pairingtypes.RelayPrivateData, error) {
	request := cwsm.activeSubscriptions[hashedParams].subscriptionOrigRequestChainMessage
	reply := cwsm.activeSubscriptions[hashedParams].subscriptionFirstReply

	// Get the unsubscribe params
	unsubscribeParams := cwsm.unsubscribeParamsExtractor(request, reply)
	utils.LavaFormatTrace("extracted unsubscribe params of subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("unsubscribeParams", unsubscribeParams),
	)

	if unsubscribeParams == "" {
		utils.LavaFormatWarning("unsubscribe params are empty", nil,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
	}

	// Craft the message data from function template
	var unsubscribeRequestData string
	var found bool
	for _, currParseDirective := range request.GetApiCollection().ParseDirectives {
		if currParseDirective.FunctionTag == spectypes.FUNCTION_TAG_UNSUBSCRIBE {
			unsubscribeRequestData = fmt.Sprintf(currParseDirective.FunctionTemplate, unsubscribeParams)
			found = true
			break
		}
	}

	if !found {
		return nil, nil, nil, utils.LavaFormatError("could not find unsubscribe parse directive for given chain message", nil,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("unsubscribeParams", unsubscribeParams),
		)
	}

	if unsubscribeRequestData == "" {
		return nil, nil, nil, utils.LavaFormatError("unsubscribe request data is empty", nil,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("unsubscribeParams", unsubscribeParams),
		)
	}

	// Craft the unsubscribe chain message
	ctx := context.Background()
	chainMessage, directiveHeaders, relayRequestData, err := cwsm.relaySender.ParseRelay(ctx, "", unsubscribeRequestData, cwsm.connectionType, dappID, consumerIp, metricsData, nil)
	if err != nil {
		return nil, nil, nil, utils.LavaFormatError("could not craft unsubscribe chain message", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("unsubscribeParams", unsubscribeParams),
			utils.LogAttr("unsubscribeRequestData", unsubscribeRequestData),
			utils.LogAttr("cwsm.connectionType", cwsm.connectionType),
		)
	}

	return chainMessage, directiveHeaders, relayRequestData, nil
}

func (cwsm *ConsumerWSSubscriptionManager) sendUnsubscribeMessage(ctx context.Context, dappID, consumerIp string, chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData, metricsData *metrics.RelayMetrics) error {
	// Send the crafted unsubscribe relay
	utils.LavaFormatTrace("sending unsubscribe relay",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
	)

	_, err := cwsm.relaySender.SendParsedRelay(ctx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
	if err != nil {
		return utils.LavaFormatError("could not send unsubscribe relay", err)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) connectDappWithSubscription(dappKey string, webSocketChan *common.SafeChannelSender[*pairingtypes.RelayReply], hashedParams string) {
	// Must be called under a lock

	cwsm.activeSubscriptions[hashedParams].connectedDapps[dappKey] = struct{}{}
	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		cwsm.connectedDapps[dappKey] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	}
	cwsm.connectedDapps[dappKey][hashedParams] = webSocketChan
}

func (cwsm *ConsumerWSSubscriptionManager) UnsubscribeAll(webSocketCtx context.Context, dappID, consumerIp string, metricsData *metrics.RelayMetrics) error {
	utils.LavaFormatTrace("want to unsubscribe all",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappID", dappID),
		utils.LogAttr("consumerIp", consumerIp),
	)

	dappKey := cwsm.relaySender.CreateDappKey(dappID, consumerIp)

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
			chainMessage, directiveHeaders, relayRequestData, err := cwsm.craftUnsubscribeMessage(hashedParams, dappID, consumerIp, metricsData)
			if err != nil {
				return nil, err
			}

			return &unsubscribeRelayData{chainMessage, directiveHeaders, relayRequestData}, nil
		}

		cwsm.verifyAndDisconnectDappFromSubscription(webSocketCtx, dappKey, hashedParams, unsubscribeRelayGetter)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) verifyAndDisconnectDappFromSubscription(
	webSocketCtx context.Context,
	dappKey string,
	hashedParams string,
	unsubscribeRelayDataGetter func() (*unsubscribeRelayData, error),
) error {
	// Must be called under lock

	sendSubscriptionNotFoundErrorToWebSocket := func() error {
		jsonError, err := json.Marshal(common.JsonRpcSubscriptionNotFoundError)
		if err != nil {
			return utils.LavaFormatError("could not marshal error response", err)
		}

		cwsm.connectedDapps[dappKey][hashedParams].Send(&pairingtypes.RelayReply{Data: jsonError})
		return nil
	}

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
		)

		return nil
	}

	if _, ok := cwsm.activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatError("no active subscription found, but the subscription is found in connectedDapps, this should never happen", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		err := sendSubscriptionNotFoundErrorToWebSocket()
		if err != nil {
			return err
		}
		return nil
	}

	if _, ok := cwsm.activeSubscriptions[hashedParams].connectedDapps[dappKey]; !ok {
		utils.LavaFormatError("active subscription found, but the dappKey is found in it's connectedDapps, this should never happen", nil,
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		err := sendSubscriptionNotFoundErrorToWebSocket()
		if err != nil {
			return err
		}
		return nil
	}

	cwsm.connectedDapps[dappKey][hashedParams].Close() // close the subscription msgs channel

	delete(cwsm.connectedDapps[dappKey], hashedParams)
	utils.LavaFormatTrace("deleted hashedParams from connected dapp's active subscriptions",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("connectedDappActiveSubs", cwsm.connectedDapps[dappKey]),
	)

	delete(cwsm.activeSubscriptions[hashedParams].connectedDapps, dappKey)
	utils.LavaFormatTrace("deleted dappKey from active subscriptions",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("activeSubConnectedDapps", cwsm.activeSubscriptions[hashedParams].connectedDapps),
	)

	if len(cwsm.activeSubscriptions[hashedParams].connectedDapps) == 0 {
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
