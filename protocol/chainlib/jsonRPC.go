package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/parser"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type JsonRPCChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	BaseChainParser
}

// NewJrpcChainParser creates a new instance of JsonRPCChainParser
func NewJrpcChainParser() (chainParser *JsonRPCChainParser, err error) {
	return &JsonRPCChainParser{}, nil
}

func (apip *JsonRPCChainParser) CraftMessage(serviceApi spectypes.ServiceApi, craftData *CraftData) (ChainMessageForSend, error) {
	if craftData != nil {
		return apip.ParseMsg("", craftData.Data, craftData.ConnectionType, nil)
	}

	msg := rpcInterfaceMessages.JsonrpcMessage{
		Version: "2.0",
		ID:      []byte("1"),
		Method:  serviceApi.GetName(),
		Params:  nil,
	}
	return apip.newChainMessage(&serviceApi, &serviceApi.ApiInterfaces[0], spectypes.NOT_APPLICABLE, msg), nil
}

// this func parses message data into chain message object
func (apip *JsonRPCChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata) (ChainMessage, error) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return nil, errors.New("JsonRPCChainParser not defined")
	}

	// connectionType is currently only used in rest API.
	// Unmarshal request
	msg, err := rpcInterfaceMessages.ParseJsonRPCMsg(data)
	if err != nil {
		return nil, err
	}

	// Check api is supported and save it in nodeMsg
	serviceApi, err := apip.getSupportedApi(msg.Method)
	if err != nil {
		return nil, utils.LavaFormatError("getSupportedApi failed", err, utils.Attribute{Key: "method", Value: msg.Method})
	}

	apiInterface := GetApiInterfaceFromServiceApi(serviceApi, connectionType)
	if apiInterface == nil {
		return nil, fmt.Errorf("could not find the interface %s in the service %s", connectionType, serviceApi.Name)
	}
	requestedBlock, err := parser.ParseBlockFromParams(msg, serviceApi.BlockParsing)
	if err != nil {
		return nil, utils.LavaFormatError("ParseBlockFromParams failed parsing block", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "blockParsing", Value: serviceApi.BlockParsing}, utils.Attribute{Key: "service_api", Value: serviceApi.Name})
	}

	nodeMsg := apip.newChainMessage(serviceApi, apiInterface, requestedBlock, *msg)
	return nodeMsg, nil
}

func (*JsonRPCChainParser) newChainMessage(serviceApi *spectypes.ServiceApi, apiInterface *spectypes.ApiInterface, requestedBlock int64, msg rpcInterfaceMessages.JsonrpcMessage) *parsedMessage {
	nodeMsg := &parsedMessage{
		serviceApi:     serviceApi,
		apiInterface:   apiInterface,
		requestedBlock: requestedBlock,
		msg:            msg,
	}
	return nodeMsg
}

// GetSpec returns saved spec
// if SetSpec is not called it will return empty spec
func (apip *JsonRPCChainParser) GetSpec() (spec spectypes.Spec) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return spectypes.Spec{}
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// return spec
	return apip.spec
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
	serverApis, taggedApis := getServiceApis(spec, spectypes.APIInterfaceJsonRPC)
	for _, name := range taggedApis {
		fmt.Println(name)
	}

	// Set the spec field of the JsonRPCChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.BaseChainParser.SetTaggedApis(taggedApis)
}

func (apip *JsonRPCChainParser) GetInternalPaths() map[string]struct{} {
	internalPaths := map[string]struct{}{}
	for _, api := range apip.serverApis {
		internalPaths[api.InternalPath] = struct{}{}
	}
	return internalPaths
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
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData, apip.spec.BlocksInFinalizationProof
}

type JsonRPCChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *metrics.RPCConsumerLogs
}

