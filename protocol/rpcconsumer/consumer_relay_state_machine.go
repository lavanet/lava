package rpcconsumer

import (
	context "context"
	"time"

	"github.com/lavanet/lava/v3/protocol/chainlib"
	common "github.com/lavanet/lava/v3/protocol/common"
	lavasession "github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/lavanet/lava/v3/protocol/metrics"
	"github.com/lavanet/lava/v3/utils"
)

type RelayStateMachine interface {
	GetProtocolMessage() chainlib.ProtocolMessage
	ShouldRetry(numberOfRetriesLaunched int) bool
	GetDebugState() bool
	GetRelayTaskChannel() chan RelayStateSendInstructions
	UpdateBatch(err error)
	GetSelection() Selection
	GetUsedProviders() *lavasession.UsedProviders
	SetRelayProcessor(relayProcessor *RelayProcessor)
}

type ConsumerRelaySender interface {
	sendRelayToProvider(ctx context.Context, protocolMessage chainlib.ProtocolMessage, relayProcessor *RelayProcessor, analytics *metrics.RelayMetrics) (errRet error)
	getProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration)
	GetChainIdAndApiInterface() (string, string)
}

type tickerMetricSetterInf interface {
	SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string)
}

type ConsumerRelayStateMachine struct {
	ctx                  context.Context // same context as user context.
	relaySender          ConsumerRelaySender
	parentRelayProcessor *RelayProcessor
	protocolMessage      chainlib.ProtocolMessage // only one should make changes to protocol message is ConsumerRelayStateMachine.
	analytics            *metrics.RelayMetrics    // first relay metrics
	selection            Selection
	debugRelays          bool
	tickerMetricSetter   tickerMetricSetterInf
	batchUpdate          chan error
	usedProviders        *lavasession.UsedProviders
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
	}
}

func (crsm *ConsumerRelayStateMachine) SetRelayProcessor(relayProcessor *RelayProcessor) {
	crsm.parentRelayProcessor = relayProcessor
}

func (crsm *ConsumerRelayStateMachine) GetUsedProviders() *lavasession.UsedProviders {
	return crsm.usedProviders
}

func (crsm *ConsumerRelayStateMachine) GetSelection() Selection {
	return crsm.selection
}

func (crsm *ConsumerRelayStateMachine) ShouldRetry(numberOfRetriesLaunched int) bool {
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
	return crsm.protocolMessage
}

type RelayStateSendInstructions struct {
	protocolMessage chainlib.ProtocolMessage
	analytics       *metrics.RelayMetrics
	err             error
	done            bool
}

func (rssi *RelayStateSendInstructions) IsDone() bool {
	return rssi.done || rssi.err != nil
}

func (crsm *ConsumerRelayStateMachine) GetRelayTaskChannel() chan RelayStateSendInstructions {
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

		readResultsFromProcessor := func() {
			// ProcessResults is reading responses while blocking until the conditions are met
			utils.LavaFormatTrace("[StateMachine] Waiting for results", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
			crsm.parentRelayProcessor.WaitForResults(processingCtx)
			// Decide if we need to resend or not
			if crsm.parentRelayProcessor.HasRequiredNodeResults() {
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

		// Send First Message, with analytics and without waiting for batch update.
		relayTaskChannel <- RelayStateSendInstructions{
			protocolMessage: crsm.GetProtocolMessage(),
			analytics:       crsm.analytics,
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
						relayTaskChannel <- RelayStateSendInstructions{
							protocolMessage: crsm.GetProtocolMessage(),
						}
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
				if crsm.ShouldRetry(crsm.usedProviders.BatchNumber()) {
					utils.LavaFormatTrace("[StateMachine] success := <-gotResults - crsm.ShouldRetry(batchNumber)", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
					relayTaskChannel <- RelayStateSendInstructions{protocolMessage: crsm.GetProtocolMessage()}
				} else {
					go validateReturnCondition(nil)
				}
				go readResultsFromProcessor()
			case <-startNewBatchTicker.C:
				// Only trigger another batch for non BestResult relays or if we didn't pass the retry limit.
				if crsm.ShouldRetry(crsm.usedProviders.BatchNumber()) {
					utils.LavaFormatTrace("[StateMachine] ticker triggered", utils.LogAttr("batch", crsm.usedProviders.BatchNumber()))
					relayTaskChannel <- RelayStateSendInstructions{protocolMessage: crsm.GetProtocolMessage()}
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
	return relayTaskChannel
}

func (crsm *ConsumerRelayStateMachine) UpdateBatch(err error) {
	crsm.batchUpdate <- err
}
