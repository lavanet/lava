package chainproxy

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/relayer/sentry"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type NodeMessage interface {
	GetServiceApi() *spectypes.ServiceApi
	Send(ctx context.Context) (*servicertypes.RelayReply, error)
}

type ChainProxy interface {
	Start(context.Context) error
	GetSentry() *sentry.Sentry
	ParseMsg([]byte) (NodeMessage, error)
}

func GetChainProxy(specId uint64, nodeUrl string, nConns uint, sentry *sentry.Sentry) (ChainProxy, error) {
	switch specId {
	case 0, 1:
		return NewEthereumChainProxy(nodeUrl, nConns, sentry), nil
	case 2:
		return NewCosmosChainProxy(nodeUrl, sentry), nil
	}
	return nil, fmt.Errorf("chain proxy for chain id (%d) not found", specId)
}
