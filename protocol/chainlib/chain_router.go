package chainlib

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const (
	sep = "/"
)

type chainRouterImpl struct {
	lock             *sync.RWMutex
	chainProxyRouter map[RouterKey]ChainProxy
}

type RouterKey string

func (rk *RouterKey) String() string {
	return string(*rk)
}

func NewRouterKey(addons []string) RouterKey {
	if len(addons) == 0 {
		return sep + sep
	}
	sort.Strings(addons)
	if addons[0] != "" {
		// add support for empty addon in all routers
		addons = append([]string{""}, addons...)
	}
	return RouterKey(sep + strings.Join(addons, sep) + sep)
}

func (cri *chainRouterImpl) getChainProxySupporting(addons []string) ChainProxy {
	cri.lock.RLock()
	defer cri.lock.RUnlock()
	wantedRouterKey := NewRouterKey(addons)
	if chainProxyRet, ok := cri.chainProxyRouter[wantedRouterKey]; ok {
		return chainProxyRet
	}
	type selection struct {
		chainProxy ChainProxy
		routerKey  RouterKey
	}
	selected := selection{}
	// check for the best endpoint that supports the requested addons
possibilitiesLoop:
	for routerKey, chainProxy := range cri.chainProxyRouter {
		for _, addon := range addons {
			addonToSearch := sep + addon + sep
			if !strings.Contains(routerKey.String(), addonToSearch) {
				if debug {
					utils.LavaFormatDebug("chainProxy not supporting", utils.Attribute{Key: "routerKey", Value: routerKey}, utils.Attribute{Key: "addonToSearch", Value: addonToSearch})
				}
				continue possibilitiesLoop
			}
		}
		// if we have a selection that fits and is more precise than what we need
		if strings.Count(selected.routerKey.String(), sep) < strings.Count(routerKey.String(), sep) && selected.routerKey != "" {
			continue
		}
		// means the current routerKey supports all of the addons requested
		selected.chainProxy = chainProxy
		selected.routerKey = routerKey
	}
	return selected.chainProxy
}

func (cri chainRouterImpl) GetSupportedExtensions() []string {
	return nil
}

func (cri chainRouterImpl) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, addons []string) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// add the parsed addon from the apiCollection
	addon := chainMessage.GetApiCollection().CollectionData.AddOn
	if addons == nil {
		addons = []string{addon}
	} else {
		addons = append(addons, addon)
	}
	selectedChainProxy := cri.getChainProxySupporting(addons)
	if selectedChainProxy == nil {
		return nil, "", nil, utils.LavaFormatError("no chain proxy supporting requested addons", nil, utils.Attribute{
			Key:   "addons",
			Value: strings.Join(addons, "/"),
		})
	}
	return selectedChainProxy.SendNodeMsg(ctx, ch, chainMessage)
}

func newChainRouter(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser, proxyConstructor func(context.Context, uint, lavasession.RPCProviderEndpoint, ChainParser) (ChainProxy, error)) (ChainRouter, error) {
	cri := chainRouterImpl{
		lock:             &sync.RWMutex{},
		chainProxyRouter: map[RouterKey]ChainProxy{},
	}
	routerProxies := map[RouterKey][]common.NodeUrl{}
	for _, nodeUrl := range rpcProviderEndpoint.NodeUrls {
		routerKey := NewRouterKey(nodeUrl.Addons)
		var nodeUrls []common.NodeUrl
		var ok bool
		if nodeUrls, ok = routerProxies[routerKey]; !ok {
			nodeUrls = []common.NodeUrl{}
		}
		routerProxies[routerKey] = append(nodeUrls, nodeUrl)
	}
	// now we know which unique chainProxies we need based on addons, create them
	for routerKey, nodeUrls := range routerProxies {
		// chainProxyConstructor will get only it's relevant nodeUrls
		rpcProviderEndpoint.NodeUrls = nodeUrls
		chainProxy, err := proxyConstructor(ctx, nConns, rpcProviderEndpoint, chainParser)
		if err != nil {
			// TODO: allow some urls to be down
			return nil, err
		}
		cri.chainProxyRouter[routerKey] = chainProxy
	}
	return cri, nil
}
