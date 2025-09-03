package rpcprovider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
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
	var errorMessage string
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

		// Check for node errors
		isNodeError, errorMessage = chainMsg.CheckResponseError(replyWrapper.RelayReply.Data, replyWrapper.StatusCode)

		// Failed fetching hash return the reply.
		if requestHashString == "" {
			utils.LavaFormatWarning("Failed to hash request, shouldn't happen", nil, utils.LogAttr("url", request.RelayData.ApiUrl), utils.LogAttr("data", string(request.RelayData.Data)))
			break // We can't perform the retries as we failed fetching the request hash.
		}
		if !isNodeError {
			// Successful relay, remove it from the cache if we have it and return a valid response.
			go psm.relayRetriesManager.RemoveHashFromCache(requestHashString)
			return replyWrapper, nil
		}

		// Check if this is an unsupported method error OR if we're using a default API and got an error
		isDefaultApi := false
		if chainMsg != nil && chainMsg.GetApi() != nil {
			isDefaultApi = strings.HasPrefix(chainMsg.GetApi().Name, chainlib.DefaultApiName)
		}
		if chainlib.IsUnsupportedMethodErrorMessage(errorMessage) || (isDefaultApi && isNodeError) {
			// Extract method name if available
			methodName := ""
			apiInterface := ""
			if chainMsg != nil && chainMsg.GetApi() != nil {
				methodName = chainMsg.GetApi().Name
				if chainMsg.GetApiCollection() != nil {
					apiInterface = chainMsg.GetApiCollection().CollectionData.ApiInterface
				}
			}

			// Comprehensive structured logging
			logMessage := "unsupported method error detected - returning error to consumer"
			if isDefaultApi && !chainlib.IsUnsupportedMethodErrorMessage(errorMessage) {
				logMessage = "default API error detected - returning error to consumer"
			}
			utils.LavaFormatInfo(logMessage,
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("error", errorMessage),
				utils.LogAttr("method", methodName),
				utils.LogAttr("api_interface", apiInterface),
				utils.LogAttr("chain_id", psm.chainId),
				utils.LogAttr("url", request.RelayData.ApiUrl),
				utils.LogAttr("consumer_address", request.RelayData.GetMetadata()),
				utils.LogAttr("session_id", request.RelaySession.SessionId),
				utils.LogAttr("relay_num", request.RelaySession.RelayNum),
				utils.LogAttr("spec_id", request.RelaySession.SpecId),
				utils.LogAttr("cu_sum", request.RelaySession.CuSum),
				utils.LogAttr("request_block", request.RelayData.RequestBlock),
				utils.LogAttr("seen_block", request.RelayData.SeenBlock),
				utils.LogAttr("timestamp", time.Now().UTC()),
				utils.LogAttr("retry_attempt", retryAttempt),
				utils.LogAttr("request_data", string(request.RelayData.Data)),
				utils.LogAttr("status_code", replyWrapper.StatusCode),
				utils.LogAttr("is_default_api", isDefaultApi),
			)

			// Return an UnsupportedMethodError to the consumer so they don't increment their CU counter
			unsupportedError := chainlib.NewUnsupportedMethodError(errors.New(errorMessage), methodName)
			return nil, unsupportedError
		}

		// On the first retry, check if this hash has already failed previously
		if retryAttempt == 0 && psm.relayRetriesManager.CheckHashInCache(requestHashString) {
			utils.LavaFormatTrace("received node error, request hash was already in cache, skipping retry")
			break
		}
		utils.LavaFormatTrace("Errored Node Message, retrying message", utils.LogAttr("retry", retryAttempt))
	}

	if isNodeError && requestHashString != "" {
		utils.LavaFormatTrace("failed all relay retries for message", utils.LogAttr("hash", requestHashString))
		go psm.relayRetriesManager.AddHashToCache(requestHashString)
	}
	return replyWrapper, nil
}
