package apilib

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type JsonRPCAPIParser struct{}

func (apip *JsonRPCAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *JsonRPCAPIParser) SetSpec(spec spectypes.Spec) {}

func NewJrpcAPIParser() (apiParser *JsonRPCAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type JsonRPCAPIListener struct{}

func (apil *JsonRPCAPIListener) Serve() {}

func NewJrpcAPIListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *JsonRPCAPIListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
