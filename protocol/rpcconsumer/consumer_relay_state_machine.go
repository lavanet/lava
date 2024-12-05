package rpcconsumer

import (
	context "context"
	"sync"
	"sync/atomic"
	"time"

	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"

	"github.com/lavanet/lava/v4/protocol/chainlib"
	common "github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavaprotocol"
	lavasession "github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/utils"
)

type RelayStateMachine interface {
	GetProtocolMessage() chainlib.ProtocolMessage
	GetDebugState() bool
	GetRelayTaskChannel() (chan RelayStateSendInstructions, error)
	UpdateBatch(err error)
	GetSelection() Selection
	GetUsedProviders() *lavasession.UsedProviders
	SetResultsChecker(resultsChecker ResultsCheckerInf)
	SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager)
}

type ResultsCheckerInf interface {
	WaitForResults(ctx context.Context) error
	HasRequiredNodeResults() (bool, int)
}

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
	selection           Selection
	debugRelays         bool
	tickerMetricSetter  tickerMetricSetterInf
	batchUpdate         chan error
	usedProviders       *lavasession.UsedProviders
	relayRetriesManager *lavaprotocol.RelayRetriesManager
	relayState          []*RelayState
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
	selection := Quorum // select the majority of node responses
	if chainlib.GetStateful(protocolMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		selection = BestResult // select the majority of node successes
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
		relayState:         make([]*RelayState, 0),
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

func (crsm *ConsumerRelayStateMachine) GetSelection() Selection {
	return crsm.selection
}

func (crsm *ConsumerRelayStateMachine) appendRelayState(nextState *RelayState) {
	crsm.relayStateLock.Lock()
	defer crsm.relayStateLock.Unlock()
	crsm.relayState = append(crsm.relayState, nextState)
}

func (crsm *ConsumerRelayStateMachine) getLatestState() *RelayState {
	crsm.relayStateLock.RLock()
	defer crsm.relayStateLock.RUnlock()
	if len(crsm.relayState) == 0 {
		return nil
	}
	return crsm.relayState[len(crsm.relayState)-1]
}

func (crsm *ConsumerRelayStateMachine) stateTransition(relayState *RelayState, numberOfNodeErrors uint64) {
	batchNumber := crsm.usedProviders.BatchNumber()
	var nextState *RelayState
	if relayState == nil { // initial state
		nextState = NewRelayState(crsm.ctx, crsm.protocolMessage, 0, crsm.relayRetriesManager, crsm.relaySender, &ArchiveStatus{})
	} else {
		nextState = NewRelayState(crsm.ctx, crsm.GetProtocolMessage(), relayState.GetStateNumber()+1, crsm.relayRetriesManager, crsm.relaySender, relayState.archiveStatus.Copy())
		nextState.upgradeToArchiveIfNeeded(batchNumber, numberOfNodeErrors)
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
	return shouldRetry
}

func (crsm *ConsumerRelayStateMachine) retryCondition(numberOfRetriesLaunched int) bool {
	if numberOfRetriesLaunched >= MaximumNumberOfTickerRelayRetries {
		return false
	}
	// best result sends to top 10 providers anyway.
	return crsm.selection != BestResult
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

type RelayStateSendInstructions struct {
	analytics  *metrics.RelayMetrics
	err        error
	done       bool
	relayState *RelayState
}

func (rssi *RelayStateSendInstructions) IsDone() bool {
	return rssi.done || rssi.err != nil
}

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
			utils.LavaFormatDebug("Relay initiated with the following timeout schedule", utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("newRelayTimeout", relayTimeout))
		}
		// Create the processing timeout prior to entering the method so it wont reset every time
		processingCtx, processingCtxCancel := context.WithTimeout(crsm.ctx, processingTimeout)
		defer processingCtxCancel()

		numberOfNodeErrorsAtomic := atomic.Uint64{}
		readResultsFromProcessor := func() {
			// ProcessResults is reading responses while blocking until the conditions are met
			utils.LavaFormatTrace("[StateMachine] Waiting for results", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
			crsm.resultsChecker.WaitForResults(processingCtx)
			// Decide if we need to resend or not
			metRequiredNodeResults, numberOfNodeErrors := crsm.resultsChecker.HasRequiredNodeResults()
			numberOfNodeErrorsAtomic.Store(uint64(numberOfNodeErrors))
			if metRequiredNodeResults {
				gotResults <- true
			} else {
				gotResults <- false
			}
		}
		go readResultsFromProcessor()
		returnCondition := make(chan error, 1)
		// Used for checking whether to return an error to the user or to allow other channels return their result first see detailed description on the switch case below
		validateReturnCondition := func(err error) {
			batchOnStart := crsm.usedProviders.BatchNumber()
			time.Sleep(15 * time.Millisecond)
			utils.LavaFormatTrace("[StateMachine] validating return condition", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
			if batchOnStart == crsm.usedProviders.BatchNumber() && crsm.usedProviders.CurrentlyUsed() == 0 {
				utils.LavaFormatTrace("[StateMachine] return condition triggered", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("err", err))
				returnCondition <- err
			}
		}

		// initialize relay state
		crsm.stateTransition(nil, 0)
		// Send First Message, with analytics and without waiting for batch update.
		relayTaskChannel <- RelayStateSendInstructions{
			analytics:  crsm.analytics,
			relayState: crsm.getLatestState(),
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
					utils.LavaFormatTrace("[StateMachine] err := <-crsm.batchUpdate", utils.LogAttr("err", err), utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors))
					// Sending a new batch failed (consumer's protocol side), handling the state machine
					consecutiveBatchErrors++                        // Increase consecutive error counter
					if consecutiveBatchErrors > SendRelayAttempts { // If we failed sending a message more than "SendRelayAttempts" time in a row.
						if crsm.usedProviders.BatchNumber() == 0 && consecutiveBatchErrors == SendRelayAttempts+1 { // First relay attempt. print on first failure only.
							utils.LavaFormatWarning("Failed Sending First Message", err, utils.LogAttr("consecutive errors", consecutiveBatchErrors))
						}
						go validateReturnCondition(err) // Check if we have ongoing messages pending return.
					} else {
						utils.LavaFormatTrace("[StateMachine] batchUpdate - err != nil - batch fail retry attempt", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors))
						// Failed sending message, but we still want to attempt sending more.
						relayTaskChannel <- RelayStateSendInstructions{relayState: crsm.getLatestState()}
					}
					continue
				}
				// Successfully sent message.
				// Reset consecutiveBatchErrors
				consecutiveBatchErrors = 0
			case success := <-gotResults:
				utils.LavaFormatTrace("[StateMachine] success := <-gotResults", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
				// If we had a successful result return what we currently have
				// Or we are done sending relays, and we have no other relays pending results.
				if success { // Check wether we can return the valid results or we need to send another relay
					relayTaskChannel <- RelayStateSendInstructions{done: true}
					return
				}
				// If should retry == true, send a new batch. (success == false)
				if crsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] success := <-gotResults - crsm.ShouldRetry(batchNumber)", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
					relayTaskChannel <- RelayStateSendInstructions{relayState: crsm.getLatestState()}
				} else {
					go validateReturnCondition(nil)
				}
				go readResultsFromProcessor()
			case <-startNewBatchTicker.C:
				// Only trigger another batch for non BestResult relays or if we didn't pass the retry limit.
				if crsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
					relayTaskChannel <- RelayStateSendInstructions{relayState: crsm.getLatestState()}
					// Add ticker launch metrics
					go crsm.tickerMetricSetter.SetRelaySentByNewBatchTickerMetric(crsm.relaySender.GetChainIdAndApiInterface())
				}
			case returnErr := <-returnCondition:
				utils.LavaFormatTrace("[StateMachine] returnErr := <-returnCondition", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
				// we use this channel because there could be a race condition between us releasing the provider and about to send the return
				// to an error happening on another relay processor's routine. this can cause an error that returns to the user
				// if we don't release the case, it will cause the success case condition to not be executed
				// detailed scenario:
				// sending first relay -> waiting -> sending second relay -> getting an error on the second relay (not returning yet) ->
				// -> (in parallel) first relay finished, removing from CurrentlyUsed providers -> checking currently used (on second failed relay) -> returning error instead of the successful relay.
				// by releasing the case we allow the channel to be chosen again by the successful case.
				relayTaskChannel <- RelayStateSendInstructions{err: returnErr, done: true}
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
				relayTaskChannel <- RelayStateSendInstructions{err: processingCtx.Err(), done: true}
				return
			}
		}
	}()
	return relayTaskChannel, nil
}

func (crsm *ConsumerRelayStateMachine) UpdateBatch(err error) {
	crsm.batchUpdate <- err
}
