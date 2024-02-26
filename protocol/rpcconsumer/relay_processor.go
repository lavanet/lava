package rpcconsumer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	MaxCallsPerRelay   = 50
	DefaultTimeout     = 20 * time.Second
	DefaultTimeoutLong = 3 * time.Minute
)

type Selection int

const (
	Quorum     Selection = iota // get the majority out of requiredSuccesses
	BestResult                  // get the best result, even if it means waiting
)

func NewRelayProcessor(ctx context.Context, usedProviders *lavasession.UsedProviders, requiredSuccesses int, chainMessage chainlib.ChainMessage) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	selection := Quorum // select the majority of node responses
	if chainlib.GetStateful(chainMessage) == common.CONSISTENCY_SELECT_ALLPROVIDERS {
		selection = BestResult // select the majority of node successes
	}
	return &RelayProcessor{
		usedProviders:          usedProviders,
		requiredSuccesses:      requiredSuccesses,
		responses:              make(chan *relayResponse, MaxCallsPerRelay), // we set it as buffered so it is not blocking
		nodeResponseErrors:     RelayErrors{relayErrors: []RelayError{}},
		protocolResponseErrors: RelayErrors{relayErrors: []RelayError{}, onFailureMergeAll: true},
		chainMessage:           chainMessage,
		guid:                   guid,
		selection:              selection,
	}
}

type RelayProcessor struct {
	usedProviders          *lavasession.UsedProviders
	responses              chan *relayResponse
	requiredSuccesses      int
	nodeResponseErrors     RelayErrors
	protocolResponseErrors RelayErrors
	successResults         []common.RelayResult
	lock                   sync.RWMutex
	chainMessage           chainlib.ChainMessage
	guid                   uint64
	selection              Selection
}

func (rp *RelayProcessor) String() string {
	rp.lock.RLock()
	nodeErrors := len(rp.nodeResponseErrors.relayErrors)
	protocolErrors := len(rp.protocolResponseErrors.relayErrors)
	results := len(rp.successResults)
	usedProviders := rp.usedProviders
	rp.lock.RUnlock()

	currentlyUsedAddresses := usedProviders.CurrentlyUsedAddresses()
	unwantedAddresses := usedProviders.UnwantedAddresses()
	return fmt.Sprintf("relayProcessor {results:%d, nodeErrors:%d, protocolErrors:%d,unwantedAddresses: %s,currentlyUsedAddresses:%s}",
		results, nodeErrors, protocolErrors, strings.Join(unwantedAddresses, ";"), strings.Join(currentlyUsedAddresses, ";"))
}

// RemoveUsed will set the provider as being currently used, if the error is one that allows a retry with the same provider, it will only be removed from currently used
// if it's not, then it will be added to unwanted providers since the same relay shouldn't send to it again
func (rp *RelayProcessor) RemoveUsed(providerAddress string, err error) {
	rp.usedProviders.RemoveUsed(providerAddress, err)
}

func (rp *RelayProcessor) GetUsedProviders() *lavasession.UsedProviders {
	return rp.usedProviders
}

// this function returns all results that came from a node, meaning success, and node errors
func (rp *RelayProcessor) NodeResults() []common.RelayResult {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.nodeResultsInner()
}

func (rp *RelayProcessor) nodeResultsInner() []common.RelayResult {
	// start with results and add to them node results
	nodeResults := rp.successResults
	for _, relayError := range rp.nodeResponseErrors.relayErrors {
		nodeResults = append(nodeResults, relayError.response.relayResult)
	}
	return nodeResults
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
	rp.successResults = append(rp.successResults, response.relayResult)
}

func (rp *RelayProcessor) setErrorResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rp.guid}, utils.Attribute{Key: "provider", Value: response.relayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.err.Error()})
	rp.protocolResponseErrors.relayErrors = append(rp.protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
}

func (rp *RelayProcessor) checkEndProcessing() bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)
	if resultsCount >= rp.requiredSuccesses {
		// we have enough successes, we can return
		return true
	}
	if rp.selection == Quorum {
		// we need a quorum of all node results
		nodeErrors := len(rp.nodeResponseErrors.relayErrors)
		if nodeErrors+resultsCount >= rp.requiredSuccesses {
			// we have enough node results for our quorum
			return true
		}
	}
	if rp.usedProviders.CurrentlyUsed() == 0 {
		// no active sessions, we can return
		return true
	}

	return false
}

