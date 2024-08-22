package rpcconsumer

import (
	context "context"
	"errors"
	"fmt"
	"strings"
	"sync"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/chainlib"
	common "github.com/lavanet/lava/v2/protocol/common"
	lavasession "github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type RelayResultManager struct {
	responses                    chan *relayResponse
	requiredSuccesses            int
	nodeResponseErrors           RelayErrors
	protocolResponseErrors       RelayErrors
	successResults               []common.RelayResult
	lock                         sync.RWMutex
	selection                    Selection
	protocolMessage              chainlib.ProtocolMessage
	consumerConsistency          *ConsumerConsistency
	guid                         uint64
	usedProviders                *lavasession.UsedProviders
	userData                     chainlib.UserData
	metricsInf                   MetricsInterface
	chainIdAndApiInterfaceGetter chainIdAndApiInterfaceGetter
}

func (rrm *RelayResultManager) String() string {
	if rrm == nil {
		return ""
	}
	rrm.lock.RLock()
	nodeErrors := len(rrm.nodeResponseErrors.relayErrors)
	protocolErrors := len(rrm.protocolResponseErrors.relayErrors)
	results := len(rrm.successResults)
	usedProviders := rrm.usedProviders
	rrm.lock.RUnlock()

	currentlyUsedAddresses := usedProviders.CurrentlyUsedAddresses()
	unwantedAddresses := usedProviders.UnwantedAddresses()
	return fmt.Sprintf("relayProcessor {results:%d, nodeErrors:%d, protocolErrors:%d,unwantedAddresses: %s,currentlyUsedAddresses:%s}",
		results, nodeErrors, protocolErrors, strings.Join(unwantedAddresses, ";"), strings.Join(currentlyUsedAddresses, ";"))
}

func (rrm *RelayResultManager) HasRequiredNodeResults() bool {
	return false
}

func (rrm *RelayResultManager) handleResponse(response *relayResponse) {
	if response == nil {
		return
	}
	if response.err != nil {
		rrm.setErrorResponse(response)
	} else {
		rrm.setValidResponse(response)
	}
}

func (rrm *RelayResultManager) readExistingResponses() {
	for {
		select {
		case response := <-rrm.responses:
			rrm.handleResponse(response)
		default:
			// No more responses immediately available, exit the loop
			return
		}
	}
}

// this function waits for the processing results, they are written by multiple go routines and read by this go routine
// it then updates the responses in their respective place, node errors, protocol errors or success results
func (rrm *RelayResultManager) WaitForResults(ctx context.Context) error {
	if rrm == nil {
		return utils.LavaFormatError("RelayResultManager.WaitForResults is nil, misuse detected", nil)
	}
	responsesCount := 0
	for {
		select {
		case response := <-rrm.responses:
			responsesCount++
			rrm.handleResponse(response)
			if rrm.checkEndProcessing(responsesCount) {
				// we can finish processing
				return nil
			}
		case <-ctx.Done():
			return utils.LavaFormatWarning("cancelled relay processor", nil, utils.LogAttr("total responses", responsesCount))
		}
	}
}

