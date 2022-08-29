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

	_, err = nodeMsg.Send(ctx)
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

	_, err = nodeMsg.Send(ctx)
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
				utils.LavaFormatInfo("error read message ws", &map[string]string{"err": err.Error()})
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				break
			}
			utils.LavaFormatInfo(" ws: in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			reply, err := SendRelay(ctx, cp, privKey, "", string(msg), "")
			if err != nil {
				utils.LavaFormatInfo("error send relay ws", &map[string]string{"err": err.Error()})
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				break
			}

			if err = c.WriteMessage(mt, reply.Data); err != nil {
				log.Println("write:", err)
				utils.LavaFormatInfo("error write message ws", &map[string]string{"err": err.Error()})
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				break
			}
			utils.LavaFormatInfo("jsonrpc out <<<", &map[string]string{"seed": msgSeed, "msg": string(reply.Data)})
		}
	})

	app.Get("/ws/:dappId", webSocketCallback)
	app.Get("/:dappId/websocket", webSocketCallback) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("jsonrpc in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body())})
		reply, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), "")
		if err != nil {
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information" %s}`, err))
		}

		utils.LavaFormatInfo("jsonrpc out <<<", &map[string]string{"seed": msgSeed, "msg": string(reply.Data)})
		return c.SendString(string(reply.Data))
	})

	app.Get("/:dappId/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("urirpc in <<<", &map[string]string{"seed": msgSeed, "msg": path})
		reply, err := SendRelay(ctx, cp, privKey, path, "", "")
		if err != nil {
			log.Println(err)
			if string(c.Body()) != "" {
				return c.SendString(fmt.Sprintf(`{"error": "unsupported api", "recommendation": "For jsonRPC use POST", "more_information": "%s"}`, err))
			}
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information" %s}`, err))
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

func (nm *TendemintRpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	//
	// Call our node
	var result json.RawMessage
	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	err = rpc.CallContext(connectCtx, &result, nm.msg.Method, nm.msg.Params)

	//
	// Wrap result back to json
	replyMsg := JsonrpcMessage{
		Version: nm.msg.Version,
		ID:      nm.msg.ID,
	}
	if err != nil {
		//
		// TODO: CallContext is limited, it does not give us the source
		// of the error or the error code if json (we need smarter error handling)
		replyMsg.Error = &jsonError{
			Code:    1, // TODO
			Message: fmt.Sprintf("%s", err),
		}
		nm.msg.Result = []byte(fmt.Sprintf("%s", err))
		return nil, err
	} else {
		replyMsg.Result = result
		nm.msg.Result = result
	}

	data, err := json.Marshal(replyMsg)
	if err != nil {
		nm.msg.Result = []byte(fmt.Sprintf("%s", err))
		return nil, err
	}
	reply := &pairingtypes.RelayReply{
		Data: data,
	}
	return reply, nil
}
