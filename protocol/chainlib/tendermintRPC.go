package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendermintChainParser struct {
	BaseChainParser
}

// NewTendermintRpcChainParser creates a new instance of TendermintChainParser
func NewTendermintRpcChainParser() (chainParser *TendermintChainParser, err error) {
	return &TendermintChainParser{}, nil
}

func (apip *TendermintChainParser) getApiCollection(connectionType string, internalPath string, addon string) (*spectypes.ApiCollection, error) {
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getApiCollection(connectionType, internalPath, addon)
}

func (apip *TendermintChainParser) getSupportedApi(name string, connectionType string) (*ApiContainer, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getSupportedApi(name, connectionType)
}

func (apip *TendermintChainParser) CraftMessage(parsing *spectypes.Parsing, connectionType string, craftData *CraftData) (ChainMessageForSend, error) {
	if craftData != nil {
		return apip.ParseMsg("", craftData.Data, craftData.ConnectionType, nil)
	}

	msg := rpcInterfaceMessages.JsonrpcMessage{
		Version: "2.0",
		ID:      []byte("1"),
		Method:  parsing.ApiName,
		Params:  nil,
	}

	apiCont, err := apip.getSupportedApi(parsing.ApiName, connectionType)
	if err != nil {
		return nil, err
	}
	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}
	tenderMsg := rpcInterfaceMessages.TendermintrpcMessage{JsonrpcMessage: msg, Path: parsing.ApiName}
	return apip.newChainMessage(apiCont.api, spectypes.NOT_APPLICABLE, tenderMsg, apiCollection), nil
}

// ParseMsg parses message data into chain message object
func (apip *TendermintChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata) (ChainMessage, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("TendermintChainParser not defined")
	}

	// connectionType is currently only used in rest api
	// Unmarshal request
	var msg rpcInterfaceMessages.JsonrpcMessage
	isJsonrpc := string(data) != ""
	if isJsonrpc {
		// Fetch pointer to message and error
		msgPtr, err := rpcInterfaceMessages.ParseJsonRPCMsg(data)
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

		msg = rpcInterfaceMessages.JsonrpcMessage{
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
					return nil, utils.LavaFormatError("Cannot parse query params", nil, utils.Attribute{Key: "params", Value: param})
				}
				params[splitParam[0]] = splitParam[1]
			}
			msg.Params = params
		} else {
			msg.Params = make(map[string]interface{}, 0)
		}
	}

	// Check api is supported and save it in nodeMsg
	apiCont, err := apip.getSupportedApi(msg.Method, connectionType)
	if err != nil {
		return nil, utils.LavaFormatError("getSupportedApi failed", err, utils.Attribute{Key: "method", Value: msg.Method})
	}

	// Extract default block parser
	blockParser := apiCont.api.BlockParsing
	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}

	// Fetch requested block, it is used for data reliability
	requestedBlock, err := parser.ParseBlockFromParams(msg, blockParser)
	if err != nil {
		return nil, utils.LavaFormatError("ParseBlockFromParams failed parsing block", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "blockParsing", Value: apiCont.api.BlockParsing})
	}
	tenderMsg := rpcInterfaceMessages.TendermintrpcMessage{JsonrpcMessage: msg, Path: ""}
	if !isJsonrpc {
		tenderMsg.Path = url // add path
	}
	nodeMsg := apip.newChainMessage(apiCont.api, requestedBlock, tenderMsg, apiCollection)
	return nodeMsg, nil
}

func (*TendermintChainParser) newChainMessage(serviceApi *spectypes.Api, requestedBlock int64, msg rpcInterfaceMessages.TendermintrpcMessage, apiCollection *spectypes.ApiCollection) ChainMessage {
	nodeMsg := &parsedMessage{
		api:            serviceApi,
		apiCollection:  apiCollection,
		requestedBlock: requestedBlock,
		msg:            msg,
	}
	return nodeMsg
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
	serverApis, taggedApis, apiCollections := getServiceApis(spec, spectypes.APIInterfaceTendermintRPC)
	apip.BaseChainParser.Construct(spec, taggedApis, serverApis, apiCollections)
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
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData, apip.spec.BlocksInFinalizationProof
}

type TendermintRpcChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *metrics.RPCConsumerLogs
}

// NewTendermintRpcChainListener creates a new instance of TendermintRpcChainListener
func NewTendermintRpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *metrics.RPCConsumerLogs) (chainListener *TendermintRpcChainListener) {
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

			ctx, cancel := context.WithCancel(context.Background())
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			defer cancel() // incase there's a problem make sure to cancel the connection
			utils.LavaFormatInfo("ws in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: msg}, utils.Attribute{Key: "dappID", Value: dappID})

			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := apil.relaySender.SendRelay(ctx, "", string(msg), "", dappID, metricsData, nil)
			go apil.logger.AddMetricForWebSocket(metricsData, err, c)
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
	websocketCallbackWithDappID := constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, apil.logger.StoreMetricData)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://HOST:PORT/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()
		msgSeed := apil.logger.GetMessageSeed()
		dappID := extractDappIDFromFiberContext(c)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection

		utils.LavaFormatInfo("in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: c.Body()}, utils.Attribute{Key: "dappID", Value: dappID})
		reply, _, err := apil.relaySender.SendRelay(ctx, "", string(c.Body()), "", dappID, metricsData, nil)
		go apil.logger.AddMetricForHttp(metricsData, err, c.GetReqHeaders())

		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "POST", c.Request().URI().String(), string(c.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := rpcInterfaceMessages.ConvertToTendermintError(errMasking, c.Body())

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()

		query := "?" + string(c.Request().URI().QueryString())
		path := c.Params("*")
		dappID := extractDappIDFromFiberContext(c)
		msgSeed := apil.logger.GetMessageSeed()
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection
		utils.LavaFormatInfo("urirpc in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: path}, utils.Attribute{Key: "dappID", Value: dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := apil.relaySender.SendRelay(ctx, path+query, "", "", dappID, metricsData, nil)
		go apil.logger.AddMetricForHttp(metricsData, err, c.GetReqHeaders())

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
	ListenWithRetry(app, apil.endpoint.NetworkAddress)
}

type tendermintRpcChainProxy struct {
	// embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
	httpNodeUrl   common.NodeUrl
	httpConnector *chainproxy.Connector
}

func NewtendermintRpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, averageBlockTime time.Duration) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	websocketUrl, httpUrl := verifyTendermintEndpoint(rpcProviderEndpoint.NodeUrls)
	cp := &tendermintRpcChainProxy{
		JrpcChainProxy: JrpcChainProxy{BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: websocketUrl}, conn: map[string]*chainproxy.Connector{}},
		httpNodeUrl:    httpUrl,
		httpConnector:  nil,
	}
	cp.addHttpConnector(ctx, nConns, httpUrl)
	return cp, cp.start(ctx, nConns, websocketUrl, nil)
}

func (cp *tendermintRpcChainProxy) addHttpConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) error {
	conn, err := chainproxy.NewConnector(ctx, nConns, nodeUrl)
	if err != nil {
		return err
	}
	cp.httpConnector = conn
	if cp.httpConnector == nil {
		return errors.New("g_conn == nil")
	}
	return nil
}