// this function defines if we should use the processor to return the result (meaning it has some insight and responses) or just return to the user
func (rp *RelayProcessor) HasResults() bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)
	nodeErrors := len(rp.nodeResponseErrors.relayErrors)
	protocolErrors := len(rp.protocolResponseErrors.relayErrors)
	return resultsCount+nodeErrors+protocolErrors > 0
}

func (rp *RelayProcessor) HasRequiredNodeResults() bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)
	if resultsCount >= rp.requiredSuccesses {
		return true
	}
	if rp.selection == Quorum {
		// we need a quorum of all node results
		nodeErrors := len(rp.nodeResponseErrors.relayErrors)
		if nodeErrors+resultsCount >= rp.requiredSuccesses {
			// we have enough node results for our quorum
			return true
		}
	}
	// on BestResult we want to retry if there is no success
	return false
}

// this function waits for the processing results, they are written by multiple go routines and read by this go routine
// it then updates the responses in their respective place, node errors, protocol errors or success results
func (rp *RelayProcessor) WaitForResults(ctx context.Context) error {
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
			if rp.checkEndProcessing() {
				// we can finish processing
				return nil
			}
		case <-ctx.Done():
			return utils.LavaFormatWarning("cancelled relay processor", nil, utils.LogAttr("total responses", responsesCount))
		}
	}
}

func (rp *RelayProcessor) processingError() (returnedResult *common.RelayResult, processingError error) {
	// TODO:
	return nil, fmt.Errorf("not implmented")
}

