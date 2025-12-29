package rpcsmartrouter

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

type SmartRouterRelaySender interface {
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

type SmartRouterRelayStateMachine struct {
	ctx                      context.Context // same context as user context.
	relaySender              SmartRouterRelaySender
	resultsChecker           ResultsCheckerInf
	analytics                *metrics.RelayMetrics // first relay metrics
	selection                relaycore.Selection
	debugRelays              bool
	tickerMetricSetter       tickerMetricSetterInf
	batchUpdate              chan error
	usedProviders            *lavasession.UsedProviders
	relayRetriesManager      *lavaprotocol.RelayRetriesManager
	relayState               []*relaycore.RelayState
	protocolMessage          chainlib.ProtocolMessage
	relayStateLock           sync.RWMutex
	consecutivePairingErrors int // Track consecutive pairing errors for circuit breaker
}

func NewSmartRouterRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender SmartRouterRelaySender,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
	tickerMetricSetter tickerMetricSetterInf,
) RelayStateMachine {
	selection := relaycore.Quorum // select the majority of node responses
	if chainlib.GetStateful(protocolMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		selection = relaycore.BestResult // select the majority of node successes
	}

	return &SmartRouterRelayStateMachine{
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

func (srsm *SmartRouterRelayStateMachine) Initialized() bool {
	return srsm.relayRetriesManager != nil && srsm.resultsChecker != nil
}

func (srsm *SmartRouterRelayStateMachine) SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager) {
	srsm.relayRetriesManager = relayRetriesManager
}

func (srsm *SmartRouterRelayStateMachine) SetResultsChecker(resultsChecker ResultsCheckerInf) {
	srsm.resultsChecker = resultsChecker
}

func (srsm *SmartRouterRelayStateMachine) GetUsedProviders() *lavasession.UsedProviders {
	return srsm.usedProviders
}

func (srsm *SmartRouterRelayStateMachine) GetSelection() relaycore.Selection {
	return srsm.selection
}

func (srsm *SmartRouterRelayStateMachine) appendRelayState(nextState *relaycore.RelayState) {
	srsm.relayStateLock.Lock()
	defer srsm.relayStateLock.Unlock()
	srsm.relayState = append(srsm.relayState, nextState)
}

func (srsm *SmartRouterRelayStateMachine) getLatestState() *relaycore.RelayState {
	srsm.relayStateLock.RLock()
	defer srsm.relayStateLock.RUnlock()
	if len(srsm.relayState) == 0 {
		return nil
	}
	return srsm.relayState[len(srsm.relayState)-1]
}

func (srsm *SmartRouterRelayStateMachine) stateTransition(relayState *relaycore.RelayState, numberOfNodeErrors uint64) {
	batchNumber := srsm.usedProviders.BatchNumber()
	var nextState *relaycore.RelayState
	if relayState == nil { // initial state
		nextState = relaycore.NewRelayState(srsm.ctx, srsm.protocolMessage, 0, srsm.relayRetriesManager, srsm.relaySender, &relaycore.ArchiveStatus{})
	} else {
		// Get the appropriate protocol message (with archive upgrade if needed) BEFORE creating RelayState
		protocolMessage := srsm.GetProtocolMessage()
		archiveStatus := relayState.GetArchiveStatus()

		// Use static function to get upgraded protocol message without creating RelayState
		upgradedProtocolMessage := relaycore.UpgradeToArchiveIfNeeded(srsm.ctx, protocolMessage, archiveStatus, srsm.relaySender, srsm.relayRetriesManager, batchNumber, numberOfNodeErrors)

		// Create the final RelayState with the correct protocol message
		nextState = relaycore.NewRelayState(srsm.ctx, upgradedProtocolMessage, relayState.GetStateNumber()+1, srsm.relayRetriesManager, srsm.relaySender, archiveStatus)
	}
	srsm.appendRelayState(nextState)
}

// Should retry implements the logic for when to send another relay.
// As well as the decision of changing the protocol message,
// into different extensions or addons based on certain conditions
func (srsm *SmartRouterRelayStateMachine) shouldRetry(numberOfNodeErrors uint64) bool {
	batchNumber := srsm.usedProviders.BatchNumber()
	shouldRetry := srsm.retryCondition(batchNumber)
	if shouldRetry {
		srsm.stateTransition(srsm.getLatestState(), numberOfNodeErrors)
	}

	utils.LavaFormatDebug("[StateMachine] shouldRetry called",
		utils.LogAttr("GUID", srsm.ctx),
		utils.LogAttr("numberOfNodeErrors", numberOfNodeErrors),
		utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()),
		utils.LogAttr("selection", srsm.selection),
		utils.LogAttr("shouldRetry", shouldRetry),
	)

	return shouldRetry
}

