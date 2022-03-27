package chainproxy

import (
	"context"
	"errors"
)

type CosmosChainProxy struct {
	nodeUrl string
}

func NewCosmosChainProxy(nodeUrl string) ChainProxy {
	return &CosmosChainProxy{
		nodeUrl: nodeUrl,
	}
}

func (cp *CosmosChainProxy) Start(context.Context) error {
	return errors.New("unsupported chain")
}

func (cp *CosmosChainProxy) ParseMsg(data []byte) (NodeMessage, error) {
	return nil, errors.New("unsupported chain")
}
