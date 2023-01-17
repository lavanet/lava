package chainlib

import (
	"context"
	"fmt"
	"time"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type JsonRPCChainParser struct{}

func (apip *JsonRPCChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	return nil, nil
}

func (apip *JsonRPCChainParser) SetSpec(spec spectypes.Spec) {}

func (apip *JsonRPCChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// TODO
	return false, 0
}

func (apip *JsonRPCChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	//TODO
	return 0, 0, 0
}

func NewJrpcChainParser() (chainParser *JsonRPCChainParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type JsonRPCChainListener struct{}

func (apil *JsonRPCChainListener) Serve() {}

func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *JsonRPCChainListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
