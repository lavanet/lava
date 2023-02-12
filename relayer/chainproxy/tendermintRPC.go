package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/lavanet/lava/relayer/metrics"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tenderminttypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

type TendemintRpcMessage struct {
	JrpcMessage
	cp   *tendermintRpcChainProxy
	path string
}

type tendermintRpcChainProxy struct {
	// embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
}

func (m TendemintRpcMessage) GetParams() interface{} {
	return m.msg.Params
}

func (m TendemintRpcMessage) GetResult() json.RawMessage {
	return m.msg.Result
}

func (m TendemintRpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *tendermintRpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []interface{}{}
	nodeMsg, err := cp.newMessage(&serviceApi, spectypes.LATEST_BLOCK, params, http.MethodGet)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Error On Send FetchLatestBlockNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	msgParsed, ok := nodeMsg.GetMsg().(*JsonrpcMessage)
	if !ok {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("FetchLatestBlockNum - nodeMsg.GetMsg().(*JsonrpcMessage) - type assertion failed, type:" + fmt.Sprintf("%s", nodeMsg.GetMsg()))
	}
	blocknum, err := parser.ParseBlockFromReply(msgParsed, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	return blocknum, nil
}

func (cp *tendermintRpcChainProxy) GetConsumerSessionManager() *lavasession.ConsumerSessionManager {
	return cp.csm
}

func (cp *tendermintRpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCK_BY_NUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg("", []byte(fmt.Sprintf(serviceApi.Parsing.FunctionTemplate, blockNum)), http.MethodGet)
	} else {
		params := make([]interface{}, 0)
		params = append(params, blockNum)
		nodeMsg, err = cp.newMessage(&serviceApi, spectypes.LATEST_BLOCK, params, http.MethodGet)
	}

	if err != nil {
		return "", err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return "", utils.LavaFormatError("Error On Send FetchBlockHashByNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	msg, ok := nodeMsg.GetMsg().(*JsonrpcMessage)
	if !ok {
		return "", fmt.Errorf("FetchBlockHashByNum - nodeMsg.GetMsg().(*JsonrpcMessage) - type assertion failed, type:" + fmt.Sprintf("%s", nodeMsg.GetMsg()))
	}
	blockData, err := parser.ParseMessageResponse(msg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", utils.LavaFormatError("Failed To Parse FetchLatestBlockNum", err, &map[string]string{
			"nodeUrl":  cp.nodeUrl,
			"Method":   msg.Method,
			"Response": string(msg.Result),
		})
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	hash, ok := blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", errors.New("hash not string parsable")
	}

	return hash, nil
}

func NewtendermintRpcChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry, csm *lavasession.ConsumerSessionManager, pLogs *PortalLogs) ChainProxy {
	return &tendermintRpcChainProxy{
		JrpcChainProxy: JrpcChainProxy{
			nodeUrl:    nodeUrl,
			nConns:     nConns,
			sentry:     sentry,
			portalLogs: pLogs,
			csm:        csm,
		},
	}
}

func (cp *tendermintRpcChainProxy) newMessage(serviceApi *spectypes.ServiceApi, requestedBlock int64, params []interface{}, connectionType string) (*TendemintRpcMessage, error) {
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

	nodeMsg := &TendemintRpcMessage{
		JrpcMessage: JrpcMessage{
			serviceApi:   serviceApi,
			apiInterface: apiInterface,
			msg: &JsonrpcMessage{
				Version: "2.0",
				ID:      []byte("1"), // TODO:: use ids
				Method:  serviceApi.GetName(),
				Params:  params,
			},
			requestedBlock: requestedBlock,
		},
		cp: cp,
	}
	return nodeMsg, nil
}

