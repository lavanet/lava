package rpcprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type activeSubscription struct {
	cancellableContext           context.Context
	cancellableContextCancelFunc context.CancelFunc
	messagesChannel              chan interface{}
	nodeSubscription             *rpcclient.ClientSubscription
	subscriptionID               string
	firstSetupReply              *pairingtypes.RelayReply
	firstSetupRequest            *pairingtypes.RelayRequest
	apiCollection                *spectypes.ApiCollection
	connectedConsumers           map[uint64]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply] // first key is epoch, second key is consumer address
}

type ProviderNodeSubscriptionManager struct {
	chainRouter         chainlib.ChainRouter
	chainParser         chainlib.ChainParser
	activeSubscriptions map[string]*activeSubscription // key is request params hash
	currentEpoch        uint64
	prevEpoch           uint64
	privKey             *btcec.PrivateKey
	lock                sync.RWMutex
}

func NewProviderNodeSubscriptionManager(chainRouter chainlib.ChainRouter, chainParser chainlib.ChainParser, currentEpoch, prevEpoch uint64, privKey *btcec.PrivateKey) *ProviderNodeSubscriptionManager {
	utils.LavaFormatTrace("NewProviderNodeSubscriptionManager", utils.LogAttr("prevEpoch", prevEpoch), utils.LogAttr("currentEpoch", currentEpoch))
	return &ProviderNodeSubscriptionManager{
		chainRouter:         chainRouter,
		chainParser:         chainParser,
		activeSubscriptions: make(map[string]*activeSubscription),
		currentEpoch:        currentEpoch,
		prevEpoch:           prevEpoch,
		privKey:             privKey,
	}
}

