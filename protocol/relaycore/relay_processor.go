package relaycore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
)

type RelayProcessor struct {
	usedProviders                *lavasession.UsedProviders
	responses                    chan *RelayResponse
	crossValidationParams        *common.CrossValidationParams // nil for Stateless/Stateful, non-nil for CrossValidation
	lock                         sync.RWMutex
	guid                         uint64
	selection                    Selection
	consistency                  Consistency
	debugRelay                   bool
	allowSessionDegradation      uint32 // used in the scenario where extension was previously used.
	metricsInf                   MetricsInterface
	chainIdAndApiInterfaceGetter ChainIdAndApiInterfaceGetter
	relayRetriesManager          *lavaprotocol.RelayRetriesManager
	ResultsManager
	RelayStateMachine
	crossValidationMap                 map[[32]byte]int
	currentCrossValidationEqualResults int
	statefulRelayTargets               []string // stores all providers that received a stateful relay
	crossValidationQueriedProviders    []string // stores all providers that were queried for cross-validation (even if response not received)
}

func NewRelayProcessor(
	ctx context.Context,
	crossValidationParams *common.CrossValidationParams, // nil for Stateless/Stateful
	consistency Consistency,
	metricsInf MetricsInterface,
	chainIdAndApiInterfaceGetter ChainIdAndApiInterfaceGetter,
	relayRetriesManager *lavaprotocol.RelayRetriesManager,
	relayStateMachine RelayStateMachine,
) *RelayProcessor {
	guid, _ := utils.GetUniqueIdentifier(ctx)
	selection := relayStateMachine.GetSelection()

	// Defensive validation - these should never fail in production as params
	// are validated at parse time, but guards against programming errors
	if selection == CrossValidation && crossValidationParams == nil {
		utils.LavaFormatFatal("CrossValidation selection requires non-nil crossValidationParams", nil)
	}
	if crossValidationParams != nil {
		if crossValidationParams.AgreementThreshold < 1 {
			utils.LavaFormatFatal("invalid cross-validation AgreementThreshold", nil,
				utils.LogAttr("AgreementThreshold", crossValidationParams.AgreementThreshold))
		}
		if crossValidationParams.MaxParticipants < 1 {
			utils.LavaFormatFatal("invalid cross-validation MaxParticipants", nil,
				utils.LogAttr("MaxParticipants", crossValidationParams.MaxParticipants))
		}
		if crossValidationParams.MaxParticipants > MaxCallsPerRelay {
			utils.LavaFormatFatal("cross-validation MaxParticipants exceeds maximum allowed",
				nil,
				utils.LogAttr("MaxParticipants", crossValidationParams.MaxParticipants),
				utils.LogAttr("MaxCallsPerRelay", MaxCallsPerRelay))
		}
	}

	relayProcessor := &RelayProcessor{
		crossValidationParams:              crossValidationParams,
		responses:                          make(chan *RelayResponse, MaxCallsPerRelay), // buffered to prevent blocking
		ResultsManager:                     NewResultsManager(guid),
		guid:                               guid,
		consistency:                        consistency,
		debugRelay:                         relayStateMachine.GetDebugState(),
		metricsInf:                         metricsInf,
		chainIdAndApiInterfaceGetter:       chainIdAndApiInterfaceGetter,
		relayRetriesManager:                relayRetriesManager,
		RelayStateMachine:                  relayStateMachine,
		selection:                          selection,
		usedProviders:                      relayStateMachine.GetUsedProviders(),
		crossValidationMap:                 make(map[[32]byte]int),
		currentCrossValidationEqualResults: 0,
	}
	relayProcessor.RelayStateMachine.SetResultsChecker(relayProcessor)
	relayProcessor.RelayStateMachine.SetRelayRetriesManager(relayRetriesManager)
	return relayProcessor
}

func (rp *RelayProcessor) GetCrossValidationParams() *common.CrossValidationParams {
	return rp.crossValidationParams
}

// getAgreementThreshold returns the agreement threshold or 1 if not in CrossValidation mode
func (rp *RelayProcessor) getAgreementThreshold() int {
	if rp.crossValidationParams != nil {
		return rp.crossValidationParams.AgreementThreshold
	}
	return 1
}

// getMaxParticipants returns the max participants or 1 if not in CrossValidation mode
func (rp *RelayProcessor) getMaxParticipants() int {
	if rp.crossValidationParams != nil {
		return rp.crossValidationParams.MaxParticipants
	}
	return 1
}

