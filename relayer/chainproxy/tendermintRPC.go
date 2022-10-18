package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
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
		return spectypes.NOT_APPLICABLE, err
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.GetMsg().(*JsonrpcMessage), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	return blocknum, nil
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
		return "", err
	}

	blockData, err := parser.ParseMessageResponse((nodeMsg.GetMsg().(*JsonrpcMessage)), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	hash, ok := blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", errors.New("hash not string parseable")
	}

	return hash, nil
}

func NewtendermintRpcChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry) ChainProxy {
	return &tendermintRpcChainProxy{
		JrpcChainProxy: JrpcChainProxy{
			nodeUrl: nodeUrl,
			nConns:  nConns,
			sentry:  sentry,
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
				AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				break
			}
			utils.LavaFormatInfo("ws: in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			// identify if the message is a subscription
			nodeMsg, err := cp.ParseMsg("", msg, "")
			if err != nil {
				AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
				continue
			}

			if nodeMsg.GetServiceApi().Category.Subscription {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel() //incase there's a problem make sure to cancel the connection
				replySrv, err := SendRelaySubscribe(ctx, cp, privKey, "", string(msg), "")
				if err != nil {
					AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}

				var reply pairingtypes.RelayReply
				err = (*replySrv).RecvMsg(&reply) //this reply contains the RPC ID
				if err != nil {
					AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					continue
				}

				utils.LavaFormatInfo("ws out >>>", &map[string]string{"seed": msgSeed, "reply": string(reply.Data)})

				for {
					err = (*replySrv).RecvMsg(&reply)
					if err != nil {
						AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
						// break
					}

					utils.LavaFormatInfo("ws out >>>", &map[string]string{"seed": msgSeed, "reply": string(reply.Data)})
				}
			} else {
				reply, err := SendRelay(ctx, cp, privKey, "", string(msg), "")
				if err != nil {
					AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					break
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed)
					break
				}
				utils.LavaFormatInfo("jsonrpc out <<<", &map[string]string{"seed": msgSeed, "msg": string(reply.Data)})
			}

		}
	})

	app.Get("/ws/:dappId", webSocketCallback)
	app.Get("/:dappId/websocket", webSocketCallback) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("http in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body())})
		reply, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), "")
		if err != nil {
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information" %s}`, GetUniqueGuidResponseForError(err)))
		}
		utils.LavaFormatInfo("http out <<<", &map[string]string{"seed": msgSeed, "msg": string(reply.Data)})
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("urirpc in <<<", &map[string]string{"seed": msgSeed, "msg": path})
		reply, err := SendRelay(ctx, cp, privKey, path, "", "")
		if err != nil {
			if string(c.Body()) != "" {
				return c.SendString(fmt.Sprintf(`{"error": "unsupported api", "recommendation": "For jsonRPC use POST", "more_information": "%s"}`, GetUniqueGuidResponseForError(err)))
			}
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information" %s}`, GetUniqueGuidResponseForError(err)))
		}
		utils.LavaFormatInfo("urirpc out <<<", &map[string]string{"seed": msgSeed, "msg": string(reply.Data)})
		return c.SendString(string(reply.Data))
	})
	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
}

func (nm *TendemintRpcMessage) Send(ctx context.Context, ch chan interface{}) (*pairingtypes.RelayReply, string, *rpcclient.ClientSubscription, error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, "", nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	params := nm.msg.Params

	// Call our node
	var result JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if ch != nil {
		sub, err = rpc.Subscribe(context.Background(), nm.msg.ID, &result, nm.msg.Method, ch, nm.msg.Params)
	} else {
		connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
		err = rpc.CallContext(connectCtx, nm.msg.ID, &result, nm.msg.Method, nm.msg.Params)
	}

	var replyMsg JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		replyMsg = JsonrpcMessage{
			Version: nm.msg.Version,
			ID:      nm.msg.ID,
		}
		replyMsg.Error = &jsonError{
			Code:    1,
			Message: fmt.Sprintf("%s", err),
		}
	} else {
		nm.msg = &result
		replyMsg = result
	}

	data, err := json.Marshal(replyMsg)
	if err != nil {
		nm.msg.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: data,
	}

	var subscriptionID string
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
