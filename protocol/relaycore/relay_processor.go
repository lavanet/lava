package relaycore

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	quorumMap                 map[[32]byte]int
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
		quorumMap:                    make(map[[32]byte]int),
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
	successResults, nodeErrorResults, protocolErrors := rp.GetResultsData()

	// NEW: Check success results for IsUnsupportedMethod flag
	for _, result := range successResults {
		if result.IsUnsupportedMethod {
			return true
		}
	}

	// Check node errors for IsUnsupportedMethod flag
	for _, nodeErrorResult := range nodeErrorResults {
		// NEW: Check flag first (preferred method)
		if nodeErrorResult.IsUnsupportedMethod {
			return true
		}

		// KEEP: Also check reply data as backup (for backward compatibility)
		if nodeErrorResult.Reply != nil && nodeErrorResult.Reply.Data != nil {
			// Check if this is an unsupported method error based on the reply
			if chainlib.IsUnsupportedMethodErrorMessageBytes(nodeErrorResult.Reply.Data) {
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
	resultsCount, nodeErrors, specialNodeErrors, protocolErrors := rp.GetResults()

	hash, hashErr := rp.getInputMsgInfoHashString()
	neededForQuorum := rp.getRequiredQuorumSize(resultsCount)
	if rp.quorumParams.Enabled() && neededForQuorum <= rp.currentQourumEqualResults ||
		!rp.quorumParams.Enabled() && resultsCount >= neededForQuorum {
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

			// Check if we need to add protocol errors retry metrics
			if protocolErrors > 0 {
				chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
				go rp.metricsInf.SetProtocolErrorRecoveredSuccessfullyMetric(chainId, apiInterface, strconv.Itoa(protocolErrors))
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

	// Only hash successful responses (not errors) for quorum tracking
	// This prevents error responses from being counted toward quorum
	if response != nil && nodeError == nil && response.Err == nil {
		// Hash the response data instead of creating canonical form - much more efficient
		hash := sha256.Sum256(response.RelayResult.GetReply().GetData())
		rp.quorumMap[hash]++
		if rp.quorumMap[hash] > rp.currentQourumEqualResults {
			rp.currentQourumEqualResults = rp.quorumMap[hash]
		}
	}

	if response != nil && response.RelayResult.Reply != nil {
		if rp.consistency != nil && response.RelayResult.Reply.LatestBlock > 0 {
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

// getRequiredQuorumSize calculates the required number of matching responses.
// When quorum feature is enabled: applies the quorum rate formula to the response count
// When quorum feature is disabled: returns the configured minimum (typically 1)
// This ensures consistent quorum calculation across all scenarios.
func (rp *RelayProcessor) getRequiredQuorumSize(responseCount int) int {
	if rp.quorumParams.Enabled() {
		return int(math.Ceil(rp.quorumParams.Rate * float64(utils.Max(rp.quorumParams.Min, responseCount))))
	}
	return rp.quorumParams.Min
}

func (rp *RelayProcessor) responsesQuorum(results []common.RelayResult, quorumSize int) (returnedResult *common.RelayResult, processingError error) {
	if quorumSize <= 0 {
		return nil, errors.New("quorumSize must be greater than zero")
	}

	type resultCount struct {
		count  int
		result common.RelayResult
	}

	countMap := make(map[[32]byte]*resultCount)
	deterministic := rp.RelayStateMachine.GetProtocolMessage().GetApi().Category.Deterministic
	var bestQosResult common.RelayResult
	bestQos := sdktypes.ZeroDec()
	nilReplies := 0
	nilReplyIdx := -1

	// Helper function to check if response data is valid and meaningful
	isValidResponse := func(data []byte) bool {
		if len(data) == 0 {
			return false
		}
		// Check for empty JSON objects/arrays that are technically valid but meaningless
		trimmed := bytes.TrimSpace(data)
		if bytes.Equal(trimmed, []byte("{}")) || bytes.Equal(trimmed, []byte("[]")) {
			return false
		}
		return true
	}

	for idx, result := range results {
		if result.Reply != nil && result.Reply.Data != nil && isValidResponse(result.Reply.Data) {
			// Hash the response data for comparison - much faster and more memory efficient
			// than creating canonical forms. SHA256 ensures reliable comparison for quorum.
			hash := sha256.Sum256(result.Reply.Data)

			if count, exists := countMap[hash]; exists {
				count.count++
			} else {
				countMap[hash] = &resultCount{
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
			return nil, utils.LavaFormatInfo("equal results count is less than requiredQuorumSize",
				utils.LogAttr("nilReplies", nilReplies),
				utils.LogAttr("results", len(results)),
				utils.LogAttr("quorumEqualResults", maxCount),
				utils.LogAttr("requiredQuorumSize", quorumSize),
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

	// Calculate the required quorum size using the unified function
	requiredQuorumSize := rp.getRequiredQuorumSize(successResultsCount)

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
		utils.LavaFormatDebug("[ProcessingResult]:", utils.LogAttr("GUID", rp.guid), utils.LogAttr("successResultsCount", successResultsCount), utils.LogAttr("quorumParams.Min", rp.quorumParams.Min), utils.LogAttr("requiredQuorumSize", requiredQuorumSize))
	}

	// there are enough successes
	if successResultsCount >= requiredQuorumSize {
		// Try to form a quorum with successes first
		result, err := rp.responsesQuorum(successResults, requiredQuorumSize)
		if err == nil {
			// Successes formed a quorum
			if len(nodeErrors) > 0 && !isSpecialApi { // if we have node errors and it's not a default api, we should degrade availability
				shouldDegradeAvailability = true
			}
			return result, nil
		}

		// Successes couldn't form a quorum (they don't match), try node errors if available
		requiredNodeErrorsQuorumSize := rp.getRequiredQuorumSize(nodeErrorCount)
		if nodeErrorCount >= requiredNodeErrorsQuorumSize {
			utils.LavaFormatInfo("Success responses didn't match, attempting node error quorum as fallback",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("successCount", successResultsCount),
				utils.LogAttr("nodeErrorCount", nodeErrorCount),
				utils.LogAttr("requiredNodeErrorsQuorumSize", requiredNodeErrorsQuorumSize),
				utils.LogAttr("quorumEnabled", rp.quorumParams.Enabled()),
			)
			nodeErrorResult, nodeErrorErr := rp.responsesQuorum(nodeErrors, requiredNodeErrorsQuorumSize)
			if nodeErrorErr == nil {
				// Node errors formed a quorum, use them as fallback
				utils.LavaFormatInfo("Using node error quorum as fallback (success responses didn't match)",
					utils.LogAttr("GUID", rp.guid),
					utils.LogAttr("successCount", successResultsCount),
					utils.LogAttr("nodeErrorCount", nodeErrorCount),
					utils.LogAttr("nodeErrorQuorum", nodeErrorResult.Quorum),
				)
				return nodeErrorResult, nil
			}
			utils.LavaFormatDebug("Node errors also failed to form quorum",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("nodeErrorCount", nodeErrorCount),
				utils.LogAttr("requiredQuorumSize", requiredNodeErrorsQuorumSize),
				utils.LogAttr("error", nodeErrorErr),
			)
		}
		// Neither successes nor node errors could form a quorum, return the error from successes
		return result, err
	}

	// there are not enough successes, let's check if there are enough node errors and protocol errors
	// Protocol errors (like consistency violations) should count as attempted responses
	totalResponses := successResultsCount + nodeErrorCount + protocolErrorCount
	if totalResponses >= rp.quorumParams.Min && rp.selection == Quorum {
		// Check if we have enough node errors to form a quorum
		// Recalculate required quorum size based on node error count
		requiredQuorumSize = rp.getRequiredQuorumSize(nodeErrorCount)
		if successResultsCount == 0 && nodeErrorCount >= requiredQuorumSize {
			// Try to form a quorum with node errors first (no successes available)
			utils.LavaFormatInfo("No success responses, attempting quorum with node errors only",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("nodeErrorCount", nodeErrorCount),
				utils.LogAttr("requiredQuorumSize", requiredQuorumSize),
				utils.LogAttr("protocolErrorCount", protocolErrorCount),
				utils.LogAttr("quorumEnabled", rp.quorumParams.Enabled()),
			)
			return rp.responsesQuorum(nodeErrors, requiredQuorumSize)
		}

		// If we couldn't form a quorum with node errors, try combining with successes
		if nodeErrorCount+successResultsCount >= requiredQuorumSize {
			utils.LavaFormatInfo("Attempting quorum with combined node errors and successes",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("successCount", successResultsCount),
				utils.LogAttr("nodeErrorCount", nodeErrorCount),
				utils.LogAttr("combinedCount", nodeErrorCount+successResultsCount),
				utils.LogAttr("requiredQuorumSize", requiredQuorumSize),
				utils.LogAttr("quorumEnabled", rp.quorumParams.Enabled()),
			)
			nodeResults := make([]common.RelayResult, 0, len(successResults)+len(nodeErrors))
			nodeResults = append(nodeResults, successResults...)
			nodeResults = append(nodeResults, nodeErrors...)
			return rp.responsesQuorum(nodeResults, requiredQuorumSize)
		}
	}

	if rp.selection == BestResult && nodeErrorCount > 0 {
		return rp.responsesQuorum(nodeErrors, rp.quorumParams.Min)
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
