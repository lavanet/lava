package relaycore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/common/types"
)

type RelayProcessor struct {
	usedProviders                *lavasession.UsedProviders
	responses                    chan *RelayResponse
	quorumParams                 common.QuorumParams
	lock                         sync.RWMutex
	guid                         uint64
	selection                    Selection
	consistency                  Consistency
	skipDataReliability          bool
	debugRelay                   bool
	allowSessionDegradation      uint32 // used in the scenario where extension was previously used.
	metricsInf                   MetricsInterface
	chainIdAndApiInterfaceGetter ChainIdAndApiInterfaceGetter
	relayRetriesManager          *lavaprotocol.RelayRetriesManager
	ResultsManager
	RelayStateMachine
	availabilityDegrader      QoSAvailabilityDegrader
	qourumMap                 map[string]int
	currentQourumEqualResults int
	statefulRelayTargets      []string // stores all providers that received a stateful relay
}

func NewRelayProcessor(
	ctx context.Context,
	quorumParams common.QuorumParams,
	consistency Consistency,
	metricsInf MetricsInterface,
	chainIdAndApiInterfaceGetter ChainIdAndApiInterfaceGetter,
	relayRetriesManager *lavaprotocol.RelayRetriesManager,
	relayStateMachine RelayStateMachine,
	availabilityDegrader QoSAvailabilityDegrader,
) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	if quorumParams.Min <= 0 {
		utils.LavaFormatFatal("invalid requirement, successes count must be greater than 0", nil, utils.LogAttr("requiredSuccesses", quorumParams.Min))
	}
	relayProcessor := &RelayProcessor{
		quorumParams:                 quorumParams,
		responses:                    make(chan *RelayResponse, MaxCallsPerRelay), // we set it as buffered so it is not blocking
		ResultsManager:               NewResultsManager(guid),
		guid:                         guid,
		consistency:                  consistency,
		debugRelay:                   relayStateMachine.GetDebugState(),
		metricsInf:                   metricsInf,
		chainIdAndApiInterfaceGetter: chainIdAndApiInterfaceGetter,
		relayRetriesManager:          relayRetriesManager,
		RelayStateMachine:            relayStateMachine,
		selection:                    relayStateMachine.GetSelection(),
		usedProviders:                relayStateMachine.GetUsedProviders(),
		availabilityDegrader:         availabilityDegrader,
		qourumMap:                    make(map[string]int),
		currentQourumEqualResults:    0,
	}
	relayProcessor.RelayStateMachine.SetResultsChecker(relayProcessor)
	relayProcessor.RelayStateMachine.SetRelayRetriesManager(relayRetriesManager)
	return relayProcessor
}

func (rp *RelayProcessor) GetQuorumParams() common.QuorumParams {
	return rp.quorumParams
}

// true if we never got an extension. (default value)
func (rp *RelayProcessor) GetAllowSessionDegradation() bool {
	return atomic.LoadUint32(&rp.allowSessionDegradation) == 0
}

// in case we had an extension and managed to get a session successfully, we prevent session degradation.
func (rp *RelayProcessor) SetDisallowDegradation() {
	atomic.StoreUint32(&rp.allowSessionDegradation, 1)
}

func (rp *RelayProcessor) SetSkipDataReliability(val bool) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	rp.skipDataReliability = val
}

func (rp *RelayProcessor) GetSkipDataReliability() bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.skipDataReliability
}

// SetStatefulRelayTargets stores the list of providers that received a stateful relay
func (rp *RelayProcessor) SetStatefulRelayTargets(providers []string) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	rp.statefulRelayTargets = providers
}

