package chainlib

import (
	"context"
	"testing"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	testcommon "github.com/lavanet/lava/testutil/common"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestChainRouterWithDisabledWebSocketInSpec(t *testing.T) {
	ctx := context.Background()
	apiInterface := spectypes.APIInterfaceJsonRPC
	chainParser, err := NewChainParser(apiInterface)
	require.NoError(t, err)

	IgnoreSubscriptionNotConfiguredError = false

	addonsOptions := []string{"-addon-", "-addon2-"}
	extensionsOptions := []string{"-test-", "-test2-", "-test3-"}

	spec := testcommon.CreateMockSpec()
	spec.ApiCollections = []*spectypes.ApiCollection{
		{
			Enabled: false,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
			ParseDirectives: []*spectypes.ParseDirective{{
				FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
			}},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        addonsOptions[0],
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        addonsOptions[1],
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
	}
	chainParser.SetSpec(spec)
	endpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{},
		ChainID:        spec.Index,
		ApiInterface:   apiInterface,
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{},
	}

	type servicesStruct struct {
		services []string
	}

	playBook := []struct {
		name     string
		services []servicesStruct
		success  bool
	}{
		{
			name: "empty services",
			services: []servicesStruct{{
				services: []string{},
			}},
			success: true,
		},
		{
			name: "one-addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}},
			success: true,
		},
		{
			name: "one-extension",
			services: []servicesStruct{{
				services: []string{extensionsOptions[0]},
			}},
			success: false,
		},
		{
			name: "one-extension with empty services",
			services: []servicesStruct{
				{
					services: []string{extensionsOptions[0]},
				},
				{
					services: []string{},
				},
			},
			success: true,
		},
		{
			name: "two-addons together",
			services: []servicesStruct{{
				services: addonsOptions,
			}},
			success: true,
		},
		{
			name: "two-addons, separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}, {
				services: []string{addonsOptions[1]},
			}},
			success: true,
		},
		{
			name: "addon + extension only",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addon + extension, addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: true,
		},
		{
			name: "two addons + extension, addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addons + extension, two addons",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], addonsOptions[1]},
			}},
			success: true,
		},
		{
			name: "addons + two extensions, addon extension",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, other addon",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[1], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[1]},
				},
			},
			success: false,
		},
		{
			name: "addons + two extensions, addon ext1, addon ext2",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[1]},
				},
			},
			success: false,
		},
		{
			name: "addons + two extensions, works",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0]},
				},
			},
			success: true,
		},
		{
			name: "addons + two extensions, works, addon2",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], addonsOptions[1]},
				},
			},
			success: true,
		},
		{
			name: "addon1 + ext, addon 2 + ext, addon 1",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[1], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0]},
				},
			},
			success: false,
		},
		{
			name: "addon1 + ext, addon 2 + ext, addon 1,addon2",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[1], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0]},
				},
				{
					services: []string{addonsOptions[1]},
				},
			},
			success: true,
		},
		{
			name: "addon, ext",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0]},
				},
				{
					services: []string{extensionsOptions[0]},
				},
			},
			success: true,
		},
	}
	for _, play := range playBook {
		t.Run(play.name, func(t *testing.T) {
			nodeUrls := []common.NodeUrl{}
			for _, service := range play.services {
				nodeUrl := common.NodeUrl{Url: "http://127.0.0.1:0"}
				nodeUrl.Addons = service.services
				nodeUrls = append(nodeUrls, nodeUrl)
			}
			endpoint.NodeUrls = nodeUrls
			_, err := GetChainRouter(ctx, 1, endpoint, chainParser)
			if play.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestChainRouterWithEnabledWebSocketInSpec(t *testing.T) {
	ctx := context.Background()
	apiInterface := spectypes.APIInterfaceJsonRPC
	chainParser, err := NewChainParser(apiInterface)
	require.NoError(t, err)

	IgnoreSubscriptionNotConfiguredError = false

	addonsOptions := []string{"-addon-", "-addon2-"}
	extensionsOptions := []string{"-test-", "-test2-", "-test3-"}

	spec := testcommon.CreateMockSpec()
	spec.ApiCollections = []*spectypes.ApiCollection{
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
			ParseDirectives: []*spectypes.ParseDirective{{
				FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
			}},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        "",
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        addonsOptions[0],
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: apiInterface,
				InternalPath: "",
				Type:         "",
				AddOn:        addonsOptions[1],
			},
			Extensions: []*spectypes.Extension{
				{
					Name:         extensionsOptions[0],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[1],
					CuMultiplier: 1,
				},
				{
					Name:         extensionsOptions[2],
					CuMultiplier: 1,
				},
			},
		},
	}
	chainParser.SetSpec(spec)
	endpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{},
		ChainID:        spec.Index,
		ApiInterface:   apiInterface,
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{},
	}

	type servicesStruct struct {
		services []string
	}

	playBook := []struct {
		name     string
		services []servicesStruct
		success  bool
	}{
		{
			name: "empty services, except websocket",
			services: []servicesStruct{{
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "only websocket",
			services: []servicesStruct{{
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "one-addon, without websocket addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "one-addon, with websocket addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "one-extension, without websocket",
			services: []servicesStruct{{
				services: []string{extensionsOptions[0]},
			}},
			success: false,
		},
		{
			name: "one-extension, with websocket",
			services: []servicesStruct{{
				services: []string{extensionsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "one-extension, with empty services, without websocket",
			services: []servicesStruct{{
				services: []string{extensionsOptions[0]},
			}, {
				services: []string{},
			}},
			success: false,
		},
		{
			name: "one-extension, with empty services, with websocket",
			services: []servicesStruct{{
				services: []string{extensionsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "two-addons together, without websocket",
			services: []servicesStruct{{
				services: addonsOptions,
			}},
			success: false,
		},
		{
			name: "two-addons together, with websocket",
			services: []servicesStruct{{
				services: append(addonsOptions, WebSocketExtension),
			}},
			success: true,
		},
		{
			name: "two-addons together, with separated websocket",
			services: []servicesStruct{{
				services: addonsOptions,
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "two-addons, separated, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}, {
				services: []string{addonsOptions[1]},
			}},
			success: false,
		},
		{
			name: "two-addons, separated, with websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}, {
				services: []string{addonsOptions[1]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "addon + extension only, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addon + extension only, with websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addon + extension only, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addon + extension, addon, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addon + extension, addon, with websocket in first",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], WebSocketExtension},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: true,
		},
		{
			name: "addon + extension, addon, with websocket in second",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "addon + extension, addon, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "two addons + extension, addon, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "two addons + extension, addon, with websocket in first",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], addonsOptions[1], extensionsOptions[0], WebSocketExtension},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "two addons + extension, addon, with websocket in second",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "two addons + extension, addon, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addons + extension, two addons, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], addonsOptions[1]},
			}},
			success: true,
		},
		{
			name: "addons + extension, two addons, with websocket in first",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], WebSocketExtension},
			}, {
				services: []string{addonsOptions[0], addonsOptions[1]},
			}},
			success: true,
		},
		{
			name: "addons + extension, two addons, with websocket in second",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], addonsOptions[1], WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "addons + extension, two addons, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], addonsOptions[1]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: true,
		},
		{
			name: "addons + two extensions, addon extension, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon extension, with websocket in first",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1], WebSocketExtension},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon extension, with websocket in second",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1], WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon extension, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon, without websocket",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon, with websocket in first",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1], WebSocketExtension},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon, with websocket in second",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon, with websocket separated",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], WebSocketExtension},
			}, {
				services: []string{WebSocketExtension},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, other addon",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, addon ext1, addon ext2",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}},
			success: false,
		},
		{
			name: "addons + two extensions, works",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0], extensionsOptions[1]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: true,
		},
		{
			name: "addons + two extensions, works, addon2",
			services: []servicesStruct{
				{
					services: []string{addonsOptions[0], extensionsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[0]},
				},
				{
					services: []string{addonsOptions[0], extensionsOptions[1]},
				},
				{
					services: []string{addonsOptions[0], addonsOptions[1]},
				},
			},
			success: true,
		},
		{
			name: "addon1 + ext, addon 2 + ext, addon 1",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}},
			success: false,
		},
		{
			name: "addon1 + ext, addon 2 + ext, addon 1,addon2",
			services: []servicesStruct{{
				services: []string{addonsOptions[0], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[1], extensionsOptions[0]},
			}, {
				services: []string{addonsOptions[0]},
			}, {
				services: []string{addonsOptions[1]},
			}},
			success: true,
		},
		{
			name: "addon, ext",
			services: []servicesStruct{{
				services: []string{addonsOptions[0]},
			}, {
				services: []string{extensionsOptions[0]},
			}},
			success: true,
		},
	}
	for _, play := range playBook {
		t.Run(play.name, func(t *testing.T) {
			nodeUrls := []common.NodeUrl{}
			for _, service := range play.services {
				nodeUrl := common.NodeUrl{Url: "http://127.0.0.1:0"}
				nodeUrl.Addons = service.services
				nodeUrls = append(nodeUrls, nodeUrl)
			}
			endpoint.NodeUrls = nodeUrls
			_, err := GetChainRouter(ctx, 1, endpoint, chainParser)
			if play.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TODO: Elad: add websocket tests