// true if we never got an extension. (default value)
func (rp *RelayProcessor) GetAllowSessionDegradation() bool {
	return atomic.LoadUint32(&rp.allowSessionDegradation) == 0
}

// in case we had an extension and managed to get a session successfully, we prevent session degradation.
func (rp *RelayProcessor) SetDisallowDegradation() {
	atomic.StoreUint32(&rp.allowSessionDegradation, 1)
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

// SetCrossValidationQueriedProviders stores the list of all providers that were queried for cross-validation
// This includes providers whose responses may not have been received (due to early exit when threshold met)
func (rp *RelayProcessor) SetCrossValidationQueriedProviders(providers []string) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	rp.crossValidationQueriedProviders = providers
}

// GetCrossValidationQueriedProviders returns the list of all providers that were queried for cross-validation
func (rp *RelayProcessor) GetCrossValidationQueriedProviders() []string {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	return rp.crossValidationQueriedProviders
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

	// Common exit condition: all responses received from all providers in the batch
	if responsesCount >= rp.usedProviders.SessionsLatestBatch() {
		utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - all responses received",
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("selection", rp.selection),
			utils.LogAttr("responsesCount", responsesCount),
			utils.LogAttr("SessionsLatestBatch", rp.usedProviders.SessionsLatestBatch()))
		return true
	}

	// Mode-specific early exit conditions
	switch rp.selection {
	case CrossValidation:
		// Early exit if we've reached the agreement threshold
		if rp.currentCrossValidationEqualResults >= rp.getAgreementThreshold() {
			utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - CrossValidation threshold met",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("agreementThreshold", rp.getAgreementThreshold()),
				utils.LogAttr("currentEqualResults", rp.currentCrossValidationEqualResults))
			return true
		}
	case Stateless, Stateful:
		// Early exit if we have a successful result
		if rp.ResultsManager.RequiredResults(1, rp.selection) {
			utils.LavaFormatDebug("[RelayProcessor] checkEndProcessing - RequiredResults met",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("selection", rp.selection))
			return true
		}
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

// isBatchRequest returns true if the current request is a batch request (e.g., JSON-RPC batch).
func (rp *RelayProcessor) isBatchRequest() bool {
	return rp.RelayStateMachine.GetProtocolMessage().IsBatch()
}

