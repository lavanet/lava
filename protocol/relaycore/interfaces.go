package relaycore

import (
	"context"

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
	GetUsedProviders() *lavasession.UsedProviders
	SetResultsChecker(resultsChecker ResultsCheckerInf)
	SetRelayRetriesManager(relayRetriesManager *lavaprotocol.RelayRetriesManager)
}

// ResultsCheckerInf interface for checking results
type ResultsCheckerInf interface {
	WaitForResults(ctx context.Context) error
	HasRequiredNodeResults(tries int) (bool, int)
	GetCrossValidationParams() common.CrossValidationParams
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
