package rpcconsumer

import (
	context "context"
	"sync"
	"sync/atomic"
	"time"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	lavasession "github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/lavanet/lava/v5/utils"
)

// Using interfaces from relaycore
type (
	RelayStateMachine = relaycore.RelayStateMachine
	ResultsCheckerInf = relaycore.ResultsCheckerInf
)

type ConsumerRelaySender interface {
	getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration)
	GetChainIdAndApiInterface() (string, string)
	ParseRelay(
		ctx context.Context,
		url string,
		req string,
		connectionType string,
		dappID string,
		consumerIp string,
		metadata []pairingtypes.Metadata,
	) (protocolMessage chainlib.ProtocolMessage, err error)
}

type tickerMetricSetterInf interface {
	SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string)
}

type ConsumerRelayStateMachine struct {
	ctx                 context.Context // same context as user context.
	relaySender         ConsumerRelaySender
	resultsChecker      ResultsCheckerInf
	analytics           *metrics.RelayMetrics // first relay metrics
	selection           relaycore.Selection
	debugRelays         bool
	tickerMetricSetter  tickerMetricSetterInf
	batchUpdate         chan error
	usedProviders       *lavasession.UsedProviders
	relayRetriesManager *lavaprotocol.RelayRetriesManager
	relayState          []*relaycore.RelayState
	protocolMessage     chainlib.ProtocolMessage
	relayStateLock      sync.RWMutex
}

func NewRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender ConsumerRelaySender,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
	tickerMetricSetter tickerMetricSetterInf,
) RelayStateMachine {
	selection := relaycore.Quorum // select the majority of node responses
	if chainlib.GetStateful(protocolMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		selection = relaycore.BestResult // select the majority of node successes
	}

	return &ConsumerRelayStateMachine{
		ctx:                ctx,
		usedProviders:      usedProviders,
		relaySender:        relaySender,
		protocolMessage:    protocolMessage,
		analytics:          analytics,
		selection:          selection,
		debugRelays:        debugRelays,
		tickerMetricSetter: tickerMetricSetter,
		batchUpdate:        make(chan error, MaximumNumberOfTickerRelayRetries),
		relayState:         make([]*relaycore.RelayState, 0),
	}
}

func (crsm *ConsumerRelayStateMachine) Initialized() bool {
	return crsm.relayRetriesManager != nil && crsm.resultsChecker != nil
}

func (crsm *ConsumerRelayStateMachine) SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager) {
	crsm.relayRetriesManager = relayRetriesManager
}

func (crsm *ConsumerRelayStateMachine) SetResultsChecker(resultsChecker ResultsCheckerInf) {
	crsm.resultsChecker = resultsChecker
}

func (crsm *ConsumerRelayStateMachine) GetUsedProviders() *lavasession.UsedProviders {
	return crsm.usedProviders
}

func (crsm *ConsumerRelayStateMachine) GetSelection() relaycore.Selection {
	return crsm.selection
}

func (crsm *ConsumerRelayStateMachine) appendRelayState(nextState *relaycore.RelayState) {
	crsm.relayStateLock.Lock()
	defer crsm.relayStateLock.Unlock()
	crsm.relayState = append(crsm.relayState, nextState)
}

func (crsm *ConsumerRelayStateMachine) getLatestState() *relaycore.RelayState {
	crsm.relayStateLock.RLock()
	defer crsm.relayStateLock.RUnlock()
	if len(crsm.relayState) == 0 {
		return nil
	}
	return crsm.relayState[len(crsm.relayState)-1]
}

