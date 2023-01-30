package chainlib

import (
	"context"
	"fmt"
	"time"

	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	grpcInterface          = "grpc"
	jsonRPCInterface       = "jsonrpc"
	restInterface          = "rest"
	tendermintRPCInterface = "tendermintrpc"
)

func NewChainParser(apiInterface string) (chainParser ChainParser, err error) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcChainParser()
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcChainParser()
	case spectypes.APIInterfaceRest:
		return NewRestChainParser()
	case spectypes.APIInterfaceGrpc:
		return NewGrpcChainParser()
	}
	return nil, fmt.Errorf("chainParser for apiInterface (%s) not found", apiInterface)
}

func NewChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *lavaprotocol.RPCConsumerLogs) (ChainListener, error) {
	switch listenEndpoint.ApiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return NewJrpcChainListener(ctx, listenEndpoint, relaySender, rpcConsumerLogs), nil
	case spectypes.APIInterfaceTendermintRPC:
		return NewTendermintRpcChainListener(ctx, listenEndpoint, relaySender, rpcConsumerLogs), nil
	case spectypes.APIInterfaceRest:
		return NewRestChainListener(ctx, listenEndpoint, relaySender, rpcConsumerLogs), nil
	case spectypes.APIInterfaceGrpc:
		return NewGrpcChainListener(ctx, listenEndpoint, relaySender, rpcConsumerLogs), nil
	}
	return nil, fmt.Errorf("chainListener for apiInterface (%s) not found", listenEndpoint.ApiInterface)
}

type ChainParser interface {
	ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error)
	SetSpec(spec spectypes.Spec)
	DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32)
	ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32)
}

type ChainMessage interface {
	GetServiceApi() *spectypes.ServiceApi
	GetInterface() *spectypes.ApiInterface
	RequestedBlock() int64
}

type RelaySender interface {
	SendRelay(
		ctx context.Context,
		url string,
		req string,
		connectionType string,
		dappID string,
		analytics *metrics.RelayMetrics,
	) (*pairingtypes.RelayReply, *pairingtypes.Relayer_RelaySubscribeClient, error)
}

type ChainListener interface {
	Serve(ctx context.Context)
}
