package apilib

import (
	"fmt"

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

func NewRestAPIListener() (apiListener *RestAPIListener) {
	return nil
}
