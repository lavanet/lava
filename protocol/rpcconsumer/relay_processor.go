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
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils"
)

type Selection int

const (
	MaxCallsPerRelay = 50
)

var relayCountOnNodeError = 2

// selection Enum, do not add other const
const (
	Quorum     Selection = iota // get the majority out of requiredSuccesses
	BestResult                  // get the best result, even if it means waiting
)

type MetricsInterface interface {
	SetRelayNodeErrorMetric(providerAddress string, chainId string, apiInterface string)
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
	lock                         sync.RWMutex
	guid                         uint64
	selection                    Selection
	consumerConsistency          *ConsumerConsistency
	skipDataReliability          bool
	debugRelay                   bool
	allowSessionDegradation      uint32 // used in the scenario where extension was previously used.
	metricsInf                   MetricsInterface
	chainIdAndApiInterfaceGetter chainIdAndApiInterfaceGetter
	relayRetriesManager          *lavaprotocol.RelayRetriesManager
	ResultsManager
	RelayStateMachine
}

func NewRelayProcessor(
	ctx context.Context,
	requiredSuccesses int,
	consumerConsistency *ConsumerConsistency,
	metricsInf MetricsInterface,
	chainIdAndApiInterfaceGetter chainIdAndApiInterfaceGetter,
	relayRetriesManager *lavaprotocol.RelayRetriesManager,
	relayStateMachine RelayStateMachine,
) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	if requiredSuccesses <= 0 {
		utils.LavaFormatFatal("invalid requirement, successes count must be greater than 0", nil, utils.LogAttr("requiredSuccesses", requiredSuccesses))
	}
	relayProcessor := &RelayProcessor{
		requiredSuccesses:            requiredSuccesses,
		responses:                    make(chan *relayResponse, MaxCallsPerRelay), // we set it as buffered so it is not blocking
		ResultsManager:               NewResultsManager(guid),
		guid:                         guid,
		consumerConsistency:          consumerConsistency,
		debugRelay:                   relayStateMachine.GetDebugState(),
		metricsInf:                   metricsInf,
		chainIdAndApiInterfaceGetter: chainIdAndApiInterfaceGetter,
		relayRetriesManager:          relayRetriesManager,
		RelayStateMachine:            relayStateMachine,
		selection:                    relayStateMachine.GetSelection(),
		usedProviders:                relayStateMachine.GetUsedProviders(),
	}
	relayProcessor.RelayStateMachine.SetResultsChecker(relayProcessor)
	relayProcessor.RelayStateMachine.SetRelayRetriesManager(relayRetriesManager)
	return relayProcessor
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

func (rp *RelayProcessor) String() string {
	if rp == nil {
		return ""
	}

	usedProviders := rp.GetUsedProviders()

	currentlyUsedAddresses := usedProviders.CurrentlyUsedAddresses()
	unwantedAddresses := usedProviders.AllUnwantedAddresses()
	return fmt.Sprintf("relayProcessor {%s, unwantedAddresses: %s,currentlyUsedAddresses:%s}",
		rp.ResultsManager.String(), strings.Join(unwantedAddresses, ";"), strings.Join(currentlyUsedAddresses, ";"))
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
	return rp.ResultsManager.NodeResults()
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

func (rp *RelayProcessor) checkEndProcessing(responsesCount int) bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	if rp.ResultsManager.RequiredResults(rp.requiredSuccesses, rp.selection) {
		return true
	}
	// check if we got all of the responses
	if responsesCount >= rp.usedProviders.SessionsLatestBatch() {
		// no active sessions, and we read all the responses, we can return
		return true
	}
	return false
}

func (rp *RelayProcessor) getInputMsgInfoHashString() (string, error) {
	hash, err := rp.RelayStateMachine.GetProtocolMessage().GetRawRequestHash()
	hashString := ""
	if err == nil {
		hashString = string(hash)
	}
	return hashString, err
}