func (pnsm *ProviderNodeSubscriptionManager) AddConsumer(ctx context.Context, request *pairingtypes.RelayRequest, chainMessage chainlib.ChainMessageForSend, consumerAddr sdk.AccAddress, consumerChannel chan<- *pairingtypes.RelayReply) (clientSubscription *rpcclient.ClientSubscription, subscriptionId string, err error) {
	utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() called", utils.LogAttr("consumerAddr", consumerAddr))

	if pnsm == nil {
		return nil, "", fmt.Errorf("ProviderNodeSubscriptionManager is nil")
	}

	hashedParams, params, err := pnsm.getHashedParams(chainMessage)
	if err != nil {
		return nil, "", err
	}

	utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() hashed params",
		utils.LogAttr("params", string(params)),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	consumerAddrString := consumerAddr.String()

	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	if pnsm.prevEpoch == pnsm.currentEpoch {
		utils.LavaFormatWarning("AddConsumer() called with prevEpoch == currentEpoch", nil, utils.LogAttr("prevEpoch", pnsm.prevEpoch), utils.LogAttr("currentEpoch", pnsm.currentEpoch))
		return nil, "", nil
	}

	var firstSetupReply *pairingtypes.RelayReply

	paramsChannelToConnectedConsumers, found := pnsm.activeSubscriptions[hashedParams]
	if found {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() found existing subscription", utils.LogAttr("params", string(params)))

		if _, ok := paramsChannelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch]; !ok { // Add the new epoch if doesn't exist
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer registered for epoch that does not exists in map, creating the new epoch map", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
		} else if _, found := paramsChannelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch][consumerAddrString]; found { // Consumer is already connected to this subscription in current epoch, dismiss
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer is already connected, returning the existing subscription", utils.LogAttr("consumerAddr", consumerAddr))
			return paramsChannelToConnectedConsumers.nodeSubscription, paramsChannelToConnectedConsumers.subscriptionID, nil
		}

		// Check if the consumer already exist in prev epoch for the same reqParamHash
		connectedConsumerChan, found := paramsChannelToConnectedConsumers.connectedConsumers[pnsm.prevEpoch][consumerAddrString]
		if found {
			// Same consumer is renewing the subscription for the new epoch, let's move him to the current one
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer already exist in prev epoch, moving him to the current one", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch][consumerAddrString] = connectedConsumerChan
			delete(paramsChannelToConnectedConsumers.connectedConsumers[pnsm.prevEpoch], consumerAddrString) // Remove the entry from old epoch
		} else {
			// Add the new entry for the consumer
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer registered for new epoch, the consumer does not exist in prev epoch", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch][consumerAddrString] = common.NewSafeChannelSender(ctx, consumerChannel)
		}

		firstSetupReply = paramsChannelToConnectedConsumers.firstSetupReply
		clientSubscription = paramsChannelToConnectedConsumers.nodeSubscription
		subscriptionId = paramsChannelToConnectedConsumers.subscriptionID
	} else {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() did not found existing subscription, creating new one")

		nodeChan := make(chan interface{})

		replyWrapper, subscriptionId, clientSubscription, _, _, err := pnsm.chainRouter.SendNodeMsg(ctx, nodeChan, chainMessage, nil)
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() subscription reply received",
			utils.LogAttr("replyWrapper", replyWrapper),
			utils.LogAttr("subscriptionId", subscriptionId),
			utils.LogAttr("clientSubscription", clientSubscription),
			utils.LogAttr("err", err),
		)
		if err != nil {
			return nil, "", utils.LavaFormatError("ProviderNodeSubscriptionManager: Subscription failed", err, utils.LogAttr("GUID", ctx), utils.LogAttr("params", params))
		}

		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return nil, "", utils.LavaFormatError("Subscription failed, relayWrapper or RelayReply are nil", nil, utils.LogAttr("GUID", ctx))
		}

		reply := replyWrapper.RelayReply

		err = pnsm.signReply(reply, consumerAddr, request, chainMessage.GetApiCollection())
		if err != nil {
			return nil, "", err
		}

		if clientSubscription == nil {
			// failed subscription, but not an error. (probably a node error)

			// Send the first message to the consumer, so it can handle the error
			SafeChannelSender := common.NewSafeChannelSender(ctx, consumerChannel)
			SafeChannelSender.Send(reply)

			return nil, "", utils.LavaFormatWarning("subscription failed, node error", nil, utils.LogAttr("GUID", ctx), utils.LogAttr("reply", reply))
		}

		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() subscription successful",
			utils.LogAttr("subscriptionId", subscriptionId),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("firstSetupReplyData", reply.Data),
		)

		cancellableCtx, cancel := context.WithCancel(context.Background())
		channelToConnectedConsumers := &activeSubscription{
			cancellableContext:           cancellableCtx,
			cancellableContextCancelFunc: cancel,
			messagesChannel:              nodeChan,
			nodeSubscription:             clientSubscription,
			subscriptionID:               subscriptionId,
			firstSetupReply:              reply,
			firstSetupRequest:            request,
			apiCollection:                chainMessage.GetApiCollection(),
			connectedConsumers:           make(map[uint64]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		}

		channelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
		channelToConnectedConsumers.connectedConsumers[pnsm.currentEpoch][consumerAddrString] = common.NewSafeChannelSender(ctx, consumerChannel)

		pnsm.activeSubscriptions[hashedParams] = channelToConnectedConsumers
		firstSetupReply = reply

		go pnsm.listenForSubscriptionMessages(ctx, nodeChan, hashedParams)
	}

	// Send the first setup message to the consumer in a go routine because the blocking listening for this channel happens after this function
	pnsm.activeSubscriptions[hashedParams].connectedConsumers[pnsm.currentEpoch][consumerAddrString].Send(firstSetupReply)

	return clientSubscription, subscriptionId, nil
}

func (pnsm *ProviderNodeSubscriptionManager) listenForSubscriptionMessages(ctx context.Context, nodeChan chan interface{}, hashedParams string) {
	utils.LavaFormatTrace("Inside ProviderNodeSubscriptionManager:startListeningForSubscription()", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
	defer utils.LavaFormatTrace("Leaving ProviderNodeSubscriptionManager:startListeningForSubscription()", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))

	ticker := time.NewTicker(15 * time.Minute) // Set a time limit of 15 minutes for the subscription
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() timeout reached, ending subscription", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
			pnsm.closeNodeSubscription(hashedParams)
			return
		case <-pnsm.activeSubscriptions[hashedParams].cancellableContext.Done():
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() subscription context is done, ending subscription", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
			pnsm.closeNodeSubscription(hashedParams)
			return
		case <-ctx.Done():
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() context done, exiting", utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
			return
		case nodeMsg := <-nodeChan:
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() got new message from node, ending subscription",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("nodeMsg", nodeMsg),
			)
			pnsm.handleNewNodeMessage(hashedParams, nodeMsg)
		}
	}
}

func (pnsm *ProviderNodeSubscriptionManager) getHashedParams(chainMessage chainlib.ChainMessageForSend) (hashedParams string, params []byte, err error) {
	rpcInputMessage := chainMessage.GetRPCMessage()
	params, err = json.Marshal(rpcInputMessage.GetParams())
	if err != nil {
		return "", nil, utils.LavaFormatError("could not marshal params", err)
	}

	hashedParams = rpcclient.CreateHashFromParams(params)
	return hashedParams, params, nil
}