// HasUnsupportedMethodErrors checks if any of the current errors are unsupported method errors.
// Note: We only check nodeErrors and protocolErrors, not successResults, because:
// - The IsUnsupportedMethod flag is only set when isNodeError=true (in consumer/smartrouter)
// - If it's a node error, it goes to nodeErrors, never to successResults
func (rp *RelayProcessor) HasUnsupportedMethodErrors() bool {
	if rp == nil {
		return false
	}

	_, nodeErrorResults, protocolErrors := rp.GetResultsData()

	// Check node errors for IsUnsupportedMethod flag
	for _, nodeErrorResult := range nodeErrorResults {
		if nodeErrorResult.IsUnsupportedMethod {
			return true
		}
	}

	// Check protocol errors (for backward compatibility with old providers that may return gRPC errors)
	for _, protocolError := range protocolErrors {
		if chainlib.IsUnsupportedMethodError(protocolError.GetError()) {
			return true
		}
		// Epoch mismatch errors should be retried, not treated as unsupported
		if lavasession.EpochMismatchError.Is(protocolError.GetError()) {
			continue
		}
		// Check if this is a non-retryable error (indicates unsupported or permanent failure)
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

	// Never retry batch requests if DisableBatchRequestRetry is enabled
	if DisableBatchRequestRetry && rp.isBatchRequest() {
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

	// CrossValidation mode: check if agreementThreshold is met
	if rp.selection == CrossValidation {
		if rp.currentCrossValidationEqualResults >= rp.getAgreementThreshold() {
			if hashErr == nil {
				go rp.relayRetriesManager.RemoveHashFromCache(hash)
			}
			if rp.debugRelay {
				utils.LavaFormatDebug("HasRequiredNodeResults CrossValidation threshold met",
					utils.LogAttr("GUID", rp.guid),
					utils.LogAttr("tries", tries),
					utils.LogAttr("agreementThreshold", rp.getAgreementThreshold()),
					utils.LogAttr("currentCrossValidationEqualResults", rp.currentCrossValidationEqualResults),
					utils.LogAttr("resultsCount", resultsCount),
				)
			}
			return true, nodeErrors
		}
		// CrossValidation doesn't retry - return true only when all expected responses received
		// (The state machine handles no-retry logic)
		if rp.debugRelay {
			utils.LavaFormatDebug("HasRequiredNodeResults CrossValidation threshold not met",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("tries", tries),
				utils.LogAttr("agreementThreshold", rp.getAgreementThreshold()),
				utils.LogAttr("currentCrossValidationEqualResults", rp.currentCrossValidationEqualResults),
				utils.LogAttr("resultsCount", resultsCount),
			)
		}
		return false, nodeErrors
	}

	// Original logic for Stateless and Stateful modes
	// For Stateless/Stateful, we need at least 1 successful response
	if resultsCount >= 1 {
		if hashErr == nil { // Incase we had a successful relay we can remove the hash from our relay retries map
			// Use a routine to run it in parallel
			go rp.relayRetriesManager.RemoveHashFromCache(hash)
		}
		// Check if we need to add node errors retry metrics
		if rp.selection == Stateless {
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
			utils.LavaFormatDebug("HasRequiredNodeResults requirements met",
				utils.LogAttr("GUID", rp.guid),
				utils.LogAttr("tries", tries),
				utils.LogAttr("resultsCount", resultsCount),
				utils.LogAttr("nodeErrors", nodeErrors),
				utils.LogAttr("specialNodeErrors", specialNodeErrors),
			)
		}
		return true, nodeErrors
	}

	if rp.selection == Stateless {
		// Check if we have enough results (need at least 1)
		needsMoreRetries := resultsCount < 1

		if !needsMoreRetries {
			// Retry on node error flow:
			shouldRetry := rp.shouldRetryRelay(resultsCount, hashErr, nodeErrors, specialNodeErrors)
			if rp.debugRelay {
				utils.LavaFormatDebug("HasRequiredNodeResults shouldRetry",
					utils.LogAttr("GUID", rp.guid),
					utils.LogAttr("shouldRetry", shouldRetry),
					utils.LogAttr("tries", tries),
					utils.LogAttr("resultsCount", resultsCount),
					utils.LogAttr("nodeErrors", nodeErrors),
					utils.LogAttr("specialNodeErrors", specialNodeErrors),
				)
			}
			return !shouldRetry, nodeErrors
		}
	}
	// on Stateful we want to retry if there is no success
	if rp.debugRelay {
		utils.LavaFormatDebug("HasRequiredNodeResults returning false",
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("tries", tries),
			utils.LogAttr("resultsCount", resultsCount),
			utils.LogAttr("nodeErrors", nodeErrors),
			utils.LogAttr("specialNodeErrors", specialNodeErrors),
		)
	}
	return false, nodeErrors
}

func (rp *RelayProcessor) handleResponse(response *RelayResponse) {
	nodeError := rp.ResultsManager.SetResponse(response, rp.RelayStateMachine.GetProtocolMessage())

	// send relay error metrics only on non stateful queries, as stateful queries always return X-1/X errors.
	if nodeError != nil && rp.selection != Stateful {
		chainId, apiInterface := rp.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface()
		go rp.metricsInf.SetRelayNodeErrorMetric(response.RelayResult.ProviderInfo.ProviderAddress, chainId, apiInterface)
		utils.LavaFormatInfo("Relay received a node error", utils.LogAttr("GUID", rp.guid), utils.LogAttr("Error", nodeError), utils.LogAttr("provider", response.RelayResult.ProviderInfo), utils.LogAttr("Request", rp.RelayStateMachine.GetProtocolMessage().GetApi().Name))
	}

	// Only hash successful responses (not errors) for cross-validation tracking
	// This prevents error responses from being counted toward cross-validation
	if response != nil && nodeError == nil && response.Err == nil {
		// Hash the response data once and cache it in the RelayResult
		hash := sha256.Sum256(response.RelayResult.GetReply().GetData())
		response.RelayResult.ResponseHash = hash // Cache the hash for later reuse
		rp.crossValidationMap[hash]++
		if rp.crossValidationMap[hash] > rp.currentCrossValidationEqualResults {
			rp.currentCrossValidationEqualResults = rp.crossValidationMap[hash]
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

func (rp *RelayProcessor) responsesCrossValidation(results []common.RelayResult, crossValidationSize int) (returnedResult *common.RelayResult, processingError error) {
	if crossValidationSize <= 0 {
		return nil, errors.New("crossValidationSize must be greater than zero")
	}

	type resultCount struct {
		count  int
		result common.RelayResult
	}

	countMap := make(map[[32]byte]*resultCount)
	nilReplies := 0
	nilReplyIdx := -1

	// Helper function to check if response data is valid
	isValidResponse := func(data []byte) bool {
		return len(data) > 0
	}

	for idx, result := range results {
		if result.Reply != nil && result.Reply.Data != nil && isValidResponse(result.Reply.Data) {
			// Use cached hash if available (set in handleResponse), otherwise compute it
			// This eliminates redundant SHA256 computation for responses already hashed
			hash := result.ResponseHash
			if hash == [32]byte{} {
				// Fallback: hash not cached (e.g., for old code paths or error cases)
				hash = sha256.Sum256(result.Reply.Data)
			}

			if count, exists := countMap[hash]; exists {
				count.count++
			} else {
				countMap[hash] = &resultCount{
					count:  1,
					result: result,
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

	if nilReplies >= crossValidationSize && maxCount < crossValidationSize {
		maxCount = nilReplies
		mostCommonResult = results[nilReplyIdx]
	}

	if maxCount < crossValidationSize {
		// Cross-validation failed: agreement threshold not reached
		// Same behavior for both deterministic and non-deterministic APIs
		if rp.selection == CrossValidation {
			return nil, utils.LavaFormatInfo("cross-validation failed: agreement threshold not reached",
				utils.LogAttr("nilReplies", nilReplies),
				utils.LogAttr("results", len(results)),
				utils.LogAttr("maxMatchingResults", maxCount),
				utils.LogAttr("agreementThreshold", crossValidationSize),
				utils.LogAttr("maxParticipants", rp.getMaxParticipants()))
		} else {
			// Stateless/Stateful modes - return original error message
			return nil, utils.LavaFormatInfo("majority count is less than crossValidationSize",
				utils.LogAttr("nilReplies", nilReplies),
				utils.LogAttr("results", len(results)),
				utils.LogAttr("maxCount", maxCount),
				utils.LogAttr("crossValidationSize", crossValidationSize))
		}
	}

	mostCommonResult.CrossValidation = maxCount
	return &mostCommonResult, nil
}

// this function returns the results according to the defined strategy
// results were stored in WaitForResults and now there's logic to select which results are returned to the user
// will return an error if we did not meet quota of replies, if we did we follow the strategies:
// if return strategy == get_first: return the first success, if none: get best node error
// if strategy == stateless get majority of node responses
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

	if rp.debugRelay {
		// adding as much debug info as possible. all successful relays, all node errors and all protocol errors
		utils.LavaFormatDebug("[Processing Result] Debug Relay",
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("selection", rp.selection),
			utils.LogAttr("agreementThreshold", rp.getAgreementThreshold()),
			utils.LogAttr("maxParticipants", rp.getMaxParticipants()))
		utils.LavaFormatDebug("[Processing Debug] number of node results", utils.LogAttr("GUID", rp.guid), utils.LogAttr("successResultsCount", successResultsCount), utils.LogAttr("nodeErrorCount", nodeErrorCount), utils.LogAttr("protocolErrorCount", protocolErrorCount))
		for idx, result := range successResults {
			utils.LavaFormatDebug("[Processing Debug] success result", utils.LogAttr("GUID", rp.guid), utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range nodeErrors {
			utils.LavaFormatDebug("[Processing Debug] node result", utils.LogAttr("GUID", rp.guid), utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		for idx, result := range protocolErrors {
			utils.LavaFormatDebug("[Processing Debug] protocol error", utils.LogAttr("GUID", rp.guid), utils.LogAttr("idx", idx), utils.LogAttr("result", result))
		}
		utils.LavaFormatDebug("[ProcessingResult]:", utils.LogAttr("GUID", rp.guid), utils.LogAttr("successResultsCount", successResultsCount))
	}

	// Process results based on selection mode
	switch rp.selection {
	case CrossValidation:
		return rp.processCrossValidationResult(successResults, successResultsCount, nodeErrorCount, rp.getAgreementThreshold())

	case Stateful:
		return rp.processStatefulResult(successResults, nodeErrors, successResultsCount, nodeErrorCount, allProvidersAddresses)

	case Stateless:
		return rp.processStatelessResult(successResults, nodeErrors, successResultsCount, nodeErrorCount, protocolErrorCount, allProvidersAddresses)

	default:
		return nil, utils.LavaFormatError("unknown selection mode", nil, utils.LogAttr("selection", rp.selection))
	}
}

// processCrossValidationResult handles result processing for CrossValidation mode.
// Only successful responses count towards consensus - node errors are ignored.
func (rp *RelayProcessor) processCrossValidationResult(
	successResults []common.RelayResult,
	successResultsCount, nodeErrorCount, requiredCrossValidationSize int,
) (*common.RelayResult, error) {
	// Check if we have enough successful responses
	if successResultsCount >= requiredCrossValidationSize {
		result, err := rp.responsesCrossValidation(successResults, requiredCrossValidationSize)
		if err == nil {
			return result, nil
		}
		// Successful responses exist but don't agree
		// Return a minimal result so headers can be attached
		return &common.RelayResult{StatusCode: http.StatusInternalServerError}, utils.LavaFormatError("cross-validation failed: successful responses did not reach agreement",
			err,
			utils.LogAttr("GUID", rp.guid),
			utils.LogAttr("successCount", successResultsCount),
			utils.LogAttr("agreementThreshold", requiredCrossValidationSize),
			utils.LogAttr("nodeErrorCount", nodeErrorCount),
		)
	}

	// Not enough successful responses
	// Return a minimal result so headers can be attached
	return &common.RelayResult{StatusCode: http.StatusInternalServerError}, utils.LavaFormatError("cross-validation failed: insufficient successful responses",
		nil,
		utils.LogAttr("GUID", rp.guid),
		utils.LogAttr("successCount", successResultsCount),
		utils.LogAttr("agreementThreshold", requiredCrossValidationSize),
		utils.LogAttr("nodeErrorCount", nodeErrorCount),
		utils.LogAttr("maxParticipants", rp.getMaxParticipants()),
	)
}

// processStatefulResult handles result processing for Stateful mode.
// Returns first success, or first node error if no successes.
// No cross-validation/consensus - just return the first available result.
func (rp *RelayProcessor) processStatefulResult(
	successResults, nodeErrors []common.RelayResult,
	successResultsCount, nodeErrorCount int,
	allProvidersAddresses []string,
) (*common.RelayResult, error) {
	// Return first success if available
	if successResultsCount > 0 {
		result := successResults[0]
		return &result, nil
	}

	// No successes, return first node error if available
	if nodeErrorCount > 0 {
		result := nodeErrors[0]
		return &result, nil
	}

	// No results at all
	return rp.buildFailureResult(nodeErrorCount, 0, allProvidersAddresses)
}

// processStatelessResult handles result processing for Stateless mode.
// Returns first success, or first node error if no successes.
// No cross-validation/consensus - retries handle getting a valid response.
func (rp *RelayProcessor) processStatelessResult(
	successResults, nodeErrors []common.RelayResult,
	successResultsCount, nodeErrorCount, protocolErrorCount int,
	allProvidersAddresses []string,
) (*common.RelayResult, error) {
	// Return first success if available
	if successResultsCount > 0 {
		result := successResults[0]
		return &result, nil
	}

	// No successes, return first node error if available
	if nodeErrorCount > 0 {
		result := nodeErrors[0]
		return &result, nil
	}

	// No results at all - return failure
	return rp.buildFailureResult(nodeErrorCount, protocolErrorCount, allProvidersAddresses)
}

// buildFailureResult constructs an error result when no consensus can be reached.
func (rp *RelayProcessor) buildFailureResult(
	nodeErrorCount, protocolErrorCount int,
	allProvidersAddresses []string,
) (*common.RelayResult, error) {
	returnedResult := &common.RelayResult{StatusCode: http.StatusInternalServerError}
	var processingError error

	if nodeErrorCount > 0 {
		// Prefer node errors over protocol errors
		nodeErr := rp.GetBestNodeErrorMessageForUser()
		processingError = nodeErr.Err
		if nodeErr.Response != nil {
			returnedResult = &nodeErr.Response.RelayResult
		}
	} else if protocolErrorCount > 0 {
		protocolErr := rp.GetBestProtocolErrorMessageForUser()
		processingError = protocolErr.Err
		if protocolErr.Response != nil {
			returnedResult = &protocolErr.Response.RelayResult
		}
	}

	returnedResult.ProviderInfo.ProviderAddress = strings.Join(allProvidersAddresses, ",")
	return returnedResult, utils.LavaFormatError("failed relay, insufficient results", processingError, utils.LogAttr("GUID", rp.guid))
}
