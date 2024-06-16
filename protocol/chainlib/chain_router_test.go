package chainlib

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	testcommon "github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
