package chainproxy

import (
	"context"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/sentry"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type CosmosMessage struct {
	cp         *CosmosChainProxy
	serviceApi *spectypes.ServiceApi
	path       string
	msg        []byte
}

type CosmosChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewCosmosChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &CosmosChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *CosmosChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *CosmosChainProxy) Start(context.Context) error {
	return nil
}

func (cp *CosmosChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	return &CosmosMessage{cp: cp, path: path, msg: data}, nil
}

func (nm *CosmosChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	return
}

func (nm *CosmosMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *CosmosMessage) Send(ctx context.Context) (*servicertypes.RelayReply, error) {
	return nil, errors.New("unsupported")
}
