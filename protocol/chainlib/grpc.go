package chainlib

import (
	"context"
	"fmt"
	"time"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type GrpcChainParser struct{}

func (apip *GrpcChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	return nil, fmt.Errorf("not implemented")
}

func (apip *GrpcChainParser) SetSpec(spec spectypes.Spec) {}

func (apip *GrpcChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// TODO
	return false, 0
}

func (apip *GrpcChainParser) CreateNodeMsg(url string, data []byte, connectionType string) (NodeMessage, error) { // has to be thread safe, reuse code within ParseMsg as common functionality
	return nil, fmt.Errorf("not implemented")
}

func (apip *GrpcChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
	return 0, 0, 0, 0
}

func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type GrpcChainListener struct{}

func (apil *GrpcChainListener) Serve() {}

func NewGrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (chainListener *GrpcChainListener) {
	// open up server for grpc implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}

func NewGrpcChainProxy(nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, chainParser ChainParser) ChainProxy {
	return nil
}