// hasUnsupportedMethodErrorsInStateMachine checks if we have unsupported method errors at state machine level
func (srsm *SmartRouterRelayStateMachine) hasUnsupportedMethodErrorsInStateMachine() bool {
	if srsm.resultsChecker == nil {
		return false
	}

	// Check if the results checker has unsupported method errors
	if relayProcessor, ok := srsm.resultsChecker.(*relaycore.RelayProcessor); ok {
		return relayProcessor.HasUnsupportedMethodErrors()
	}

	return false
}

func (srsm *SmartRouterRelayStateMachine) retryCondition(numberOfRetriesLaunched int) bool {
	utils.LavaFormatTrace("[StateMachine] retryCondition", utils.LogAttr("numberOfRetriesLaunched", numberOfRetriesLaunched), utils.LogAttr("GUID", srsm.ctx), utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()), utils.LogAttr("selection", srsm.selection))

	// Never retry if we detect unsupported method errors at state machine level
	if srsm.hasUnsupportedMethodErrorsInStateMachine() {
		utils.LavaFormatTrace("[StateMachine] retryCondition: unsupported method detected, no retry", utils.LogAttr("GUID", srsm.ctx))
		return false
	}

	if srsm.resultsChecker.GetQuorumParams().Enabled() && numberOfRetriesLaunched > srsm.resultsChecker.GetQuorumParams().Max {
		return false
	} else if numberOfRetriesLaunched >= MaximumNumberOfTickerRelayRetries {
		return false
	}
	// best result sends to top 10 providers anyway.
	return srsm.selection != relaycore.BestResult
}

func (srsm *SmartRouterRelayStateMachine) GetDebugState() bool {
	return srsm.debugRelays
}

func (srsm *SmartRouterRelayStateMachine) GetProtocolMessage() chainlib.ProtocolMessage {
	latestState := srsm.getLatestState()
	if latestState == nil { // failed fetching latest state
		return srsm.protocolMessage
	}
	return latestState.GetProtocolMessage()
}

// Using RelayStateSendInstructions from relaycore
type RelayStateSendInstructions = relaycore.RelayStateSendInstructions

