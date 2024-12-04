package chainlib

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	testcommon "github.com/lavanet/lava/v4/testutil/common"
	"github.com/lavanet/lava/v4/utils"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

var (
	listenerAddressTcp  = "localhost:0"
	listenerAddressHttp = ""
	listenerAddressWs   = ""
)

type TimeServer int64

func TestChainRouterWithDisabledWebSocketInSpec(t *testing.T) {
	ctx := context.Background()
	apiInterface := spectypes.APIInterfaceJsonRPC
	chainParser, err := NewChainParser(apiInterface)
	require.NoError(t, err)

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
				nodeUrl := common.NodeUrl{Url: listenerAddressHttp}
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
				nodeUrl := common.NodeUrl{Url: listenerAddressHttp}
				nodeUrl.Addons = service.services
				nodeUrls = append(nodeUrls, nodeUrl)
				nodeUrl.Url = listenerAddressWs
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

type chainProxyMock struct {
	endpoint lavasession.RPCProviderEndpoint
}

func (m *chainProxyMock) GetChainProxyInformation() (common.NodeUrl, string) {
	urlStr := ""
	if len(m.endpoint.NodeUrls) > 0 {
		urlStr = m.endpoint.NodeUrls[0].UrlStr()
	}
	return common.NodeUrl{}, urlStr
}

func (m *chainProxyMock) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	return nil, "", nil, nil
}

type PolicySt struct {
	addons       []string
	extensions   []string
	apiInterface string
}

func (a PolicySt) GetSupportedAddons(string) ([]string, error) {
	return a.addons, nil
}

func (a PolicySt) GetSupportedExtensions(string) ([]epochstoragetypes.EndpointService, error) {
	ret := []epochstoragetypes.EndpointService{}
	for _, ext := range a.extensions {
		ret = append(ret, epochstoragetypes.EndpointService{Extension: ext, ApiInterface: a.apiInterface})
	}
	return ret, nil
}

