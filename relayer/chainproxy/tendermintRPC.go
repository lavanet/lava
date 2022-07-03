package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
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

func (m TendemintRpcMessage) GetParams() []interface{} {
	return m.msg.Params
}

func (m TendemintRpcMessage) GetResult() json.RawMessage {
	return m.msg.Result
}

func (m TendemintRpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *tendermintRpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM) //TODO:: move to const
	if !ok {
		return parser.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []interface{}{}
	nodeMsg, err := cp.newMessage(&serviceApi, serviceApi.GetName(), parser.LATEST_BLOCK, params)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.GetMsg().(*JsonrpcMessage), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	return blocknum, nil
}

func (cp *tendermintRpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM) //TODO:: move to const
	if !ok {
		return "", errors.New(spectypes.GET_BLOCK_BY_NUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg("", []byte(fmt.Sprintf(serviceApi.Parsing.FunctionTemplate, blockNum)))
	} else {
		params := make([]interface{}, 0)
		params = append(params, blockNum)
		nodeMsg, err = cp.newMessage(&serviceApi, serviceApi.GetName(), parser.LATEST_BLOCK, params)
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

func (cp *tendermintRpcChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	//
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
		if strings.Contains(path[idx+1:], "=") {
			params_raw := strings.Split(path[idx+1:], "&") //list with structure ['height=0x500',...]
			msg.Params = make([]interface{}, len(params_raw))
			for i := range params_raw {
				msg.Params[i] = params_raw[i]
			}
		} else {
			msg.Params = make([]interface{}, 0)
		}
		//convert the list of strings to a list of interfaces
	}
	//
	// Check api is supported and save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(msg.Method)
	if err != nil {
		return nil, err
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

	app.Get("/ws/:dappId", websocket.New(func(c *websocket.Conn) {
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Println("ws: in <<< ", string(msg))

			reply, err := SendRelay(ctx, cp, privKey, "", string(msg))
			if err != nil {
				log.Println(err)
				break
			}

			if err = c.WriteMessage(mt, reply.Data); err != nil {
				log.Println("write:", err)
				break
			}
			log.Println("out >>> ", string(reply.Data))
		}
	}))

	app.Post("/:dappId", func(c *fiber.Ctx) error {
		log.Println("jsonrpc in <<< ", string(c.Body()))
		reply, err := SendRelay(ctx, cp, privKey, "", string(c.Body()))
		if err != nil {
			log.Println(err)
			return nil
		}

		log.Println("out >>> ", string(reply.Data))
		return c.SendString(string(reply.Data))
	})

	app.Use(func(c *fiber.Ctx) error {
		path := c.OriginalURL()
		if len(path) > 1 && path[0] == '/' {
			path = path[1:]
		}
		log.Println("urirpc in <<< ", string(path))
		reply, err := SendRelay(ctx, cp, privKey, path, "")
		if err != nil {
			log.Println(err)
			//
			// TODO: better errors
			return c.SendString(`{"error": "unsupported api"}`)
		}
		log.Println("out >>> ", string(reply.Data))
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
	err = rpc.CallContext(ctx, &result, nm.msg.Method, nm.msg.Params...)

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
	} else {
		replyMsg.Result = result
		nm.msg.Result = result
	}

	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, err
	}
	reply := &pairingtypes.RelayReply{
		Data: data,
	}
	return reply, nil
}