func (srsm *SmartRouterRelayStateMachine) GetRelayTaskChannel() (chan RelayStateSendInstructions, error) {
	if !srsm.Initialized() {
		return nil, utils.LavaFormatError("SmartRouterRelayStateMachine was not initialized properly", nil)
	}
	relayTaskChannel := make(chan RelayStateSendInstructions)
	go func() {
		// A channel to be notified processing was done, true means we have results and can return
		gotResults := make(chan bool, 1)
		processingTimeout, relayTimeout := srsm.relaySender.getProcessingTimeout(srsm.GetProtocolMessage())
		if srsm.debugRelays {
			utils.LavaFormatDebug("Relay initiated with the following timeout schedule", utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("newRelayTimeout", relayTimeout), utils.LogAttr("GUID", srsm.ctx))
		}
		// Create the processing timeout prior to entering the method so it wont reset every time
		processingCtx, processingCtxCancel := context.WithTimeout(srsm.ctx, processingTimeout)
		defer processingCtxCancel()

		numberOfNodeErrorsAtomic := atomic.Uint64{}
		readResultsFromProcessor := func() {
			// ProcessResults is reading responses while blocking until the conditions are met
			utils.LavaFormatTrace("[StateMachine] Waiting for results", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
			srsm.resultsChecker.WaitForResults(processingCtx)
			// Decide if we need to resend or not
			metRequiredNodeResults, numberOfNodeErrors := srsm.resultsChecker.HasRequiredNodeResults(srsm.usedProviders.BatchNumber())
			numberOfNodeErrorsAtomic.Store(uint64(numberOfNodeErrors))
			gotResults <- metRequiredNodeResults
		}
		go readResultsFromProcessor()
		returnCondition := make(chan error, 1)
		// Used for checking whether to return an error to the user or to allow other channels return their result first see detailed description on the switch case below
		validateReturnCondition := func(err error) {
			batchOnStart := srsm.usedProviders.BatchNumber()
			time.Sleep(15 * time.Millisecond)
			utils.LavaFormatTrace("[StateMachine] validating return condition", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
			if batchOnStart == srsm.usedProviders.BatchNumber() && srsm.usedProviders.CurrentlyUsed() == 0 {
				utils.LavaFormatTrace("[StateMachine] return condition triggered", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("err", err), utils.LogAttr("GUID", srsm.ctx))
				returnCondition <- err
			}
		}

		// initialize relay state
		srsm.stateTransition(nil, 0)
		// Send First Message, with analytics and without waiting for batch update.
		relayTaskChannel <- RelayStateSendInstructions{
			Analytics:      srsm.analytics,
			RelayState:     srsm.getLatestState(),
			NumOfProviders: srsm.resultsChecker.GetQuorumParams().Min,
		}

		// Initialize parameters
		startNewBatchTicker := time.NewTicker(relayTimeout) // Every relay timeout we send a new batch
		defer startNewBatchTicker.Stop()
		consecutiveBatchErrors := 0
		// Start the relay state machine
		for {
			select {
			// Getting batch update for either errors sending message or successful batches
			case err := <-srsm.batchUpdate:
				if err != nil { // Error handling
					utils.LavaFormatTrace("[StateMachine] err := <-srsm.batchUpdate", utils.LogAttr("err", err), utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", srsm.ctx))
					// Sending a new batch failed (consumer's protocol side), handling the state machine
					consecutiveBatchErrors++ // Increase consecutive error counter

					// Circuit breaker: Check if this is a pairing error
					if lavasession.PairingListEmptyError.Is(err) {
						srsm.consecutivePairingErrors++
						utils.LavaFormatDebug("[StateMachine] Detected PairingListEmptyError",
							utils.LogAttr("GUID", srsm.ctx),
							utils.LogAttr("consecutivePairingErrors", srsm.consecutivePairingErrors),
							utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()),
						)

						// Circuit breaker: If we hit pairing errors 2 times in a row, stop retrying
						// After first pairing error, all blocked providers are exhausted and added to unwanted list
						// Second pairing error confirms no providers available - all subsequent attempts will be identical
						if srsm.consecutivePairingErrors >= 2 {
							utils.LavaFormatWarning("Circuit breaker triggered: All providers exhausted, stopping retries",
								nil,
								utils.LogAttr("GUID", srsm.ctx),
								utils.LogAttr("consecutivePairingErrors", srsm.consecutivePairingErrors),
								utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()),
								utils.LogAttr("timesSaved", "~8 seconds of futile retries avoided"),
							)
							go validateReturnCondition(err)
							continue
						}
					} else {
						// Reset pairing error counter if we got a different error type
						srsm.consecutivePairingErrors = 0
					}

					if consecutiveBatchErrors > SendRelayAttempts { // If we failed sending a message more than "SendRelayAttempts" time in a row.
						if srsm.usedProviders.BatchNumber() == 0 && consecutiveBatchErrors == SendRelayAttempts+1 { // First relay attempt. print on first failure only.
							utils.LavaFormatWarning("Failed Sending First Message", err, utils.LogAttr("consecutive errors", consecutiveBatchErrors), utils.LogAttr("GUID", srsm.ctx))
						}
						go validateReturnCondition(err) // Check if we have ongoing messages pending return.
					} else {
						utils.LavaFormatTrace("[StateMachine] batchUpdate - err != nil - batch fail retry attempt", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", srsm.ctx))
						// Failed sending message, but we still want to attempt sending more.
						relayTaskChannel <- RelayStateSendInstructions{RelayState: srsm.getLatestState(), NumOfProviders: 1}
					}
					continue
				}
				// Successfully sent message.
				// Reset consecutiveBatchErrors and pairing error counter
				consecutiveBatchErrors = 0
				srsm.consecutivePairingErrors = 0
			case success := <-gotResults:
				utils.LavaFormatTrace("[StateMachine] success := <-gotResults", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
				// If we had a successful result return what we currently have
				// Or we are done sending relays, and we have no other relays pending results.
				if success { // Check wether we can return the valid results or we need to send another relay
					utils.LavaFormatTrace("[StateMachine] successfully sent message", utils.LogAttr("GUID", srsm.ctx))
					// Reset pairing error counter on success
					srsm.consecutivePairingErrors = 0
					relayTaskChannel <- RelayStateSendInstructions{Done: true}
					return
				}

				// Check if global timeout expired before spawning retry
				select {
				case <-processingCtx.Done():
					// Timeout expired! Stop retrying immediately
					utils.LavaFormatWarning("[StateMachine] Global timeout expired, stopping retries",
						nil,
						utils.LogAttr("GUID", srsm.ctx),
						utils.LogAttr("processingTimeout", processingTimeout),
						utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()),
					)
					relayTaskChannel <- RelayStateSendInstructions{Err: processingCtx.Err(), Done: true}
					return
				default:
					// Timeout not expired yet, safe to continue
				}

				// If should retry == true, send a new batch. (success == false)
				if srsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] LavaFormatTrace success := <-gotResults - srsm.ShouldRetry(batchNumber)", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: srsm.getLatestState(), NumOfProviders: 1}
				} else {
					go validateReturnCondition(nil)
				}
			go readResultsFromProcessor()
		// case <-startNewBatchTicker.C:
		// 	// Only trigger another batch for non BestResult relays or if we didn't pass the retry limit.
		// 	if srsm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
		// 		utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
		// 		relayTaskChannel <- RelayStateSendInstructions{RelayState: srsm.getLatestState(), NumOfProviders: 1}
		// 		// Add ticker launch metrics
		// 		go srsm.tickerMetricSetter.SetRelaySentByNewBatchTickerMetric(srsm.relaySender.GetChainIdAndApiInterface())
		// 	}
		case returnErr := <-returnCondition:
				utils.LavaFormatTrace("[StateMachine] returnErr := <-returnCondition", utils.LogAttr("batch", srsm.usedProviders.BatchNumber()), utils.LogAttr("GUID", srsm.ctx))
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
				userData := srsm.GetProtocolMessage().GetUserData()
				utils.LavaFormatWarning("Relay Got processingCtx timeout", nil,
					utils.LogAttr("processingTimeout", processingTimeout),
					utils.LogAttr("dappId", userData.DappId),
					utils.LogAttr("consumerIp", userData.ConsumerIp),
					utils.LogAttr("protocolMessage.GetApi().Name", srsm.GetProtocolMessage().GetApi().Name),
					utils.LogAttr("GUID", srsm.ctx),
					utils.LogAttr("batchNumber", srsm.usedProviders.BatchNumber()),
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

func (srsm *SmartRouterRelayStateMachine) UpdateBatch(err error) {
	srsm.batchUpdate <- err
}