// Deciding wether we should send a relay retry attempt based on the node error
func (rp *RelayProcessor) shouldRetryRelay(resultsCount int, hashErr error, nodeErrors int, hash string) bool {
	// Retries will be performed based on the following scenarios:
	// 1. If relayCountOnNodeError > 0
	// 2. If we have 0 successful relays and we have only node errors.
	// 3. Hash calculation was successful.
	// 4. Number of retries < relayCountOnNodeError.
	if relayCountOnNodeError > 0 && resultsCount == 0 && hashErr == nil {
		if nodeErrors <= relayCountOnNodeError {
			// TODO: check chain message retry on archive. (this feature will be added in the generic parsers feature)

			// Check if user specified to disable caching - OR
			// Check hash already exist, if it does, we don't want to retry
			if !rp.relayRetriesManager.CheckHashInCache(hash) {
				// If we didn't find the hash in the hash map we can retry
				utils.LavaFormatTrace("retrying on relay error", utils.LogAttr("retry_number", nodeErrors), utils.LogAttr("hash", utils.ToHexString(hash)))
				go rp.metricsInf.SetNodeErrorAttemptMetric(rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface())
				return false
			}
			utils.LavaFormatTrace("found hash in map wont retry", utils.LogAttr("hash", utils.ToHexString(hash)))
		} else {
			// We failed enough times. we need to add this to our hash map so we don't waste time on it again.
			chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
			utils.LavaFormatWarning("Failed to recover retries on node errors, might be an invalid input", nil,
				utils.LogAttr("api", rp.RelayStateMachine.GetProtocolMessage().GetApi().Name),
				utils.LogAttr("params", rp.RelayStateMachine.GetProtocolMessage().GetRPCMessage().GetParams()),
				utils.LogAttr("chainId", chainId),
				utils.LogAttr("apiInterface", apiInterface),
				utils.LogAttr("hash", utils.ToHexString(hash)),
			)
			rp.relayRetriesManager.AddHashToCache(hash)
		}
	}
	// Do not perform a retry
	return true
}

func (rp *RelayProcessor) HasRequiredNodeResults() (bool, int) {
	if rp == nil {
		return false, 0
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount, nodeErrors, _ := rp.GetResults()

	hash, hashErr := rp.getInputMsgInfoHashString()
	if resultsCount >= rp.requiredSuccesses {
		if hashErr == nil { // Incase we had a successful relay we can remove the hash from our relay retries map
			// Use a routine to run it in parallel
			go rp.relayRetriesManager.RemoveHashFromCache(hash)
		}
		// Check if we need to add node errors retry metrics
		if rp.selection == Quorum {
			// If nodeErrors length is larger than 0, our retry mechanism was activated. we add our metrics now.
			if nodeErrors > 0 {
				chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
				go rp.metricsInf.SetNodeErrorRecoveredSuccessfullyMetric(chainId, apiInterface, strconv.Itoa(nodeErrors))
			}
		}
		return true, nodeErrors
	}
	if rp.selection == Quorum {
		// We need a quorum of all node results
		if nodeErrors+resultsCount >= rp.requiredSuccesses {
			// Retry on node error flow:
			return rp.shouldRetryRelay(resultsCount, hashErr, nodeErrors, hash), nodeErrors
		}
	}
	// on BestResult we want to retry if there is no success
	return false, nodeErrors
}

func (rp *RelayProcessor) handleResponse(response *relayResponse) {
	nodeError := rp.ResultsManager.SetResponse(response, rp.RelayStateMachine.GetProtocolMessage())
	// send relay error metrics only on non stateful queries, as stateful queries always return X-1/X errors.
	if nodeError != nil && rp.selection != BestResult {
		chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
		go rp.metricsInf.SetRelayNodeErrorMetric(response.relayResult.ProviderInfo.ProviderAddress, chainId, apiInterface)
		utils.LavaFormatInfo("Relay received a node error", utils.LogAttr("Error", nodeError), utils.LogAttr("provider", response.relayResult.ProviderInfo), utils.LogAttr("Request", rp.RelayStateMachine.GetProtocolMessage().GetApi().Name))
	}

	if response != nil && response.relayResult.GetReply().GetLatestBlock() > 0 {
		// set consumer consistency when possible
		blockSeen := response.relayResult.GetReply().GetLatestBlock()
		rp.consumerConsistency.SetSeenBlock(blockSeen, rp.RelayStateMachine.GetProtocolMessage().GetUserData())
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
			return utils.LavaFormatDebug("cancelled relay processor", utils.LogAttr("total responses", responsesCount))
		}
	}
}

