package chainlib

import (
	"context"
	"encoding/json"
	"sync"

	rpcclient "github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type activeSubscriptionHolder struct {
	firstSubscriptionReply       *pairingtypes.RelayReply
	subscriptionOrigRequest      *pairingtypes.RelayRequest
	subscriptionOrigChainMessage ChainMessage
	replyServer                  *pairingtypes.Relayer_RelaySubscribeClient
	connectedDapps               map[string]chan<- *pairingtypes.RelayReply // key is dapp key
}

type ConsumerWSSubscriptionManager struct {
	activeSubscriptions         map[string]activeSubscriptionHolder // key is params hash
	relaySender                 RelaySender
	consumerSessionManager      *lavasession.ConsumerSessionManager
	chainParser                 ChainParser
	refererData                 *RefererData
	longLastingProvidersStorage *lavasession.LongLastingProvidersStorage
	lock                        sync.RWMutex
}

func NewConsumerWSSubscriptionManager(consumerSessionManager *lavasession.ConsumerSessionManager, relaySender RelaySender, refererData *RefererData, chainParser ChainParser, longLastingProvidersStorage *lavasession.LongLastingProvidersStorage) *ConsumerWSSubscriptionManager {
	return &ConsumerWSSubscriptionManager{
		activeSubscriptions:         make(map[string]activeSubscriptionHolder),
		consumerSessionManager:      consumerSessionManager,
		chainParser:                 chainParser,
		refererData:                 refererData,
		relaySender:                 relaySender,
		longLastingProvidersStorage: longLastingProvidersStorage,
	}
}

func (cwsm *ConsumerWSSubscriptionManager) StartSubscription(webSocketCtx context.Context, chainMessage ChainMessage, directiveHeaders map[string]string, relayRequestData *pairingtypes.RelayPrivateData, dappID, consumerIp string, metricsData *metrics.RelayMetrics, websocketChan chan<- *pairingtypes.RelayReply) (*pairingtypes.RelayReply, error) {
	hashedParams, _, err := cwsm.getHashedParams(chainMessage)
	if err != nil {
		return nil, utils.LavaFormatError("could not marshal params", err)
	}

	dappKey := cwsm.relaySender.CreateSubscriptionKey(dappID, consumerIp)

	// Remove the websocket from the active subscriptions, when the websocket is closed
	go func() {
		<-webSocketCtx.Done()

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		utils.LavaFormatTrace("removing websocket from active subscriptions",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		delete(cwsm.activeSubscriptions[hashedParams].connectedDapps, dappKey)
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

		activeSubscription.connectedDapps[dappKey] = websocketChan
		return activeSubscription.firstSubscriptionReply, nil
	} else {
		utils.LavaFormatTrace("could not find active subscription for given params, creating new one",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("params", chainMessage.GetRPCMessage().GetParams()),
		)

		relayResult, err := cwsm.relaySender.SendParsedRelay(webSocketCtx, dappID, consumerIp, metricsData, chainMessage, directiveHeaders, relayRequestData)
		if err != nil {
			return nil, utils.LavaFormatError("could not send subscription relay", err)
		}

		utils.LavaFormatTrace("got relay result from SendRelay",
			utils.LogAttr("GUID", webSocketCtx),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("relayResult", relayResult),
		)

		replyServer := relayResult.GetReplyServer()
		var reply pairingtypes.RelayReply
		if replyServer != nil { // TODO: Handle nil replyServer
			select {
			case <-(*replyServer).Context().Done():
				utils.LavaFormatTrace("reply server context canceled",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)

				return nil, utils.LavaFormatError("context canceled", nil)
			default:
				err := (*replyServer).RecvMsg(&reply)
				if err != nil {
					return nil, utils.LavaFormatError("could not read reply from reply server", err)
				}

				utils.LavaFormatTrace("successfully got first reply",
					utils.LogAttr("GUID", webSocketCtx),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					utils.LogAttr("reply", reply),
				)

				providerAddr := relayResult.ProviderInfo.ProviderAddress
				cwsm.longLastingProvidersStorage.AddProvider(providerAddr)

				// Need to be run once for subscription
				go cwsm.listenForSubscriptionMessages(dappKey, replyServer, hashedParams, providerAddr)
			}
		} else {
			return nil, utils.LavaFormatTrace("reply server is nil",
				utils.LogAttr("GUID", webSocketCtx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
		}

		cwsm.activeSubscriptions[hashedParams] = activeSubscriptionHolder{
			firstSubscriptionReply:       &reply,
			replyServer:                  replyServer,
			subscriptionOrigRequest:      relayResult.Request,
			subscriptionOrigChainMessage: chainMessage,
			connectedDapps:               map[string]chan<- *pairingtypes.RelayReply{dappKey: websocketChan},
		}

		return &reply, nil
	}
}

func (cwsm *ConsumerWSSubscriptionManager) listenForSubscriptionMessages(dappKey string, replyServer *pairingtypes.Relayer_RelaySubscribeClient, hashedParams string, providerAddr string) {
	defer func() {
		// Only gets here when there is an issue with the connection to the provider or the connection's context is canceled
		// Then, we close all active connections with dapps

		// TODO: Test this with provider subscription timeout

		cwsm.lock.Lock()
		defer cwsm.lock.Unlock()

		utils.LavaFormatTrace("closing all connected dapps for closed subscription connection", utils.LogAttr("params", hashedParams))

		cwsm.longLastingProvidersStorage.RemoveProvider(providerAddr)
		cwsm.relaySender.CancelSubscriptionContext(dappKey)

		for _, activeChan := range cwsm.activeSubscriptions[hashedParams].connectedDapps {
			close(activeChan)
		}

		delete(cwsm.activeSubscriptions, hashedParams)
	}()

	for {
		select {
		case <-(*replyServer).Context().Done():
			utils.LavaFormatTrace("reply server context canceled", utils.LogAttr("params", hashedParams))
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

	filteredHeaders, _, ignoredHeaders := cwsm.chainParser.HandleHeaders(subMsg.Metadata, activeSubscription.subscriptionOrigChainMessage.GetApiCollection(), spectypes.Header_pass_reply)
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

	for _, websocketChannel := range cwsm.activeSubscriptions[hashedParams].connectedDapps {
		websocketChannel <- subMsg
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
