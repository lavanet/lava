package rpcconsumer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	MaxCallsPerRelay                   = 50
	NumberOfRetriesAllowedOnNodeErrors = 2 // we will try maximum additional 2 relays on node errors
)

type Selection int

const (
	Quorum     Selection = iota // get the majority out of requiredSuccesses
	BestResult                  // get the best result, even if it means waiting
)

type MetricsInterface interface {
	SetRelayNodeErrorMetric(chainId string, apiInterface string)
	SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string)
	SetNodeErrorAttemptMetric(chainId string, apiInterface string)
}

type chainIdAndApiInterfaceGetter interface {
	GetChainIdAndApiInterface() (string, string)
}

type RelayProcessor struct {
	usedProviders                *lavasession.UsedProviders
	responses                    chan *relayResponse
	requiredSuccesses            int
	nodeResponseErrors           RelayErrors
	protocolResponseErrors       RelayErrors
	successResults               []common.RelayResult
	lock                         sync.RWMutex
	chainMessage                 chainlib.ChainMessage
	guid                         uint64
	selection                    Selection
	consumerConsistency          *ConsumerConsistency
	dappID                       string
	consumerIp                   string
	skipDataReliability          bool
	debugRelay                   bool
	allowSessionDegradation      uint32 // used in the scenario where extension was previously used.
	metricsInf                   MetricsInterface
	chainIdAndApiInterfaceGetter chainIdAndApiInterfaceGetter
	disableRelayRetry            bool
	relayRetriesManager          *RelayRetriesManager
}

func NewRelayProcessor(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	requiredSuccesses int,
	chainMessage chainlib.ChainMessage,
	consumerConsistency *ConsumerConsistency,
	dappID string,
	consumerIp string,
	debugRelay bool,
	metricsInf MetricsInterface,
	chainIdAndApiInterfaceGetter chainIdAndApiInterfaceGetter,
	disableRelayRetry bool,
	relayRetriesManager *RelayRetriesManager,
) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	selection := Quorum // select the majority of node responses
	if chainlib.GetStateful(chainMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		selection = BestResult // select the majority of node successes
	}
	if requiredSuccesses <= 0 {
		utils.LavaFormatFatal("invalid requirement, successes count must be greater than 0", nil, utils.LogAttr("requiredSuccesses", requiredSuccesses))
	}
	return &RelayProcessor{
		usedProviders:                usedProviders,
		requiredSuccesses:            requiredSuccesses,
		responses:                    make(chan *relayResponse, MaxCallsPerRelay), // we set it as buffered so it is not blocking
		nodeResponseErrors:           RelayErrors{relayErrors: []RelayError{}},
		protocolResponseErrors:       RelayErrors{relayErrors: []RelayError{}, onFailureMergeAll: true},
		chainMessage:                 chainMessage,
		guid:                         guid,
		selection:                    selection,
		consumerConsistency:          consumerConsistency,
		dappID:                       dappID,
		consumerIp:                   consumerIp,
		debugRelay:                   debugRelay,
		metricsInf:                   metricsInf,
		chainIdAndApiInterfaceGetter: chainIdAndApiInterfaceGetter,
		disableRelayRetry:            disableRelayRetry,
		relayRetriesManager:          relayRetriesManager,
	}
}

// true if we never got an extension. (default value)
func (rp *RelayProcessor) GetAllowSessionDegradation() bool {
	return atomic.LoadUint32(&rp.allowSessionDegradation) == 0
}

// in case we had an extension and managed to get a session successfully, we prevent session degradation.
func (rp *RelayProcessor) SetDisallowDegradation() {
	atomic.StoreUint32(&rp.allowSessionDegradation, 1)
}

func (rp *RelayProcessor) setSkipDataReliability(val bool) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	rp.skipDataReliability = val
}

func (rp *RelayProcessor) getSkipDataReliability() bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.skipDataReliability
}

func (rp *RelayProcessor) ShouldRetry(numberOfRetriesLaunched int) bool {
	if numberOfRetriesLaunched >= MaximumNumberOfTickerRelayRetries {
		return false
	}
	return rp.selection != BestResult
}

