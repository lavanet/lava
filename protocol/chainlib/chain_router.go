package chainlib

import (
	"context"
	"strings"
	"sync"

	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type chainRouterEntry struct {
	ChainProxy
	addonsSupported map[string]struct{}
	methodsRouted   map[string]struct{}
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
	chainProxyRouter map[string][]chainRouterEntry // key is routing key
}

func (cri *chainRouterImpl) GetChainProxySupporting(ctx context.Context, addon string, extensions []string, method string, internalPath string) (ChainProxy, error) {
	cri.lock.RLock()
	defer cri.lock.RUnlock()

	// check if that specific method has a special route, if it does apply it to the router key
	wantedRouterKey := lavasession.NewRouterKey(extensions)
	wantedRouterKey.ApplyInternalPath(internalPath)
	wantedRouterKeyStr := wantedRouterKey.String()
	if chainProxyEntries, ok := cri.chainProxyRouter[wantedRouterKeyStr]; ok {
		for _, chainRouterEntry := range chainProxyEntries {
			if chainRouterEntry.isSupporting(addon) {
				// check if the method is supported
				if len(chainRouterEntry.methodsRouted) > 0 {
					if _, ok := chainRouterEntry.methodsRouted[method]; !ok {
						continue
					}
					utils.LavaFormatTrace("chainProxy supporting method routing selected",
						utils.LogAttr("addon", addon),
						utils.LogAttr("wantedRouterKey", wantedRouterKeyStr),
						utils.LogAttr("method", method),
					)
				}
				if wantedRouterKeyStr != lavasession.GetEmptyRouterKey().String() { // add trailer only when router key is not default (||)
					grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeExtension, wantedRouterKeyStr))
				}
				return chainRouterEntry.ChainProxy, nil
			}

			utils.LavaFormatTrace("chainProxy supporting extensions but not supporting addon",
				utils.LogAttr("addon", addon),
				utils.LogAttr("wantedRouterKey", wantedRouterKeyStr),
			)
		}
		// no support for this addon
		return nil, utils.LavaFormatError("no chain proxy supporting requested addon", nil, utils.Attribute{Key: "addon", Value: addon})
	}
	// no support for these extensions
	return nil, utils.LavaFormatError("no chain proxy supporting requested extensions and internal path", nil,
		utils.LogAttr("extensions", extensions),
		utils.LogAttr("internalPath", internalPath),
		utils.LogAttr("supported", cri.chainProxyRouter),
	)
}

func (cri chainRouterImpl) ExtensionsSupported(extensions []string) bool {
	routerKey := lavasession.NewRouterKey(extensions).String()
	_, ok := cri.chainProxyRouter[routerKey]
	return ok
}

func (cri chainRouterImpl) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, extensions []string) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, proxyUrl common.NodeUrl, chainId string, err error) {
	// add the parsed addon from the apiCollection
	addon := chainMessage.GetApiCollection().CollectionData.AddOn
	selectedChainProxy, err := cri.GetChainProxySupporting(ctx, addon, extensions, chainMessage.GetApi().Name, chainMessage.GetApiCollection().CollectionData.InternalPath)
	if err != nil {
		return nil, "", nil, common.NodeUrl{}, "", err
	}
	relayReply, subscriptionID, relayReplyServer, err = selectedChainProxy.SendNodeMsg(ctx, ch, chainMessage)
	proxyUrl, chainId = selectedChainProxy.GetChainProxyInformation()
	return relayReply, subscriptionID, relayReplyServer, proxyUrl, chainId, err
}