func (crsm *ConsumerRelayStateMachine) stateTransition(relayState *relaycore.RelayState, numberOfNodeErrors uint64) {
	batchNumber := crsm.usedProviders.BatchNumber()
	var nextState *relaycore.RelayState
	if relayState == nil { // initial state
		nextState = relaycore.NewRelayState(crsm.ctx, crsm.protocolMessage, 0, crsm.relayRetriesManager, crsm.relaySender, &relaycore.ArchiveStatus{})
	} else {
		// Get the appropriate protocol message (with archive upgrade if needed) BEFORE creating RelayState
		protocolMessage := crsm.GetProtocolMessage()
		archiveStatus := relayState.GetArchiveStatus()

		// Use static function to get upgraded protocol message without creating RelayState
		upgradedProtocolMessage := relaycore.UpgradeToArchiveIfNeeded(crsm.ctx, protocolMessage, archiveStatus, crsm.relaySender, crsm.relayRetriesManager, batchNumber, numberOfNodeErrors)

		// Create the final RelayState with the correct protocol message
		nextState = relaycore.NewRelayState(crsm.ctx, upgradedProtocolMessage, relayState.GetStateNumber()+1, crsm.relayRetriesManager, crsm.relaySender, archiveStatus)
	}
	crsm.appendRelayState(nextState)
}

// Should retry implements the logic for when to send another relay.
// As well as the decision of changing the protocol message,
// into different extensions or addons based on certain conditions
func (crsm *ConsumerRelayStateMachine) shouldRetry(numberOfNodeErrors uint64) bool {
	batchNumber := crsm.usedProviders.BatchNumber()
	shouldRetry := crsm.retryCondition(batchNumber)
	if shouldRetry {
		crsm.stateTransition(crsm.getLatestState(), numberOfNodeErrors)
	}

	utils.LavaFormatDebug("[StateMachine] shouldRetry called",
		utils.LogAttr("GUID", crsm.ctx),
		utils.LogAttr("numberOfNodeErrors", numberOfNodeErrors),
		utils.LogAttr("batchNumber", crsm.usedProviders.BatchNumber()),
		utils.LogAttr("selection", crsm.selection),
		utils.LogAttr("shouldRetry", shouldRetry),
	)

	return shouldRetry
}

// hasUnsupportedMethodErrorsInStateMachine checks if we have unsupported method errors at state machine level
func (crsm *ConsumerRelayStateMachine) hasUnsupportedMethodErrorsInStateMachine() bool {
	if crsm.resultsChecker == nil {
		return false
	}

	// Check if the results checker has unsupported method errors
	if relayProcessor, ok := crsm.resultsChecker.(*relaycore.RelayProcessor); ok {
		return relayProcessor.HasUnsupportedMethodErrors()
	}

	return false
}

func (crsm *ConsumerRelayStateMachine) retryCondition(numberOfRetriesLaunched int) bool {
	utils.LavaFormatTrace("[StateMachine] retryCondition", utils.LogAttr("numberOfRetriesLaunched", numberOfRetriesLaunched), utils.LogAttr("GUID", crsm.ctx), utils.LogAttr("batchNumber", crsm.usedProviders.BatchNumber()), utils.LogAttr("selection", crsm.selection))

	// If unsupported method returned, no retry - stop now
	if crsm.hasUnsupportedMethodErrorsInStateMachine() {
		utils.LavaFormatTrace("[StateMachine] retryCondition: unsupported method detected, no retry", utils.LogAttr("GUID", crsm.ctx))
		return false
	}

	// If quorum is disabled, check for success: if success stop, otherwise retry
	if !crsm.resultsChecker.GetQuorumParams().Enabled() {
		if crsm.resultsChecker.HasSuccessfulResults() {
			utils.LavaFormatTrace("[StateMachine] retryCondition: quorum disabled with success, no retry",
				utils.LogAttr("GUID", crsm.ctx))
			return false
		} else {
			utils.LavaFormatTrace("[StateMachine] retryCondition: quorum disabled with no success, allow failover retry",
				utils.LogAttr("GUID", crsm.ctx))
			return true
		}
	}

	// If quorum is enabled, check retry limit, retry until quorum met
	if numberOfRetriesLaunched > crsm.resultsChecker.GetQuorumParams().Max {
		return false
	}
	if numberOfRetriesLaunched >= MaximumNumberOfTickerRelayRetries {
		return false
	}

	// BestResult selection (stateful) doesn't retry, Quorum selection retries until quorum met
	return crsm.selection != relaycore.BestResult
}

