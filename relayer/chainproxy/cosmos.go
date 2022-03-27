package chainproxy

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/sentry"
)

type CosmosChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewCosmosChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	return &CosmosChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *CosmosChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *CosmosChainProxy) Start(context.Context) error {
	return errors.New("unsupported chain")
}

func (cp *CosmosChainProxy) ParseMsg(data []byte) (NodeMessage, error) {
	return nil, errors.New("unsupported chain")
}

func (nm *CosmosChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {

	return
}
