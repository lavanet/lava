package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type JsonrpcMessage struct {
	Version string               `json:"jsonrpc,omitempty"`
	ID      json.RawMessage      `json:"id,omitempty"`
	Method  string               `json:"method,omitempty"`
	Params  interface{}          `json:"params,omitempty"`
	Error   *rpcclient.JsonError `json:"error,omitempty"`
	Result  json.RawMessage      `json:"result,omitempty"`
}

type JrpcMessage struct {
	cp             *JrpcChainProxy
	serviceApi     *spectypes.ServiceApi
	msg            *JsonrpcMessage
	requestedBlock int64
}

func (j *JrpcMessage) GetMsg() interface{} {
	return j.msg
}

func (j *JrpcMessage) setMessageResult(result json.RawMessage) {
	j.msg.Result = result
}

func convertMsg(rpcMsg *rpcclient.JsonrpcMessage) (*JsonrpcMessage, error) {
	// Return an error if the message was not sent
	if rpcMsg == nil {
		return nil, ErrFailedToConvertMessage
	}

	msg := &JsonrpcMessage{
		Version: rpcMsg.Version,
		ID:      rpcMsg.ID,
		Method:  rpcMsg.Method,
		Params:  rpcMsg.Params,
		Error:   rpcMsg.Error,
		Result:  rpcMsg.Result,
	}

	return msg, nil
}

type JrpcChainProxy struct {
	conn       *Connector
	nConns     uint
	nodeUrl    string
	sentry     *sentry.Sentry
	csm        *lavasession.ConsumerSessionManager
	portalLogs *PortalLogs
	cache      *performance.Cache
}

func NewJrpcChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry, csm *lavasession.ConsumerSessionManager, pLogs *PortalLogs) ChainProxy {
	return &JrpcChainProxy{
		nodeUrl:    nodeUrl,
		nConns:     nConns,
		sentry:     sentry,
		csm:        csm,
		portalLogs: pLogs,
		cache:      nil,
	}
}

func (cp *JrpcChainProxy) SetCache(cache *performance.Cache) {
	cp.cache = cache
}

func (cp *JrpcChainProxy) GetCache() *performance.Cache {
	return cp.cache
}

func (cp *JrpcChainProxy) GetConsumerSessionManager() *lavasession.ConsumerSessionManager {
	return cp.csm
}

func (cp *JrpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []interface{}{}
	nodeMsg, err := cp.NewMessage(&serviceApi, spectypes.LATEST_BLOCK, params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Error On Send FetchLatestBlockNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.msg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Failed To Parse FetchLatestBlockNum", err, &map[string]string{
			"nodeUrl":  cp.nodeUrl,
			"Method":   nodeMsg.msg.Method,
			"Response": string(nodeMsg.msg.Result),
		})
	}

	return blocknum, nil
}

func (cp *JrpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCK_BY_NUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg("", []byte(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum)), "")
	} else {
		params := make([]interface{}, 0)
		params = append(params, blockNum)
		nodeMsg, err = cp.NewMessage(&serviceApi, spectypes.LATEST_BLOCK, params)
	}

	if err != nil {
		return "", err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return "", utils.LavaFormatError("Error On Send FetchBlockHashByNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}
	// log.Println("%s", reply)
	msgParsed, ok := nodeMsg.GetMsg().(*JsonrpcMessage)
	if !ok {
		return "", fmt.Errorf("FetchBlockHashByNum - nodeMsg.GetMsg().(*JsonrpcMessage) - type assertion failed, type:" + fmt.Sprintf("%s", nodeMsg.GetMsg()))
	}
	blockData, err := parser.ParseMessageResponse(msgParsed, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	parsedIndexString, ok := blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", fmt.Errorf("FetchBlockHashByNum - blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string) - type assertion failed, type:" + fmt.Sprintf("%s", blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX]))
	}
	return parsedIndexString, nil
}

func (cp JsonrpcMessage) GetParams() interface{} {
	return cp.Params
}

func (cp JsonrpcMessage) GetResult() json.RawMessage {
	return cp.Result
}

func (cp JsonrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *JrpcChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *JrpcChainProxy) Start(ctx context.Context) error {
	cp.conn = NewConnector(ctx, cp.nConns, cp.nodeUrl)
	if cp.conn == nil {
		return errors.New("g_conn == nil")
	}

	return nil
}

