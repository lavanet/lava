package relaycore

import (
	context "context"
	"errors"
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
// Retry decisions are centralized in the policy engine (RelayPolicyInf).
type UnifiedRelayStateMachine struct {
	ctx                   context.Context
	relaySender           RelaySenderInf
	resultsChecker        ResultsCheckerInf
	analytics             *metrics.RelayMetrics
	selection             Selection
	crossValidationParams *common.CrossValidationParams
	debugRelays           bool
	batchUpdate           chan error
	usedProviders         *lavasession.UsedProviders
	relayRetriesManager   *lavaprotocol.RelayRetriesManager
	relayState            []*RelayState
	protocolMessage       chainlib.ProtocolMessage
	relayStateLock        sync.RWMutex
	config                StateMachineConfig
	policy                RelayPolicyInf
}

func NewUnifiedRelayStateMachine(
	ctx context.Context,
	usedProviders *lavasession.UsedProviders,
	relaySender RelaySenderInf,
	protocolMessage chainlib.ProtocolMessage,
	analytics *metrics.RelayMetrics,
	debugRelays bool,
	config StateMachineConfig,
	policy RelayPolicyInf,
) (RelayStateMachine, error) {
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
		policy:                policy,
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

// stateTransition creates the next relay state. If the policy returned a mutation,
// it is applied here via applyMutation. Otherwise falls back to UpgradeToArchiveIfNeeded.
func (sm *UnifiedRelayStateMachine) stateTransition(relayState *RelayState, numberOfNodeErrors uint64, mutation *MutationOutput) {
	batchNumber := sm.usedProviders.BatchNumber()
	var nextState *RelayState
	if relayState == nil {
		nextState = NewRelayState(sm.ctx, sm.protocolMessage, 0, sm.relayRetriesManager, sm.relaySender, &ArchiveStatus{})
	} else {
		protocolMessage := sm.GetProtocolMessage()
		archiveStatus := relayState.GetArchiveStatus()

		var upgradedProtocolMessage chainlib.ProtocolMessage
		if mutation != nil && (mutation.ArchiveAction != ArchiveNoChange || mutation.CacheHashes) {
			upgradedProtocolMessage = sm.applyMutation(protocolMessage, archiveStatus, *mutation)
		} else {
			// Fallback: policy.Decide() returned no archive mutation (e.g. epoch-mismatch
			// retry at step 4, or initial state). Legacy UpgradeToArchiveIfNeeded applies
			// batch-number-based archive logic that mirrors decideMutation(). Both paths
			// must stay in sync until the fallback is eliminated.
			upgradedProtocolMessage = UpgradeToArchiveIfNeeded(sm.ctx, protocolMessage, archiveStatus, sm.relaySender, sm.relayRetriesManager, batchNumber, numberOfNodeErrors)
		}

		nextState = NewRelayState(sm.ctx, upgradedProtocolMessage, relayState.GetStateNumber()+1, sm.relayRetriesManager, sm.relaySender, archiveStatus)
	}
	sm.appendRelayState(nextState)
}

// applyMutation applies the policy's archive/cache mutation to the protocol message.
func (sm *UnifiedRelayStateMachine) applyMutation(protocolMessage chainlib.ProtocolMessage, archiveStatus *ArchiveStatus, mutation MutationOutput) chainlib.ProtocolMessage {
	if mutation.CacheHashes {
		cacheBlockHashes(protocolMessage, archiveStatus, sm.relayRetriesManager)
	}

	switch mutation.ArchiveAction {
	case ArchiveAdd:
		return addArchiveExtension(sm.ctx, protocolMessage, archiveStatus, sm.relaySender)
	case ArchiveRemove:
		return removeArchiveExtension(sm.ctx, protocolMessage, archiveStatus, sm.relaySender)
	default:
		return protocolMessage
	}
}

// getResultsSummary retrieves the ResultsSummary from the results checker.
func (sm *UnifiedRelayStateMachine) getResultsSummary() ResultsSummary {
	if sm.resultsChecker == nil {
		return ResultsSummary{}
	}
	return sm.resultsChecker.GetResultsSummary()
}

// buildDecisionInput assembles the DecisionInput for the policy engine.
func (sm *UnifiedRelayStateMachine) buildDecisionInput(numberOfNodeErrors uint64, isTickerHedge bool) DecisionInput {
	latestState := sm.getLatestState()
	var archiveStatus *ArchiveStatus
	if latestState != nil {
		archiveStatus = latestState.GetArchiveStatus()
	}

	return DecisionInput{
		Selection:     sm.selection,
		AttemptNumber: sm.usedProviders.BatchNumber(),
		IsBatch:       sm.protocolMessage.IsBatch(),
		Summary:       sm.getResultsSummary(),
		ArchiveStatus: archiveStatus,
		NodeErrors:    numberOfNodeErrors,
		IsTickerHedge: isTickerHedge,
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
func (sm *UnifiedRelayStateMachine) checkAndHandleTimeout(
	processingCtx context.Context,
	relayTaskChannel chan RelayStateSendInstructions,
	processingTimeout time.Duration,
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
		utils.LogAttr("consecutiveBatchErrors", sm.policy.GetConsecutiveBatchErrors()),
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
		sm.stateTransition(nil, 0, nil)

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

		// Start the relay state machine
		for {
			// SmartRouter: Priority check for processing timeout before select
			if sm.config.EnableTimeoutPriority {
				if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, "priority_check") {
					return
				}
			}

			select {
			case err := <-sm.batchUpdate:
				isPairingListEmpty := err != nil && errors.Is(err, lavasession.PairingListEmptyError)
				result := sm.policy.OnSendRelayResult(err, isPairingListEmpty)

				switch result {
				case SendSuccess:
					// continue to select loop
				case SendStop:
					if sm.usedProviders.BatchNumber() == 0 && sm.policy.GetConsecutiveBatchErrors() == sm.config.SendRelayAttempts+1 {
						utils.LavaFormatWarning("Failed Sending First Message", err, utils.LogAttr("consecutive errors", sm.policy.GetConsecutiveBatchErrors()), utils.LogAttr("GUID", sm.ctx))
					}
					go validateReturnCondition(err)
				case SendRetry:
					if sm.config.EnableTimeoutPriority {
						if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, "batchUpdate_error") {
							return
						}
					}
					utils.LavaFormatTrace("[StateMachine] batchUpdate - send retry", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
				}

			case success := <-gotResults:
				utils.LavaFormatTrace("[StateMachine] success := <-gotResults", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
				if success {
					utils.LavaFormatTrace("[StateMachine] successfully sent message", utils.LogAttr("GUID", sm.ctx))
					relayTaskChannel <- RelayStateSendInstructions{Done: true}
					return
				}

				if sm.config.EnableTimeoutPriority {
					if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, "gotResults_retry") {
						return
					}
				}

				nodeErrors := numberOfNodeErrorsAtomic.Load()
				output := sm.policy.Decide(sm.buildDecisionInput(nodeErrors, false))
				utils.LavaFormatDebug("[StateMachine] policy.Decide",
					utils.LogAttr("GUID", sm.ctx),
					utils.LogAttr("action", output.Action),
					utils.LogAttr("reason", output.Reason),
					utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
				)

				if output.Action == ActionRetry {
					sm.stateTransition(sm.getLatestState(), nodeErrors, &output.Mutation)
					relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
				} else {
					// Don't return immediately — in-flight relays from earlier batches
					// may still succeed. validateReturnCondition waits 15ms and checks
					// whether any relays are still CurrentlyUsed before concluding.
					go validateReturnCondition(nil)
				}
				go readResultsFromProcessor()

			case <-startNewBatchTicker.C:
				if sm.config.EnableTimeoutPriority {
					if sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, "ticker_retry") {
						return
					}
				}

				nodeErrors := numberOfNodeErrorsAtomic.Load()
				output := sm.policy.Decide(sm.buildDecisionInput(nodeErrors, true))
				if output.Action == ActionRetry {
					utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
					sm.stateTransition(sm.getLatestState(), nodeErrors, &output.Mutation)
					relayTaskChannel <- RelayStateSendInstructions{RelayState: sm.getLatestState(), NumOfProviders: 1}
					if sm.analytics != nil {
						sm.analytics.HedgeCount++
					}
				}

			case returnErr := <-returnCondition:
				utils.LavaFormatTrace("[StateMachine] returnErr := <-returnCondition", utils.LogAttr("batch", sm.usedProviders.BatchNumber()), utils.LogAttr("GUID", sm.ctx))
				relayTaskChannel <- RelayStateSendInstructions{Err: returnErr, Done: true}
				return

			case <-processingCtx.Done():
				if sm.config.EnableTimeoutPriority {
					sm.checkAndHandleTimeout(processingCtx, relayTaskChannel, processingTimeout, "processingCtx_done_backup")
				} else {
					userData := sm.GetProtocolMessage().GetUserData()
					utils.LavaFormatWarning("Relay Got processingCtx timeout", nil,
						utils.LogAttr("processingTimeout", processingTimeout),
						utils.LogAttr("dappId", userData.DappId),
						utils.LogAttr("consumerIp", userData.ConsumerIp),
						utils.LogAttr("protocolMessage.GetApi().Name", sm.GetProtocolMessage().GetApi().Name),
						utils.LogAttr("GUID", sm.ctx),
						utils.LogAttr("batchNumber", sm.usedProviders.BatchNumber()),
						utils.LogAttr("consecutiveBatchErrors", sm.policy.GetConsecutiveBatchErrors()),
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