// this function returns the results according to the defined strategy
// results were stored in WaitForResults and now there's logic to select which results are returned to the user
// will return an error if we did not meet quota of replies, if we did we follow the strategies:
// if return strategy == get_first: return the first success, if none: get best node error
// if strategy == quorum get majority of node responses
// on error: we will return a placeholder relayResult, with a provider address and a status code
func (rp *RelayProcessor) ProcessingResult() (returnedResult *common.RelayResult, processingError error) {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	// there are enough successes
	if len(rp.successResults) > rp.requiredSuccesses {
		return rp.responsesQuorum(rp.successResults, rp.requiredSuccesses)
	}
	nodeResults := rp.nodeResultsInner()
	// there are not enough successes, let's check if there are enough node errors

	if len(nodeResults) > rp.requiredSuccesses {
		if rp.selection == Quorum {
			return rp.responsesQuorum(nodeResults, rp.requiredSuccesses)
		} else if rp.selection == BestResult && len(rp.successResults) > len(rp.nodeResponseErrors.relayErrors) {
			// we have more than half succeeded, quorum will be
			return rp.responsesQuorum(rp.successResults, (rp.requiredSuccesses+1)/2)
		}
	}
	var bestErrorMessage RelayError
	// we don't have enough for a quorum, prefer a node error on protocol errors
	if len(rp.nodeResponseErrors.relayErrors) > 0 { // if we have node errors, we prefer returning them over protocol errors.
		bestErrorMessage = rp.nodeResponseErrors.GetBestErrorMessageForUser()
	} else if len(rp.protocolResponseErrors.relayErrors) > 0 { // if we have protocol errors at this point return the best one
		bestErrorMessage = rp.protocolResponseErrors.GetBestErrorMessageForUser()
	}

	returnedResult = &common.RelayResult{}

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

// func (rpccs *RPCConsumerServer) getBestResult(timeout time.Duration, responses chan *relayResponse, numberOfSessions int, chainMessage chainlib.ChainMessage) *relayResponse {
// 	responsesReceived := 0
// 	nodeResponseErrors := &RelayErrors{relayErrors: []RelayError{}}
// 	protocolResponseErrors := &RelayErrors{relayErrors: []RelayError{}, onFailureMergeAll: true}
// 	// a helper function to fetch the best response (prioritize node over protocol)
// 	getBestResponseBetweenNodeAndProtocolErrors := func() (*relayResponse, error) {
// 		if len(nodeResponseErrors.relayErrors) > 0 { // if we have node errors, we prefer returning them over protocol errors.
// 			bestErrorMessage := nodeResponseErrors.GetBestErrorMessageForUser()
// 			return bestErrorMessage.response, nil
// 		}
// 		if len(protocolResponseErrors.relayErrors) > 0 { // if we have protocol errors at this point return the best one
// 			protocolsBestErrorMessage := protocolResponseErrors.GetBestErrorMessageForUser()
// 			return protocolsBestErrorMessage.response, nil
// 		}
// 		return nil, fmt.Errorf("failed getting best response")
// 	}
// 	startTime := time.Now()
// 	for {
// 		select {
// 		case response := <-responses:
// 			// increase responses received
// 			responsesReceived++
// 			if response.err == nil {
// 				// validate if its a error response (from the node not the provider)
// 				foundError, errorMessage := chainMessage.CheckResponseError(response.relayResult.Reply.Data, response.relayResult.StatusCode)
// 				// print debug only when we have multiple responses
// 				if numberOfSessions > 1 {
// 					utils.LavaFormatDebug("Got Response", utils.LogAttr("responsesReceived", responsesReceived), utils.LogAttr("out_of", numberOfSessions), utils.LogAttr("foundError", foundError), utils.LogAttr("errorMessage", errorMessage), utils.LogAttr("Status code", response.relayResult.StatusCode))
// 				}
// 				if foundError {
// 					// this is a node error, meaning we still didn't get a good response.
// 					// we will choose to wait until there will be a response or timeout happens
// 					// if timeout happens we will take the majority of response messages
// 					nodeResponseErrors.relayErrors = append(nodeResponseErrors.relayErrors, RelayError{err: fmt.Errorf(errorMessage), ProviderInfo: response.relayResult.ProviderInfo, response: response})
// 				} else {
// 					// Return the first successful response
// 					return response // returning response
// 				}
// 			} else {
// 				// we want to keep the error message in a separate response error structure
// 				// in case we got only errors and we want to return the best one
// 				protocolResponseErrors.relayErrors = append(protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
// 			}

// 			// check if this is the last response we are going to receive
// 			// we get here only if all other responses including this one are not valid responses
// 			// (whether its a node error or protocol errors)
// 			if responsesReceived == numberOfSessions {
// 				bestRelayResult, err := getBestResponseBetweenNodeAndProtocolErrors()
// 				if err == nil { // successfully sent the channel response
// 					return bestRelayResult
// 				}
// 				// if we got here, we for some reason failed to fetch both the best node error and the protocol error
// 				// it indicates mostly an unwanted behavior.
// 				utils.LavaFormatWarning("failed getting best error message for both node and protocol", nil,
// 					utils.LogAttr("nodeResponseErrors", nodeResponseErrors),
// 					utils.LogAttr("protocolsBestErrorMessage", protocolResponseErrors),
// 					utils.LogAttr("numberOfSessions", numberOfSessions),
// 				)
// 				return response
// 			}
// 		case <-time.After(timeout + 3*time.Second - time.Since(startTime)):
// 			// Timeout occurred, try fetching the best result we have, prefer node errors over protocol errors
// 			bestRelayResponse, err := getBestResponseBetweenNodeAndProtocolErrors()
// 			if err == nil { // successfully sent the channel response
// 				return bestRelayResponse
// 			}
// 			// failed fetching any error, getting here indicates a real context timeout happened.
// 			return &relayResponse{common.RelayResult{}, NoResponseTimeout}
// 		}
// 	}
// }

func GetTimeoutForProcessing(relayTimeout time.Duration, chainMessage chainlib.ChainMessage) time.Duration {
	ctxTimeout := DefaultTimeout
	if chainlib.IsHangingApi(chainMessage) || chainMessage.GetApi().ComputeUnits > 100 || chainlib.GetStateful(chainMessage) == common.CONSISTENCY_SELECT_ALLPROVIDERS {
		ctxTimeout = DefaultTimeoutLong
	}
	if relayTimeout > ctxTimeout {
		ctxTimeout = relayTimeout
	}
	return ctxTimeout
}