func (cp *JrpcChainProxy) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	if api, ok := cp.sentry.GetSpecApiByName(name); ok {
		if !api.Enabled {
			return nil, errors.New("api is disabled")
		}
		return &api, nil
	}

	return nil, errors.New("JRPC api not supported")
}

func (cp *JrpcChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	// connectionType is currently only used in rest API.
	// Unmarshal request
	var msg JsonrpcMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	//
	// Check api is supported and save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(msg.Method)
	if err != nil {
		return nil, utils.LavaFormatError("getSupportedApi failed", err, &map[string]string{"method": msg.Method})
	}
	requestedBlock, err := parser.ParseBlockFromParams(msg, serviceApi.BlockParsing)
	if err != nil {
		return nil, err
	}
	nodeMsg := &JrpcMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		msg:            &msg,
		requestedBlock: requestedBlock,
	}
	return nodeMsg, nil
}

func (cp *JrpcChainProxy) NewMessage(serviceApi *spectypes.ServiceApi, requestedBlock int64, params []interface{}) (*JrpcMessage, error) {
	method := serviceApi.GetName()
	serviceApi, err := cp.getSupportedApi(method)
	if err != nil {
		return nil, err
	}

	nodeMsg := &JrpcMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		requestedBlock: requestedBlock,
		msg: &JsonrpcMessage{
			Version: "2.0",
			ID:      []byte("1"), // TODO:: use ids
			Method:  method,
			Params:  params,
		},
	}
	return nodeMsg, nil
}

func (cp *JrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	app.Use("/ws/:dappId", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("jsonRpc-WebSocket")

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
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), "", "", msgSeed, err)
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				break
			}
			utils.LavaFormatInfo("ws in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // incase there's a problem make sure to cancel the connection
			dappID := ExtractDappIDFromWebsocketConnection(c)
			reply, replyServer, err := SendRelay(ctx, cp, privKey, "", string(msg), "", dappID)
			if err != nil {
				cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				continue
			}

			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}
				cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						// break
					}

					cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}
				cp.portalLogs.LogRequestAndResponse("jsonrpc ws msg", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
			}
		}
	})
	websocketCallbackWithDappID := ConstructFiberCallbackWithDappIDExtraction(webSocketCallback)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("jsonRpc-http post")
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		dappID := ExtractDappIDFromFiberContext(c)
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body()), "dappID": dappID})
		reply, _, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), "", dappID)
		if err != nil {
			msgSeed := cp.portalLogs.GetUniqueGuidResponseForError(err)
			cp.portalLogs.LogRequestAndResponse("jsonrpc http", true, "POST", c.Request().URI().String(), string(c.Body()), "", msgSeed, err)
			return c.SendString(fmt.Sprintf(`{"error": {"code":-32000,"message":"%s"}}`, err))
		}
		cp.portalLogs.LogRequestAndResponse("jsonrpc http", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), "", nil)
		return c.SendString(string(reply.Data))
	})

	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
}

func (nm *JrpcMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *JrpcMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *JrpcMessage) Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, "", nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	// Call our node
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if ch != nil {
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nm.msg.ID, nm.msg.Method, ch, nm.msg.Params)
	} else {
		connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
		rpcMessage, err = rpc.CallContext(connectCtx, nm.msg.ID, nm.msg.Method, nm.msg.Params)
	}

	var replyMsg JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		replyMsg = JsonrpcMessage{
			Version: nm.msg.Version,
			ID:      nm.msg.ID,
		}
		replyMsg.Error = &rpcclient.JsonError{
			Code:    1,
			Message: fmt.Sprintf("%s", err),
		}
		// this later causes returning an error
	} else {
		replyMessage, err = convertMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC error", err, nil)
		}

		nm.msg = replyMessage
		replyMsg = *replyMessage
	}

	data, err := json.Marshal(replyMsg)
	if err != nil {
		nm.msg.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: data,
	}

	if ch != nil {
		subscriptionID, err = strconv.Unquote(string(replyMsg.Result))
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("Subscription failed", err, nil)
		}
	}
	if replyMsg.Error != nil {
		return reply, "", nil, utils.LavaFormatError(replyMsg.Error.Message, nil, nil)
	}

	return reply, subscriptionID, sub, err
}