func (crsm *ConsumerRelayStateMachine) GetDebugState() bool {
	return crsm.debugRelays
}

func (crsm *ConsumerRelayStateMachine) GetProtocolMessage() chainlib.ProtocolMessage {
	latestState := crsm.getLatestState()
	if latestState == nil { // failed fetching latest state
		return crsm.protocolMessage
	}
	return latestState.GetProtocolMessage()
}

// Using RelayStateSendInstructions from relaycore
type RelayStateSendInstructions = relaycore.RelayStateSendInstructions

func (crsm *ConsumerRelayStateMachine) GetRelayTaskChannel() (chan RelayStateSendInstructions, error) {
	if !crsm.Initialized() {
		return nil, utils.LavaFormatError("ConsumerRelayStateMachine was not initialized properly", nil)
	}
	relayTaskChannel := make(chan RelayStateSendInstructions)
	go func() {
		// A channel to be notified processing was done, true means we have results and can return
		gotResults := make(chan bool, 1)
		processingTimeout, relayTimeout := crsm.relaySender.getProcessingTimeout(crsm.GetProtocolMessage())
		if crsm.debugRelays {
			utils.LavaFormatDebug("Relay initiated with the following timeout schedule", utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("newRelayTimeout", relayTimeout), utils.LogAttr("GUID", crsm.ctx))
		}
		// Create the processing timeout prior to entering the method so it wont reset every time
		processingCtx, processingCtxCancel := context.WithTimeout(crsm.ctx, processingTimeout)
		defer processingCtxCancel()

		numberOfNodeErrorsAtomic := atomic.Uint64{}
		readResultsFromProcessor := func() {
			// ProcessResults is reading responses while blocking until the conditions are met
			utils.LavaFormatTrace("[StateMachine] Waiting for results", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
			crsm.resultsChecker.WaitForResults(processingCtx)
			// Decide if we need to resend or not
			metRequiredNodeResults, numberOfNodeErrors := crsm.resultsChecker.HasRequiredNodeResults(crsm.usedProviders.BatchNumber())
			numberOfNodeErrorsAtomic.Store(uint64(numberOfNodeErrors))
			gotResults <- metRequiredNodeResults
		}
		go readResultsFromProcessor()
		returnCondition := make(chan error, 1)
		// Used for checking whether to return an error to the user or to allow other channels return their result first see detailed description on the switch case below
		validateReturnCondition := func(err error) {
			batchOnStart := crsm.usedProviders.BatchNumber()
			time.Sleep(15 * time.Millisecond)
			utils.LavaFormatTrace("[StateMachine] validating return condition", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
			if batchOnStart == crsm.usedProviders.BatchNumber() && crsm.usedProviders.CurrentlyUsed() == 0 {
				utils.LavaFormatTrace("[StateMachine] return condition triggered", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("err", err), utils.LogAttr("GUID", crsm.ctx))
				returnCondition <- err
			}
		}

		// initialize relay state
		crsm.stateTransition(nil, 0)
		// Send First Message, with analytics and without waiting for batch update.
		relayTaskChannel <- RelayStateSendInstructions{
			Analytics:      crsm.analytics,
			RelayState:     crsm.getLatestState(),
			NumOfProviders: crsm.resultsChecker.GetQuorumParams().Min,
		}

		// Initialize parameters
		startNewBatchTicker := time.NewTicker(relayTimeout) // Every relay timeout we send a new batch
		defer startNewBatchTicker.Stop()
		consecutiveBatchErrors := 0
		// Start the relay state machine
		for {
			select {
			// Getting batch update for either errors sending message or successful batches
			case err := <-crsm.batchUpdate:
				if err != nil { // Error handling
					utils.LavaFormatTrace("[StateMachine] err := <-crsm.batchUpdate", utils.LogAttr("err", err), utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", crsm.ctx))
					// Sending a new batch failed (consumer's protocol side), handling the state machine
					consecutiveBatchErrors++                        // Increase consecutive error counter
					if consecutiveBatchErrors > SendRelayAttempts { // If we failed sending a message more than "SendRelayAttempts" time in a row.
						if crsm.usedProviders.BatchNumber() == 0 && consecutiveBatchErrors == SendRelayAttempts+1 { // First relay attempt. print on first failure only.
							utils.LavaFormatWarning("Failed Sending First Message", err, utils.LogAttr("consecutive errors", consecutiveBatchErrors), utils.LogAttr("GUID", crsm.ctx))
						}
						go validateReturnCondition(err) // Check if we have ongoing messages pending return.
					} else {
						utils.LavaFormatTrace("[StateMachine] batchUpdate - err != nil - batch fail retry attempt", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", crsm.ctx))
						// Failed sending message, but we still want to attempt sending more.
						relayTaskChannel <- RelayStateSendInstructions{RelayState: crsm.getLatestState(), NumOfProviders: 1}
					}
					continue
				}
				// Successfully sent message.
				// Reset consecutiveBatchErrors
				consecutiveBatchErrors = 0
			case success := <-gotResults:
				utils.LavaFormatTrace("[StateMachine] success := <-gotResults", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
				// If we had a successful result return what we currently have
				// Or we are done sending relays, and we have no other relays pending results.
				if success { // Check wether we can return the valid results or we need to send another relay
					utils.LavaFormatTrace("[StateMachine] successfully sent message", utils.LogAttr("GUID", crsm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{Done: true}
					return
				}
				// If should retry == true, send a new batch. (success == false)
				if crsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] LavaFormatTrace success := <-gotResults - crsm.ShouldRetry(batchNumber)", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: crsm.getLatestState(), NumOfProviders: 1}
				} else {
					// No retry needed, wait for return condition to be validated
					go validateReturnCondition(nil)
				}
				// Always spawn readResultsFromProcessor to keep checking for new results
				go readResultsFromProcessor()
			case <-startNewBatchTicker.C:
				// Only trigger another batch for non BestResult relays or if we didn't pass the retry limit.
				if crsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: crsm.getLatestState(), NumOfProviders: 1}
					// Add ticker launch metrics
					go crsm.tickerMetricSetter.SetRelaySentByNewBatchTickerMetric(crsm.relaySender.GetChainIdAndApiInterface())
				}
			case returnErr := <-returnCondition:
				utils.LavaFormatTrace("[StateMachine] returnErr := <-returnCondition", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", crsm.ctx))
				// we use this channel because there could be a race condition between us releasing the provider and about to send the return
				// to an error happening on another relay processor's routine. this can cause an error that returns to the user
				// if we don't release the case, it will cause the success case condition to not be executed
				// detailed scenario:
				// sending first relay -> waiting -> sending second relay -> getting an error on the second relay (not returning yet) ->
				// -> (in parallel) first relay finished, removing from CurrentlyUsed providers -> checking currently used (on second failed relay) -> returning error instead of the successful relay.
				// by releasing the case we allow the channel to be chosen again by the successful case.
				relayTaskChannel <- RelayStateSendInstructions{Err: returnErr, Done: true}
				return
			case <-processingCtx.Done():
				// In case we got a processing timeout we return context deadline exceeded to the user.
				userData := crsm.GetProtocolMessage().GetUserData()
				utils.LavaFormatWarning("Relay Got processingCtx timeout", nil,
					utils.LogAttr("processingTimeout", processingTimeout),
					utils.LogAttr("dappId", userData.DappId),
					utils.LogAttr("consumerIp", userData.ConsumerIp),
					utils.LogAttr("protocolMessage.GetApi().Name", crsm.GetProtocolMessage().GetApi().Name),
					utils.LogAttr("GUID", crsm.ctx),
					utils.LogAttr("batchNumber", crsm.usedProviders.BatchNumber()),
					utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors),
				)
				// returning the context error
				relayTaskChannel <- RelayStateSendInstructions{Err: processingCtx.Err(), Done: true}
				return
			}
		}
	}()
	return relayTaskChannel, nil
}

func (crsm *ConsumerRelayStateMachine) UpdateBatch(err error) {
	crsm.batchUpdate <- err
}
