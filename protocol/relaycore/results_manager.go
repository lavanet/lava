package relaycore

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

type ResultsManager interface {
	String() string
	NodeResults() []common.RelayResult
	RequiredResults(requiredSuccesses int, selection Selection) bool
	ProtocolErrors() uint64
	HasResults() bool
	GetResults() (success int, nodeErrors int, specialNodeErrors int, protocolErrors int)
	GetResultsData() (successResults []common.RelayResult, nodeErrors []common.RelayResult, protocolErrors []RelayError)
	SetResponse(response *RelayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error)
	GetBestNodeErrorMessageForUser() RelayError
	GetBestProtocolErrorMessageForUser() RelayError
	NodeErrors() (ret []common.RelayResult)
}

type ResultsManagerInst struct {
	nodeResponseErrors     RelayErrors
	protocolResponseErrors RelayErrors
	successResults         []common.RelayResult
	lock                   sync.RWMutex
	guid                   uint64
}

func NewResultsManager(guid uint64) ResultsManager {
	return &ResultsManagerInst{
		guid:                   guid,
		nodeResponseErrors:     RelayErrors{RelayErrors: []RelayError{}},
		protocolResponseErrors: RelayErrors{RelayErrors: []RelayError{}, OnFailureMergeAll: true},
	}
}

func (rp *ResultsManagerInst) setErrorResponse(response *RelayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rp.guid}, utils.Attribute{Key: "provider", Value: response.RelayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.Err.Error()})
	utils.LavaFormatError(
		"could not send relay to provider",
		response.Err,
		utils.Attribute{Key: "GUID", Value: rp.guid},
		utils.Attribute{Key: "provider", Value: response.RelayResult.ProviderInfo.ProviderAddress},
		utils.Attribute{Key: "statusCode", Value: response.RelayResult.StatusCode},
		utils.Attribute{Key: "providerTrailer", Value: response.RelayResult.ProviderTrailer},
	)
	rp.protocolResponseErrors.AddError(RelayError{Err: response.Err, ProviderInfo: response.RelayResult.ProviderInfo, Response: response})
}

// only when locked
func (rp *ResultsManagerInst) nodeResultsInner() []common.RelayResult {
	// start with results and add to them node results
	nodeResults := rp.successResults
	nodeResults = append(nodeResults, rp.NodeErrors()...)
	return nodeResults
}

// only when locked
func (rp *ResultsManagerInst) NodeErrors() (ret []common.RelayResult) {
	for _, relayError := range rp.nodeResponseErrors.RelayErrors {
		ret = append(ret, relayError.Response.RelayResult)
	}
	return ret
}

// returns an error if and only if it was a parsed node error
func (rp *ResultsManagerInst) setValidResponse(response *RelayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error) {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
	// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
	reqBlock, _ := protocolMessage.RequestedBlock()
	if reqBlock == spectypes.LATEST_BLOCK {
		// TODO: when we turn on dataReliability on latest call UpdateLatest, until then we turn it off always
		// modifiedOnLatestReq := rp.chainMessage.UpdateLatestBlockInMessage(response.RelayResult.Reply.LatestBlock, false)
		// if !modifiedOnLatestReq {
		response.RelayResult.Finalized = false // shut down data reliability
		// }
	}

	if response.RelayResult.Reply == nil {
		utils.LavaFormatError("got to setValidResponse with nil Reply",
			response.Err,
			utils.LogAttr("ProviderInfo", response.RelayResult.ProviderInfo),
			utils.LogAttr("StatusCode", response.RelayResult.StatusCode),
			utils.LogAttr("Finalized", response.RelayResult.Finalized),
			utils.LogAttr("Quorum", response.RelayResult.Quorum),
		)
		return nil
	}

	// on subscribe results, we just append to successful results instead of parsing results because we already have a validation.
	if chainlib.IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
		rp.successResults = append(rp.successResults, response.RelayResult)
		return nil
	}

	// check response error
	foundError, errorMessage := protocolMessage.CheckResponseError(response.RelayResult.Reply.Data, response.RelayResult.StatusCode)
	if foundError {
		// this is a node error, meaning we still didn't get a good response.
		// we may choose to wait until there will be a response or timeout happens
		// if we decide to wait and timeout happens we will take the majority of response messages
		err := fmt.Errorf("%s", errorMessage)
		// Log node error payload and headers for troubleshooting
		// also log the original request payload and request headers if available
		reqPayload := ""
		var reqHeaders interface{}
		if response.RelayResult.Request != nil && response.RelayResult.Request.RelayData != nil {
			reqPayload = string(response.RelayResult.Request.RelayData.Data)
			reqHeaders = response.RelayResult.Request.RelayData.Metadata
		}
		// Get request URL safely
		requestUrl := ""
		if protocolMessage.RelayPrivateData() != nil {
			requestUrl = protocolMessage.RelayPrivateData().ApiUrl
		}

		utils.LavaFormatError(
			"received node error reply from provider",
			err,
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("provider", response.RelayResult.ProviderInfo),
			utils.LogAttr("statusCode", response.RelayResult.StatusCode),
			utils.LogAttr("api", protocolMessage.GetApi().Name),
			utils.LogAttr("requestUrl", requestUrl),
			utils.LogAttr("payload", string(response.RelayResult.Reply.Data)),
			utils.LogAttr("headers", response.RelayResult.Reply.Metadata),
			utils.LogAttr("requestPayload", reqPayload),
			utils.LogAttr("requestHeaders", reqHeaders),
		)
		rp.nodeResponseErrors.AddError(RelayError{Err: err, ProviderInfo: response.RelayResult.ProviderInfo, Response: response})
		return err
	}
	rp.successResults = append(rp.successResults, response.RelayResult)
	return nil
}

