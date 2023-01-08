package apilib

import (
	"fmt"

	spectypes "github.com/lavanet/lava/x/spec/types"
)

type GrpcAPIParser struct{}

func (apip *GrpcAPIParser) ParseMsg(url string, data []byte, connectionType string) (APIMessage, error) {
	return nil, nil
}

func (apip *GrpcAPIParser) SetSpec(spec spectypes.Spec) {}

func NewGrpcAPIParser() (apiParser *GrpcAPIParser, err error) {
	return nil, fmt.Errorf("not implemented")
}

type GrpcAPIListener struct{}

func (apil *GrpcAPIListener) Serve() {}

func NewGrpcAPIListener() (apiListener *GrpcAPIListener) {
	return nil
}
