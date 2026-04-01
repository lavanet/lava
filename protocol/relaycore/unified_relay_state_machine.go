package relaycore

import (
	context "context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	lavasession "github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
)

// UnifiedRelayStateMachine is the single state machine implementation used by both
// Consumer and SmartRouter. Behavior differences are controlled via StateMachineConfig.
type UnifiedRelayStateMachine struct {
	ctx                      context.Context
	relaySender              RelaySenderInf
	resultsChecker           ResultsCheckerInf
	analytics                *metrics.RelayMetrics
	selection                Selection
	crossValidationParams    *common.CrossValidationParams
	debugRelays              bool
	batchUpdate              chan error
	usedProviders            *lavasession.UsedProviders
	relayRetriesManager      *lavaprotocol.RelayRetriesManager
	relayState               []*RelayState
	protocolMessage          chainlib.ProtocolMessage
	relayStateLock           sync.RWMutex
	consecutivePairingErrors int // Used only when config.EnableCircuitBreaker is true
	config                   StateMachineConfig
}

func NewUnifiedRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender RelaySenderInf,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
	config StateMachineConfig,
) (RelayStateMachine, error) {
	// Check cross-validation headers FIRST (highest priority)
	// This is the SINGLE SOURCE OF TRUTH for determining if cross-validation is enabled
	crossValidationParams, headersPresent, err := protocolMessage.GetCrossValidationParameters()

	var selection Selection
	var cvParams *common.CrossValidationParams

	if headersPresent && err != nil {
		return nil, utils.LavaFormatError("invalid cross-validation headers", err, utils.LogAttr("GUID", ctx))
	} else if headersPresent {
		selection = CrossValidation
		cvParams = &crossValidationParams
		utils.LavaFormatDebug("[StateMachine] CrossValidation mode enabled",
			utils.LogAttr("maxParticipants", crossValidationParams.MaxParticipants),
			utils.LogAttr("agreementThreshold", crossValidationParams.AgreementThreshold),
			utils.LogAttr("GUID", ctx))
	} else if chainlib.GetStateful(protocolMessage) == common.CONSISTENCY_SELECT_ALL_PROVIDERS {
		selection = Stateful
	} else {
		selection = Stateless
	}

	return &UnifiedRelayStateMachine{
		ctx:                   ctx,
		usedProviders:         usedProviders,
		relaySender:           relaySender,
		protocolMessage:       protocolMessage,
		analytics:             analytics,
		selection:             selection,
		crossValidationParams: cvParams,
		debugRelays:           debugRelays,
		batchUpdate:           make(chan error, config.MaxRetries),
		relayState:            make([]*RelayState, 0),
		config:                config,
	}, nil
}

func (sm *UnifiedRelayStateMachine) Initialized() bool {
	return sm.relayRetriesManager != nil && sm.resultsChecker != nil
}

func (sm *UnifiedRelayStateMachine) SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager) {
	sm.relayRetriesManager = relayRetriesManager
}

func (sm *UnifiedRelayStateMachine) SetResultsChecker(resultsChecker ResultsCheckerInf) {
	sm.resultsChecker = resultsChecker
}

func (sm *UnifiedRelayStateMachine) GetUsedProviders() *lavasession.UsedProviders {
	return sm.usedProviders
}

func (sm *UnifiedRelayStateMachine) GetSelection() Selection {
	return sm.selection
}

func (sm *UnifiedRelayStateMachine) GetCrossValidationParams() *common.CrossValidationParams {
	return sm.crossValidationParams
}

func (sm *UnifiedRelayStateMachine) appendRelayState(nextState *RelayState) {
	sm.relayStateLock.Lock()
	defer sm.relayStateLock.Unlock()
	sm.relayState = append(sm.relayState, nextState)
}

func (sm *UnifiedRelayStateMachine) getLatestState() *RelayState {
	sm.relayStateLock.RLock()
	defer sm.relayStateLock.RUnlock()
	if len(sm.relayState) == 0 {
		return nil
	}
	return sm.relayState[len(sm.relayState)-1]
}