func (rm *ResultsManagerInst) GetResults() (success int, nodeErrors int, specialNodeErrors int, protocolErrors int) {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	specialErrorPatterns := []string{"The node does not track the shard ID"}
	for _, err := range rm.nodeResponseErrors.RelayErrors {
		if err.Response != nil && err.Response.RelayResult.Reply != nil && err.Response.RelayResult.Reply.Data != nil {
			for _, specialErrorPattern := range specialErrorPatterns {
				if strings.Contains(string(err.Response.RelayResult.Reply.Data), specialErrorPattern) {
					specialNodeErrors++
				}
			}
		}
	}
	nodeErrors = len(rm.nodeResponseErrors.RelayErrors) - specialNodeErrors
	protocolErrors = len(rm.protocolResponseErrors.RelayErrors)
	success = len(rm.successResults)
	return success, nodeErrors, specialNodeErrors, protocolErrors
}

func (rm *ResultsManagerInst) String() string {
	results, nodeErrors, specialNodeErrors, protocolErrors := rm.GetResults()
	return fmt.Sprintf("resultsManager {success %d, nodeErrors:%d, specialNodeErrors:%d, protocolErrors:%d}", results, nodeErrors, specialNodeErrors, protocolErrors)
}

// this function returns all results that came from a node, meaning success, and node errors
func (rp *ResultsManagerInst) NodeResults() []common.RelayResult {
	if rp == nil {
		return nil
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.nodeResultsInner()
}

func (rp *ResultsManagerInst) ProtocolErrors() uint64 {
	if rp == nil {
		return 0
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return uint64(len(rp.protocolResponseErrors.RelayErrors))
}

func (rp *ResultsManagerInst) RequiredResults(requiredSuccesses int, selection Selection) bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)
	if resultsCount >= requiredSuccesses {
		// we have enough successes, we can return
		utils.LavaFormatDebug("Reached RequiredResults", utils.LogAttr("resultsCount", resultsCount), utils.LogAttr("requiredSuccesses", requiredSuccesses), utils.LogAttr("GUID", rp.guid))
		return true
	}
	// Only count successful results for quorum validation
	return false
}

// this function defines if we should use the manager to return the result (meaning it has some insight and responses) or just return to the user
func (rp *ResultsManagerInst) HasResults() bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)
	nodeErrors := len(rp.nodeResponseErrors.RelayErrors)
	protocolErrors := len(rp.protocolResponseErrors.RelayErrors)
	return resultsCount+nodeErrors+protocolErrors > 0
}

func (rp *ResultsManagerInst) SetResponse(response *RelayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error) {
	if response == nil {
		return nil
	}
	if response.Err != nil {
		rp.setErrorResponse(response)
	} else {
		return rp.setValidResponse(response, protocolMessage)
	}
	return nil
}

func (rp *ResultsManagerInst) GetResultsData() (successResults []common.RelayResult, nodeErrors []common.RelayResult, protocolErrors []RelayError) {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.successResults, rp.NodeErrors(), rp.protocolResponseErrors.RelayErrors
}

func (rp *ResultsManagerInst) GetBestNodeErrorMessageForUser() RelayError {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.nodeResponseErrors.GetBestErrorMessageForUser()
}

func (rp *ResultsManagerInst) GetBestProtocolErrorMessageForUser() RelayError {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.protocolResponseErrors.GetBestErrorMessageForUser()
}
