package rpcprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type ChannelToConnectedConsumers struct {
	CancellableContext           context.Context
	CancellableContextCancelFunc context.CancelFunc
	MessagesChannel              chan interface{}
	NodeSubscription             *rpcclient.ClientSubscription
	SubscriptionID               string
	FirstSetupReply              *pairingtypes.RelayReply
	FirstSetupRequest            *pairingtypes.RelayRequest
	ApiCollection                *spectypes.ApiCollection
	ConnectedConsumers           map[uint64]map[string]*SafeChannelSender[*pairingtypes.RelayReply] // first key is epoch, second key is consumer address
}

type ProviderNodeSubscriptionManager struct {
	chainRouter       chainlib.ChainRouter
	chainParser       chainlib.ChainParser
	openSubscriptions map[string]*ChannelToConnectedConsumers // key is request params hash
	currentEpoch      uint64
	prevEpoch         uint64
	privKey           *btcec.PrivateKey
	lock              sync.RWMutex
}

func NewProviderNodeSubscriptionManager(chainRouter chainlib.ChainRouter, chainParser chainlib.ChainParser, currentEpoch, prevEpoch uint64, privKey *btcec.PrivateKey) *ProviderNodeSubscriptionManager {
	utils.LavaFormatTrace("NewProviderNodeSubscriptionManager", utils.LogAttr("prevEpoch", prevEpoch), utils.LogAttr("currentEpoch", currentEpoch))
	return &ProviderNodeSubscriptionManager{
		chainRouter:       chainRouter,
		chainParser:       chainParser,
		openSubscriptions: make(map[string]*ChannelToConnectedConsumers),
		currentEpoch:      currentEpoch,
		prevEpoch:         prevEpoch,
		privKey:           privKey,
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
		utils.LogAttr("hashedParams", pnsm.readableHashedParams(hashedParams)),
	)

	consumerAddrString := consumerAddr.String()

	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	if pnsm.prevEpoch == pnsm.currentEpoch {
		utils.LavaFormatWarning("AddConsumer() called with prevEpoch == currentEpoch", nil, utils.LogAttr("prevEpoch", pnsm.prevEpoch), utils.LogAttr("currentEpoch", pnsm.currentEpoch))
		return nil, "", nil
	}

	var firstSetupReply *pairingtypes.RelayReply

	paramsChannelToConnectedConsumers, found := pnsm.openSubscriptions[hashedParams]
	if found {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() found existing subscription", utils.LogAttr("params", string(params)))

		if _, ok := paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch]; !ok { // Add the new epoch if doesn't exist
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer registered for epoch that does not exists in map, creating the new epoch map", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch] = make(map[string]*SafeChannelSender[*pairingtypes.RelayReply])
		} else if _, found := paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch][consumerAddrString]; found { // Consumer is already connected to this subscription in current epoch, dismiss
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer is already connected, returning the existing subscription", utils.LogAttr("consumerAddr", consumerAddr))
			return paramsChannelToConnectedConsumers.NodeSubscription, paramsChannelToConnectedConsumers.SubscriptionID, nil
		}

		// Check if the consumer already exist in prev epoch for the same reqParamHash
		connectedConsumerChan, found := paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.prevEpoch][consumerAddrString]
		if found {
			// Same consumer is renewing the subscription for the new epoch, let's move him to the current one
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer already exist in prev epoch, moving him to the current one", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch][consumerAddrString] = connectedConsumerChan
			delete(paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.prevEpoch], consumerAddrString) // Remove the entry from old epoch
		} else {
			// Add the new entry for the consumer
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() consumer registered for new epoch, the consumer does not exist in prev epoch", utils.LogAttr("consumerAddr", consumerAddr))
			paramsChannelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch][consumerAddrString] = NewSafeChannelSender(ctx, consumerChannel)
		}

		firstSetupReply = paramsChannelToConnectedConsumers.FirstSetupReply
		clientSubscription = paramsChannelToConnectedConsumers.NodeSubscription
		subscriptionId = paramsChannelToConnectedConsumers.SubscriptionID
	} else {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() did not found existing subscription, creating new one")

		nodeChan := make(chan interface{})

		replyWrapper, subscriptionId, clientSubscription, _, _, err := pnsm.chainRouter.SendNodeMsg(ctx, nodeChan, chainMessage, nil)
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
			SafeChannelSender := NewSafeChannelSender(ctx, consumerChannel)
			SafeChannelSender.Send(reply)

			return nil, "", utils.LavaFormatWarning("subscription failed, node error", nil, utils.LogAttr("GUID", ctx), utils.LogAttr("reply", reply))
		}

		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:AddConsumer() subscription successful",
			utils.LogAttr("subscriptionId", subscriptionId),
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("firstSetupReplyData", reply.Data),
		)

		cancellableCtx, cancel := context.WithCancel(context.Background())
		channelToConnectedConsumers := &ChannelToConnectedConsumers{
			CancellableContext:           cancellableCtx,
			CancellableContextCancelFunc: cancel,
			MessagesChannel:              nodeChan,
			NodeSubscription:             clientSubscription,
			SubscriptionID:               subscriptionId,
			FirstSetupReply:              reply,
			FirstSetupRequest:            request,
			ApiCollection:                chainMessage.GetApiCollection(),
			ConnectedConsumers:           make(map[uint64]map[string]*SafeChannelSender[*pairingtypes.RelayReply]),
		}

		channelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch] = make(map[string]*SafeChannelSender[*pairingtypes.RelayReply])
		channelToConnectedConsumers.ConnectedConsumers[pnsm.currentEpoch][consumerAddrString] = NewSafeChannelSender(ctx, consumerChannel)

		pnsm.openSubscriptions[hashedParams] = channelToConnectedConsumers
		firstSetupReply = reply

		go pnsm.listenForSubscriptionMessages(ctx, nodeChan, hashedParams)
	}

	// Send the first setup message to the consumer in a go routine because the blocking listening for this channel happens after this function
	pnsm.openSubscriptions[hashedParams].ConnectedConsumers[pnsm.currentEpoch][consumerAddrString].Send(firstSetupReply)

	return clientSubscription, subscriptionId, nil
}

