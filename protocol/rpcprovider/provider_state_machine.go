package rpcprovider

import (
	"context"
	"time"

	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

type ProviderRelaySender interface {
	SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage chainlib.ChainMessageForSend, extensions []string) (relayReply *chainlib.RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error)
}

type ProviderStateMachine struct {
	relayRetriesManager lavaprotocol.RelayRetriesManagerInf
	chainId             string
	relaySender         ProviderRelaySender
	numberOfRetries     int
}

func NewProviderStateMachine(chainId string, relayRetriesManager lavaprotocol.RelayRetriesManagerInf, relaySender ProviderRelaySender, numberOfRetries int) *ProviderStateMachine {
	return &ProviderStateMachine{
		relayRetriesManager: relayRetriesManager,
		chainId:             chainId,
		relaySender:         relaySender,
		numberOfRetries:     numberOfRetries,
	}
}

func (psm *ProviderStateMachine) SendNodeMessage(ctx context.Context, chainMsg chainlib.ChainMessage, request *pairingtypes.RelayRequest) (*chainlib.RelayReplyWrapper, error) {
	hash, err := chainMsg.GetRawRequestHash()
	requestHashString := ""
	if err != nil {
		utils.LavaFormatWarning("Failed converting message to hash", err, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
	} else {
		requestHashString = string(hash)
	}

	var replyWrapper *chainlib.RelayReplyWrapper
	var isNodeError bool
	for retryAttempt := 0; retryAttempt <= psm.numberOfRetries; retryAttempt++ {
		sendTime := time.Now()
		replyWrapper, _, _, _, _, err = psm.relaySender.SendNodeMsg(ctx, nil, chainMsg, request.RelayData.Extensions)
		if err != nil {
			return nil, utils.LavaFormatError("Sending chainMsg failed", err, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if replyWrapper == nil || replyWrapper.RelayReply == nil {
			return nil, utils.LavaFormatError("Relay Wrapper returned nil without an error", nil, utils.LogAttr("attempt", retryAttempt), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		if debugLatency {
			utils.LavaFormatDebug("node reply received", utils.LogAttr("attempt", retryAttempt), utils.LogAttr("timeTaken", time.Since(sendTime)), utils.LogAttr("GUID", ctx), utils.LogAttr("specID", psm.chainId))
		}

		// Failed fetching hash return the reply.
		if requestHashString == "" {
			utils.LavaFormatWarning("Failed to hash request, shouldn't happen", nil, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
			break // We can't perform the retries as we failed fetching the request hash.
		}

		// Check for node errors
		isNodeError, _ = chainMsg.CheckResponseError(replyWrapper.RelayReply.Data, replyWrapper.StatusCode)
		if !isNodeError {
			// Successful relay, remove it from the cache if we have it and return a valid response.
			go psm.relayRetriesManager.RemoveHashFromCache(requestHashString)
			return replyWrapper, nil
		}

		// On the first retry, check if this hash has already failed previously
		if retryAttempt == 0 && psm.relayRetriesManager.CheckHashInCache(requestHashString) {
			utils.LavaFormatTrace("received node error, request hash was already in cache, skipping retry")
			break
		}
		utils.LavaFormatTrace("Errored Node Message, retrying message", utils.LogAttr("retry", retryAttempt))
	}

	if isNodeError {
		utils.LavaFormatTrace("failed all relay retries for message")
		go psm.relayRetriesManager.AddHashToCache(requestHashString)
	}
	return replyWrapper, nil
}
