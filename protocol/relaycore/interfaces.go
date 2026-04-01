package relaycore

import (
	"context"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
)

// RelayStateMachine interface for managing relay state
type RelayStateMachine interface {
	GetProtocolMessage() chainlib.ProtocolMessage
	GetDebugState() bool
	GetRelayTaskChannel() (chan RelayStateSendInstructions, error)
	UpdateBatch(err error)
	GetSelection() Selection
	GetCrossValidationParams() *common.CrossValidationParams // nil for Stateless/Stateful, non-nil for CrossValidation
	GetUsedProviders() *lavasession.UsedProviders
	SetResultsChecker(resultsChecker ResultsCheckerInf)
	SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager)
}

// ResultsCheckerInf interface for checking results
type ResultsCheckerInf interface {
	WaitForResults(ctx context.Context) error
	HasRequiredNodeResults(tries int) (bool, int)
	GetCrossValidationParams() *common.CrossValidationParams // nil for Stateless/Stateful, non-nil for CrossValidation
}

// MetricsInterface for relay processor metrics
type MetricsInterface interface {
	SetRelayNodeErrorMetric(providerAddress string, chainId string, apiInterface string)
	SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string)
	SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string)
}

// ChainIdAndApiInterfaceGetter interface
type ChainIdAndApiInterfaceGetter interface {
	GetChainIdAndApiInterface() (string, string)
}

// RelayStateSendInstructions struct for relay instructions
type RelayStateSendInstructions struct {
	Analytics      *metrics.RelayMetrics
	Err            error
	Done           bool
	RelayState     *RelayState
	NumOfProviders int
}

func (rssi *RelayStateSendInstructions) IsDone() bool {
	return rssi.Done || rssi.Err != nil
}

// RelaySenderInf is the unified interface for relay senders (Consumer and SmartRouter).
// Both ConsumerRelaySender and SmartRouterRelaySender have identical signatures.
type RelaySenderInf interface {
	RelayParserInf
	GetProcessingTimeout(chainMessage chainlib.ChainMessage) (processingTimeout time.Duration, relayTimeout time.Duration)
	GetChainIdAndApiInterface() (string, string)
}

// TickerMetricSetterInf interface for setting ticker metrics
type TickerMetricSetterInf interface {
	SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string)
}

// StateMachineConfig configures behavior differences between Consumer and SmartRouter state machines
type StateMachineConfig struct {
	// EnableCircuitBreaker enables the PairingListEmptyError circuit breaker (SmartRouter only)
	EnableCircuitBreaker bool
	// CircuitBreakerThreshold is the number of consecutive pairing errors before tripping (default: 2)
	CircuitBreakerThreshold int
	// EnableTimeoutPriority enables priority timeout checks before each select case (SmartRouter only)
	EnableTimeoutPriority bool
	// EnableUnsupportedMethodCheck enables checking for unsupported method errors before retry (Consumer only)
	EnableUnsupportedMethodCheck bool
	// MaxRetries is the maximum number of ticker relay retries
	MaxRetries int
	// SendRelayAttempts is the number of consecutive batch errors before giving up
	SendRelayAttempts int
}