func TestChainRouterWithMethodRoutes(t *testing.T) {
	ctx := context.Background()
	apiInterface := spectypes.APIInterfaceRest
	chainParser, err := NewChainParser(apiInterface)
	require.NoError(t, err)

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
			Apis: []*spectypes.Api{
				{
					Enabled: true,
					Name:    "api-1",
				},
				{
					Enabled: true,
					Name:    "api-2",
				},
				{
					Enabled: true,
					Name:    "api-8",
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
			ParseDirectives: []*spectypes.ParseDirective{{
				FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
			}},
			Apis: []*spectypes.Api{
				{
					Enabled: true,
					Name:    "api-3",
				},
				{
					Enabled: true,
					Name:    "api-4",
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
	const extMarker = "::ext::"
	playBook := []struct {
		name            string
		nodeUrls        []common.NodeUrl
		success         bool
		apiToUrlMapping map[string]string
	}{
		{
			name: "addon routing",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
					Addons:  []string{addonsOptions[0]},
				},
				{
					Url:    "ws:-0-",
					Addons: []string{addonsOptions[0]},
				},
				{
					Url:     "-1-",
					Methods: []string{"api-2"},
				},
			},
			success: true,
			apiToUrlMapping: map[string]string{
				"api-1": "-0-",
				"api-2": "-1-",
				"api-3": "-0-",
			},
		},
		{
			name: "basic method routing",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{},
				},
				{
					Url:     "-1-",
					Methods: []string{"api-2"},
				},
				{
					Url:     "ws:-1-",
					Methods: []string{},
				},
			},
			success: true,
			apiToUrlMapping: map[string]string{
				"api-1": "-0-",
				"api-2": "-1-",
			},
		},
		{
			name: "method routing with extension",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{},
				},
				{
					Url:    "-1-",
					Addons: []string{extensionsOptions[0]},
				},
				{
					Url:     "-2-",
					Methods: []string{"api-2"},
					Addons:  []string{extensionsOptions[0]},
				},
			},
			success: true,
			apiToUrlMapping: map[string]string{
				"api-1": "-0-",
				"api-2": "-0-",
				"api-1" + extMarker + extensionsOptions[0]: "-1-",
				"api-2" + extMarker + extensionsOptions[0]: "-2-",
			},
		},
		{
			name: "method routing with two extensions",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{},
				},
				{
					Url:    "-1-",
					Addons: []string{extensionsOptions[0]},
				},
				{
					Url:     "-2-",
					Methods: []string{"api-2"},
					Addons:  []string{extensionsOptions[0]},
				},
				{
					Url:    "-3-",
					Addons: []string{extensionsOptions[1]},
				},
				{
					Url:     "-4-",
					Methods: []string{"api-8"},
					Addons:  []string{extensionsOptions[1]},
				},
			},
			success: true,
			apiToUrlMapping: map[string]string{
				"api-1": "-0-",
				"api-2": "-0-",
				"api-1" + extMarker + extensionsOptions[0]: "-1-",
				"api-2" + extMarker + extensionsOptions[0]: "-2-",
				"api-1" + extMarker + extensionsOptions[1]: "-3-",
				"api-8" + extMarker + extensionsOptions[1]: "-4-",
			},
		},
		{
			name: "two method routings with extension",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{},
				},
				{
					Url:    "-1-",
					Addons: []string{extensionsOptions[0]},
				},
				{
					Url:     "-2-",
					Methods: []string{"api-2"},
					Addons:  []string{extensionsOptions[0]},
				},
				{
					Url:     "-3-",
					Methods: []string{"api-8"},
					Addons:  []string{extensionsOptions[0]},
				},
				{
					Url:     "ws:-1-",
					Methods: []string{},
				},
			},
			success: true,
			apiToUrlMapping: map[string]string{
				"api-1": "-0-",
				"api-2": "-0-",
				"api-1" + extMarker + extensionsOptions[0]: "-1-",
				"api-2" + extMarker + extensionsOptions[0]: "-2-",
				"api-8" + extMarker + extensionsOptions[0]: "-3-",
			},
		},
		{
			name: "method routing without base",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{"api-1"},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{"api-1"},
				},
			},
			success: false,
		},
		{
			name: "method routing without base with extension",
			nodeUrls: []common.NodeUrl{
				{
					Url:     "-0-",
					Methods: []string{},
				},
				{
					Url:     "ws:-0-",
					Methods: []string{},
				},
				{
					Url:     "-1-",
					Addons:  []string{extensionsOptions[0]},
					Methods: []string{"api-1"},
				},
			},
			success: false,
		},
	}
	mockProxyConstructor := func(_ context.Context, _ uint, endp lavasession.RPCProviderEndpoint, _ ChainParser) (ChainProxy, error) {
		mockChainProxy := &chainProxyMock{endpoint: endp}
		return mockChainProxy, nil
	}
	for _, play := range playBook {
		t.Run(play.name, func(t *testing.T) {
			endpoint.NodeUrls = play.nodeUrls
			policy := PolicySt{
				addons:       addonsOptions,
				extensions:   extensionsOptions,
				apiInterface: apiInterface,
			}
			chainParser.SetPolicy(policy, spec.Index, apiInterface)
			chainRouter, err := newChainRouter(ctx, 1, *endpoint, chainParser, mockProxyConstructor)
			if play.success {
				require.NoError(t, err)
				for api, url := range play.apiToUrlMapping {
					extension := extensionslib.ExtensionInfo{}
					if strings.Contains(api, extMarker) {
						splitted := strings.Split(api, extMarker)
						api = splitted[0]
						extension.ExtensionOverride = []string{splitted[1]}
					}
					chainMsg, err := chainParser.ParseMsg(api, nil, "", nil, extension)
					require.NoError(t, err)
					chainProxy, err := chainRouter.GetChainProxySupporting(ctx, chainMsg.GetApiCollection().CollectionData.AddOn, common.GetExtensionNames(chainMsg.GetExtensions()), api, "")
					require.NoError(t, err)
					_, urlFromProxy := chainProxy.GetChainProxyInformation()
					require.Equal(t, url, urlFromProxy, "chainMsg: %+v, ---chainRouter: %+v", chainMsg, chainRouter)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func createRPCServer() net.Listener {
	listener, err := net.Listen("tcp", listenerAddressTcp)
	if err != nil {
		log.Fatal("Listener error: ", err)
	}

	app := fiber.New(fiber.Config{
		JSONEncoder: gojson.Marshal,
		JSONDecoder: gojson.Unmarshal,
	})
	app.Use(favicon.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer c.Close()
		for {
			// Read message from WebSocket
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}

			// Print the message to the console
			log.Printf("Received: %s", message)

			// Echo the message back
			err = c.WriteMessage(mt, message)
			if err != nil {
				log.Println("Write error:", err)
				break
			}
		}
	}))

	listenerAddressTcp = listener.Addr().String()
	listenerAddressHttp = "http://" + listenerAddressTcp
	listenerAddressWs = "ws://" + listenerAddressTcp + "/ws"
	// Serve accepts incoming HTTP connections on the listener l, creating
	// a new service goroutine for each. The service goroutines read requests
	// and then call handler to reply to them
	go app.Listener(listener)

	return listener
}

func TestMain(m *testing.M) {
	listener := createRPCServer()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := rpcclient.DialContext(ctx, listenerAddressHttp)
		_, err2 := rpcclient.DialContext(ctx, listenerAddressWs)
		if err2 != nil {
			utils.LavaFormatDebug("waiting for grpc server to launch")
			continue
		}
		if err != nil {
			utils.LavaFormatDebug("waiting for grpc server to launch")
			continue
		}
		cancel()
		break
	}

	utils.LavaFormatDebug("listening on", utils.LogAttr("address", listenerAddressHttp))

	// Start running tests.
	code := m.Run()
	listener.Close()
	os.Exit(code)
}

func TestChainRouterWithInternalPaths(t *testing.T) {
	type play struct {
		name                       string
		specApiCollections         []*spectypes.ApiCollection
		apiInterface               string
		nodeUrls                   []common.NodeUrl
		expectedServicesToNodeUrls map[string][]common.NodeUrl
		expectedError              bool
	}

	playBook := []play{}

	apiInterfaces := []string{spectypes.APIInterfaceJsonRPC, spectypes.APIInterfaceTendermintRPC}
	for _, apiInterface := range apiInterfaces {
		playBook = append(playBook, []play{
			{
				name:         "no_internal_paths_in_spec__single_http_node_url_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||": {{Url: "https://localhost:1234", InternalPath: ""}},
				},
			},
			{
				name:         "no_internal_paths_in_spec__multiple_http_node_urls_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:5678",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||": {
						{Url: "https://localhost:1234", InternalPath: ""},
						{Url: "https://localhost:5678", InternalPath: ""},
					},
				},
			},
			{
				name:         "no_internal_paths_in_spec__single_ws_node_url__should_error",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "wss://localhost:1234/ws",
						InternalPath: "",
					},
				},
				expectedError: true,
			},
			{
				name:         "no_internal_paths_in_spec__both_ws_and_http_node_urls",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "wss://localhost:1234/ws",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":          {{Url: "https://localhost:1234", InternalPath: ""}},
					"|websocket|": {{Url: "wss://localhost:1234/ws", InternalPath: ""}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__single_http_node_url_configured__not_covering_all_internal_paths",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:1234/Y", InternalPath: "/Y"}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__multiple_http_node_urls_configured__covering_some_internal_paths",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:1234/X",
						InternalPath: "/X",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:1234/Y", InternalPath: "/Y"}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__multiple_http_node_urls_configured__covering_all_internal_paths",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:1234/X",
						InternalPath: "/X",
					},
					{
						Url:          "https://localhost:1234/Y",
						InternalPath: "/Y",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:1234/Y", InternalPath: "/Y"}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__multiple_http_node_urls_configured__no_root_internal_path__should_error",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234/X",
						InternalPath: "/X",
					},
					{
						Url:          "https://localhost:1234/Y",
						InternalPath: "/Y",
					},
				},
				expectedError: true,
			},
			{
				name:         "with_internal_paths_in_spec__multiple_http_node_urls_and_ws_configured__covering_all_internal_paths",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "wss://localhost:1234/ws",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:5678",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:5678/X",
						InternalPath: "/X",
					},
					{
						Url:          "https://localhost:9012/Y",
						InternalPath: "/Y",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:5678", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:5678/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:9012/Y", InternalPath: "/Y"}},
					"|websocket|":         {{Url: "wss://localhost:1234/ws", InternalPath: ""}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__only_root_http_and_ws_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "wss://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:1234/Y", InternalPath: "/Y"}},
					"|websocket|":         {{Url: "wss://localhost:1234", InternalPath: ""}},
				},
			},
			{
				name:         "with_internal_paths_in_spec__only_root_http_and_ws_and_one_out_of_two_internal_paths_are_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/Y",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "https://localhost:5678/X",
						InternalPath: "/X",
					},
					{
						Url:          "wss://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:5678/X", InternalPath: "/X"}},
					"||internal-path:/Y|": {{Url: "https://localhost:1234/Y", InternalPath: "/Y"}},
					"|websocket|":         {{Url: "wss://localhost:1234", InternalPath: ""}},
				},
			},
			{
				name:         "with_internal_paths_and_ws_internal_paths_in_spec__only_http_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                  {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|": {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
				},
			},
			{
				name:         "with_internal_paths_and_ws_internal_paths_in_spec__http_and_ws_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "wss://localhost:5678",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                            {{Url: "https://localhost:1234", InternalPath: ""}},
					"||internal-path:/X|":           {{Url: "https://localhost:1234/X", InternalPath: "/X"}},
					"|websocket|":                   {{Url: "wss://localhost:5678", InternalPath: ""}},
					"|websocket|internal-path:/WS|": {{Url: "wss://localhost:5678/WS", InternalPath: "/WS"}},
				},
			},
			{
				name:         "with_internal_paths_and_multiple_ws_internal_paths_in_spec__http_and_ws_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
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
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "",
					},
					{
						Url:          "wss://localhost:1234",
						InternalPath: "",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||":                            {{Url: "https://localhost:1234", InternalPath: ""}},
					"|websocket|":                   {{Url: "wss://localhost:1234", InternalPath: ""}},
					"|websocket|internal-path:/WS|": {{Url: "wss://localhost:1234/WS", InternalPath: "/WS"}},
					"|websocket|internal-path:/X|":  {{Url: "wss://localhost:1234/X", InternalPath: "/X"}},
				},
			},
			{
				name:         "with_internal_paths_and_mixed_internal_paths_in_spec_and_root_is_disabled_http_only_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: false,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "wss://localhost:1234",
						InternalPath: "/WS",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"|websocket|internal-path:/WS|": {{Url: "wss://localhost:1234", InternalPath: "/WS"}},
				},
			},
			{
				name:         "with_internal_paths_and_mixed_internal_paths_in_spec_and_root_is_disabled_ws_only_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: false,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "wss://localhost:1234",
						InternalPath: "/WS",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"|websocket|internal-path:/WS|": {{Url: "wss://localhost:1234", InternalPath: "/WS"}},
				},
			},
			{
				name:         "with_internal_paths_and_mixed_internal_paths_in_spec_and_root_is_disabled_http_and_ws_is_configured",
				apiInterface: apiInterface,
				specApiCollections: []*spectypes.ApiCollection{
					{
						Enabled: false,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "",
							Type:         "POST",
							AddOn:        "",
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/X",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
					},
					{
						Enabled: true,
						CollectionData: spectypes.CollectionData{
							ApiInterface: apiInterface,
							InternalPath: "/WS",
							Type:         "POST",
							AddOn:        "",
						},
						InheritanceApis: []*spectypes.CollectionData{
							{
								ApiInterface: apiInterface,
								InternalPath: "",
								Type:         "POST",
								AddOn:        "",
							},
						},
						ParseDirectives: []*spectypes.ParseDirective{{
							FunctionTag: spectypes.FUNCTION_TAG_SUBSCRIBE,
						}},
					},
				},
				nodeUrls: []common.NodeUrl{
					{
						Url:          "https://localhost:1234",
						InternalPath: "/X",
					},
					{
						Url:          "wss://localhost:1234",
						InternalPath: "/WS",
					},
				},
				expectedServicesToNodeUrls: map[string][]common.NodeUrl{
					"||internal-path:/X|":           {{Url: "https://localhost:1234", InternalPath: "/X"}},
					"|websocket|internal-path:/WS|": {{Url: "wss://localhost:1234", InternalPath: "/WS"}},
				},
			},
		}...)
	}

	for _, play := range playBook {
		t.Run(play.apiInterface+"__"+play.name, func(t *testing.T) {
			chainParser, err := NewChainParser(play.apiInterface)
			require.NoError(t, err)

			spec := testcommon.CreateMockSpec()
			spec.ApiCollections = play.specApiCollections
			chainParser.SetSpec(spec)

			endpoint := lavasession.RPCProviderEndpoint{
				NetworkAddress: lavasession.NetworkAddressData{},
				ChainID:        spec.Index,
				ApiInterface:   play.apiInterface,
				Geolocation:    1,
				NodeUrls:       play.nodeUrls,
			}

			chainRouter := &chainRouterImpl{}

			nodeUrlsByService, err := chainRouter.BatchNodeUrlsByServices(endpoint, chainParser)
			if play.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, len(play.expectedServicesToNodeUrls), len(nodeUrlsByService), nodeUrlsByService)
			actualNodeUrlsCount := 0
			for routerKey, actualEndpoint := range nodeUrlsByService {
				// Check that the router key is in the expected services
				require.Contains(t, play.expectedServicesToNodeUrls, routerKey, routerKey)
				actualNodeUrlsCount += len(actualEndpoint.NodeUrls)

				expectedNodeUrls := play.expectedServicesToNodeUrls[routerKey]
				require.Len(t, actualEndpoint.NodeUrls, len(expectedNodeUrls),
					fmt.Sprintf("RouterKey: %v, NodeUrls: %v", routerKey, actualEndpoint.NodeUrls))

				for _, actualNodeUrl := range actualEndpoint.NodeUrls {
					found := false
					for _, expectedNodeUrls := range expectedNodeUrls {
						if expectedNodeUrls.Url == actualNodeUrl.Url && expectedNodeUrls.InternalPath == actualNodeUrl.InternalPath {
							found = true
							break
						}
					}
					require.True(t, found, actualNodeUrl)
				}
			}
		})
	}
}