func (pnsm *ProviderNodeSubscriptionManager) listenForSubscriptionMessages(ctx context.Context, nodeChan chan interface{}, hashedParams string) {
	for {
		select {
		case <-pnsm.openSubscriptions[hashedParams].CancellableContext.Done():
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() pnsm.openSubscriptions[hashedParams].CancellableContext.Done(), exiting", utils.LogAttr("hashedParams", pnsm.readableHashedParams(hashedParams)))
			return
		case <-ctx.Done():
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() context done, exiting", utils.LogAttr("hashedParams", pnsm.readableHashedParams(hashedParams)))
			return
		case nodeMsg := <-nodeChan:
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

	return string(sigs.HashMsg(params)), params, nil
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
	for epoch, consumerAddrToChannel := range pnsm.openSubscriptions[hashedParams].ConnectedConsumers {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() sending to consumers in epoch", utils.LogAttr("epoch", epoch), utils.LogAttr("hashedParams", pnsm.readableHashedParams(hashedParams)))

		for consumerAddrString, consumerChannel := range consumerAddrToChannel {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:startListeningForSubscription() sending to consumer", utils.LogAttr("consumerAddr", consumerAddrString), utils.LogAttr("hashedParams", pnsm.readableHashedParams(hashedParams)))

			request := pnsm.openSubscriptions[hashedParams].FirstSetupRequest
			apiCollection := pnsm.openSubscriptions[hashedParams].ApiCollection

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
		utils.LogAttr("consumerAddr", consumerAddr),
		utils.LogAttr("params", params),
	)

	consumerAddrString := consumerAddr.String()

	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	openSubscriptions, ok := pnsm.openSubscriptions[hashedParams]
	if !ok {
		utils.LavaFormatWarning("ProviderNodeSubscriptionManager:RemoveConsumer() no subscription found for params", nil,
			utils.LogAttr("consumerAddr", consumerAddr),
			utils.LogAttr("params", params),
		)
		return nil
	}

	// Remove consumer from all epochs
	for epoch, connectedConsumers := range openSubscriptions.ConnectedConsumers {
		if _, ok := connectedConsumers[consumerAddrString]; ok {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() found consumer in epoch",
				utils.LogAttr("consumerAddr", consumerAddr),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("params", params),
				utils.LogAttr("connectedConsumers", connectedConsumers),
			)

			if closeConsumerChannel {
				connectedConsumers[consumerAddrString].Close()
			}

			delete(pnsm.openSubscriptions[hashedParams].ConnectedConsumers[epoch], consumerAddrString)
		} else {
			utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() consumer not found in epoch",
				utils.LogAttr("consumerAddr", consumerAddr),
				utils.LogAttr("epoch", epoch),
				utils.LogAttr("params", params),
				utils.LogAttr("connectedConsumers", connectedConsumers),
			)
		}
	}

	connectedConsumersCount := len(pnsm.openSubscriptions[hashedParams].ConnectedConsumers[pnsm.prevEpoch])
	if connectedConsumersCount == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in previous epoch, closing subscription",
			utils.LogAttr("params", params),
			utils.LogAttr("epoch", pnsm.prevEpoch),
		)
		delete(pnsm.openSubscriptions[hashedParams].ConnectedConsumers, pnsm.prevEpoch)
	} else {
		utils.LavaFormatTrace(fmt.Sprintf("ProviderNodeSubscriptionManager:RemoveConsumer() still have %v consumers left in this subscription for previous epoch", connectedConsumersCount),
			utils.LogAttr("params", params),
			utils.LogAttr("connectedConsumers", pnsm.openSubscriptions[hashedParams].ConnectedConsumers),
		)
	}

	connectedConsumersCount = len(pnsm.openSubscriptions[hashedParams].ConnectedConsumers[pnsm.currentEpoch])
	if connectedConsumersCount == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in current epoch, closing subscription",
			utils.LogAttr("params", params),
			utils.LogAttr("epoch", pnsm.currentEpoch),
		)
		delete(pnsm.openSubscriptions[hashedParams].ConnectedConsumers, pnsm.currentEpoch)
	} else {
		utils.LavaFormatTrace(fmt.Sprintf("ProviderNodeSubscriptionManager:RemoveConsumer() still have %v consumers left in this subscription for current epoch", connectedConsumersCount),
			utils.LogAttr("params", params),
			utils.LogAttr("connectedConsumers", pnsm.openSubscriptions[hashedParams].ConnectedConsumers),
		)
	}

	epochsConnected := len(pnsm.openSubscriptions[hashedParams].ConnectedConsumers)
	if epochsConnected == 0 {
		utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() no more consumers in subscription, closing subscription", utils.LogAttr("params", params))

		pnsm.openSubscriptions[hashedParams].NodeSubscription.Unsubscribe()
		pnsm.openSubscriptions[hashedParams].CancellableContextCancelFunc()
		close(pnsm.openSubscriptions[hashedParams].MessagesChannel)
		delete(pnsm.openSubscriptions, hashedParams)
	}

	utils.LavaFormatTrace("ProviderNodeSubscriptionManager:RemoveConsumer() removed consumer", utils.LogAttr("consumerAddr", consumerAddr), utils.LogAttr("params", params))
	return nil
}

func (pnsm *ProviderNodeSubscriptionManager) readableHashedParams(hashedParams string) string {
	return fmt.Sprintf("%x", hashedParams)
}

func (pnsm *ProviderNodeSubscriptionManager) UpdateEpoch(epoch uint64) {
	pnsm.lock.Lock()
	defer pnsm.lock.Unlock()

	utils.LavaFormatTrace("ProviderNodeSubscriptionManager: UpdateEpoch", utils.LogAttr("epoch", epoch))

	// TODO: Disconnect all prev epoch consumers, and close all streams
	// To test, disable the context cancellation in the consumer, and see that the context is still cancelled from provider

	for _, channelToConnectedConsumers := range pnsm.openSubscriptions {
		if _, ok := channelToConnectedConsumers.ConnectedConsumers[pnsm.prevEpoch]; ok {
			for consumerAddr, consumerChannel := range channelToConnectedConsumers.ConnectedConsumers[pnsm.prevEpoch] {
				consumerChannel.Close()
				delete(channelToConnectedConsumers.ConnectedConsumers[pnsm.prevEpoch], consumerAddr)
			}
		}
	}

	pnsm.prevEpoch = pnsm.currentEpoch
	pnsm.currentEpoch = epoch
}
