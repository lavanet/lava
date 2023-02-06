package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendermintChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi
}

// NewTendermintRpcChainParser creates a new instance of TendermintChainParser
func NewTendermintRpcChainParser() (chainParser *TendermintChainParser, err error) {
	return &TendermintChainParser{}, nil
}

// ParseMsg parses message data into chain message object
func (apip *TendermintChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("TendermintChainParser not defined")
	}

	// connectionType is currently only used in rest api
	// Unmarshal request
	var msg chainproxy.JsonrpcMessage
	if string(data) != "" {
		// Fetch pointer to message and error
		msgPtr, err := chainproxy.ParseJsonRPCMsg(data)
		if err != nil {
			return nil, err
		}

		// Assign value of pointer to msg
		msg = *msgPtr
	} else {
		// assuming URI
		var parsedMethod string
		idx := strings.Index(url, "?")
		if idx == -1 {
			parsedMethod = url
		} else {
			parsedMethod = url[0:idx]
		}

		msg = chainproxy.JsonrpcMessage{
			ID:      []byte("1"),
			Version: "2.0",
			Method:  parsedMethod,
		}
		if strings.Contains(url[idx+1:], "=") {
			params := make(map[string]interface{})
			rawParams := strings.Split(url[idx+1:], "&") // list with structure ['height=0x500',...]
			for _, param := range rawParams {
				splitParam := strings.Split(param, "=")
				if len(splitParam) != 2 {
					return nil, utils.LavaFormatError("Cannot parse query params", nil, &map[string]string{"params": param})
				}
				params[splitParam[0]] = splitParam[1]
			}
			msg.Params = params
		} else {
			msg.Params = make(map[string]interface{}, 0)
		}
	}

	// Check api is supported and save it in nodeMsg
	serviceApi, err := apip.getSupportedApi(msg.Method)
	if err != nil {
		return nil, utils.LavaFormatError("getSupportedApi failed", err, &map[string]string{"method": msg.Method})
	}

	// Extract default block parser
	blockParser := serviceApi.BlockParsing

	// Find matched api interface by connection type
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

	// Check if custom block parser exists in the api interface
	// Use custom block parser only for URI calls
	if apiInterface.GetOverwriteBlockParsing() != nil && url != "" {
		blockParser = *apiInterface.GetOverwriteBlockParsing()
	}

	// Fetch requested block, it is used for data reliability
	requestedBlock, err := parser.ParseBlockFromParams(msg, blockParser)
	if err != nil {
		return nil, err
	}

	nodeMsg := &parsedMessage{
		serviceApi:     serviceApi,
		apiInterface:   apiInterface,
		requestedBlock: requestedBlock,
		msg:            msg,
	}
	return nodeMsg, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *TendermintChainParser) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("TendermintChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := apip.serverApis[name]

	// Return an error if spec does not exist
	if !ok {
		return nil, errors.New("tendermintRPC api not supported")
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
	}

	return &api, nil
}

// SetSpec sets the spec for the TendermintChainParser
func (apip *TendermintChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	serverApis, taggedApis := getServiceApis(spec, spectypes.APIInterfaceTendermintRPC)

	// Set the spec field of the TendermintChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.taggedApis = taggedApis
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *TendermintChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the TendermintChainParser instance exists
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
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData, spec.BlocksInFinalizationProof)
func (apip *TendermintChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
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

type TendermintRpcChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *common.RPCConsumerLogs
}

// NewTendermintRpcChainListener creates a new instance of TendermintRpcChainListener
func NewTendermintRpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *common.RPCConsumerLogs) (chainListener *TendermintRpcChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &TendermintRpcChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

// Serve http server for TendermintRpcChainListener
func (apil *TendermintRpcChainListener) Serve(ctx context.Context) {
	// Guard that the TendermintChainParser instance exists
	if apil == nil {
		return
	}

	// Setup HTTP Server
	app := fiber.New(fiber.Config{})
	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface

	app.Use(favicon.New())

	app.Use("/ws/:dappId", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("tendermint-WebSocket")
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	webSocketCallback := websocket.New(func(c *websocket.Conn) {
		var (
			mt  int
			msg []byte
			err error
		)
		msgSeed := apil.logger.GetMessageSeed()
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				break
			}
			dappID := extractDappIDFromWebsocketConnection(c)
			utils.LavaFormatInfo("ws in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg), "dappID": dappID})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // incase there's a problem make sure to cancel the connection
			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := apil.relaySender.SendRelay(ctx, "", string(msg), http.MethodGet, dappID, metricsData)
			go apil.logger.AddMetric(metricsData, err != nil)
			if err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						// break
					}
					apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
			}
		}
	})
	websocketCallbackWithDappID := constructFiberCallbackWithDappIDExtraction(webSocketCallback)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("tendermint-WebSocket")
		msgSeed := apil.logger.GetMessageSeed()
		dappID := extractDappIDFromFiberContext(c)
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body()), "dappID": dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := apil.relaySender.SendRelay(ctx, "", string(c.Body()), http.MethodGet, dappID, metricsData)
		go apil.logger.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "POST", c.Request().URI().String(), string(c.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := chainproxy.ConvertToTendermintError(errMasking, c.Body())

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		apil.logger.LogStartTransaction("tendermint-WebSocket")

		query := "?" + string(c.Request().URI().QueryString())
		path := c.Params("*")
		dappID := ""
		if len(c.Route().Params) > 1 {
			dappID = c.Route().Params[1]
			dappID = strings.ReplaceAll(dappID, "*", "")
		}
		msgSeed := apil.logger.GetMessageSeed()
		utils.LavaFormatInfo("urirpc in <<<", &map[string]string{"seed": msgSeed, "msg": path, "dappID": dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := apil.relaySender.SendRelay(ctx, path+query, "", http.MethodGet, dappID, metricsData)
		go apil.logger.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "GET", c.Request().URI().String(), "", errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			if string(c.Body()) != "" {
				errMasking = addAttributeToError("recommendation", "For jsonRPC use POST", errMasking)
			}

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "GET", c.Request().URI().String(), "", string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})
	//
	// Go
	err := app.Listen(apil.endpoint.NetworkAddress)
	if err != nil {
		utils.LavaFormatError("app.Listen(listenAddr)", err, nil)
	}
}

