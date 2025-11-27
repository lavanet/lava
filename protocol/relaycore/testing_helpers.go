package relaycore

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavaprotocol"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// Testing helpers for relay processor tests

type RelayProcessorMetricsMock struct{}

func (romm *RelayProcessorMetricsMock) SetRelayNodeErrorMetric(providerAddress, chainId, apiInterface string) {
}

func (romm *RelayProcessorMetricsMock) SetNodeErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
}

func (romm *RelayProcessorMetricsMock) SetProtocolErrorRecoveredSuccessfullyMetric(chainId string, apiInterface string, attempt string) {
}

func (romm *RelayProcessorMetricsMock) SetRelaySentByNewBatchTickerMetric(chainId string, apiInterface string) {
}

func (romm *RelayProcessorMetricsMock) SetNodeErrorAttemptMetric(chainId string, apiInterface string) {
}

func (romm *RelayProcessorMetricsMock) GetChainIdAndApiInterface() (string, string) {
	return "testId", "testInterface"
}

var (
	RelayRetriesManagerInstance = lavaprotocol.NewRelayRetriesManager()
	RelayProcessorMetrics       = &RelayProcessorMetricsMock{}
)

func SendSuccessRespJsonRpc(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
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
	response := &RelayResponse{
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

func SendSuccessResp(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &RelayResponse{
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

func SendProtocolError(relayProcessor *RelayProcessor, provider string, delay time.Duration, err error) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), err)
	response := &RelayResponse{
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

func SendNodeError(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &RelayResponse{
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

func SendNodeErrorJsonRpc(relayProcessor *RelayProcessor, provider string, delay time.Duration) {
	time.Sleep(delay)
	id, _ := json.Marshal(1)
	res := rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      id,
		Error:   &rpcclient.JsonError{Code: 1, Message: "test"},
	}
	resBytes, _ := json.Marshal(res)

	relayProcessor.GetUsedProviders().RemoveUsed(provider, lavasession.NewRouterKey(nil), nil)
	response := &RelayResponse{
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
