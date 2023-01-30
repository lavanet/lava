package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	rpcInterface = "jsonrpc"
)

type JsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"` // version of the JSON-RPC protocol used
	ID      json.RawMessage `json:"id,omitempty"`      // identifier of the request
	Method  string          `json:"method,omitempty"`  // name of the method to be invoked
	Params  interface{}     `json:"params,omitempty"`  // parameters for the method
}

// GetParams returns the parameters of the JSON-RPC message
func (jm JsonrpcMessage) GetParams() interface{} {
	return jm.Params
}

// ParseBlock parses the input string and returns a block number
func (jm JsonrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

// GetResult TODO we need it to be able to use parses
// Remove when we deprecated old code
func (cp JsonrpcMessage) GetResult() json.RawMessage {
	return nil
}

type JsonRPCChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi
}

// NewJrpcChainParser creates a new instance of JsonRPCChainParser
func NewJrpcChainParser() (chainParser *JsonRPCChainParser, err error) {
	return &JsonRPCChainParser{}, nil
}

// ParseMsg parses message data into chain message object
func (apip *JsonRPCChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	// connectionType is currently only used in rest API.
	// Unmarshal request
	var msg JsonrpcMessage

	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	// Check api is supported and save it in nodeMsg
	serviceApi, err := apip.getSupportedApi(msg.Method)
	if err != nil {
		return nil, utils.LavaFormatError("getSupportedApi failed", err, &map[string]string{"method": msg.Method})
	}

	var apiInterface *spectypes.ApiInterface = nil
	for i := range serviceApi.ApiInterfaces {
		if serviceApi.ApiInterfaces[i].Type == connectionType {
			apiInterface = &serviceApi.ApiInterfaces[i]
			break
		}
	}
	if apiInterface == nil {
		return nil, fmt.Errorf("could not find the interface %s in the service %s", connectionType, serviceApi.Name)
	}

	requestedBlock, err := parser.ParseBlockFromParams(msg, serviceApi.BlockParsing)
	if err != nil {
		return nil, err
	}
	nodeMsg := &parsedMessage{
		serviceApi:     serviceApi,
		apiInterface:   apiInterface,
		requestedBlock: requestedBlock,
	}
	return nodeMsg, nil
}

// SetSpec sets the spec for the JsonRPCChainParser
func (apip *JsonRPCChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	serverApis, taggedApis := getServiceApis(spec, rpcInterface)

	// Set the spec field of the JsonRPCChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.taggedApis = taggedApis
}

// getSupportedApi fetches service api from spec by name
func (apip *JsonRPCChainParser) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return nil, errors.New("JsonRPCChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := apip.serverApis[name]

	// Return an error if spec does not exist
	if !ok {
		return nil, errors.New("JRPC api not supported")
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
	}

	return &api, nil
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *JsonRPCChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return false, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Return enabled and data reliability threshold from spec
	return apip.spec.Enabled, apip.spec.GetReliabilityThreshold()
}

// ChainBlockStats returns block stats from spec
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData)
func (apip *JsonRPCChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return 0, 0, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Convert average block time from int64 -> time.Duration
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Second

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData
}

type JsonRPCChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *lavaprotocol.RPCConsumerLogs
}

// NewJrpcChainListener creates a new instance of JsonRPCChainListener
func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *lavaprotocol.RPCConsumerLogs) (chainListener *JsonRPCChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &JsonRPCChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

func (apil *JsonRPCChainListener) Serve(ctx context.Context) {
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	app.Use("/ws/:dappId", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("jsonRpc-WebSocket")
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface

	webSocketCallback := websocket.New(func(c *websocket.Conn) {
		var (
			mt  int
			msg []byte
			err error
		)
		msgSeed := apil.logger.GetMessageSeed()
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
				break
			}
			utils.LavaFormatInfo("ws in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // incase there's a problem make sure to cancel the connection
			dappID := extractDappIDFromWebsocketConnection(c)
			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := apil.relaySender.SendRelay(ctx, "", string(msg), http.MethodGet, dappID, metricsData)
			go apil.logger.AddMetric(metricsData, err != nil)
			if err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
						// break
					}

					apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
			}
		}
	})
	websocketCallbackWithDappID := constructFiberCallbackWithDappIDExtraction(webSocketCallback)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("jsonRpc-http post")
		msgSeed := apil.logger.GetMessageSeed()
		dappID := extractDappIDFromFiberContext(c)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body()), "dappID": dappID})

		reply, _, err := apil.relaySender.SendRelay(ctx, "", string(c.Body()), http.MethodGet, dappID, metricsData)
		go apil.logger.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("jsonrpc http", true, "POST", c.Request().URI().String(), string(c.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("jsonrpc http",
			false,
			"POST",
			c.Request().URI().String(),
			string(c.Body()),
			string(reply.Data),
			msgSeed,
			nil,
		)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	// Go
	err := app.Listen(apil.endpoint.NetworkAddress)
	if err != nil {
		utils.LavaFormatError("app.Listen(listenAddr)", err, nil)
	}
}