func (sm *UnifiedRelayStateMachine) stateTransition(relayState *RelayState, numberOfNodeErrors uint64) {
	batchNumber := sm.usedProviders.BatchNumber()
	var nextState *RelayState
	if relayState == nil {
		nextState = NewRelayState(sm.ctx, sm.protocolMessage, 0, sm.relayRetriesManager, sm.relaySender, &ArchiveStatus{})
	} else {
		protocolMessage := sm.GetProtocolMessage()
		archiveStatus := relayState.GetArchiveStatus()
		upgradedProtocolMessage := UpgradeToArchiveIfNeeded(sm.ctx, protocolMessage, archiveStatus, sm.relaySender, sm.relayRetriesManager, batchNumber, numberOfNodeErrors)
		nextState = NewRelayState(sm.ctx, upgradedProtocolMessage, relayState.GetStateNumber()+1, sm.relayRetriesManager, sm.relaySender, archiveStatus)
	}
	sm.appendRelayState(nextState)
}

func (sm *UnifiedRelayStateMachine) shouldRetry(numberOfNodeErrors uint64) bool {
	batchNumber := sm.usedProviders.BatchNumber()
	shouldRetry := sm.retryCondition(batchNumber)
	if shouldRetry {
		sm.stateTransition(sm.getLatestState(), numberOfNodeErrors)
	}

	utils.LavaFormatDebug("[StateMachine] shouldRetry called",
		utils.LogAttr("GUID", sm.ctx),
		utils.LogAttr("numberOfNodeErrors", numberOfNodeErrors),
		utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
		utils.LogAttr("selection", sm.selection),
		utils.LogAttr("shouldRetry", shouldRetry),
	)

	return shouldRetry
}

// hasNonRetryableUserFacingErrorsInStateMachine checks if we have non-retryable user-facing errors
// (unsupported method or user input errors) at the state machine level.
func (sm *UnifiedRelayStateMachine) hasNonRetryableUserFacingErrorsInStateMachine() bool {
	if sm.resultsChecker == nil {
		return false
	}
	if relayProcessor, ok := sm.resultsChecker.(*RelayProcessor); ok {
		return relayProcessor.HasNonRetryableUserFacingErrors()
	}
	return false
}

func (sm *UnifiedRelayStateMachine) retryCondition(numberOfRetriesLaunched int) bool {
	utils.LavaFormatTrace("[StateMachine] retryCondition", utils.LogAttr("numberOfRetriesLaunched", numberOfRetriesLaunched), utils.LogAttr("GUID", sm.ctx), utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()), utils.LogAttr("selection", sm.selection))

	switch sm.selection {
	case CrossValidation:
		// No retries - each provider gets exactly one chance (Consumer only, SmartRouter falls to default)
		utils.LavaFormatTrace("[StateMachine] retryCondition: CrossValidation mode, no retry", utils.LogAttr("GUID", sm.ctx))
		return false

	case Stateful:
		utils.LavaFormatTrace("[StateMachine] retryCondition: Stateful mode, no retry", utils.LogAttr("GUID", sm.ctx))
		return false

	case Stateless:
		if DisableBatchRequestRetry && sm.protocolMessage.IsBatch() {
			utils.LavaFormatTrace("[StateMachine] retryCondition: batch request retry disabled, no retry", utils.LogAttr("GUID", sm.ctx))
			return false
		}
		if sm.hasNonRetryableUserFacingErrorsInStateMachine() {
			utils.LavaFormatTrace("[StateMachine] retryCondition: non-retryable user-facing error detected, no retry", utils.LogAttr("GUID", sm.ctx))
			return false
		}
		if numberOfRetriesLaunched >= sm.config.MaxRetries {
			utils.LavaFormatTrace("[StateMachine] retryCondition: max retries reached, no retry", utils.LogAttr("GUID", sm.ctx), utils.LogAttr("numberOfRetriesLaunched", numberOfRetriesLaunched))
			return false
		}
		utils.LavaFormatTrace("[StateMachine] retryCondition: Stateless mode, will retry", utils.LogAttr("GUID", sm.ctx))
		return true

	default:
		utils.LavaFormatTrace("[StateMachine] retryCondition: unknown selection, no retry", utils.LogAttr("GUID", sm.ctx), utils.LogAttr("selection", sm.selection))
		return false
	}
}

func (sm *UnifiedRelayStateMachine) GetDebugState() bool {
	return sm.debugRelays
}

func (sm *UnifiedRelayStateMachine) GetProtocolMessage() chainlib.ProtocolMessage {
	latestState := sm.getLatestState()
	if latestState == nil {
		return sm.protocolMessage
	}
	return latestState.GetProtocolMessage()
}