func (rp *RelayProcessor) String() string {
	if rp == nil {
		return ""
	}
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

func (rp *RelayProcessor) GetUsedProviders() *lavasession.UsedProviders {
	if rp == nil {
		utils.LavaFormatError("RelayProcessor.GetUsedProviders is nil, misuse detected", nil)
		return nil
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.usedProviders
}

// this function returns all results that came from a node, meaning success, and node errors
func (rp *RelayProcessor) NodeResults() []common.RelayResult {
	if rp == nil {
		return nil
	}
	rp.readExistingResponses()
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.nodeResultsInner()
}

// only when locked
func (rp *RelayProcessor) nodeResultsInner() []common.RelayResult {
	// start with results and add to them node results
	nodeResults := rp.successResults
	nodeResults = append(nodeResults, rp.nodeErrors()...)
	return nodeResults
}

// only when locked
func (rp *RelayProcessor) nodeErrors() (ret []common.RelayResult) {
	for _, relayError := range rp.nodeResponseErrors.relayErrors {
		ret = append(ret, relayError.response.relayResult)
	}
	return ret
}

func (rp *RelayProcessor) ProtocolErrors() uint64 {
	if rp == nil {
		return 0
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return uint64(len(rp.protocolResponseErrors.relayErrors))
}

func (rp *RelayProcessor) SetResponse(response *relayResponse) {
	if rp == nil {
		return
	}
	if response == nil {
		return
	}
	rp.responses <- response
}

func (rp *RelayProcessor) setValidResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()

	// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
	// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
	reqBlock, _ := rp.chainMessage.RequestedBlock()
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
		return
	}
	// no error, update the seen block
	blockSeen := response.relayResult.Reply.LatestBlock
	// nil safe
	rp.consumerConsistency.SetSeenBlock(blockSeen, rp.dappID, rp.consumerIp)
	// check response error
	foundError, errorMessage := rp.chainMessage.CheckResponseError(response.relayResult.Reply.Data, response.relayResult.StatusCode)
	if foundError {
		// this is a node error, meaning we still didn't get a good response.
		// we may choose to wait until there will be a response or timeout happens
		// if we decide to wait and timeout happens we will take the majority of response messages
		err := fmt.Errorf(errorMessage)
		rp.nodeResponseErrors.relayErrors = append(rp.nodeResponseErrors.relayErrors, RelayError{err: err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
		// send relay error metrics only on non stateful queries, as stateful queries always return X-1/X errors.
		if rp.selection != BestResult {
			go rp.metricsInf.SetRelayNodeErrorMetric(rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface())
			utils.LavaFormatInfo("Relay received a node error", utils.LogAttr("Error", err), utils.LogAttr("provider", response.relayResult.ProviderInfo), utils.LogAttr("Request", rp.chainMessage.GetApi().Name), utils.LogAttr("requested_block", reqBlock))
		}
		return
	}
	rp.successResults = append(rp.successResults, response.relayResult)
}

func (rp *RelayProcessor) setErrorResponse(response *relayResponse) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rp.guid}, utils.Attribute{Key: "provider", Value: response.relayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.err.Error()})
	rp.protocolResponseErrors.relayErrors = append(rp.protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
}

func (rp *RelayProcessor) checkEndProcessing(responsesCount int) bool {
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
	// check if we got all of the responses
	if responsesCount >= rp.usedProviders.SessionsLatestBatch() {
		// no active sessions, and we read all the responses, we can return
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

func (rp *RelayProcessor) getInputMsgInfoHashString() (string, error) {
	hash, err := rp.chainMessage.GetInputMsgInfoHash()
	hashString := ""
	if err == nil {
		hashString = string(hash)
	}
	return hashString, err
}

func (rp *RelayProcessor) HasRequiredNodeResults() bool {
	if rp == nil {
		return false
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount := len(rp.successResults)

	hash, hashErr := rp.getInputMsgInfoHashString()
	if resultsCount >= rp.requiredSuccesses {
		if hashErr == nil { // Incase we had a successful relay we can remove the hash from our relay retries map
			// Use a routine to run it in parallel
			go rp.relayRetriesManager.RemoveHashFromMap(hash)
		}
		// Check if we need to add node errors retry metrics
		if rp.selection == Quorum {
			// If nodeErrors length is larger than 0, our retry mechanism was activated. we add our metrics now.
			nodeErrors := len(rp.nodeResponseErrors.relayErrors)
			if nodeErrors > 0 {
				chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
				go rp.metricsInf.SetNodeErrorRecoveredSuccessfullyMetric(chainId, apiInterface, strconv.Itoa(nodeErrors))
			}
		}
		return true
	}
	if rp.selection == Quorum {
		// We need a quorum of all node results
		nodeErrors := len(rp.nodeResponseErrors.relayErrors)
		if nodeErrors+resultsCount >= rp.requiredSuccesses {
			// Retry on node error flow:
			if !rp.disableRelayRetry { // In case we want to try again if we have a node error.
				if resultsCount == 0 { // Only if we have 0 successful relays and we have only node errors.
					// Only continue the retry flow if we managed to parse hash.
					if hashErr == nil {
						// Only send a maximum of NumberOfRetriesAllowedOnNodeErrors retries.
						if nodeErrors <= NumberOfRetriesAllowedOnNodeErrors {
							// TODO: check chain message retry on archive. (this feature will be added in the generic parsers feature)

							// Check hash already exist, if it does, we don't want to retry
							if !rp.relayRetriesManager.CheckHashInMap(hash) {
								// If we didn't find the hash in the hash map we can retry
								utils.LavaFormatTrace("retrying on relay error", utils.LogAttr("retry_number", nodeErrors), utils.LogAttr("hash", hash))
								go rp.metricsInf.SetNodeErrorAttemptMetric(rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface())
								return false
							}
							utils.LavaFormatTrace("found hash in map wont retry", utils.LogAttr("hash", hash))
						} else {
							// We failed enough times. we need to add this to our hash map so we don't waste time on it again.
							utils.LavaFormatTrace("adding hash to hash map after NumberOfRetriesAllowedOnNodeErrors errors", utils.LogAttr("hash", hash))
							rp.relayRetriesManager.AddHashToMap(hash)
						}
					}
				}
			}
			// we have enough node results for our quorum
			return true
		}
	}
	// on BestResult we want to retry if there is no success
	return false
}

func (rp *RelayProcessor) handleResponse(response *relayResponse) {
	if response == nil {
		return
	}
	if response.err != nil {
		rp.setErrorResponse(response)
	} else {
		rp.setValidResponse(response)
	}
}

func (rp *RelayProcessor) readExistingResponses() {
	for {
		select {
		case response := <-rp.responses:
			rp.handleResponse(response)
		default:
			// No more responses immediately available, exit the loop
			return
		}
	}
}

// this function waits for the processing results, they are written by multiple go routines and read by this go routine
// it then updates the responses in their respective place, node errors, protocol errors or success results
func (rp *RelayProcessor) WaitForResults(ctx context.Context) error {
	if rp == nil {
		return utils.LavaFormatError("RelayProcessor.WaitForResults is nil, misuse detected", nil)
	}
	responsesCount := 0
	for {
		select {
		case response := <-rp.responses:
			responsesCount++
			rp.handleResponse(response)
			if rp.checkEndProcessing(responsesCount) {
				// we can finish processing
				return nil
			}
		case <-ctx.Done():
			return utils.LavaFormatWarning("cancelled relay processor", nil, utils.LogAttr("total responses", responsesCount))
		}
	}
}

func (rp *RelayProcessor) responsesQuorum(results []common.RelayResult, quorumSize int) (returnedResult *common.RelayResult, processingError error) {
	if quorumSize <= 0 {
		return nil, errors.New("quorumSize must be greater than zero")
	}
	countMap := make(map[string]int) // Map to store the count of each unique result.Reply.Data
	deterministic := rp.chainMessage.GetApi().Category.Deterministic
	var bestQosResult common.RelayResult
	bestQos := sdktypes.ZeroDec()
	nilReplies := 0
	nilReplyIdx := -1
	for idx, result := range results {
		if result.Reply != nil && result.Reply.Data != nil {
			countMap[string(result.Reply.Data)]++
			if !deterministic {
				if result.ProviderInfo.ProviderQoSExcellenceSummery.IsNil() || result.ProviderInfo.ProviderStake.Amount.IsNil() {
					continue
				}
				currentResult := result.ProviderInfo.ProviderQoSExcellenceSummery.MulInt(result.ProviderInfo.ProviderStake.Amount)
				if currentResult.GTE(bestQos) {
					bestQos.Set(currentResult)
					bestQosResult = result
				}
			}
		} else {
			nilReplies++
			nilReplyIdx = idx
		}
	}
	var mostCommonResult common.RelayResult
	var maxCount int
	for _, result := range results {
		if result.Reply != nil && result.Reply.Data != nil {
			count := countMap[string(result.Reply.Data)]
			if count > maxCount {
				maxCount = count
				mostCommonResult = result
			}
		}
	}

	if nilReplies >= quorumSize && maxCount < quorumSize {
		// we don't have a quorum with a valid response, but we have a quorum with an empty one
		maxCount = nilReplies
		mostCommonResult = results[nilReplyIdx]
	}
	// Check if the majority count is less than quorumSize
	if maxCount < quorumSize {
		if !deterministic {
			// non deterministic apis might not have a quorum
			// instead of failing get the best one
			bestQosResult.Quorum = 1
			return &bestQosResult, nil
		}
		return nil, utils.LavaFormatInfo("majority count is less than quorumSize", utils.LogAttr("nilReplies", nilReplies), utils.LogAttr("results", len(results)), utils.LogAttr("maxCount", maxCount), utils.LogAttr("quorumSize", quorumSize))
	}
	mostCommonResult.Quorum = maxCount
	return &mostCommonResult, nil
}

// this function returns the results according to the defined strategy
// results were stored in WaitForResults and now there's logic to select which results are returned to the user
// will return an error if we did not meet quota of replies, if we did we follow the strategies:
// if return strategy == get_first: return the first success, if none: get best node error
// if strategy == quorum get majority of node responses
// on error: we will return a placeholder relayResult, with a provider address and a status code
func (rp *RelayProcessor) ProcessingResult() (returnedResult *common.RelayResult, processingError error) {
	if rp == nil {
		return nil, utils.LavaFormatError("RelayProcessor.ProcessingResult is nil, misuse detected", nil)
	}

	// this must be here before the lock because this function locks
	allProvidersAddresses := rp.GetUsedProviders().UnwantedAddresses()

	rp.lock.RLock()
	defer rp.lock.RUnlock()
	// there are enough successes
	successResultsCount := len(rp.successResults)
	if successResultsCount >= rp.requiredSuccesses {
		return rp.responsesQuorum(rp.successResults, rp.requiredSuccesses)
	}
	nodeResults := rp.nodeResultsInner()
	// there are not enough successes, let's check if there are enough node errors

	if rp.debugRelay {
		// adding as much debug info as possible. all successful relays, all node errors and all protocol errors
		utils.LavaFormatDebug("[Processing Result] Debug Relay", utils.LogAttr("rp.requiredSuccesses", rp.requiredSuccesses))
		utils.LavaFormatDebug("[Processing Debug] number of node results", utils.LogAttr("len(rp.successResults)", len(rp.successResults)), utils.LogAttr("len(rp.nodeResponseErrors.relayErrors)", len(rp.nodeResponseErrors.relayErrors)), utils.LogAttr("len(rp.protocolResponseErrors.relayErrors)", len(rp.protocolResponseErrors.relayErrors)))
		for idx, result := range rp.successResults {
			utils.LavaFormatDebug("[Processing Debug] success result", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range rp.nodeResponseErrors.relayErrors {
			utils.LavaFormatDebug("[Processing Debug] node result", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range rp.protocolResponseErrors.relayErrors {
			utils.LavaFormatDebug("[Processing Debug] protocol error", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
	}

	if len(nodeResults) >= rp.requiredSuccesses {
		if rp.selection == Quorum {
			return rp.responsesQuorum(nodeResults, rp.requiredSuccesses)
		} else if rp.selection == BestResult && successResultsCount > len(rp.nodeResponseErrors.relayErrors) {
			// we have more than half succeeded, and we are success oriented
			return rp.responsesQuorum(rp.successResults, (rp.requiredSuccesses+1)/2)
		}
	}
	// we don't have enough for a quorum, prefer a node error on protocol errors
	if len(rp.nodeResponseErrors.relayErrors) >= rp.requiredSuccesses { // if we have node errors, we prefer returning them over protocol errors.
		nodeErr := rp.nodeResponseErrors.GetBestErrorMessageForUser()
		return &nodeErr.response.relayResult, nil
	}

	// if we got here we trigger a protocol error
	returnedResult = &common.RelayResult{StatusCode: http.StatusInternalServerError}
	if len(rp.nodeResponseErrors.relayErrors) > 0 { // if we have node errors, we prefer returning them over protocol errors, even if it's just the one
		nodeErr := rp.nodeResponseErrors.GetBestErrorMessageForUser()
		processingError = nodeErr.err
		errorResponse := nodeErr.response
		if errorResponse != nil {
			returnedResult = &errorResponse.relayResult
		}
	} else if len(rp.protocolResponseErrors.relayErrors) > 0 {
		protocolErr := rp.protocolResponseErrors.GetBestErrorMessageForUser()
		processingError = protocolErr.err
		errorResponse := protocolErr.response
		if errorResponse != nil {
			returnedResult = &errorResponse.relayResult
		}
	}
	returnedResult.ProviderInfo.ProviderAddress = strings.Join(allProvidersAddresses, ",")
	return returnedResult, utils.LavaFormatError("failed relay, insufficient results", processingError)
}
