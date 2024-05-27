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
	replyServer                         *pairingtypes.Relayer_RelaySubscribeClient
	closeSubscriptionChan               chan *unsubscribeRelayData
	connectedDapps                      map[string]struct{} // key is dapp key
}

type connectedDapp struct {
	activeSubscriptions map[string]struct{} // key is hashed params
	webSocketChannel    chan<- *pairingtypes.RelayReply
}

type ConsumerWSSubscriptionManager struct {
	connectedDapps              map[string]*connectedDapp            // key is dapp key
	activeSubscriptions         map[string]*activeSubscriptionHolder // key is params hash
	relaySender                 RelaySender
	consumerSessionManager      *lavasession.ConsumerSessionManager
	chainParser                 ChainParser
	refererData                 *RefererData
	connectionType              string
	longLastingProvidersStorage *lavasession.LongLastingProvidersStorage
	unsubscribeParamsExtractor  func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string
	lock                        sync.RWMutex
}

func NewConsumerWSSubscriptionManager(
	consumerSessionManager *lavasession.ConsumerSessionManager,
	relaySender RelaySender,
	refererData *RefererData,
	connectionType string,
	chainParser ChainParser,
	longLastingProvidersStorage *lavasession.LongLastingProvidersStorage,
	unsubscribeParamsExtractor func(request ChainMessage, reply *rpcclient.JsonrpcMessage) string,
) *ConsumerWSSubscriptionManager {
	return &ConsumerWSSubscriptionManager{
		connectedDapps:              make(map[string]*connectedDapp),
		activeSubscriptions:         make(map[string]*activeSubscriptionHolder),
		consumerSessionManager:      consumerSessionManager,
		chainParser:                 chainParser,
		refererData:                 refererData,
		relaySender:                 relaySender,
		connectionType:              connectionType,
		longLastingProvidersStorage: longLastingProvidersStorage,
		unsubscribeParamsExtractor:  unsubscribeParamsExtractor,
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

	// Remove the websocket from the active subscriptions, when the websocket is closed
	go func() {
		<-webSocketCtx.Done()

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		utils.LavaFormatTrace("websocket context is done, removing websocket from active subscriptions",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		if _, ok := cwsm.connectedDapps[dappKey]; ok {
			// The websocket can be closed before the first reply is received, so we need to check if the dapp was even added to the connectedDapps map
			cwsm.connectedDapps[dappKey].activeSubscriptions = make(map[string]struct{}) // reset connected subscriptions, so it'll be deleted completely
			cwsm.disconnectDappWithSubscription(webSocketCtx, dappKey, hashedParams, nil)
		}

		close(websocketRepliesChan)
	}()

	cwsm.lock.Lock()
	defer cwsm.lock.Unlock()

	activeSubscription, found := cwsm.activeSubscriptions[hashedParams]
	if found {
		// Add to existing subscription
		utils.LavaFormatTrace("found active subscription for given params",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		cwsm.connectDappWithSubscription(dappKey, websocketRepliesChan, hashedParams)

		return activeSubscription.firstSubscriptionReply, websocketRepliesChan, nil
	}

	utils.LavaFormatTrace("could not find active subscription for given params, creating new one",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
	)

	relayResult, err := cwsm.relaySender.SendParsedRelay(webSocketCtx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
	if err != nil {
		return nil, nil, utils.LavaFormatError("could not send subscription relay", err)
	}

	utils.LavaFormatTrace("got relay result from SendRelay",
		utils.LogAttr("GUID", webSocketCtx),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("relayResult", relayResult),
	)

	replyServer := relayResult.GetReplyServer()
	var reply pairingtypes.RelayReply
	if replyServer == nil { // TODO: Handle nil replyServer
		return nil, nil, utils.LavaFormatTrace("reply server is nil",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
	}

	select {
	case <-(*replyServer).Context().Done(): // Make sure the reply server is open
		utils.LavaFormatTrace("reply server context canceled",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		return nil, nil, utils.LavaFormatError("context canceled", nil)
	default:
		err := (*replyServer).RecvMsg(&reply)
		if err != nil {
			return nil, nil, utils.LavaFormatError("could not read reply from reply server", err)
		}

		utils.LavaFormatTrace("successfully got first reply",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("reply", reply),
		)
	}

	providerAddr := relayResult.ProviderInfo.ProviderAddress
	cwsm.longLastingProvidersStorage.AddProvider(providerAddr)

	// Parse the reply
	var replyJson rpcclient.JsonrpcMessage
	err = json.Unmarshal(reply.Data, &replyJson)
	if err != nil {
		return nil, nil, utils.LavaFormatError("could not parse reply into json", err, utils.LogAttr("reply", reply.Data))
	}

	closeSubscriptionChan := make(chan *unsubscribeRelayData)
	cwsm.activeSubscriptions[hashedParams] = &activeSubscriptionHolder{
		firstSubscriptionReply:              &reply,
		replyServer:                         replyServer,
		subscriptionOrigRequest:             relayResult.Request,
		subscriptionOrigRequestChainMessage: chainMessage,
		subscriptionFirstReply:              &replyJson,
		closeSubscriptionChan:               closeSubscriptionChan,
		connectedDapps:                      map[string]struct{}{dappKey: {}},
	}

	cwsm.connectDappWithSubscription(dappKey, websocketRepliesChan, hashedParams)

	// Need to be run once for subscription
	go cwsm.listenForSubscriptionMessages(webSocketCtx, dappID, consumerIp, replyServer, hashedParams, providerAddr, metricsData, closeSubscriptionChan)

	return &reply, websocketRepliesChan, nil

}

func (cwsm *ConsumerWSSubscriptionManager) listenForSubscriptionMessages(
	webSocketCtx context.Context,
	dappID string,
	consumerIp string,
	replyServer *pairingtypes.Relayer_RelaySubscribeClient,
	hashedParams string,
	providerAddr string,
	metricsData *metrics.RelayMetrics,
	closeSubscriptionChan chan *unsubscribeRelayData,
) {

	var unsubscribeData *unsubscribeRelayData

	defer func() {
		// Only gets here when there is an issue with the connection to the provider or the connection's context is canceled
		// Then, we close all active connections with dapps

		// TODO: Test this with provider subscription timeout

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		utils.LavaFormatTrace("closing all connected dapps for closed subscription connection",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		cwsm.activeSubscriptions[hashedParams].connectedDapps = make(map[string]struct{}) // disconnect all dapps at once from active subscription

		// Close all remaining active connections
		for _, connectedDapp := range cwsm.connectedDapps {
			delete(connectedDapp.activeSubscriptions, hashedParams)
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

			chainMessage, directiveHeaders, relayRequestData, err = cwsm.craftUnsubscribeMessage(hashedParams, dappID, consumerIp, metricsData)
			if err != nil {
				utils.LavaFormatError("could not craft unsubscribe message", err)
				return
			}

			stringJson, err := json.Marshal(chainMessage.GetRPCMessage())
			if err != nil {
				utils.LavaFormatError("could not marshal chain message", err)
				return
			}

			utils.LavaFormatTrace("crafted unsubscribe message to send to the provider",
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("chainMessage", string(stringJson)),
			)
		}

		err = cwsm.sendUnsubscribeMessage(context.Background(), dappID, consumerIp, chainMessage, directiveHeaders, relayRequestData, metricsData)
		if err != nil {
			utils.LavaFormatError("could not send unsubscribe message", err, utils.LogAttr("GUID", webSocketCtx))
		}

		delete(cwsm.activeSubscriptions, hashedParams)

		cwsm.longLastingProvidersStorage.RemoveProvider(providerAddr)
		cwsm.relaySender.CancelSubscriptionContext(hashedParams)
	}()

	for {
		select {
		case unsubscribeData = <-closeSubscriptionChan:
			utils.LavaFormatTrace("requested to close subscription connection", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
			return
		case <-(*replyServer).Context().Done():
			utils.LavaFormatTrace("reply server context canceled", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
			return
		default:
			var reply pairingtypes.RelayReply
			err := (*replyServer).RecvMsg(&reply)
			if err != nil {
				// TODO: handle error better
				utils.LavaFormatTrace("error reading from subscription stream", utils.LogAttr("original error", err.Error()))
				return
			}

			cwsm.handleSubscriptionNodeMessage(hashedParams, &reply, providerAddr)
		}
	}
}

func (cwsm *ConsumerWSSubscriptionManager) handleSubscriptionNodeMessage(hashedParams string, subMsg *pairingtypes.RelayReply, providerAddr string) {
	cwsm.lock.RLock()
	defer cwsm.lock.RUnlock()

	activeSubscription := cwsm.activeSubscriptions[hashedParams]

	filteredHeaders, _, ignoredHeaders := cwsm.chainParser.HandleHeaders(subMsg.Metadata, activeSubscription.subscriptionOrigRequestChainMessage.GetApiCollection(), spectypes.Header_pass_reply)
	subMsg.Metadata = filteredHeaders
	err := lavaprotocol.VerifyRelayReply(context.Background(), subMsg, activeSubscription.subscriptionOrigRequest, providerAddr)
	if err != nil {
		utils.LavaFormatError("Failed VerifyRelayReply on subscription message", err,
			utils.LogAttr("subMsg", subMsg),
			utils.LogAttr("originalRequest", activeSubscription.subscriptionOrigRequest),
		)
		return
	}

	subMsg.Metadata = append(subMsg.Metadata, ignoredHeaders...)

	for connectedDappKey := range cwsm.activeSubscriptions[hashedParams].connectedDapps {
		if _, ok := cwsm.connectedDapps[connectedDappKey]; !ok {
			utils.LavaFormatError("connected dapp not found", nil,
				utils.LogAttr("connectedDappKey", connectedDappKey),
				utils.LogAttr("hashedParams", hashedParams),
				utils.LogAttr("activeSubscriptions[hashedParams].connectedDapps", cwsm.activeSubscriptions[hashedParams].connectedDapps),
				utils.LogAttr("connectedDapps", cwsm.connectedDapps),
			)
			continue
		}

		cwsm.connectedDapps[connectedDappKey].webSocketChannel <- subMsg
	}
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

	// Look for active connection
	if _, ok := cwsm.activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatDebug("no active subscription found",
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("consumerIp", consumerIp),
		)

		jsonError, err := json.Marshal(common.JsonRpcSubscriptionNotFoundError)
		if err != nil {
			return utils.LavaFormatError("could not marshal error response", err)
		}

		cwsm.connectedDapps[dappKey].webSocketChannel <- &pairingtypes.RelayReply{Data: jsonError}
		return nil
	}

	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		utils.LavaFormatDebug("dapp is not connected",
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("consumerIp", consumerIp),
		)

		jsonError, err := json.Marshal(common.JsonRpcSubscriptionNotFoundError)
		if err != nil {
			return utils.LavaFormatError("could not marshal error response", err)
		}

		cwsm.connectedDapps[dappKey].webSocketChannel <- &pairingtypes.RelayReply{Data: jsonError}
		return nil
	}

	if _, ok := cwsm.connectedDapps[dappKey].activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatDebug("no active subscription found for given dapp",
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("consumerIp", consumerIp),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		jsonError, err := json.Marshal(common.JsonRpcSubscriptionNotFoundError)
		if err != nil {
			return utils.LavaFormatError("could not marshal error response", err)
		}

		cwsm.connectedDapps[dappKey].webSocketChannel <- &pairingtypes.RelayReply{Data: jsonError}
		return nil
	}

	// Remove the websocket from the active subscriptions, when the websocket is closed
	cwsm.disconnectDappWithSubscription(webSocketCtx, dappKey, hashedParams, &unsubscribeRelayData{chainMessage, directiveHeaders, relayRequestData})
	return nil
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
	_, err := cwsm.relaySender.SendParsedRelay(ctx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
	if err != nil {
		return utils.LavaFormatError("could not send unsubscribe relay", err)
	}

	return nil
}

func (cwsm *ConsumerWSSubscriptionManager) connectDappWithSubscription(dappKey string, webSocketChan chan<- *pairingtypes.RelayReply, hashedParams string) {
	// Must be called under a lock

	cwsm.activeSubscriptions[hashedParams].connectedDapps[dappKey] = struct{}{}
	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		cwsm.connectedDapps[dappKey] = &connectedDapp{
			activeSubscriptions: map[string]struct{}{hashedParams: {}},
			webSocketChannel:    webSocketChan,
		}
	} else {
		cwsm.connectedDapps[dappKey].activeSubscriptions[hashedParams] = struct{}{}
	}
}

func (cwsm *ConsumerWSSubscriptionManager) disconnectDappWithSubscription(ctx context.Context, dappKey string, hashedParams string, unsubscribeRelayData *unsubscribeRelayData) {
	// Must be called under a lock

	if _, ok := cwsm.connectedDapps[dappKey]; !ok {
		utils.LavaFormatDebug("dapp is not connected",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		return
	}

	if _, ok := cwsm.activeSubscriptions[hashedParams]; !ok {
		utils.LavaFormatDebug("no active subscription found for given hashedParams",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		return
	}

	delete(cwsm.connectedDapps[dappKey].activeSubscriptions, hashedParams)
	utils.LavaFormatTrace("deleted hashedParams from connected dapp's active subscriptions",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("connectedDappActiveSubs", cwsm.connectedDapps[dappKey].activeSubscriptions),
	)

	delete(cwsm.activeSubscriptions[hashedParams].connectedDapps, dappKey)
	utils.LavaFormatTrace("deleted dappKey from active subscriptions",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("dappKey", dappKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("activeSubConnectedDapps", cwsm.activeSubscriptions[hashedParams].connectedDapps),
	)

	if len(cwsm.activeSubscriptions[hashedParams].connectedDapps) == 0 {
		// No more dapps are connected, close the subscription with provider
		utils.LavaFormatTrace("no more dapps are connected to subscription, closing subscription",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("dappKey", dappKey),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		// Close subscription with provider
		go func() {
			// In a go routine because the reading routine is also locking on new messages from the node
			// So we need to release the lock here, and let the last message be sent, and then the channel will be released
			cwsm.activeSubscriptions[hashedParams].closeSubscriptionChan <- unsubscribeRelayData
		}()
	}
}