// checkAndHandleTimeout checks if processingCtx has expired and handles cleanup if so.
// Only used when config.EnableTimeoutPriority is true (SmartRouter mode).
func (sm *UnifiedRelayStateMachine) checkAndHandleTimeout(
	processingCtx context.Context,
	relayTaskChannel chan RelayStateSendInstructions,
	processingTimeout time.Duration,
	consecutiveBatchErrors int,
	location string,
) bool {
	if processingCtx.Err() == nil {
		return false
	}

	userData := sm.GetProtocolMessage().GetUserData()
	utils.LavaFormatWarning("Relay processing timeout expired",
		nil,
		utils.LogAttr("location", location),
		utils.LogAttr("processingTimeout", processingTimeout),
		utils.LogAttr("dappId", userData.DappId),
		utils.LogAttr("consumerIp", userData.ConsumerIp),
		utils.LogAttr("api", sm.GetProtocolMessage().GetApi().Name),
		utils.LogAttr("GUID", sm.ctx),
		utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
		utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors),
	)

	relayTaskChannel <- RelayStateSendInstructions{Err: processingCtx.Err(), Done: true}
	return true
}

func (sm *UnifiedRelayStateMachine) GetRelayTaskChannel() (chan RelayStateSendInstructions, error) {
	if !sm.Initialized() {
		return nil, utils.LavaFormatError("UnifiedRelayStateMachine was not initialized properly", nil)
	}

	relayTaskChannel := make(chan RelayStateSendInstructions, 1)
	go func() {
		gotResults := make(chan bool, 1)
		processingTimeout, relayTimeout := sm.relaySender.GetProcessingTimeout(sm.GetProtocolMessage())
		if sm.debugRelays {
			utils.LavaFormatDebug("Relay initiated with the following timeout schedule", utils.LogAttr("processingTimeout", processingTimeout), utils.LogAttr("newRelayTimeout", relayTimeout), utils.LogAttr("GUID", sm.ctx))
		}
		processingCtx, processingCtxCancel := context.WithTimeout(sm.ctx, processingTimeout)
		defer processingCtxCancel()

		numberOfNodeErrorsAtomic := atomic.Uint64{}
		readResultsFromProcessor := func() {
			utils.LavaFormatTrace("[StateMachine] Waiting for results", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
			sm.resultsChecker.WaitForResults(processingCtx)
			metRequiredNodeResults, numberOfNodeErrors := sm.resultsChecker.HasRequiredNodeResults(sm.usedProviders.BatchNumber())
			numberOfNodeErrorsAtomic.Store(uint64(numberOfNodeErrors))
			gotResults <- metRequiredNodeResults
		}
		go readResultsFromProcessor()
		returnCondition := make(chan error, 1)
		validateReturnCondition := func(err error) {
			batchOnStart := sm.usedProviders.BatchNumber()
			time.Sleep(15 * time.Millisecond)
			utils.LavaFormatTrace("[StateMachine] validating return condition", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
			if batchOnStart == sm.usedProviders.BatchNumber() && sm.usedProviders.CurrentlyUsed() == 0 {
				utils.LavaFormatTrace("[StateMachine] return condition triggered", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("err", err), utils.LogAttr("GUID", sm.ctx))
				returnCondition <- err
			}
		}

		// initialize relay state
		sm.stateTransition(nil, 0)

		// Determine number of providers for initial batch
		var numProviders int
		if sm.selection == CrossValidation && sm.crossValidationParams != nil {
			numProviders = sm.crossValidationParams.MaxParticipants
		} else {
			numProviders = 1
		}

		// Send First Message
		relayTaskChannel <- RelayStateSendInstructions{
			Analytics:      sm.analytics,
			RelayState:     sm.getLatestState(),
			NumOfProviders: numProviders,
		}

		// Initialize parameters
		startNewBatchTicker := time.NewTicker(relayTimeout)
		defer startNewBatchTicker.Stop()
		consecutiveBatchErrors := 0

		// Start the relay state machine
		for {
			// SmartRouter: Priority check for processing timeout before select
			if sm.config.EnableTimeoutPriority {
				if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, consecutiveBatchErrors, "priority_check") {
					return
				}
			}

			select {
			case err := <-sm.batchUpdate:
				if err != nil {
					utils.LavaFormatTrace("[StateMachine] err := <-sm.batchUpdate", utils.LogAttr("err", err), utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", sm.ctx))
					consecutiveBatchErrors++

					// Circuit breaker logic (SmartRouter only)
					if sm.config.EnableCircuitBreaker {
						if lavasession.PairingListEmptyError.Is(err) {
							sm.consecutivePairingErrors++
							utils.LavaFormatDebug("[StateMachine] Detected PairingListEmptyError",
								utils.LogAttr("GUID", sm.ctx),
								utils.LogAttr("consecutivePairingErrors", sm.consecutivePairingErrors),
								utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
							)
							if sm.consecutivePairingErrors >= sm.config.CircuitBreakerThreshold {
								utils.LavaFormatWarning("Circuit breaker triggered: All providers exhausted, stopping retries",
									nil,
									utils.LogAttr("GUID", sm.ctx),
									utils.LogAttr("consecutivePairingErrors", sm.consecutivePairingErrors),
									utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
									utils.LogAttr("timesSaved", "~8 seconds of futile retries avoided"),
								)
								go validateReturnCondition(err)
								continue
							}
						} else {
							sm.consecutivePairingErrors = 0
						}
					}

					if consecutiveBatchErrors > sm.config.SendRelayAttempts {
						if sm.usedProviders.BatchNumber() == 0 && consecutiveBatchErrors == sm.config.SendRelayAttempts+1 {
							utils.LavaFormatWarning("Failed Sending First Message", err, utils.LogAttr("consecutive errors", consecutiveBatchErrors), utils.LogAttr("GUID", sm.ctx))
						}
						go validateReturnCondition(err)
					} else {
						// SmartRouter: Check timeout before retry
						if sm.config.EnableTimeoutPriority {
							if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, consecutiveBatchErrors, "batchUpdate_error") {
								return
							}
						}
						utils.LavaFormatTrace("[StateMachine] batchUpdate - err != nil - batch fail retry attempt", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors), utils.LogAttr("GUID", sm.ctx))
						relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
					}
					continue
				}
				// Successfully sent message
				consecutiveBatchErrors = 0
				if sm.config.EnableCircuitBreaker {
					sm.consecutivePairingErrors = 0
				}

			case success := <-gotResults:
				utils.LavaFormatTrace("[StateMachine] success := <-gotResults", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
				if success {
					utils.LavaFormatTrace("[StateMachine] successfully sent message", utils.LogAttr("GUID", sm.ctx))
					if sm.config.EnableCircuitBreaker {
						sm.consecutivePairingErrors = 0
					}
					relayTaskChannel <- RelayStateSendInstructions{Done: true}
					return
				}

				// SmartRouter: Check timeout before retry
				if sm.config.EnableTimeoutPriority {
					if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, consecutiveBatchErrors, "gotResults_retry") {
						return
					}
				}

				if sm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] success := <-gotResults - sm.ShouldRetry(batchNumber)", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
				} else {
					go validateReturnCondition(nil)
				}
				go readResultsFromProcessor()

			case <-startNewBatchTicker.C:
				// SmartRouter: Check timeout before retry
				if sm.config.EnableTimeoutPriority {
					if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, consecutiveBatchErrors, "ticker_retry") {
						return
					}
				}
				if sm.shouldRetry(numberOfNodeErrorsAtomic.Load()) {
					utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
				}

			case returnErr := <-returnCondition:
				utils.LavaFormatTrace("[StateMachine] returnErr := <-returnCondition", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
				relayTaskChannel <- RelayStateSendInstructions{Err: returnErr, Done: true}
				return

			case <-processingCtx.Done():
				if sm.config.EnableTimeoutPriority {
					// Backup case for SmartRouter - should rarely trigger due to priority checks
					sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, consecutiveBatchErrors, "processingCtx_done_backup")
				} else {
					// Consumer: standard timeout handling
					userData := sm.GetProtocolMessage().GetUserData()
					utils.LavaFormatWarning("Relay Got processingCtx timeout", nil,
						utils.LogAttr("processingTimeout", processingTimeout),
						utils.LogAttr("dappId", userData.DappId),
						utils.LogAttr("consumerIp", userData.ConsumerIp),
						utils.LogAttr("protocolMessage.GetApi().Name", sm.GetProtocolMessage().GetApi().Name),
						utils.LogAttr("GUID", sm.ctx),
						utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
						utils.LogAttr("consecutiveBatchErrors", consecutiveBatchErrors),
					)
					relayTaskChannel <- RelayStateSendInstructions{Err: processingCtx.Err(), Done: true}
				}
				return
			}
		}
	}()
	return relayTaskChannel, nil
}

func (sm *UnifiedRelayStateMachine) UpdateBatch(err error) {
	sm.batchUpdate <- err
}
