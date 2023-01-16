package lavaprotocol

import (
	"context"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	TimePerCU                = uint64(100 * time.Millisecond)
	MinimumTimePerRelayDelay = time.Second
	AverageWorldLatency      = 200 * time.Millisecond
)

type RelayRequestCommonData struct {
	ChainID        string `protobuf:"bytes,1,opt,name=chainID,proto3" json:"chainID,omitempty"`
	ConnectionType string `protobuf:"bytes,2,opt,name=connection_type,json=connectionType,proto3" json:"connection_type,omitempty"`
	ApiUrl         string `protobuf:"bytes,3,opt,name=api_url,json=apiUrl,proto3" json:"api_url,omitempty"`
	Data           []byte `protobuf:"bytes,6,opt,name=data,proto3" json:"data,omitempty"`
	RequestBlock   int64  `protobuf:"varint,11,opt,name=request_block,json=requestBlock,proto3" json:"request_block,omitempty"`
}

func NewRelayRequestCommonData(chainID string, connectionType string, apiUrl string, data []byte, requestBlock int64) RelayRequestCommonData {
	return RelayRequestCommonData{
		ChainID:        chainID,
		ConnectionType: connectionType,
		ApiUrl:         apiUrl,
		Data:           data,
		RequestBlock:   requestBlock,
	}
}

func ConstructRelayRequest(ctx context.Context, privKey *btcec.PrivateKey, chainID string, relayRequestCommonData RelayRequestCommonData, providerPublicAddress string, consumerSession *lavasession.SingleConsumerSession, epoch int64, reportedProviders []byte) (*pairingtypes.RelayRequest, error) {
	relayRequest := &pairingtypes.RelayRequest{
		Provider:              providerPublicAddress,
		ConnectionType:        relayRequestCommonData.ConnectionType,
		ApiUrl:                relayRequestCommonData.ApiUrl,
		Data:                  relayRequestCommonData.Data,
		SessionId:             uint64(consumerSession.SessionId),
		ChainID:               chainID,
		CuSum:                 consumerSession.CuSum + consumerSession.LatestRelayCu, // add the latestRelayCu which will be applied when session is returned properly
		BlockHeight:           epoch,
		RelayNum:              consumerSession.RelayNum + lavasession.RelayNumberIncrement, // increment the relay number. which will be applied when session is returned properly
		RequestBlock:          relayRequestCommonData.RequestBlock,
		QoSReport:             consumerSession.QoSInfo.LastQoSReport,
		DataReliability:       nil,
		UnresponsiveProviders: reportedProviders,
	}
	sig, err := sigs.SignRelay(privKey, *relayRequest)
	if err != nil {
		return nil, err
	}
	relayRequest.Sig = sig
	return relayRequest, nil
}

func GetTimePerCu(cu uint64) time.Duration {
	return time.Duration(cu*TimePerCU) + MinimumTimePerRelayDelay
}

func UpdateRequestedBlock(request *pairingtypes.RelayRequest, response *pairingtypes.RelayReply) {
	// since sometimes the user is sending requested block that is a magic like latest, or earliest we need to specify to the reliability what it is
	request.RequestBlock = ReplaceRequestedBlock(request.RequestBlock, response.LatestBlock)
}

func ReplaceRequestedBlock(requestedBlock int64, latestBlock int64) int64 {
	switch requestedBlock {
	case spectypes.LATEST_BLOCK:
		return latestBlock
	case spectypes.SAFE_BLOCK:
		return latestBlock
	case spectypes.FINALIZED_BLOCK:
		return latestBlock
	case spectypes.EARLIEST_BLOCK:
		return spectypes.NOT_APPLICABLE // TODO: add support for earliest block reliability
	}
	return requestedBlock
}