func (cp *tendermintRpcChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	// connectionType is currently only used in rest api
	// Unmarshal request
	var msg JsonrpcMessage
	if string(data) != "" {
		// assuming jsonrpc
		err := json.Unmarshal(data, &msg)
		if err != nil {
			return nil, err
		}
	} else {
		// assuming URI
		var parsedMethod string
		idx := strings.Index(path, "?")
		if idx == -1 {
			parsedMethod = path
		} else {
			parsedMethod = path[0:idx]
		}

		msg = JsonrpcMessage{
			ID:      []byte("1"),
			Version: "2.0",
			Method:  parsedMethod,
		}
		if strings.Contains(path[idx+1:], "=") {
			params := make(map[string]interface{})
			rawParams := strings.Split(path[idx+1:], "&") // list with structure ['height=0x500',...]
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
	serviceApi, err := cp.getSupportedApi(msg.Method)
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
	if apiInterface.GetOverwriteBlockParsing() != nil && path != "" {
		blockParser = *apiInterface.GetOverwriteBlockParsing()
	}

	// Fetch requested block, it is used for data reliability
	requestedBlock, err := parser.ParseBlockFromParams(msg, blockParser)
	if err != nil {
		return nil, err
	}

	var extraTimeout time.Duration
	if apiInterface.Category.HangingApi {
		extraTimeout = time.Duration(cp.sentry.GetAverageBlockTime()) * time.Millisecond
	}

	nodeMsg := &TendemintRpcMessage{
		JrpcMessage: JrpcMessage{
			serviceApi:           serviceApi,
			apiInterface:         apiInterface,
			msg:                  &msg,
			requestedBlock:       requestedBlock,
			extendContextTimeout: extraTimeout,
		},
		path: path,
		cp:   cp,
	}
	return nodeMsg, nil
}

func (cp *tendermintRpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})
	chainID := cp.GetSentry().ChainID
	apiInterface := cp.GetSentry().ApiInterface

	app.Use(favicon.New())

	app.Use("/ws/:dappId", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("tendermint-WebSocket")
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
		msgSeed := cp.portalLogs.GetMessageSeed()
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				break
			}
			dappID := ExtractDappIDFromWebsocketConnection(c)
			utils.LavaFormatInfo("ws in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg), "dappID": dappID})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // incase there's a problem make sure to cancel the connection
			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := SendRelay(ctx, cp, privKey, "", string(msg), http.MethodGet, dappID, metricsData)
			go cp.portalLogs.AddMetric(metricsData, err != nil)
			if err != nil {
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						// break
					}
					cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
			}
		}
	})
	websocketCallbackWithDappID := ConstructFiberCallbackWithDappIDExtraction(webSocketCallback)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("tendermint-WebSocket")
		msgSeed := cp.portalLogs.GetMessageSeed()
		dappID := ExtractDappIDFromFiberContext(c)
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body()), "dappID": dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), http.MethodGet, dappID, metricsData)
		go cp.portalLogs.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := cp.portalLogs.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			cp.portalLogs.LogRequestAndResponse("tendermint http in/out", true, "POST", c.Request().URI().String(), string(c.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json (tendermint) response
			response := convertToTendermintError(errMasking, c.Body())

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		cp.portalLogs.LogRequestAndResponse("tendermint http in/out", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("tendermint-WebSocket")

		query := "?" + string(c.Request().URI().QueryString())
		path := c.Params("*")
		dappID := ExtractDappIDFromFiberContext(c)
		msgSeed := cp.portalLogs.GetMessageSeed()
		utils.LavaFormatInfo("urirpc in <<<", &map[string]string{"seed": msgSeed, "msg": path, "dappID": dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := SendRelay(ctx, cp, privKey, path+query, "", http.MethodGet, dappID, metricsData)
		go cp.portalLogs.AddMetric(metricsData, err != nil)
		if err != nil {
			// Get unique GUID response
			errMasking := cp.portalLogs.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			cp.portalLogs.LogRequestAndResponse("tendermint http in/out", true, "GET", c.Request().URI().String(), "", errMasking, msgSeed, err)

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
		cp.portalLogs.LogRequestAndResponse("tendermint http in/out", false, "GET", c.Request().URI().String(), "", string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})
	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		utils.LavaFormatError("app.Listen(listenAddr)", err, nil)
	}
}

func convertToTendermintError(errString string, inputInfo []byte) string {
	var msg JsonrpcMessage
	err := json.Unmarshal(inputInfo, &msg)
	if err == nil {
		id, errId := idFromRawMessage(msg.ID)
		if errId != nil {
			utils.LavaFormatError("error idFromRawMessage", errId, nil)
			return InternalErrorString
		}
		res, merr := json.Marshal(&RPCResponse{
			JSONRPC: msg.Version,
			ID:      id,
			Error:   convertErrorToRPCError(errString, LavaErrorCode),
		})
		if merr != nil {
			utils.LavaFormatError("convertToTendermintError json.Marshal", merr, nil)
			return InternalErrorString
		}
		return string(res)
	}
	utils.LavaFormatError("error convertToTendermintError", err, nil)
	return InternalErrorString
}

func getTendermintRPCError(jsonError *rpcclient.JsonError) (*tenderminttypes.RPCError, error) {
	var rpcError *tenderminttypes.RPCError
	if jsonError != nil {
		errData, ok := (jsonError.Data).(string)
		if !ok {
			return nil, utils.LavaFormatError("(rpcMsg.Error.Data).(string) conversion failed", nil, &map[string]string{"data": fmt.Sprintf("%v", jsonError.Data)})
		}
		rpcError = &tenderminttypes.RPCError{
			Code:    jsonError.Code,
			Message: jsonError.Message,
			Data:    errData,
		}
	}
	return rpcError, nil
}

func convertErrorToRPCError(errString string, code int) *tenderminttypes.RPCError {
	var rpcError *tenderminttypes.RPCError
	unmarshalError := json.Unmarshal([]byte(errString), &rpcError)
	if unmarshalError != nil {
		utils.LavaFormatWarning("Failed unmarshalling error tendermintrpc", unmarshalError, &map[string]string{"err": errString})
		rpcError = &tenderminttypes.RPCError{
			Code:    code,
			Message: "Rpc Error",
			Data:    errString,
		}
	}
	return rpcError
}

type jsonrpcId interface {
	isJSONRPCID()
}

// JSONRPCStringID a wrapper for JSON-RPC string IDs
type JSONRPCStringID string

func (JSONRPCStringID) isJSONRPCID()      {}
func (id JSONRPCStringID) String() string { return string(id) }

// JSONRPCIntID a wrapper for JSON-RPC integer IDs
type JSONRPCIntID int

func (JSONRPCIntID) isJSONRPCID()      {}
func (id JSONRPCIntID) String() string { return fmt.Sprintf("%d", id) }

func idFromRawMessage(rawID json.RawMessage) (jsonrpcId, error) {
	var idInterface interface{}
	err := json.Unmarshal(rawID, &idInterface)
	if err != nil {
		return nil, utils.LavaFormatError("failed to unmarshal id from response", err, &map[string]string{"id": fmt.Sprintf("%v", rawID)})
	}

	switch id := idInterface.(type) {
	case string:
		return JSONRPCStringID(id), nil
	case float64:
		// json.Unmarshal uses float64 for all numbers
		return JSONRPCIntID(int(id)), nil
	default:
		typ := reflect.TypeOf(id)
		return nil, utils.LavaFormatError("failed to unmarshal id not a string or float", err, &map[string]string{"id": fmt.Sprintf("%v", rawID), "id type": fmt.Sprintf("%v", typ)})
	}
}

type RPCResponse struct {
	JSONRPC string                    `json:"jsonrpc"`
	ID      jsonrpcId                 `json:"id,omitempty"`
	Result  json.RawMessage           `json:"result,omitempty"`
	Error   *tenderminttypes.RPCError `json:"error,omitempty"`
}

func convertTendermintMsg(rpcMsg *rpcclient.JsonrpcMessage) (*RPCResponse, error) {
	// Return an error if the message was not sent
	if rpcMsg == nil {
		return nil, ErrFailedToConvertMessage
	}
	rpcError, err := getTendermintRPCError(rpcMsg.Error)
	if err != nil {
		return nil, err
	}

	jsonid, err := idFromRawMessage(rpcMsg.ID)
	if err != nil {
		return nil, err
	}
	msg := &RPCResponse{
		JSONRPC: rpcMsg.Version,
		ID:      jsonid,
		Result:  rpcMsg.Result,
		Error:   rpcError,
	}

	return msg, nil
}

// Send sends either Tendermint RPC or URI call depending on the type
func (nm *TendemintRpcMessage) Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// If path exists then the call is URI
	if nm.path != "" {
		return nm.SendURI(ctx, ch)
	}

	// Else do RPC call
	return nm.SendRPC(ctx, ch)
}

// SendURI sends URI HTTP call
func (nm *TendemintRpcMessage) SendURI(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// check if the input channel is not nil
	if ch != nil {
		// return an error if the channel is not nil
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on Tendermint URI", nil, nil)
	}

	// create a new http client with a timeout set by the getTimePerCu function
	httpClient := http.Client{
		Timeout: getTimePerCu(nm.serviceApi.ComputeUnits),
	}

	// construct the url by concatenating the node url with the path variable
	url := nm.cp.nodeUrl + "/" + nm.path

	// create a new http request
	connectCtx, cancel := context.WithTimeout(ctx, getTimePerCu(nm.serviceApi.ComputeUnits)+nm.GetExtraContextTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(connectCtx, http.MethodGet, url, nil)
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
func (nm *TendemintRpcMessage) SendRPC(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get rpc connection from the connection pool
	rpc, err := nm.cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}

	// return the rpc connection to the pool after the function completes
	defer nm.cp.conn.ReturnRpc(rpc)

	// create variables for the rpc message and reply message
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *RPCResponse
	var sub *rpcclient.ClientSubscription

	// If ch is not nil do subscription
	if ch != nil {
		// subscribe to the rpc call if the channel is not nil
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nm.msg.ID, nm.msg.Method, ch, nm.msg.Params)
	} else {
		// create a context with a timeout set by the getTimePerCu function
		connectCtx, cancel := context.WithTimeout(ctx, getTimePerCu(nm.serviceApi.ComputeUnits)+nm.GetExtraContextTimeout())
		defer cancel()
		// perform the rpc call
		rpcMessage, err = rpc.CallContext(connectCtx, nm.msg.ID, nm.msg.Method, nm.msg.Params)
	}

	var replyMsg *RPCResponse
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			// Not an rpc error, return provider error without disclosing the endpoint address
			return nil, "", nil, utils.LavaFormatError("Failed Sending Message", context.DeadlineExceeded, nil)
		}
		id, idErr := idFromRawMessage(nm.msg.ID)
		if idErr != nil {
			return nil, "", nil, utils.LavaFormatError("Failed parsing ID when getting rpc error", idErr, nil)
		}
		replyMsg = &RPCResponse{
			JSONRPC: nm.msg.Version,
			ID:      id,
			Error:   convertErrorToRPCError(err.Error(), -1), // TODO: fetch error code from err.
		}
	} else {
		replyMessage, err = convertTendermintMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermingRPC error", err, nil)
		}

		replyMsg = replyMessage
		nm.msg.Result = replyMessage.Result
	}

	// marshal the jsonrpc message to json
	data, err := json.Marshal(replyMsg)
	if err != nil {
		nm.msg.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	// create a new relay reply struct
	reply := &pairingtypes.RelayReply{
		Data: data,
	}

	if ch != nil {
		// get the params for the rpc call
		params := nm.msg.Params

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