// GetStatefulRelayTargets returns the list of providers that received a stateful relay
func (rp *RelayProcessor) GetStatefulRelayTargets() []string {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.statefulRelayTargets
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

func (rp *RelayProcessor) SetResponse(response *RelayResponse) {
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
	if rp.ResultsManager.RequiredResults(rp.quorumParams.Min, rp.selection) {
		utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - RequiredResults", utils.LogAttr("GUID", rp.guid), utils.LogAttr("requiredSuccesses", rp.quorumParams.Min), utils.LogAttr("selection", rp.selection))
		return true
	}
	// check if we got all of the responses
	if responsesCount >= rp.usedProviders.SessionsLatestBatch() {
		utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - SessionsLatestBatch", utils.LogAttr("GUID", rp.guid), utils.LogAttr("responsesCount", responsesCount), utils.LogAttr("SessionsLatestBatch", rp.usedProviders.SessionsLatestBatch()))
		// no active sessions, and we read all the responses, we can return
		return true
	}
	utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - false", utils.LogAttr("GUID", rp.guid), utils.LogAttr("responsesCount", responsesCount), utils.LogAttr("SessionsLatestBatch", rp.usedProviders.SessionsLatestBatch()))
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

// HasUnsupportedMethodErrors checks if any of the current errors are unsupported method errors
func (rp *RelayProcessor) HasUnsupportedMethodErrors() bool {
	if rp == nil {
		return false
	}

	// Get actual error data to check for unsupported method errors
	_, nodeErrorResults, protocolErrors := rp.GetResultsData()

	// Check node errors
	for _, nodeErrorResult := range nodeErrorResults {
		if nodeErrorResult.Reply != nil && nodeErrorResult.Reply.Data != nil {
			// Check if this is an unsupported method error based on the reply
			if chainlib.IsUnsupportedMethodErrorMessage(string(nodeErrorResult.Reply.Data)) {
				return true
			}
		}
	}

	// Check protocol errors
	for _, protocolError := range protocolErrors {
		// Check if this is an unsupported method error
		if chainlib.IsUnsupportedMethodError(protocolError.GetError()) {
			return true
		}
		// Check for epoch mismatch errors that should be retried
		if lavasession.EpochMismatchError.Is(protocolError.GetError()) {
			// Epoch mismatch errors should be retried, not treated as unsupported
			continue
		}
		// Also check if we shouldn't retry this error
		if !chainlib.ShouldRetryError(protocolError.GetError()) {
			return true
		}
	}

	return false
}

// Deciding wether we should send a relay retry attempt based on the node error
func (rp *RelayProcessor) shouldRetryRelay(resultsCount int, hashErr error, nodeErrors int, specialNodeErrors int) bool {
	utils.LavaFormatDebug("shouldRetryRelay called", utils.LogAttr("GUID", rp.guid), utils.LogAttr("resultsCount", resultsCount), utils.LogAttr("hashErr", hashErr), utils.LogAttr("nodeErrors", nodeErrors), utils.LogAttr("specialNodeErrors", specialNodeErrors))

	// Never retry if we detect unsupported method errors
	if rp.HasUnsupportedMethodErrors() {
		return false
	}

	// Check if we have epoch mismatch errors that warrant retry
	_, _, protocolErrors := rp.GetResultsData()
	hasEpochMismatchError := false
	for _, protocolError := range protocolErrors {
		if lavasession.EpochMismatchError.Is(protocolError.GetError()) {
			hasEpochMismatchError = true
			break
		}
	}

	// Retries will be performed based on the following scenarios:
	// 1. If RelayCountOnNodeError > 0
	// 2. If we have 0 successful relays and we have only node errors.
	// 3. If we have epoch mismatch errors (temporary synchronization issues)
	// 4. Number of retries < RelayCountOnNodeError.
	if hasEpochMismatchError && resultsCount == 0 {
		// Allow retries for epoch mismatch errors even with higher tolerance
		return true
	}

	if RelayCountOnNodeError > 0 && resultsCount == 0 && hashErr == nil {
		if nodeErrors <= RelayCountOnNodeError && specialNodeErrors <= RelayCountOnNodeError {
			return true
		}
	}
	// Do not perform a retry
	return false
}

func (rp *RelayProcessor) HasRequiredNodeResults(tries int) (bool, int) {
	if rp == nil {
		return false, 0
	}
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	resultsCount, nodeErrors, specialNodeErrors, _ := rp.GetResults()

	hash, hashErr := rp.getInputMsgInfoHashString()
	neededForQuorum := rp.getActualQuorumSize(resultsCount)
	if rp.quorumParams.Enabled() && neededForQuorum <= rp.currentQourumEqualResults ||
		!rp.quorumParams.Enabled() && resultsCount >= rp.quorumParams.Min {
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
		if rp.debugRelay {
			utils.LavaFormatDebug("HasRequiredNodeResults quorum met",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("tries", tries),
				utils.LogAttr("neededForQuorum", neededForQuorum),
				utils.LogAttr("quorumParams.Min", rp.quorumParams.Min),
				utils.LogAttr("resultsCount", resultsCount),
				utils.LogAttr("nodeErrors", nodeErrors),
				utils.LogAttr("specialNodeErrors", specialNodeErrors),
				utils.LogAttr("currentQourumEqualResults", rp.currentQourumEqualResults),
			)
		}
		return true, nodeErrors
	}
	if rp.selection == Quorum {
		// We need a quorum of all node results

		// Define retryOnNodeErrorFlow based on quorum feature status
		var retryForQuorumNeeded bool

		if rp.quorumParams.Enabled() {
			// Quorum feature enabled: check if quorum is still mathematically possible and quorum is not yet reached
			// Only count successful results for quorum calculation
			maxRemainingProviders := rp.quorumParams.Max - resultsCount
			// The following line checks if, after accounting for the maximum possible additional successful responses (maxRemainingProviders)
			// and the current highest number of matching responses (rp.currentQourumEqualResults), it is still mathematically possible
			// to reach the required quorum threshold (calculated as quorum rate * max providers).
			// If not, then retrying is not needed because quorum cannot be achieved anymore.
			retryForQuorumNeeded = maxRemainingProviders+rp.currentQourumEqualResults >= int(math.Ceil(rp.quorumParams.Rate*float64(rp.quorumParams.Max)))
			if rp.debugRelay {
				utils.LavaFormatDebug("HasRequiredNodeResults retryForQuorumNeeded calculation", utils.LogAttr("GUID", rp.guid),
					utils.LogAttr("maxRemainingProviders", maxRemainingProviders),
					utils.LogAttr("rp.currentQourumEqualResults", rp.currentQourumEqualResults),
					utils.LogAttr("retryForQuorumNeeded", retryForQuorumNeeded),
				)
			}
		} else {
			// Quorum feature disabled: check if we have enough results for quorum
			retryForQuorumNeeded = !(resultsCount >= neededForQuorum)
		}

		if !retryForQuorumNeeded {
			// Retry on node error flow:
			shouldRetry := rp.shouldRetryRelay(resultsCount, hashErr, nodeErrors, specialNodeErrors)
			if rp.debugRelay {
				utils.LavaFormatDebug("HasRequiredNodeResults shouldRetry",
					utils.LogAttr("GUID", rp.guid),
					utils.LogAttr("shouldRetry", shouldRetry),
					utils.LogAttr("tries", tries),
					utils.LogAttr("neededForQuorum", neededForQuorum),
					utils.LogAttr("quorumParams.Min", rp.quorumParams.Min),
					utils.LogAttr("resultsCount", resultsCount),
					utils.LogAttr("nodeErrors", nodeErrors),
					utils.LogAttr("specialNodeErrors", specialNodeErrors),
					utils.LogAttr("currentQourumEqualResults", rp.currentQourumEqualResults),
				)
			}
			return !shouldRetry, nodeErrors
		}
	}
	// on BestResult we want to retry if there is no success
	if rp.debugRelay {
		utils.LavaFormatDebug("HasRequiredNodeResults returning false",
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("tries", tries),
			utils.LogAttr("neededForQuorum", neededForQuorum),
			utils.LogAttr("quorumParams.Min", rp.quorumParams.Min),
			utils.LogAttr("resultsCount", resultsCount),
			utils.LogAttr("nodeErrors", nodeErrors),
			utils.LogAttr("specialNodeErrors", specialNodeErrors),
			utils.LogAttr("currentQourumEqualResults", rp.currentQourumEqualResults),
		)
	}
	return false, nodeErrors
}

func (rp *RelayProcessor) handleResponse(response *RelayResponse) {
	nodeError := rp.ResultsManager.SetResponse(response, rp.RelayStateMachine.GetProtocolMessage())

	// send relay error metrics only on non stateful queries, as stateful queries always return X-1/X errors.
	if nodeError != nil && rp.selection != BestResult {
		chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
		go rp.metricsInf.SetRelayNodeErrorMetric(response.RelayResult.ProviderInfo.ProviderAddress, chainId, apiInterface)
		utils.LavaFormatInfo("Relay received a node error", utils.LogAttr("Error", nodeError), utils.LogAttr("provider", response.RelayResult.ProviderInfo), utils.LogAttr("Request", rp.RelayStateMachine.GetProtocolMessage().GetApi().Name))
	}

	if response != nil {
		canonicalForm, err := types.CreateCanonicalForm(response.RelayResult.GetReply().GetData())
		if err == nil {
			rp.qourumMap[canonicalForm]++
			if rp.qourumMap[canonicalForm] > rp.currentQourumEqualResults {
				rp.currentQourumEqualResults = rp.qourumMap[canonicalForm]
			}
		}

		if rp.consistency != nil && response.RelayResult.Reply != nil && response.RelayResult.Reply.LatestBlock > 0 {
			// set consistency when possible
			blockSeen := response.RelayResult.Reply.LatestBlock
			rp.consistency.SetSeenBlock(blockSeen, rp.RelayStateMachine.GetProtocolMessage().GetUserData())
		}
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

// getActualQuorumSize calculates the actual quorum size based on quorum rate and succeeded replies
func (rp *RelayProcessor) getActualQuorumSize(succeededReplies int) int {
	return int(math.Ceil(rp.quorumParams.Rate * float64(utils.Max(rp.quorumParams.Min, succeededReplies))))
}

func (rp *RelayProcessor) responsesQuorum(results []common.RelayResult, quorumSize int) (returnedResult *common.RelayResult, processingError error) {
	if quorumSize <= 0 {
		return nil, errors.New("quorumSize must be greater than zero")
	}

	// Declare originalQuorumSize at function level for use in error messages
	var originalQuorumSize int

	// When quorum feature is enabled, use the actual calculated quorum size for all comparisons
	if rp.quorumParams.Enabled() {
		// Preserve the original quorum size for error reporting
		originalQuorumSize = quorumSize

		actualQuorumSize := rp.getActualQuorumSize(len(results))
		quorumSize = actualQuorumSize
	}

	type resultCount struct {
		count  int
		result common.RelayResult
	}

	countMap := make(map[string]*resultCount)
	deterministic := rp.RelayStateMachine.GetProtocolMessage().GetApi().Category.Deterministic
	var bestQosResult common.RelayResult
	bestQos := sdktypes.ZeroDec()
	nilReplies := 0
	nilReplyIdx := -1

	for idx, result := range results {
		if result.Reply != nil && result.Reply.Data != nil {
			// Create canonical form for comparison (handles both JSON and binary data)
			canonicalForm, err := types.CreateCanonicalForm(result.Reply.Data)
			if err != nil {
				utils.LavaFormatError("failed to create canonical form", err, utils.LogAttr("result", result.Reply.Data))
				continue
			}

			if count, exists := countMap[canonicalForm]; exists {
				count.count++
			} else {
				countMap[canonicalForm] = &resultCount{
					count:  1,
					result: result,
				}
			}

			if !deterministic {
				if result.ProviderInfo.ProviderReputationSummary.IsNil() || result.ProviderInfo.ProviderStake.Amount.IsNil() {
					continue
				}
				currentResult := result.ProviderInfo.ProviderReputationSummary.MulInt(result.ProviderInfo.ProviderStake.Amount)
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

	var maxCount int
	var mostCommonResult common.RelayResult
	for _, count := range countMap {
		if count.count > maxCount {
			maxCount = count.count
			mostCommonResult = count.result
		}
	}

	if nilReplies >= quorumSize && maxCount < quorumSize {
		maxCount = nilReplies
		mostCommonResult = results[nilReplyIdx]
	}

	if maxCount < quorumSize {
		if !deterministic {
			bestQosResult.Quorum = 1
			return &bestQosResult, nil
		}
		// Only apply quorum logic when quorum feature is enabled
		if rp.quorumParams.Enabled() {
			return nil, utils.LavaFormatInfo("equal results count is less than actualQuorumSize",
				utils.LogAttr("nilReplies", nilReplies),
				utils.LogAttr("results", len(results)),
				utils.LogAttr("quorumEqualResults", maxCount),
				utils.LogAttr("actualQuorumSize", quorumSize),
				utils.LogAttr("originalQuorumSize", originalQuorumSize),
				utils.LogAttr("succeededReplies", len(results)),
				utils.LogAttr("quorumRate", rp.quorumParams.Rate))
		} else {
			// Quorum feature disabled - return original error message
			return nil, utils.LavaFormatInfo("majority count is less than quorumSize",
				utils.LogAttr("nilReplies", nilReplies),
				utils.LogAttr("results", len(results)),
				utils.LogAttr("maxCount", maxCount),
				utils.LogAttr("quorumSize", quorumSize))
		}
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
	shouldDegradeAvailability := false

	rp.lock.RLock()
	defer rp.lock.RUnlock()

	isSpecialApi := rp.GetProtocolMessage().IsDefaultApi() || rp.GetProtocolMessage().GetApi().Category.Stateful == common.CONSISTENCY_SELECT_ALL_PROVIDERS
	successResults, nodeErrors, protocolErrors := rp.GetResultsData()
	successResultsCount, nodeErrorCount, protocolErrorCount := len(successResults), len(nodeErrors), len(protocolErrors)

	defer func() {
		if shouldDegradeAvailability {
			if rp.availabilityDegrader == nil {
				utils.LavaFormatWarning("Availability degrader is nil, skipping availability degradation", nil)
				return
			}
			for _, result := range nodeErrors {
				session := result.Request.RelaySession
				utils.LavaFormatDebug("Degrading availability for provider",
					utils.LogAttr("provider", result.ProviderInfo.ProviderAddress),
					utils.LogAttr("epoch", session.Epoch),
					utils.LogAttr("sessionId", session.SessionId),
				)
				rp.availabilityDegrader.DegradeAvailability(uint64(session.Epoch), int64(session.SessionId))
			}
		}
	}()

	// Calculate the required quorum size
	var requiredQuorumSize int
	if rp.quorumParams.Enabled() {
		requiredQuorumSize = rp.getActualQuorumSize(successResultsCount)
	} else {
		requiredQuorumSize = rp.quorumParams.Min
	}

	if rp.debugRelay {
		utils.LavaFormatDebug("[ProcessingResult]:", utils.LogAttr("GUID", rp.guid), utils.LogAttr("successResultsCount", successResultsCount), utils.LogAttr("quorumParams.Min", rp.quorumParams.Min), utils.LogAttr("requiredQuorumSize", requiredQuorumSize))
	}

	// there are enough successes
	if successResultsCount >= requiredQuorumSize {
		if len(nodeErrors) > 0 && !isSpecialApi { // if we have node errors and it's not a default api, we should degrade availability
			shouldDegradeAvailability = true
		}
		return rp.responsesQuorum(successResults, requiredQuorumSize)
	}

	//in case quorum feature is disabled, we can return the node errors if they meet the quorum size (which is 1 when quorum feature is disabled)
	if !rp.quorumParams.Enabled() && nodeErrorCount >= requiredQuorumSize {
		if len(nodeErrors) > 0 && !isSpecialApi { // if we have node errors and it's not a default api, we should degrade availability
			shouldDegradeAvailability = true
		}
		return rp.responsesQuorum(nodeErrors, requiredQuorumSize)
	}

	if rp.debugRelay {
		// adding as much debug info as possible. all successful relays, all node errors and all protocol errors
		utils.LavaFormatDebug("[Processing Result] Debug Relay", utils.LogAttr("rp.quorumParams.Min", rp.quorumParams.Min))
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

	// there are not enough successes, let's check if there are enough node errors and protocol errors
	// Protocol errors (like consistency violations) should count as attempted responses
	totalResponses := successResultsCount + nodeErrorCount + protocolErrorCount
	if totalResponses >= rp.quorumParams.Min {
		if rp.selection == Quorum {
			// When quorum is enabled, we need to ensure we have enough successful responses
			// to actually meet quorum requirements, not just enough total attempts
			if rp.quorumParams.Enabled() && successResultsCount < rp.quorumParams.Min {
				// We have enough total responses, but not enough successful ones for quorum
				return nil, utils.LavaFormatError("insufficient successful responses for quorum",
					nil,
					utils.LogAttr("successResultsCount", successResultsCount),
					utils.LogAttr("nodeErrorCount", nodeErrorCount),
					utils.LogAttr("protocolErrorCount", protocolErrorCount),
					utils.LogAttr("quorumMin", rp.quorumParams.Min),
					utils.LogAttr("totalResponses", totalResponses))
			}
			// Only combine if we have at least some successes
			// If we have ONLY node errors but didn't meet quorum as checked above,
			// combining them won't help since they'd still be the same responses
			if successResultsCount > 0 {
				nodeResults := make([]common.RelayResult, 0, len(successResults)+len(nodeErrors))
				nodeResults = append(nodeResults, successResults...)
				nodeResults = append(nodeResults, nodeErrors...)
				return rp.responsesQuorum(nodeResults, requiredQuorumSize)
			}
		}
	}

	// Not enough successful results - continue waiting for more responses
	// if we got here we trigger a protocol error
	returnedResult = &common.RelayResult{StatusCode: http.StatusInternalServerError}
	if nodeErrorCount > 0 { // if we have node errors, we prefer returning them over protocol errors, even if it's just the one
		nodeErr := rp.GetBestNodeErrorMessageForUser()
		processingError = nodeErr.Err
		errorResponse := nodeErr.Response
		if errorResponse != nil {
			returnedResult = &errorResponse.RelayResult
		}
	} else if protocolErrorCount > 0 {
		protocolErr := rp.GetBestProtocolErrorMessageForUser()
		processingError = protocolErr.Err
		errorResponse := protocolErr.Response
		if errorResponse != nil {
			returnedResult = &errorResponse.RelayResult
		}
	}
	returnedResult.ProviderInfo.ProviderAddress = strings.Join(allProvidersAddresses, ",")
	return returnedResult, utils.LavaFormatError("failed relay, insufficient results", processingError)
}