type tendermintRpcChainProxy struct {
	// embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
	nodeUrl string
}

func NewtendermintRpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint) (ChainProxy, error) {
	cp := &tendermintRpcChainProxy{nodeUrl: rpcProviderEndpoint.NodeUrl}
	return cp, cp.start(ctx, nConns, rpcProviderEndpoint.NodeUrl)
}

func (cp *tendermintRpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	rpc, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	defer cp.conn.ReturnRpc(rpc)
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(chainproxy.TendermintrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast RPCInput from chainMessage", nil, &map[string]string{"rpcMessage": fmt.Sprintf("%+v", rpcInputMessage)})
	}
	if nodeMessage.Path != "" {
		return cp.SendURI(ctx, &nodeMessage, ch, chainMessage)
	}

	// Else do RPC call
	return cp.SendRPC(ctx, &nodeMessage, ch, chainMessage)
}

func (cp *tendermintRpcChainProxy) SendURI(ctx context.Context, nodeMessage *chainproxy.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// check if the input channel is not nil
	if ch != nil {
		// return an error if the channel is not nil
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on Tendermint URI", nil, nil)
	}

	// create a new http client with a timeout set by the getTimePerCu function
	httpClient := http.Client{
		Timeout: LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits),
	}

	// construct the url by concatenating the node url with the path variable
	url := cp.nodeUrl + "/" + nodeMessage.Path

	// create a new http request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", nil, err
	}

	// send the http request and get the response
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}

	// close the response body
	if res.Body != nil {
		defer res.Body.Close()
	}

	// read the response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct with the response body as the data
	reply := &pairingtypes.RelayReply{
		Data: body,
	}

	return reply, "", nil, nil
}

// SendRPC sends Tendermint HTTP or WebSockets call
func (cp *tendermintRpcChainProxy) SendRPC(ctx context.Context, nodeMessage *chainproxy.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get rpc connection from the connection pool
	rpc, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	// return the rpc connection to the pool after the function completes
	defer cp.conn.ReturnRpc(rpc)

	// create variables for the rpc message and reply message
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *chainproxy.RPCResponse
	var sub *rpcclient.ClientSubscription

	// If ch is not nil do subscription
	if ch != nil {
		// subscribe to the rpc call if the channel is not nil
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		// create a context with a timeout set by the getTimePerCu function
		connectCtx, cancel := context.WithTimeout(ctx, LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits))
		defer cancel()
		// perform the rpc call
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params)
	}

	var replyMsg *chainproxy.RPCResponse
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			// Not an rpc error, return provider error without disclosing the endpoint address
			return nil, "", nil, utils.LavaFormatError("Failed Sending Message", context.DeadlineExceeded, nil)
		}
		id, idErr := chainproxy.IdFromRawMessage(nodeMessage.ID)
		if idErr != nil {
			return nil, "", nil, utils.LavaFormatError("Failed parsing ID when getting rpc error", idErr, nil)
		}
		replyMsg = &chainproxy.RPCResponse{
			JSONRPC: nodeMessage.Version,
			ID:      id,
			Error:   chainproxy.ConvertErrorToRPCError(err.Error(), -1), // TODO: extract code from error status / message
		}
	} else {
		replyMessage, err = chainproxy.ConvertTendermintMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermingRPC error", err, nil)
		}

		replyMsg = replyMessage
	}

	// marshal the jsonrpc message to json
	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct
	reply := &pairingtypes.RelayReply{
		Data: data,
	}

	if ch != nil {
		// get the params for the rpc call
		params := nodeMessage.Params

		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown params type on tendermint subscribe", nil, nil)
		}
		subscriptionID, ok = paramsMap["query"].(string)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown subscriptionID type on tendermint subscribe", nil, nil)
		}
	}

	return reply, subscriptionID, sub, err
}
