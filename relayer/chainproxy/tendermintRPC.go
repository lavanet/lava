package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendemintRpcMessage struct {
	JrpcMessage
	cp *tendermintRpcChainProxy
}

type tendermintRpcChainProxy struct {
	//embedding the jrpc chain proxy because the only diff is on parse message
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
	nodeMsg, err := cp.newMessage(&serviceApi, serviceApi.GetName(), spectypes.LATEST_BLOCK, params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Error On Send FetchLatestBlockNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.GetMsg().(*JsonrpcMessage), serviceApi.Parsing.ResultParsing)
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
		nodeMsg, err = cp.ParseMsg("", []byte(fmt.Sprintf(serviceApi.Parsing.FunctionTemplate, blockNum)), "")
	} else {
		params := make([]interface{}, 0)
		params = append(params, blockNum)
		nodeMsg, err = cp.newMessage(&serviceApi, serviceApi.GetName(), spectypes.LATEST_BLOCK, params)
	}

	if err != nil {
		return "", err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return "", utils.LavaFormatError("Error On Send FetchBlockHashByNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	msg := (nodeMsg.GetMsg().(*JsonrpcMessage))
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

func (cp *tendermintRpcChainProxy) newMessage(serviceApi *spectypes.ServiceApi, method string, requestedBlock int64, params []interface{}) (*TendemintRpcMessage, error) {
	nodeMsg := &TendemintRpcMessage{
		JrpcMessage: JrpcMessage{serviceApi: serviceApi,
			msg: &JsonrpcMessage{
				Version: "2.0",
				ID:      []byte("1"), //TODO:: use ids
				Method:  method,
				Params:  params,
			},
			requestedBlock: requestedBlock},
		cp: cp,
	}
	return nodeMsg, nil
}

func (cp *tendermintRpcChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	// connectionType is currently only used only in rest api
	// Unmarshal request
	var msg JsonrpcMessage
	if string(data) != "" {
		//assuming jsonrpc
		err := json.Unmarshal(data, &msg)
		if err != nil {
			return nil, err
		}

	} else {
		//assuming URI
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
		} //other parameters don't matter
		// TODO: will be easier to parse the params in a map instead of an array, as calling with a map should be now supported
		if strings.Contains(path[idx+1:], "=") {
			params_raw := strings.Split(path[idx+1:], "&") //list with structure ['height=0x500',...]
			params := make([]interface{}, len(params_raw))
			for i := range params_raw {
				params[i] = params_raw[i]
			}
			msg.Params = params
		} else {
			msg.Params = make([]interface{}, 0)
		}
		//convert the list of strings to a list of interfaces
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

	nodeMsg := &TendemintRpcMessage{
		JrpcMessage: JrpcMessage{serviceApi: serviceApi,
			msg: &msg, requestedBlock: requestedBlock},
		cp: cp,
	}
	return nodeMsg, nil
}

func (cp *tendermintRpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

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
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), "", "", msgSeed, err)
				break
			}
			utils.LavaFormatInfo("ws in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() //incase there's a problem make sure to cancel the connection
			dappID := ExtractDappIDFromWebsocketConnection(c)
			reply, replyServer, err := SendRelay(ctx, cp, privKey, "", string(msg), http.MethodGet, dappID)
			if err != nil {
				cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
				cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				continue
			}

			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) //this reply contains the RPC ID
				if err != nil {
					cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}
				cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
						cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						// break
					}

					cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					cp.portalLogs.LogRequestAndResponse("tendermint ws", true, "ws", c.LocalAddr().String(), string(msg), "", msgSeed, err)
					cp.portalLogs.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}
				cp.portalLogs.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), "", nil)
			}
		}
	})
	websocketCallbackWithDappID := ConstructFiberCallbackWithDappIDExtraction(webSocketCallback)
	app.Get("/ws/:dappId", websocketCallbackWithDappID)
	app.Get("/:dappId/websocket", websocketCallbackWithDappID) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("tendermint-WebSocket")
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		dappID := ExtractDappIDFromFiberContext(c)
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body()), "dappID": dappID})
		reply, _, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), http.MethodGet, dappID)
		if err != nil {
			msgSeed := cp.portalLogs.GetUniqueGuidResponseForError(err)
			cp.portalLogs.LogRequestAndResponse("tendermint http in/out", true, "POST", c.Request().URI().String(), string(c.Body()), "", msgSeed, err)
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information": %s}`, msgSeed))
		}
		cp.portalLogs.LogRequestAndResponse("tendermint http in/out", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), msgSeed, nil)
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("tendermint-WebSocket")
		path := c.Params("*")
		dappID := ""
		if len(c.Route().Params) > 1 {
			dappID = c.Route().Params[1]
			dappID = strings.Replace(dappID, "*", "", -1)
		}
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("urirpc in <<<", &map[string]string{"seed": msgSeed, "msg": path, "dappID": dappID})
		reply, _, err := SendRelay(ctx, cp, privKey, path, "", http.MethodGet, dappID)
		if err != nil {
			msgSeed := cp.portalLogs.GetUniqueGuidResponseForError(err)
			cp.portalLogs.LogRequestAndResponse("tendermint http in/out", true, "GET", c.Request().URI().String(), "", "", msgSeed, err)
			if string(c.Body()) != "" {
				return c.SendString(fmt.Sprintf(`{"error": "unsupported api", "recommendation": "For jsonRPC use POST", "more_information": "%s"}`, msgSeed))
			}
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information": %s}`, msgSeed))
		}
		cp.portalLogs.LogRequestAndResponse("tendermint http in/out", false, "GET", c.Request().URI().String(), "", string(reply.Data), "", nil)
		return c.SendString(string(reply.Data))
	})
	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
}

func (nm *TendemintRpcMessage) Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, "", nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	params := nm.msg.Params

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
	} else {
		replyMessage, err = convertMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermingRPC error", err, nil)
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
		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown params type on tendermint subscribe", nil, nil)
		}
		subscriptionID, ok = paramsMap["query"].(string)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown subscriptionID type on tendermint subscribe", nil, nil)
		}
	}
	if replyMsg.Error != nil {
		return reply, "", nil, utils.LavaFormatError(replyMsg.Error.Message, nil, nil)
	}

	return reply, subscriptionID, sub, err
}