func (rp *RelayProcessor) responsesQuorum(results []common.RelayResult, quorumSize int) (returnedResult *common.RelayResult, processingError error) {
	if quorumSize <= 0 {
		return nil, errors.New("quorumSize must be greater than zero")
	}
	countMap := make(map[string]int) // Map to store the count of each unique result.Reply.Data
	deterministic := rp.RelayStateMachine.GetProtocolMessage().GetApi().Category.Deterministic
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
	allProvidersAddresses := rp.GetUsedProviders().AllUnwantedAddresses()

	rp.lock.RLock()
	defer rp.lock.RUnlock()

	successResults, nodeErrors, protocolErrors := rp.GetResultsData()
	successResultsCount, nodeErrorCount, protocolErrorCount := len(successResults), len(nodeErrors), len(protocolErrors)
	// there are enough successes
	if successResultsCount >= rp.requiredSuccesses {
		return rp.responsesQuorum(successResults, rp.requiredSuccesses)
	}

	if rp.debugRelay {
		// adding as much debug info as possible. all successful relays, all node errors and all protocol errors
		utils.LavaFormatDebug("[Processing Result] Debug Relay", utils.LogAttr("rp.requiredSuccesses", rp.requiredSuccesses))
		utils.LavaFormatDebug("[Processing Debug] number of node results", utils.LogAttr("successResultsCount", successResultsCount), utils.LogAttr("nodeErrorCount", nodeErrorCount), utils.LogAttr("protocolErrorCount", protocolErrorCount))
		for idx, result := range successResults {
			utils.LavaFormatDebug("[Processing Debug] success result", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range nodeErrors {
			utils.LavaFormatDebug("[Processing Debug] node result", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range protocolErrors {
			utils.LavaFormatDebug("[Processing Debug] protocol error", utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
	}

	// there are not enough successes, let's check if there are enough node errors
	if successResultsCount+nodeErrorCount >= rp.requiredSuccesses {
		if rp.selection == Quorum {
			nodeResults := make([]common.RelayResult, 0, len(successResults)+len(nodeErrors))
			nodeResults = append(nodeResults, successResults...)
			nodeResults = append(nodeResults, nodeErrors...)
			return rp.responsesQuorum(nodeResults, rp.requiredSuccesses)
		} else if rp.selection == BestResult && successResultsCount > nodeErrorCount {
			// we have more than half succeeded, and we are success oriented
			return rp.responsesQuorum(successResults, (rp.requiredSuccesses+1)/2)
		}
	}
	// we don't have enough for a quorum, prefer a node error on protocol errors
	if nodeErrorCount >= rp.requiredSuccesses { // if we have node errors, we prefer returning them over protocol errors.
		nodeErr := rp.GetBestNodeErrorMessageForUser()
		return &nodeErr.response.relayResult, nil
	}

	// if we got here we trigger a protocol error
	returnedResult = &common.RelayResult{StatusCode: http.StatusInternalServerError}
	if nodeErrorCount > 0 { // if we have node errors, we prefer returning them over protocol errors, even if it's just the one
		nodeErr := rp.GetBestNodeErrorMessageForUser()
		processingError = nodeErr.err
		errorResponse := nodeErr.response
		if errorResponse != nil {
			returnedResult = &errorResponse.relayResult
		}
	} else if protocolErrorCount > 0 {
		protocolErr := rp.GetBestProtocolErrorMessageForUser()
		processingError = protocolErr.err
		errorResponse := protocolErr.response
		if errorResponse != nil {
			returnedResult = &errorResponse.relayResult
		}
	}
	returnedResult.ProviderInfo.ProviderAddress = strings.Join(allProvidersAddresses, ",")
	return returnedResult, utils.LavaFormatError("failed relay, insufficient results", processingError)
}