func (cp *tendermintRpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(rpcInterfaceMessages.TendermintrpcMessage)
	if !ok {
		_, ok := rpcInputMessage.(*rpcInterfaceMessages.TendermintrpcMessage)
		return nil, "", nil, utils.LavaFormatError("invalid message type in tendermintrpc failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage}, utils.Attribute{Key: "ptrCast", Value: ok})
	}
	if nodeMessage.Path != "" {
		return cp.SendURI(ctx, &nodeMessage, ch, chainMessage)
	}

	// Else do RPC call
	return cp.SendRPC(ctx, &nodeMessage, ch, chainMessage)
}

func (cp *tendermintRpcChainProxy) SendURI(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// check if the input channel is not nil
	if ch != nil {
		// return an error if the channel is not nil
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on Tendermint URI", nil)
	}

	// create a new http client with a timeout set by the getTimePerCu function
	httpClient := http.Client{
		Timeout: common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits),
	}

	// construct the url by concatenating the node url with the path variable
	url := cp.httpNodeUrl.Url + "/" + nodeMessage.Path

	// create context
	relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits)
	// check if this API is hanging (waiting for block confirmation)
	if chainMessage.GetApi().Category.HangingApi {
		relayTimeout += cp.averageBlockTime
	}
	connectCtx, cancel := cp.httpNodeUrl.LowerContextTimeout(ctx, relayTimeout)
	defer cancel()

	// create a new http request
	req, err := http.NewRequestWithContext(connectCtx, http.MethodGet, cp.httpNodeUrl.AuthConfig.AddAuthPath(url), nil)
	if err != nil {
		return nil, "", nil, err
	}

	cp.httpNodeUrl.SetAuthHeaders(ctx, req.Header.Set)

	cp.httpNodeUrl.SetIpForwardingIfNecessary(ctx, req.Header.Set)
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
	body, err := io.ReadAll(res.Body)
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
func (cp *tendermintRpcChainProxy) SendRPC(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get rpc connection from the connection pool
	var rpc *rpcclient.Client
	if ch != nil {
		internalPath := chainMessage.GetApiCollection().CollectionData.InternalPath
		rpc, err = cp.conn[internalPath].GetRpc(ctx, true)
		if err != nil {
			return nil, "", nil, err
		}
		// return the rpc connection to the websocket pool after the function completes
		defer cp.conn[internalPath].ReturnRpc(rpc)
	} else {
		rpc, err = cp.httpConnector.GetRpc(ctx, true)
		if err != nil {
			return nil, "", nil, err
		}
		// return the rpc connection to the http pool after the function completes
		defer cp.httpConnector.ReturnRpc(rpc)
	}

	// create variables for the rpc message and reply message
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *rpcInterfaceMessages.RPCResponse
	var sub *rpcclient.ClientSubscription

	// If ch is not nil do subscription
	if ch != nil {
		// subscribe to the rpc call if the channel is not nil
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		// create a context with a timeout set by the LocalNodeTimePerCu function
		relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits)
		// check if this API is hanging (waiting for block confirmation)
		if chainMessage.GetApi().Category.HangingApi {
			relayTimeout += cp.averageBlockTime
		}
		cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)

		connectCtx, cancel := cp.NodeUrl.LowerContextTimeout(ctx, relayTimeout)
		defer cancel()
		// perform the rpc call
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params)
		if err != nil && connectCtx.Err() == context.DeadlineExceeded {
			// Not an rpc error, return provider error without disclosing the endpoint address
			return nil, "", nil, utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
		}
	}

	var replyMsg *rpcInterfaceMessages.RPCResponse
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		id, idErr := rpcInterfaceMessages.IdFromRawMessage(nodeMessage.ID)
		if idErr != nil {
			return nil, "", nil, utils.LavaFormatError("Failed parsing ID when getting rpc error", idErr)
		}
		replyMsg = &rpcInterfaceMessages.RPCResponse{
			JSONRPC: nodeMessage.Version,
			ID:      id,
			Error:   rpcInterfaceMessages.ConvertErrorToRPCError(err.Error(), -1), // TODO: extract code from error status / message
		}
	} else {
		replyMessage, err = rpcInterfaceMessages.ConvertTendermintMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermingRPC error", err)
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
			return nil, "", nil, utils.LavaFormatError("unknown params type on tendermint subscribe", nil)
		}
		subscriptionID, ok = paramsMap["query"].(string)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown subscriptionID type on tendermint subscribe", nil)
		}
	}

	return reply, subscriptionID, sub, err
}