func (pnsm *ProviderNodeSubscriptionManager) convertNodeMsgToMarshalledJsonRpcResponse(data interface{}, apiCollection *spectypes.ApiCollection) ([]byte, error) {
	msg, ok := data.(*rpcclient.JsonrpcMessage)
	if !ok {
		return nil, fmt.Errorf("data is not a *rpcclient.JsonrpcMessage, but: %T", data)
	}

	var convertedMsg any
	var err error

	switch apiCollection.GetCollectionData().ApiInterface {
	case spectypes.APIInterfaceJsonRPC:
		convertedMsg, err = rpcInterfaceMessages.ConvertJsonRPCMsg(msg)
		if err != nil {
			return nil, err
		}
	case spectypes.APIInterfaceTendermintRPC:
		convertedMsg, err = rpcInterfaceMessages.ConvertTendermintMsg(msg)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported API interface: %s", apiCollection.GetCollectionData().ApiInterface)
	}

	marshalledMsg, err := json.Marshal(convertedMsg)
	if err != nil {
		return nil, err
	}

	return marshalledMsg, nil
}

func (pnsm *ProviderNodeSubscriptionManager) signReply(reply *pairingtypes.RelayReply, consumerAddr sdk.AccAddress, request *pairingtypes.RelayRequest, apiCollection *spectypes.ApiCollection) error {
	var ignoredMetadata []pairingtypes.Metadata
	reply.Metadata, _, ignoredMetadata = pnsm.chainParser.HandleHeaders(reply.Metadata, apiCollection, spectypes.Header_pass_reply)

	dataReliabilityEnabled, _ := pnsm.chainParser.DataReliabilityParams()

	reply, err := lavaprotocol.SignRelayResponse(consumerAddr, *request, pnsm.privKey, reply, dataReliabilityEnabled)
	if err != nil {
		return err
	}

	reply.Metadata = append(reply.Metadata, ignoredMetadata...) // appended here only after signing
	return nil
}

func (pnsm *ProviderNodeSubscriptionManager) handleNewNodeMessage(hashedParams string, nodeMsg interface{}) {
	pnsm.lock.RLock()
	defer pnsm.lock.RUnlock()

	// Sending message to all connected consumers, for all epochs
	for epoch, consumerAddrToChannel := range pnsm.activeSubscriptions[hashedParams].connectedConsumers {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() sending to consumers in epoch", utils.LogAttr("epoch", epoch), utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))

		for consumerAddrString, consumerChannel := range consumerAddrToChannel {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() sending to consumer", utils.LogAttr("consumerAddr", consumerAddrString), utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))

			request := pnsm.activeSubscriptions[hashedParams].firstSetupRequest
			apiCollection := pnsm.activeSubscriptions[hashedParams].apiCollection

			marshalledNodeMsg, err := pnsm.convertNodeMsgToMarshalledJsonRpcResponse(nodeMsg, apiCollection)
			if err != nil {
				utils.LavaFormatError("error converting node message", err)
				return
			}

			relayMessageFromNode := &pairingtypes.RelayReply{
				Data:     marshalledNodeMsg,
				Metadata: []pairingtypes.Metadata{},
			}

			consumerAddr, err := sdk.AccAddressFromBech32(consumerAddrString)
			if err != nil {
				utils.LavaFormatError("error unmarshaling consumer address", err)
				return
			}

			err = pnsm.signReply(relayMessageFromNode, consumerAddr, request, apiCollection)
			if err != nil {
				utils.LavaFormatError("error signing reply", err)
				return
			}

			consumerChannel.Send(relayMessageFromNode)
		}
	}
}