// NewJrpcChainListener creates a new instance of JsonRPCChainListener
func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *metrics.RPCConsumerLogs) (chainListener *JsonRPCChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &JsonRPCChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

// Serve http server for JsonRPCChainListener
func (apil *JsonRPCChainListener) Serve(ctx context.Context) {
	// Guard that the JsonRPCChainListener instance exists
	if apil == nil {
		return
	}
	test_mode := common.IsTestMode(ctx)
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	app.Use("/ws/:dappId", func(c *fiber.Ctx) error {
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

	webSocketCallback := websocket.New(func(websockConn *websocket.Conn) {
		var (
			messageType int
			msg         []byte
			err         error
		)
		msgSeed := apil.logger.GetMessageSeed()
		for {
			if messageType, msg, err = websockConn.ReadMessage(); err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
				break
			}
			dappID := extractDappIDFromWebsocketConnection(websockConn)

			ctx, cancel := context.WithCancel(context.Background())
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			defer cancel() // incase there's a problem make sure to cancel the connection
			utils.LavaFormatDebug("ws in <<<", utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "msg", Value: msg}, utils.Attribute{Key: "dappID", Value: dappID})
			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := apil.relaySender.SendRelay(ctx, "", string(msg), http.MethodPost, dappID, metricsData, nil)
			go apil.logger.AddMetricForWebSocket(metricsData, err, websockConn)

			if err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}

				if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
						break
					}

					// If portal cant write to the client
					if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
						cancel()
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
						// break
					}

					apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				}
			} else {
				if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC)
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
			}
		}
	})
	websocketCallbackWithDappID := constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, apil.logger.StoreMetricData)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://HOST:PORT/1/websocket requests.

	app.Post("/:dappId/*", func(fiberCtx *fiber.Ctx) error {
		endTx := apil.logger.LogStartTransaction("jsonRpc-http post")
		defer endTx()
		msgSeed := apil.logger.GetMessageSeed()
		dappID := extractDappIDFromFiberContext(fiberCtx)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		utils.LavaFormatInfo("in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: fiberCtx.Body()}, utils.Attribute{Key: "dappID", Value: dappID})
		if test_mode {
			apil.logger.LogTestMode(fiberCtx)
		}
		reply, _, err := apil.relaySender.SendRelay(ctx, "", string(fiberCtx.Body()), http.MethodPost, dappID, metricsData, nil)
		go apil.logger.AddMetricForHttp(metricsData, err, fiberCtx.GetReqHeaders())
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("jsonrpc http", true, "POST", fiberCtx.Request().URI().String(), string(fiberCtx.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			fiberCtx.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return fiberCtx.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("jsonrpc http",
			false,
			"POST",
			fiberCtx.Request().URI().String(),
			string(fiberCtx.Body()),
			string(reply.Data),
			msgSeed,
			nil,
		)

		// Return json response
		return fiberCtx.SendString(string(reply.Data))
	})

	// Go
	ListenWithRetry(app, apil.endpoint.NetworkAddress)
}

type JrpcChainProxy struct {
	BaseChainProxy
	conn map[string]*chainproxy.Connector
}

func NewJrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, averageBlockTime time.Duration, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	cp := &JrpcChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: nodeUrl},
		conn:           map[string]*chainproxy.Connector{},
	}
	verifyRPCEndpoint(nodeUrl.Url)
	internalPaths := map[string]struct{}{}
	jsonRPCChainParser, ok := chainParser.(*JsonRPCChainParser)
	if ok {
		internalPaths = jsonRPCChainParser.GetInternalPaths()
	}
	return cp, cp.start(ctx, nConns, nodeUrl, internalPaths)
}

func (cp *JrpcChainProxy) start(ctx context.Context, nConns uint, nodeUrl common.NodeUrl, internalPaths map[string]struct{}) error {
	if len(internalPaths) == 0 {
		internalPaths = map[string]struct{}{"": {}} // add default path
	}
	basePath := nodeUrl.Url
	for path := range internalPaths {
		nodeUrl.Url = basePath + path
		conn, err := chainproxy.NewConnector(ctx, nConns, nodeUrl)
		if err != nil {
			return err
		}
		cp.conn[path] = conn
		if cp.conn == nil {
			return errors.New("g_conn == nil")
		}
	}
	return nil
}

func (cp *JrpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get node
	internalPath := chainMessage.GetServiceApi().InternalPath
	rpc, err := cp.conn[internalPath].GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	defer cp.conn[internalPath].ReturnRpc(rpc)
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(rpcInterfaceMessages.JsonrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage})
	}
	// Call our node
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *rpcInterfaceMessages.JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if ch != nil {
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits)
		// check if this API is hanging (waiting for block confirmation)
		if chainMessage.GetInterface().Category.HangingApi {
			relayTimeout += cp.averageBlockTime
		}
		cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)
		connectCtx, cancel := cp.NodeUrl.LowerContextTimeout(ctx, relayTimeout)
		defer cancel()
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params)
		if err != nil && connectCtx.Err() == context.DeadlineExceeded {
			// Not an rpc error, return provider error without disclosing the endpoint address
			return nil, "", nil, utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
		}
	}

	var replyMsg rpcInterfaceMessages.JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		utils.LavaFormatDebug("received an error from SendNodeMsg", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err})
		replyMsg = rpcInterfaceMessages.JsonrpcMessage{
			Version: nodeMessage.Version,
			ID:      nodeMessage.ID,
		}
		replyMsg.Error = &rpcclient.JsonError{
			Code:    1,
			Message: fmt.Sprintf("%s", err),
		}
		// this later causes returning an error
	} else {
		replyMessage, err = rpcInterfaceMessages.ConvertJsonRPCMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC error", err, utils.Attribute{Key: "GUID", Value: ctx})
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
			return nil, "", nil, utils.LavaFormatError("Subscription failed", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}

	return reply, subscriptionID, sub, err
}
