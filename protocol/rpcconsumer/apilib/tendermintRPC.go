package apilib

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/relayer/lavasession"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendermintAPIParser struct{}

func (apip *TendermintAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *TendermintAPIParser) SetSpec(spec spectypes.Spec) {}

func NewTendermintRpcAPIParser() (apiParser *TendermintAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type TendermintRpcAPIListener struct{}

func (apil *TendermintRpcAPIListener) Serve() {}

func NewTendermintRpcAPIListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, apiPArser APIParser, relaySender RelaySender) (apiListener *TendermintRpcAPIListener) {
	return nil
}
