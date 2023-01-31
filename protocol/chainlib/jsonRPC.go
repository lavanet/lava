package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/relayer/parser"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
		msg:            &msg,
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
	serverApis, taggedApis := getServiceApis(spec, jsonRPCInterface)

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
		return nil, errors.New("jsonRPC api not supported")
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
func (apip *JsonRPCChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return 0, 0, 0, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Convert average block time from int64 -> time.Duration
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Second

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData, apip.spec.BlocksInFinalizationProof
}

type JsonRPCChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *common.RPCConsumerLogs
}

// NewJrpcChainListener creates a new instance of JsonRPCChainListener
func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *common.RPCConsumerLogs) (chainListener *JsonRPCChainListener) {
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

type JrpcChainProxy struct {
	conn    *chainproxy.Connector
	nConns  uint
	nodeUrl string
}

func NewJrpcChainProxy(nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint) ChainProxy {
	cp := &JrpcChainProxy{nConns: nConns, nodeUrl: rpcProviderEndpoint.NodeUrl}

	return cp
}

func (cp *JrpcChainProxy) Start(ctx context.Context) error {
	cp.conn = chainproxy.NewConnector(ctx, cp.nConns, cp.nodeUrl)
	if cp.conn == nil {
		return errors.New("g_conn == nil")
	}

	return nil
}

func (cp *JrpcChainProxy) SendNodeMsg(ctx context.Context, path string, data []byte, connectionType string, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get node
	rpc, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	defer cp.conn.ReturnRpc(rpc)
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(chainproxy.JsonrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast RPCInput from chainMessage", nil, &map[string]string{"rpcMessage": fmt.Sprintf("%+v", rpcInputMessage)})
	}
	// Call our node
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *chainproxy.JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if ch != nil {
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		connectCtx, cancel := context.WithTimeout(ctx, LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits))
		defer cancel()
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params)
	}

	var replyMsg chainproxy.JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		replyMsg = chainproxy.JsonrpcMessage{
			Version: nodeMessage.Version,
			ID:      nodeMessage.ID,
		}
		replyMsg.Error = &rpcclient.JsonError{
			Code:    1,
			Message: fmt.Sprintf("%s", err),
		}
		// this later causes returning an error
	} else {
		replyMessage, err = chainproxy.ConvertJsonRPCMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC error", err, nil)
		}
		replyMsg = *replyMessage
	}

	retData, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: retData,
	}

	if ch != nil {
		subscriptionID, err = strconv.Unquote(string(replyMsg.Result))
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("Subscription failed", err, nil)
		}
	}

	return reply, subscriptionID, sub, err
}