// batch nodeUrls with the same addons together in a copy
func (cri *chainRouterImpl) BatchNodeUrlsByServices(rpcProviderEndpoint lavasession.RPCProviderEndpoint) (map[string]lavasession.RPCProviderEndpoint, error) {
	returnedBatch := map[string]lavasession.RPCProviderEndpoint{}
	routesToCheck := map[string]bool{}
	methodRoutes := map[string]int{}
	httpRouteSet := false
	firstWsRouterKey := lavasession.GetEmptyRouterKey()
	firstWsNodeUrl := common.NodeUrl{}
	isFirstWsSet := false
	for _, nodeUrl := range rpcProviderEndpoint.NodeUrls {
		routerKey := lavasession.NewRouterKey(nodeUrl.Addons)
		if len(nodeUrl.Methods) > 0 {
			// all methods defined here will go to the same batch
			methodRoutesUnique := strings.Join(nodeUrl.Methods, ",")
			var existing int
			var ok bool
			if existing, ok = methodRoutes[methodRoutesUnique]; !ok {
				methodRoutes[methodRoutesUnique] = len(methodRoutes)
				existing = len(methodRoutes)
			}
			routerKey.ApplyMethodsRoute(existing)
		}
		routerKey.ApplyInternalPath(nodeUrl.InternalPath)
		isWs, err := IsUrlWebSocket(nodeUrl.Url)
		// Some parsing may fail because of gRPC
		if err == nil && isWs {
			// save the first ws router key and nodeUrl for later use
			if !isFirstWsSet {
				isFirstWsSet = true
				firstWsRouterKey = routerKey
				firstWsNodeUrl = nodeUrl
			}

			// now change the router key to fit the websocket extension key.
			nodeUrl.Addons = append(nodeUrl.Addons, WebSocketExtension)
			routerKey.SetExtensions(nodeUrl.Addons)
		} else {
			httpRouteSet = true
		}
		cri.addRouterKeyToBatch(nodeUrl, returnedBatch, routerKey, rpcProviderEndpoint)
	}

	// check if batch has http configured, if not, add a websocket one
	// prefer one without internal path
	if !httpRouteSet {
		websocketRouterKey := lavasession.GetEmptyRouterKey()
		websocketRouterKey.SetExtensions([]string{WebSocketExtension})
		if websocketRouter, ok := returnedBatch[websocketRouterKey.String()]; ok {
			// we have a websocket route with no internal paths
			returnedBatch[lavasession.GetEmptyRouterKey().String()] = lavasession.RPCProviderEndpoint{
				NetworkAddress: websocketRouter.NetworkAddress,
				ChainID:        websocketRouter.ChainID,
				ApiInterface:   websocketRouter.ApiInterface,
				Geolocation:    websocketRouter.Geolocation,
				NodeUrls:       websocketRouter.NodeUrls,
			}
		} else {
			firstWsRoute := returnedBatch[firstWsRouterKey.String()]
			returnedBatch[firstWsRouterKey.String()] = lavasession.RPCProviderEndpoint{
				NetworkAddress: firstWsRoute.NetworkAddress,
				ChainID:        firstWsRoute.ChainID,
				ApiInterface:   firstWsRoute.ApiInterface,
				Geolocation:    firstWsRoute.Geolocation,
				NodeUrls:       []common.NodeUrl{firstWsNodeUrl},
			}
		}
	}

	if len(returnedBatch) == 0 {
		return nil, utils.LavaFormatError("invalid batch, routes are empty", nil, utils.LogAttr("endpoint", rpcProviderEndpoint))
	}
	// validate all defined method routes have a regular route
	for routerKey, valid := range routesToCheck {
		if !valid {
			return nil, utils.LavaFormatError("invalid batch, missing regular route for method route", nil, utils.LogAttr("routerKey", routerKey))
		}
	}
	utils.LavaFormatDebug("batched nodeUrls by services", utils.LogAttr("batch", returnedBatch))
	return returnedBatch, nil
}

func (*chainRouterImpl) addRouterKeyToBatch(nodeUrl common.NodeUrl, returnedBatch map[string]lavasession.RPCProviderEndpoint, routerKey lavasession.RouterKey, rpcProviderEndpoint lavasession.RPCProviderEndpoint) {
	routerKeyString := routerKey.String()
	if existingEndpoint, ok := returnedBatch[routerKeyString]; !ok {
		returnedBatch[routerKeyString] = lavasession.RPCProviderEndpoint{
			NetworkAddress: rpcProviderEndpoint.NetworkAddress,
			ChainID:        rpcProviderEndpoint.ChainID,
			ApiInterface:   rpcProviderEndpoint.ApiInterface,
			Geolocation:    rpcProviderEndpoint.Geolocation,
			NodeUrls:       []common.NodeUrl{nodeUrl},
		}
	} else {
		// setting the incoming url first as it might be http while existing is websocket. (we prioritize http over ws when possible)
		existingEndpoint.NodeUrls = append([]common.NodeUrl{nodeUrl}, existingEndpoint.NodeUrls...)
		returnedBatch[routerKeyString] = existingEndpoint
	}
}