func (rrm *RelayResultManager) responsesQuorum(results []common.RelayResult, quorumSize int) (returnedResult *common.RelayResult, processingError error) {
	if quorumSize <= 0 {
		return nil, errors.New("quorumSize must be greater than zero")
	}
	countMap := make(map[string]int) // Map to store the count of each unique result.Reply.Data
	deterministic := rrm.protocolMessage.GetApi().Category.Deterministic
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

func (rrm *RelayResultManager) setErrorResponse(response *relayResponse) {
	rrm.lock.Lock()
	defer rrm.lock.Unlock()
	utils.LavaFormatDebug("could not send relay to provider", utils.Attribute{Key: "GUID", Value: rrm.guid}, utils.Attribute{Key: "provider", Value: response.relayResult.ProviderInfo.ProviderAddress}, utils.Attribute{Key: "error", Value: response.err.Error()})
	rrm.protocolResponseErrors.relayErrors = append(rrm.protocolResponseErrors.relayErrors, RelayError{err: response.err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
}

func (rrm *RelayResultManager) checkEndProcessing(responsesCount int) bool {
	rrm.lock.RLock()
	defer rrm.lock.RUnlock()
	resultsCount := len(rrm.successResults)
	if resultsCount >= rrm.requiredSuccesses {
		// we have enough successes, we can return
		return true
	}
	if rrm.selection == Quorum {
		// we need a quorum of all node results
		nodeErrors := len(rrm.nodeResponseErrors.relayErrors)
		if nodeErrors+resultsCount >= rrm.requiredSuccesses {
			// we have enough node results for our quorum
			return true
		}
	}
	// check if we got all of the responses
	if responsesCount >= rrm.usedProviders.SessionsLatestBatch() {
		// no active sessions, and we read all the responses, we can return
		return true
	}
	return false
}

// this function defines if we should use the processor to return the result (meaning it has some insight and responses) or just return to the user
func (rrm *RelayResultManager) HasResults() bool {
	if rrm == nil {
		return false
	}
	rrm.lock.RLock()
	defer rrm.lock.RUnlock()
	resultsCount := len(rrm.successResults)
	nodeErrors := len(rrm.nodeResponseErrors.relayErrors)
	protocolErrors := len(rrm.protocolResponseErrors.relayErrors)
	return resultsCount+nodeErrors+protocolErrors > 0
}

func (rrm *RelayResultManager) setValidResponse(response *relayResponse) {
	rrm.lock.Lock()
	defer rrm.lock.Unlock()

	// future relay requests and data reliability requests need to ask for the same specific block height to get consensus on the reply
	// we do not modify the chain message data on the consumer, only it's requested block, so we let the provider know it can't put any block height it wants by setting a specific block height
	reqBlock, _ := rrm.protocolMessage.RequestedBlock()
	if reqBlock == spectypes.LATEST_BLOCK {
		// TODO: when we turn on dataReliability on latest call UpdateLatest, until then we turn it off always
		// modifiedOnLatestReq := rrm.chainMessage.UpdateLatestBlockInMessage(response.relayResult.Reply.LatestBlock, false)
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
	rrm.consumerConsistency.SetSeenBlock(blockSeen, rrm.userData.DappId, rrm.userData.ConsumerIp)
	// on subscribe results, we just append to successful results instead of parsing results because we already have a validation.
	if chainlib.IsFunctionTagOfType(rrm.protocolMessage, spectypes.FUNCTION_TAG_SUBSCRIBE) {
		rrm.successResults = append(rrm.successResults, response.relayResult)
		return
	}

	// check response error
	foundError, errorMessage := rrm.protocolMessage.CheckResponseError(response.relayResult.Reply.Data, response.relayResult.StatusCode)
	if foundError {
		// this is a node error, meaning we still didn't get a good response.
		// we may choose to wait until there will be a response or timeout happens
		// if we decide to wait and timeout happens we will take the majority of response messages
		err := fmt.Errorf("%s", errorMessage)
		rrm.nodeResponseErrors.relayErrors = append(rrm.nodeResponseErrors.relayErrors, RelayError{err: err, ProviderInfo: response.relayResult.ProviderInfo, response: response})
		// send relay error metrics only on non stateful queries, as stateful queries always return X-1/X errors.
		if rrm.selection != BestResult {
			go rrm.metricsInf.SetRelayNodeErrorMetric(rrm.chainIdAndApiInterfaceGetter.GetChainIdAndApiInterface())
			utils.LavaFormatInfo("Relay received a node error", utils.LogAttr("Error", err), utils.LogAttr("provider", response.relayResult.ProviderInfo), utils.LogAttr("Request", rrm.protocolMessage.GetApi().Name), utils.LogAttr("requested_block", reqBlock))
		}
		return
	}
	rrm.successResults = append(rrm.successResults, response.relayResult)
}
