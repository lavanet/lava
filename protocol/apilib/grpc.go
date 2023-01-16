package apilib

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type GrpcAPIParser struct{}

func (apip *GrpcAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *GrpcAPIParser) SetSpec(spec spectypes.Spec) {}

func (apip *GrpcAPIParser) DataReliabilityEnabled() bool {
	// TODO
	return false
}

func (apip *GrpcAPIParser) GetBlockDistanceForFinalizedData() uint32 {
	//TODO
	return 0
}

func NewGrpcAPIParser() (apiParser *GrpcAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type GrpcAPIListener struct{}

func (apil *GrpcAPIListener) Serve() {}

func NewGrpcAPIListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *GrpcAPIListener) {
	// open up server for grpc implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
