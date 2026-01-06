// Package relaycoretest provides test utilities for relay processor tests.
// This package should only be imported by test files.
package relaycoretest

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/relaycore"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// RelayProcessorMetricsMock is a mock implementation of metrics interfaces for testing
type RelayProcessorMetricsMock struct{}

func (romm *RelayProcessorMetricsMock) SetRelayNodeErrorMetric(providerAddress, chainId, apiInterface string) {
}

func (romm *RelayProcessorMetricsMock) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
}

func (romm *RelayProcessorMetricsMock) SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
}

func (romm *RelayProcessorMetricsMock) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
}

func (romm *RelayProcessorMetricsMock) GetChainIdAndApiInterface() (string, string) {
	return "testId", "testInterface"
}

var (
	// RelayRetriesManagerInstance is a shared instance for tests
	RelayRetriesManagerInstance = lavaprotocol.NewRelayRetriesManager()
	// RelayProcessorMetrics is a mock metrics implementation for tests
	RelayProcessorMetrics = &RelayProcessorMetricsMock{}
)

// SendSuccessRespJsonRpc sends a successful JSON-RPC response to the relay processor
func SendSuccessRespJsonRpc(relayProcessor *relaycore.RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	id, _ := json.Marshal(1)
	resultBody, _ := json.Marshal(map[string]string{"result": "success"})
	res := rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      id,
		Result:  resultBody,
	}
	resBytes, _ := json.Marshal(res)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relaycore.RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: resBytes, LatestBlock: 1},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   200,
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}

// SendSuccessResp sends a successful response to the relay processor
func SendSuccessResp(relayProcessor *relaycore.RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relaycore.RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte("ok"), LatestBlock: 1},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   200,
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}

// SendProtocolError sends a protocol error response to the relay processor
func SendProtocolError(relayProcessor *relaycore.RelayProcessor, provider string, delay time.Duration, err error) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), err)
	response := &relaycore.RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte(`{"message":"bad","code":123}`)},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   0,
		},
		Err: err,
	}
	relayProcessor.SetResponse(response)
}

// SendNodeError sends a node error response to the relay processor
func SendNodeError(relayProcessor *relaycore.RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relaycore.RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: []byte(`{"message":"bad","code":123}`)},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   500,
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}

// SendNodeErrorJsonRpc sends a JSON-RPC node error response to the relay processor
func SendNodeErrorJsonRpc(relayProcessor *relaycore.RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	id, _ := json.Marshal(1)
	res := rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      id,
		Error:   &rpcclient.JsonError{Code: 1, Message: "test"},
	}
	resBytes, _ := json.Marshal(res)

	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &relaycore.RelayResponse{
		RelayResult: common.RelayResult{
			Request: &pairingtypes.RelayRequest{
				RelaySession: &pairingtypes.RelaySession{},
				RelayData:    &pairingtypes.RelayPrivateData{},
			},
			Reply:        &pairingtypes.RelayReply{Data: resBytes},
			ProviderInfo: common.ProviderInfo{ProviderAddress: provider},
			StatusCode:   500,
		},
		Err: nil,
	}
	relayProcessor.SetResponse(response)
}