func (pnsm *ProviderNodeSubscriptionManager) RemoveConsumer(ctx context.Context, chainMessage chainlib.ChainMessageForSend, consumerAddr sdk.AccAddress, closeConsumerChannel bool) error {
	if pnsm == nil {
		return nil
	}

	hashedParams, params, err := pnsm.getHashedParams(chainMessage)
	if err != nil {
		return err
	}

	utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() requested to remove consumer from subscription",
		utils.LogAttr("GUID", ctx),
		utils.LogAttr("consumerAddr", consumerAddr),
		utils.LogAttr("params", params),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	consumerAddrString := consumerAddr.String()

	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	openSubscriptions, ok := pnsm.activeSubscriptions[hashedParams]
	if !ok {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no subscription found for params, subscription is already closed",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		return nil
	}

	// Remove consumer from all epochs
	for epoch, connectedConsumers := range openSubscriptions.connectedConsumers {
		if _, ok := connectedConsumers[consumerAddrString]; ok {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() found consumer in epoch",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("consumerAddr", consumerAddr),
				utils.LogAttr("params", params),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("connectedConsumers", connectedConsumers),
			)

			if closeConsumerChannel {
				utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() closing consumer channel",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("consumerAddr", consumerAddr),
					utils.LogAttr("params", params),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					utils.LogAttr("epoch", epoch),
				)
				connectedConsumers[consumerAddrString].Close()
			}

			delete(pnsm.activeSubscriptions[hashedParams].connectedConsumers[epoch], consumerAddrString)
			if len(pnsm.activeSubscriptions[hashedParams].connectedConsumers[epoch]) == 0 {
				utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in epoch, closing epoch",
					utils.LogAttr("GUID", ctx),
					utils.LogAttr("consumerAddr", consumerAddr),
					utils.LogAttr("params", params),
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
					utils.LogAttr("epoch", epoch),
				)
				delete(pnsm.activeSubscriptions[hashedParams].connectedConsumers, epoch)
			}
		} else {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() consumer not found in epoch",
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("consumerAddr", consumerAddr),
				utils.LogAttr("params", params),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("connectedConsumers", connectedConsumers),
			)
		}
	}

	connectedConsumersCount := len(pnsm.activeSubscriptions[hashedParams].connectedConsumers[pnsm.prevEpoch])
	if connectedConsumersCount == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in previous epoch, closing subscription",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("epoch", pnsm.prevEpoch),
		)
		delete(pnsm.activeSubscriptions[hashedParams].connectedConsumers, pnsm.prevEpoch)
	} else {
		utils.LavaFormatTrace(fmt.Sprintf("ProviderNodeSubscriptionManager:RemoveConsumer() still have %v consumers left in this subscription for previous epoch", connectedConsumersCount),
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("connectedConsumers", pnsm.activeSubscriptions[hashedParams].connectedConsumers),
			utils.LogAttr("epoch", pnsm.prevEpoch),
		)
	}

	connectedConsumersCount = len(pnsm.activeSubscriptions[hashedParams].connectedConsumers[pnsm.currentEpoch])
	if connectedConsumersCount == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in current epoch, closing subscription",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("epoch", pnsm.currentEpoch),
		)
		delete(pnsm.activeSubscriptions[hashedParams].connectedConsumers, pnsm.currentEpoch)
	} else {
		utils.LavaFormatTrace(fmt.Sprintf("ProviderNodeSubscriptionManager:RemoveConsumer() still have %v consumers left in this subscription for current epoch", connectedConsumersCount),
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("epoch", pnsm.currentEpoch),
			utils.LogAttr("connectedConsumers", pnsm.activeSubscriptions[hashedParams].connectedConsumers),
		)
	}

	epochsConnected := len(pnsm.activeSubscriptions[hashedParams].connectedConsumers)
	if epochsConnected == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in subscription, closing subscription",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)

		pnsm.activeSubscriptions[hashedParams].cancellableContextCancelFunc() // This will trigger the cancellation of the subscription
	} else {
		utils.LavaFormatTrace(fmt.Sprintf("ProviderNodeSubscriptionManager:RemoveConsumer() still have %v epochs left in this subscription", epochsConnected),
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			utils.LogAttr("connectedConsumers", pnsm.activeSubscriptions[hashedParams].connectedConsumers),
		)
	}

	utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() removed consumer", utils.LogAttr("consumerAddr", consumerAddr), utils.LogAttr("params", params))
	return nil
}

func (pnsm *ProviderNodeSubscriptionManager) closeNodeSubscription(hashedParams string) error {
	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	if _, ok := pnsm.activeSubscriptions[hashedParams]; !ok {
		return utils.LavaFormatError("closeNodeSubscription called with hashedParams that does not exist", nil, utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)))
	}

	// Disconnect all connected consumers
	for epoch, connectedConsumers := range pnsm.activeSubscriptions[hashedParams].connectedConsumers {
		for consumerAddrString, consumerChannel := range connectedConsumers {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:closeNodeSubscription() closing consumer channel",
				utils.LogAttr("consumerAddr", consumerAddrString),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			consumerChannel.Close()
		}
	}

	pnsm.activeSubscriptions[hashedParams].nodeSubscription.Unsubscribe()
	close(pnsm.activeSubscriptions[hashedParams].messagesChannel)
	delete(pnsm.activeSubscriptions, hashedParams)

	return nil
}
