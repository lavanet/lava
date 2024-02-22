package rpcconsumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	MaxCallsPerRelay = 50
)

func NewRelayProcessor(ctx context.Context, usedProviders *lavasession.UsedProviders, requiredSuccesses int, chainMessage chainlib.ChainMessage) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	return &RelayProcessor{
		usedProviders:          usedProviders,
		requiredResults:        requiredSuccesses,
		responses:              make(chan *relayResponse, MaxCallsPerRelay), // we set it as buffered so it is not blocking
		nodeResponseErrors:     &RelayErrors{relayErrors: []RelayError{}},
		protocolResponseErrors: &RelayErrors{relayErrors: []RelayError{}, onFailureMergeAll: true},
		chainMessage:           chainMessage,
		guid:                   guid,
		// TODO: handle required errors
		requiredErrors: requiredSuccesses,
	}
}

type RelayProcessor struct {
	usedProviders          *lavasession.UsedProviders
	responses              chan *relayResponse
	requiredResults        int
	nodeResponseErrors     *RelayErrors
	protocolResponseErrors *RelayErrors
	results                []common.RelayResult
	lock                   sync.RWMutex
	chainMessage           chainlib.ChainMessage
	errorRelayResult       common.RelayResult
	guid                   uint64
	requiredErrors         int
}

func (rp *RelayProcessor) String() string {
	// TODO:
	return ""
}

func (rp *RelayProcessor) RemoveUsed(providerAddress string, err error) {
	// TODO:
}

func (rp *RelayProcessor) GetUsedProviders() *lavasession.UsedProviders {
	return rp.usedProviders
}

func (rp *RelayProcessor) ComparableResults() []common.RelayResult {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	// TODO: add nodeResponseErrors
	return rp.results
}

func (rp *RelayProcessor) ProtocolErrors() uint64 {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return uint64(len(rp.protocolResponseErrors.relayErrors))
}

func (rp *RelayProcessor) SetResponse(response *relayResponse) {
	if response == nil {
		return
	}
	rp.responses <- response
}

func (rp *RelayProcessor) setValidResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	foundError, errorMessage := rp.chainMessage.CheckResponseError(response.relayResult.Reply.Data, response.relayResult.StatusCode)
	if foundError {
		// this is a node error, meaning we still didn't get a good response.
		// we may choose to wait until there will be a response or timeout happens
		// if we decide to wait and timeout happens we will take the majority of response messages
		err := fmt.Errorf(errorMessage)
		rp.nodeResponseErrors.relayErrors = append(rp.nodeResponseErrors.relayErrors, RelayError{err: err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
		return
	}
	// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
	// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
	reqBlock, _ := rp.chainMessage.RequestedBlock()
	if reqBlock == spectypes.LATEST_BLOCK {
		modifiedOnLatestReq := rp.chainMessage.UpdateLatestBlockInMessage(response.relayResult.Request.RelayData.RequestBlock, false)
		if !modifiedOnLatestReq {
			response.relayResult.Finalized = false // shut down data reliability
		}
	}
	rp.results = append(rp.results, response.relayResult)
	return
}

func (rp *RelayProcessor) setErrorResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rp.guid}, utils.Attribute{Key: "provider", Value: response.relayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.err.Error()})
	rp.protocolResponseErrors.relayErrors = append(rp.protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
}

func (rp *RelayProcessor) CheckEndProcessing() bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.results)
	if resultsCount >= rp.requiredResults {
		return true
	}
	nodeErrors := len(rp.nodeResponseErrors.relayErrors)
	protocolErrors := len(rp.protocolResponseErrors.relayErrors)
	if resultsCount+nodeErrors+protocolErrors >= rp.requiredErrors {
		return true
	}
	return false
}

func (rp *RelayProcessor) HasResponses() bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.results)
	nodeErrors := len(rp.nodeResponseErrors.relayErrors)
	protocolErrors := len(rp.protocolResponseErrors.relayErrors)
	return resultsCount+nodeErrors+protocolErrors > 0
}

func (rp *RelayProcessor) ProcessResults(ctx context.Context) ([]common.RelayResult, error) {
	responsesCount := 0
	for {
		select {
		case response := <-rp.responses:
			responsesCount++
			if response.err != nil {
				rp.setErrorResponse(response)
			} else {
				rp.setValidResponse(response)
			}
			if rp.CheckEndProcessing() {
				// we can finish processing
				return rp.ProcessingResult()
			}
		case <-ctx.Done():
			utils.LavaFormatWarning("cancelled relay processor", nil, utils.LogAttr("total responses", responsesCount))
			return rp.ProcessingResult()
		}
	}
}

func (rp *RelayProcessor) ProcessingResult() ([]common.RelayResult, error) {
	// when getting an error from all the results
	// rp.errorRelayResult.ProviderInfo.ProviderAddress += relayResult.ProviderInfo.ProviderAddress
	// if relayResult.GetStatusCode() != 0 {
	// 	// keep the error status code
	// 	rp.errorRelayResult.StatusCode = relayResult.GetStatusCode()
	// }

	// if len(relayResults) == 0 {
	// 	rpccs.appendHeadersToRelayResult(ctx, errorRelayResult, retries)
	// 	// suggest the user to add the timeout flag
	// 	if uint64(timeouts) == retries && retries > 0 {
	// 		utils.LavaFormatDebug("all relays timeout", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors.relayErrors})
	// 		return errorRelayResult, utils.LavaFormatError("Failed all relay retries due to timeout consider adding 'lava-relay-timeout' header to extend the allowed timeout duration", nil, utils.Attribute{Key: "GUID", Value: ctx})
	// 	}
	// 	bestRelayError := relayErrors.GetBestErrorMessageForUser()
	// 	return errorRelayResult, utils.LavaFormatError("Failed all retries", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.LogAttr("error", bestRelayError.err))
	// } else if len(relayErrors.relayErrors) > 0 {
	// 	utils.LavaFormatDebug("relay succeeded but had some errors", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "errors", Value: relayErrors})
	// }
	return nil, fmt.Errorf("TODO")
}
