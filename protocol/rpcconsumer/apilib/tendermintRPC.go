package apilib

import (
	"fmt"

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

func NewTendermintRpcAPIListener() (apiListener *TendermintRpcAPIListener) {
	return nil
}
