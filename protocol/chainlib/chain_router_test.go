package chainlib

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	testcommon "github.com/lavanet/lava/v2/testutil/common"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestChainRouter(t *testing.T) {
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
			name: "one-extension works",
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