func newChainRouter(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser, proxyConstructor func(context.Context, uint, lavasession.RPCProviderEndpoint, ChainParser) (ChainProxy, error)) (*chainRouterImpl, error) {
	chainProxyRouter := map[string][]chainRouterEntry{}
	cri := chainRouterImpl{
		lock: &sync.RWMutex{},
	}
	requiredMap := map[string]struct{}{}     // key is requirement
	supportedMap := map[string]requirement{} // key is requirement
	rpcProviderEndpointBatch, err := cri.BatchNodeUrlsByServices(rpcProviderEndpoint)
	if err != nil {
		return nil, err
	}
	for _, rpcProviderEndpointEntry := range rpcProviderEndpointBatch {
		addons, extensions, err := chainParser.SeparateAddonsExtensions(append(rpcProviderEndpointEntry.NodeUrls[0].Addons, ""))
		if err != nil {
			return nil, err
		}
		addonsSupportedMap := map[string]struct{}{}
		// this function calculated all routing combinations and populates them for verification at the end of the function
		updateRouteCombinations := func(extensions, addons []string) (fullySupportedRouterKey lavasession.RouterKey) {
			allExtensionsRouterKey := lavasession.NewRouterKey(extensions)
			requirement := requirement{
				RouterKey: allExtensionsRouterKey,
				addon:     "",
			}
			for _, addon := range addons {
				populateRequiredForAddon(addon, extensions, requiredMap)
				requirement.addon = addon
				supportedMap[requirement.String()] = requirement
				addonsSupportedMap[addon] = struct{}{}
			}
			return allExtensionsRouterKey
		}
		routerKey := updateRouteCombinations(extensions, addons)
		routerKey.ApplyInternalPath(rpcProviderEndpointEntry.NodeUrls[0].InternalPath)
		routerKeyStr := routerKey.String()
		methodsRouted := map[string]struct{}{}
		methods := rpcProviderEndpointEntry.NodeUrls[0].Methods
		if len(methods) > 0 {
			for _, method := range methods {
				methodsRouted[method] = struct{}{}
			}
		}

		chainProxy, err := proxyConstructor(ctx, nConns, rpcProviderEndpointEntry, chainParser)
		if err != nil {
			// TODO: allow some urls to be down
			return nil, err
		}
		chainRouterEntryInst := chainRouterEntry{
			ChainProxy:      chainProxy,
			addonsSupported: addonsSupportedMap,
			methodsRouted:   methodsRouted,
		}
		if chainRouterEntries, ok := chainProxyRouter[routerKeyStr]; !ok {
			chainProxyRouter[routerKeyStr] = []chainRouterEntry{chainRouterEntryInst}
		} else {
			if len(methodsRouted) > 0 {
				// if there are routed methods we want this in the beginning to intercept them
				chainProxyRouter[routerKeyStr] = append([]chainRouterEntry{chainRouterEntryInst}, chainRouterEntries...)
			} else {
				chainProxyRouter[routerKeyStr] = append(chainRouterEntries, chainRouterEntryInst)
			}
		}
	}
	if len(requiredMap) > len(supportedMap) {
		return nil, utils.LavaFormatError("not all requirements supported in chainRouter, missing extensions or addons in definitions", nil, utils.Attribute{Key: "required", Value: requiredMap}, utils.Attribute{Key: "supported", Value: supportedMap})
	}

	_, apiCollection, hasSubscriptionInSpec := chainParser.GetParsingByTag(spectypes.FUNCTION_TAG_SUBSCRIBE)
	// validating we have websocket support for subscription supported specs.
	webSocketSupported := false
	for _, requirement := range supportedMap {
		if requirement.IsRequirementMet(WebSocketExtension) {
			webSocketSupported = true
			break
		}
	}
	if hasSubscriptionInSpec && apiCollection.Enabled && !webSocketSupported {
		err := utils.LavaFormatError("subscriptions are applicable for this chain, but websocket is not provided in 'supported' map. By not setting ws/wss your provider wont be able to accept ws subscriptions, therefore might receive less rewards and lower QOS score.", nil,
			utils.LogAttr("apiInterface", apiCollection.CollectionData.ApiInterface),
			utils.LogAttr("supportedMap", supportedMap),
			utils.LogAttr("required", WebSocketExtension),
		)
		if !IgnoreSubscriptionNotConfiguredError {
			return nil, err
		}
	}

	// make sure all chainProxyRouter entries have one without a method routing
	for routerKey, chainRouterEntries := range chainProxyRouter {
		// get the last entry, if it has methods routed, we need to error out
		lastEntry := chainRouterEntries[len(chainRouterEntries)-1]
		if len(lastEntry.methodsRouted) > 0 {
			return nil, utils.LavaFormatError("last entry in chainProxyRouter has methods routed, this means no chainProxy supports all methods", nil, utils.LogAttr("routerKey", routerKey))
		}
	}

	cri.chainProxyRouter = chainProxyRouter
	utils.LavaFormatDebug("chainRouter created", utils.LogAttr("chainProxyRouter", chainProxyRouter))

	return &cri, nil
}

type requirement struct {
	lavasession.RouterKey
	addon string
}

func (rs *requirement) String() string {
	return rs.RouterKey.String() + "addon:" + rs.addon + lavasession.RouterKeySeparator
}

func (rs *requirement) IsRequirementMet(requirement string) bool {
	return rs.RouterKey.HasExtension(requirement) || strings.Contains(rs.addon, requirement)
}

func populateRequiredForAddon(addon string, extensions []string, required map[string]struct{}) {
	requirement := requirement{
		RouterKey: lavasession.NewRouterKey(extensions),
		addon:     addon,
	}

	requirementKey := requirement.String()
	if _, ok := required[requirementKey]; ok {
		// already handled
		return
	}

	required[requirementKey] = struct{}{}
	for i := 0; i < len(extensions); i++ {
		extensionsWithoutI := make([]string, len(extensions)-1)
		copy(extensionsWithoutI[:i], extensions[:i])
		copy(extensionsWithoutI[i:], extensions[i+1:])
		populateRequiredForAddon(addon, extensionsWithoutI, required)
	}
}
