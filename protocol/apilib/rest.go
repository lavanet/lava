package apilib

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestAPIParser struct{}

func (apip *RestAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *RestAPIParser) SetSpec(spec spectypes.Spec) {}

func NewRestAPIParser() (apiParser *RestAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type RestAPIListener struct{}

func (apil *RestAPIListener) Serve() {}

func NewRestAPIListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender) (apiListener *RestAPIListener) {
	// open up server for http implementing the api requested (currently implemented in serve_portal in chainproxy, endpoint at listenEndpoint
	// when receiving the data such as url, rpc data, headers (connectionType), use relaySender to wrap verify and send that data
	return nil
}
