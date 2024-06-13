package chainlib

import (
	"context"
	"sync"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	chainProxyRouter map[lavasession.RouterKey][]chainRouterEntry
}

func (cri *chainRouterImpl) getChainProxySupporting(ctx context.Context, addon string, extensions []string) (ChainProxy, error) {
	cri.lock.RLock()
	defer cri.lock.RUnlock()
	wantedRouterKey := lavasession.NewRouterKey(extensions)
	if chainProxyEntries, ok := cri.chainProxyRouter[wantedRouterKey]; ok {
		for _, chainRouterEntry := range chainProxyEntries {
			if chainRouterEntry.isSupporting(addon) {
				if wantedRouterKey != lavasession.GetEmptyRouterKey() { // add trailer only when router key is not default (||)
					grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeExtension, string(wantedRouterKey)))
				}
				return chainRouterEntry.ChainProxy, nil
			}

			utils.LavaFormatTrace("chainProxy supporting extensions but not supporting addon",
				utils.LogAttr("addon", addon),
				utils.LogAttr("wantedRouterKey", wantedRouterKey),
			)
		}
		// no support for this addon
		return nil, utils.LavaFormatError("no chain proxy supporting requested addon", nil, utils.Attribute{Key: "addon", Value: addon})
	}
	// no support for these extensions
	return nil, utils.LavaFormatError("no chain proxy supporting requested extensions", nil, utils.Attribute{Key: "extensions", Value: extensions})
}

func (cri chainRouterImpl) ExtensionsSupported(extensions []string) bool {
	routerKey := lavasession.NewRouterKey(extensions)
	_, ok := cri.chainProxyRouter[routerKey]
	return ok
}

func (cri chainRouterImpl) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, extensions []string) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error) {
	// add the parsed addon from the apiCollection
	addon := chainMessage.GetApiCollection().CollectionData.AddOn
	selectedChainProxy, err := cri.getChainProxySupporting(ctx, addon, extensions)
	if err != nil {
		return nil, "", nil, common.NodeUrl{}, "", err
	}
	relayReply, subscriptionID, relayReplyServer, err = selectedChainProxy.SendNodeMsg(ctx, ch, chainMessage)
	proxyUrl, chainId = selectedChainProxy.GetChainProxyInformation()
	return relayReply, subscriptionID, relayReplyServer, proxyUrl, chainId, err
}

// batch nodeUrls with the same addons together in a copy
func batchNodeUrlsByServices(rpcProviderEndpoint lavasession.RPCProviderEndpoint) map[lavasession.RouterKey]lavasession.RPCProviderEndpoint {
	returnedBatch := map[lavasession.RouterKey]lavasession.RPCProviderEndpoint{}
	for _, nodeUrl := range rpcProviderEndpoint.NodeUrls {
		if existingEndpoint, ok := returnedBatch[lavasession.NewRouterKey(nodeUrl.Addons)]; !ok {
			returnedBatch[lavasession.NewRouterKey(nodeUrl.Addons)] = lavasession.RPCProviderEndpoint{
				NetworkAddress: rpcProviderEndpoint.NetworkAddress,
				ChainID:        rpcProviderEndpoint.ChainID,
				ApiInterface:   rpcProviderEndpoint.ApiInterface,
				Geolocation:    rpcProviderEndpoint.Geolocation,
				NodeUrls:       []common.NodeUrl{nodeUrl}, // add existing nodeUrl to the batch
			}
		} else {
			existingEndpoint.NodeUrls = append(existingEndpoint.NodeUrls, nodeUrl)
			returnedBatch[lavasession.NewRouterKey(nodeUrl.Addons)] = existingEndpoint
		}
	}
	return returnedBatch
}

func newChainRouter(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser, proxyConstructor func(context.Context, uint, lavasession.RPCProviderEndpoint, ChainParser) (ChainProxy, error)) (ChainRouter, error) {
	chainProxyRouter := map[lavasession.RouterKey][]chainRouterEntry{}

	requiredMap := map[requirementSt]struct{}{}
	supportedMap := map[requirementSt]struct{}{}
	rpcProviderEndpointBatch := batchNodeUrlsByServices(rpcProviderEndpoint)
	for _, rpcProviderEndpointEntry := range rpcProviderEndpointBatch {
		addons, extensions, err := chainParser.SeparateAddonsExtensions(append(rpcProviderEndpointEntry.NodeUrls[0].Addons, ""))
		if err != nil {
			return nil, err
		}
		addonsSupportedMap := map[string]struct{}{}
		// this function calculated all routing combinations and populates them for verification at the end of the function
		updateRouteCombinations := func(extensions, addons []string) (fullySupportedRouterKey lavasession.RouterKey) {
			allExtensionsRouterKey := lavasession.NewRouterKey(extensions)
			requirement := requirementSt{
				extensions: allExtensionsRouterKey,
				addon:      "",
			}
			for _, addon := range addons {
				populateRequiredForAddon(addon, extensions, requiredMap)
				requirement.addon = addon
				supportedMap[requirement] = struct{}{}
				addonsSupportedMap[addon] = struct{}{}
			}
			return allExtensionsRouterKey
		}
		routerKey := updateRouteCombinations(extensions, addons)
		chainProxy, err := proxyConstructor(ctx, nConns, rpcProviderEndpointEntry, chainParser)
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
	if len(requiredMap) > len(supportedMap) {
		return nil, utils.LavaFormatError("not all requirements supported in chainRouter, missing extensions or addons in definitions", nil, utils.Attribute{Key: "required", Value: requiredMap}, utils.Attribute{Key: "supported", Value: supportedMap})
	}

	cri := chainRouterImpl{
		lock:             &sync.RWMutex{},
		chainProxyRouter: chainProxyRouter,
	}
	return cri, nil
}

type requirementSt struct {
	extensions lavasession.RouterKey
	addon      string
}

func populateRequiredForAddon(addon string, extensions []string, required map[requirementSt]struct{}) {
	if len(extensions) == 0 {
		required[requirementSt{
			extensions: lavasession.NewRouterKey([]string{}),
			addon:      addon,
		}] = struct{}{}
		return
	}
	requirement := requirementSt{
		extensions: lavasession.NewRouterKey(extensions),
		addon:      addon,
	}
	if _, ok := required[requirement]; ok {
		// already handled
		return
	}
	required[requirement] = struct{}{}
	for i := 0; i < len(extensions); i++ {
		extensionsWithoutI := make([]string, len(extensions)-1)
		copy(extensionsWithoutI[:i], extensions[:i])
		copy(extensionsWithoutI[i:], extensions[i+1:])
		populateRequiredForAddon(addon, extensionsWithoutI, required)
	}
}
