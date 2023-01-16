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
	return nil, nil
}

func (apip *GrpcChainParser) SetSpec(spec spectypes.Spec) {}

func (apip *GrpcChainParser) DataReliabilityEnabled() bool {
	// TODO
	return false
}

func (apip *GrpcChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	//TODO
	return 0, 0, 0
}

func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type GrpcChainListener struct{}

func (apil *GrpcChainListener) Serve() {}

func NewGrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *GrpcChainListener) {
	// open up server for grpc implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
