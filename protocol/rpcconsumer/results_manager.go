package rpcconsumer

import (
	"fmt"
	"sync"

	"github.com/lavanet/lava/v2/protocol/chainlib"
	common "github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type ResultsManager interface {
	String() string
	NodeResults() []common.RelayResult
	RequiredResults(requiredSuccesses int, selection Selection) bool
	ProtocolErrors() uint64
	HasResults() bool
	GetResults() (success int, nodeErrors int, protocolErrors int)
	GetResultsData() (successResults []common.RelayResult, nodeErrors []common.RelayResult, protocolErrors []RelayError)
	SetResponse(response *relayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error)
	GetBestNodeErrorMessageForUser() RelayError
	GetBestProtocolErrorMessageForUser() RelayError
	nodeErrors() (ret []common.RelayResult)
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
		nodeResponseErrors:     RelayErrors{relayErrors: []RelayError{}},
		protocolResponseErrors: RelayErrors{relayErrors: []RelayError{}, onFailureMergeAll: true},
	}
}

func (rp *ResultsManagerInst) setErrorResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rp.guid}, utils.Attribute{Key: "provider", Value: response.relayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.err.Error()})
	rp.protocolResponseErrors.relayErrors = append(rp.protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
}

// only when locked
func (rp *ResultsManagerInst) nodeResultsInner() []common.RelayResult {
	// start with results and add to them node results
	nodeResults := rp.successResults
	nodeResults = append(nodeResults, rp.nodeErrors()...)
	return nodeResults
}

// only when locked
func (rp *ResultsManagerInst) nodeErrors() (ret []common.RelayResult) {
	for _, relayError := range rp.nodeResponseErrors.relayErrors {
		ret = append(ret, relayError.response.relayResult)
	}
	return ret
}

// returns an error if and only if it was a parsed node error
func (rp *ResultsManagerInst) setValidResponse(response *relayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error) {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
	// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
	reqBlock, _ := protocolMessage.RequestedBlock()
	if reqBlock == spectypes.LATEST_BLOCK {
		// TODO: when we turn on dataReliability on latest call UpdateLatest, until then we turn it off always
		// modifiedOnLatestReq := rp.chainMessage.UpdateLatestBlockInMessage(response.relayResult.Reply.LatestBlock, false)
		// if !modifiedOnLatestReq {
		response.relayResult.Finalized = false // shut down data reliability
		// }
	}

	if response.relayResult.Reply == nil {
		utils.LavaFormatError("got to setValidResponse with nil Reply",
			response.err,
			utils.LogAttr("ProviderInfo", response.relayResult.ProviderInfo),
			utils.LogAttr("StatusCode", response.relayResult.StatusCode),
			utils.LogAttr("Finalized", response.relayResult.Finalized),
			utils.LogAttr("Quorum", response.relayResult.Quorum),
		)
		return nil
	}

	// on subscribe results, we just append to successful results instead of parsing results because we already have a validation.
	if chainlib.IsFunctionTagOfType(protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
		rp.successResults = append(rp.successResults, response.relayResult)
		return nil
	}

	// check response error
	foundError, errorMessage := protocolMessage.CheckResponseError(response.relayResult.Reply.Data, response.relayResult.StatusCode)
	if foundError {
		// this is a node error, meaning we still didn't get a good response.
		// we may choose to wait until there will be a response or timeout happens
		// if we decide to wait and timeout happens we will take the majority of response messages
		err := fmt.Errorf("%s", errorMessage)
		rp.nodeResponseErrors.relayErrors = append(rp.nodeResponseErrors.relayErrors, RelayError{err: err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
		return err
	}
	rp.successResults = append(rp.successResults, response.relayResult)
	return nil
}

func (rm *ResultsManagerInst) GetResults() (success int, nodeErrors int, protocolErrors int) {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	nodeErrors = len(rm.nodeResponseErrors.relayErrors)
	protocolErrors = len(rm.protocolResponseErrors.relayErrors)
	success = len(rm.successResults)
	return success, nodeErrors, protocolErrors
}

func (rm *ResultsManagerInst) String() string {
	results, nodeErrors, protocolErrors := rm.GetResults()
	return fmt.Sprintf("resultsManager {success %d, nodeErrors:%d, protocolErrors:%d}", results, nodeErrors, protocolErrors)
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
	return uint64(len(rp.protocolResponseErrors.relayErrors))
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
		return true
	}
	if selection == Quorum {
		// we need a quorum of all node results
		nodeErrors := len(rp.nodeResponseErrors.relayErrors)
		if nodeErrors+resultsCount >= requiredSuccesses {
			// we have enough node results for our quorum
			return true
		}
	}
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
	nodeErrors := len(rp.nodeResponseErrors.relayErrors)
	protocolErrors := len(rp.protocolResponseErrors.relayErrors)
	return resultsCount+nodeErrors+protocolErrors > 0
}

func (rp *ResultsManagerInst) SetResponse(response *relayResponse, protocolMessage chainlib.ProtocolMessage) (nodeError error) {
	if response == nil {
		return nil
	}
	if response.err != nil {
		rp.setErrorResponse(response)
	} else {
		return rp.setValidResponse(response, protocolMessage)
	}
	return nil
}

func (rp *ResultsManagerInst) GetResultsData() (successResults []common.RelayResult, nodeErrors []common.RelayResult, protocolErrors []RelayError) {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.successResults, rp.nodeErrors(), rp.protocolResponseErrors.relayErrors
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
