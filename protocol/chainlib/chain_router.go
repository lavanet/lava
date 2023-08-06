package chainlib

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const (
	sep = "|"
)

type chainRouterEntry struct {
	ChainProxy
	addonsSupported map[string]struct{}
}

func (cre *chainRouterEntry) isSupporting(addon string) bool {
	if addon == "" {
		return true
	}
	if _, ok := cre.addonsSupported[addon]; ok {
		return true
	}
	return false
}

type chainRouterImpl struct {
	lock             *sync.RWMutex
	chainProxyRouter map[RouterKey][]chainRouterEntry
}

type RouterKey string

func NewRouterKey(extensions []string) RouterKey {
	// make sure addons have no repetitions
	uniqueExtensions := map[string]struct{}{}
	for _, extension := range extensions {
		uniqueExtensions[extension] = struct{}{}
	}
	uniqueExtensionsSlice := []string{}
	for addon := range uniqueExtensions { // we are sorting this anyway so we don't have to keep order
		uniqueExtensionsSlice = append(uniqueExtensionsSlice, addon)
	}
	sort.Strings(uniqueExtensionsSlice)
	return RouterKey(sep + strings.Join(uniqueExtensionsSlice, sep) + sep)
}

func (cri *chainRouterImpl) getChainProxySupporting(addon string, extensions []string) (ChainProxy, error) {
	cri.lock.RLock()
	defer cri.lock.RUnlock()
	wantedRouterKey := NewRouterKey(extensions)
	if chainProxyEntries, ok := cri.chainProxyRouter[wantedRouterKey]; ok {
		for _, chainRouterEntry := range chainProxyEntries {
			if chainRouterEntry.isSupporting(addon) {
				return chainRouterEntry.ChainProxy, nil
			}
			if debug {
				utils.LavaFormatDebug("chainProxy supporting extensions but not supporting addon", utils.Attribute{Key: "addon", Value: addon}, utils.Attribute{Key: "wantedRouterKey", Value: wantedRouterKey})
			}
		}
		// no support for this addon
		return nil, utils.LavaFormatError("no chain proxy supporting requested addon", nil, utils.Attribute{Key: "addon", Value: addon})
	}
	// no support for these extensions
	return nil, utils.LavaFormatError("no chain proxy supporting requested extensions", nil, utils.Attribute{Key: "extensions", Value: extensions})
}

func (cri chainRouterImpl) ExtensionsSupported(extensions []string) bool {
	routerKey := NewRouterKey(extensions)
	_, ok := cri.chainProxyRouter[routerKey]
	return ok
}

func (cri chainRouterImpl) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, extensions []string) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// add the parsed addon from the apiCollection
	addon := chainMessage.GetApiCollection().CollectionData.AddOn
	selectedChainProxy, err := cri.getChainProxySupporting(addon, extensions)
	if err != nil {
		return nil, "", nil, err
	}
	return selectedChainProxy.SendNodeMsg(ctx, ch, chainMessage)
}

func newChainRouter(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser, proxyConstructor func(context.Context, uint, lavasession.RPCProviderEndpoint, ChainParser) (ChainProxy, error)) (ChainRouter, error) {
	chainProxyRouter := map[RouterKey][]chainRouterEntry{}

	for _, nodeUrl := range rpcProviderEndpoint.NodeUrls {
		addons, extensions, err := chainParser.SeparateAddonsExtensions(nodeUrl.Addons)
		if err != nil {
			return nil, err
		}
		routerKey := NewRouterKey(extensions)
		addonsSupportedMap := map[string]struct{}{}
		for _, addon := range addons {
			addonsSupportedMap[addon] = struct{}{}
		}
		chainProxy, err := proxyConstructor(ctx, nConns, rpcProviderEndpoint, chainParser)
		if err != nil {
			// TODO: allow some urls to be down
			return nil, err
		}
		chainRouterEntryInst := chainRouterEntry{
			ChainProxy:      chainProxy,
			addonsSupported: addonsSupportedMap,
		}
		if chainRouterEntries, ok := chainProxyRouter[routerKey]; !ok {
			chainProxyRouter[routerKey] = []chainRouterEntry{chainRouterEntryInst}
		} else {
			chainProxyRouter[routerKey] = append(chainRouterEntries, chainRouterEntryInst)
		}
	}

	cri := chainRouterImpl{
		lock:             &sync.RWMutex{},
		chainProxyRouter: chainProxyRouter,
	}
	return cri, nil
}
