package apilib

import (
	"context"
	"fmt"
	"time"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendermintAPIParser struct{}

func (apip *TendermintAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *TendermintAPIParser) SetSpec(spec spectypes.Spec) {}

func (apip *TendermintAPIParser) DataReliabilityEnabled() bool {
	// TODO
	return false
}

func (apip *TendermintAPIParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	//TODO
	return 0, 0, 0
}

func NewTendermintRpcAPIParser() (apiParser *TendermintAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type TendermintRpcAPIListener struct{}

func (apil *TendermintRpcAPIListener) Serve() {}

func NewTendermintRpcAPIListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *TendermintRpcAPIListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
