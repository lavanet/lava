package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  []interface{}   `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type JrpcMessage struct {
	cp             *JrpcChainProxy
	serviceApi     *spectypes.ServiceApi
	msg            *JsonrpcMessage
	requestedBlock int64
}

func (j *JrpcMessage) setMessageResult(result json.RawMessage) {
	j.msg.Result = result
}

type JrpcChainProxy struct {
	conn    *Connector
	nConns  uint
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewJrpcChainProxy(nodeUrl string, nConns uint, sentry *sentry.Sentry) ChainProxy {
	return &JrpcChainProxy{
		nodeUrl: nodeUrl,
		nConns:  nConns,
		sentry:  sentry,
	}
}

type rpcInputReply struct {
	Data json.RawMessage
}

func (r *rpcInputReply) GetParams() []interface{}               { return nil }
func (r *rpcInputReply) GetResult() json.RawMessage             { return r.Data }
func (r *rpcInputReply) ParseBlock(block string) (int64, error) { return 0, nil }

func (cp *JrpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag("getBlockNumber") //TODO:: move to const
	if !ok {
		return -1, errors.New("getBlockNumber tag function not found")
	}

	params := []interface{}{}
	nodeMsg := cp.NewMessage(&serviceApi, serviceApi.GetName(), 0, params)

	reply, err := nodeMsg.Send(ctx)
	if err != nil {
		return -1, err
	}
	log.Println("%s", reply)

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.msg, serviceApi.ResultParsing)
	if err != nil {
		return -1, err
	}

	return blocknum, nil
}
func (cp *JrpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag("getBlockByNumber") //TODO:: move to const
	if !ok {
		return "", errors.New("getBlockNumber tag function not found")
	}

	params := []interface{}{}
	if serviceApi.GetFunctionParams().Type == "array" {
		retArr := make([]interface{}, 0)
		for _, paramConfig := range serviceApi.GetFunctionParams().Params {

			switch paramConfig.Type {
			case "printformat":
				retArr = append(retArr, fmt.Sprintf(paramConfig.Template, blockNum))
			case "plain":
				retArr = append(retArr, paramConfig.Value)
			}

		}
		params = retArr
	}

	nodeMsg := cp.NewMessage(&serviceApi, serviceApi.GetName(), 0, params)

	reply, err := nodeMsg.Send(ctx)
	if err != nil {
		return "", err
	}
	log.Println("%s", reply)

	blockData, err := parser.ParseMessageResponse(nodeMsg.msg, serviceApi.ResultParsing)
	if err != nil {
		return "", err
	}

	return blockData[0].(string), nil
}

func (cp JsonrpcMessage) GetParams() []interface{} {
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

	return nil, errors.New("api not supported")
}

func (cp *JrpcChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	//
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
		return nil, err
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

func (cp *JrpcChainProxy) ParseMsgResponse(nodeMsg NodeMessage) (NodeMessage, error) {
	// err := parser.ParseResponse(nodeMsg)
	// if err != nil {
	// 	return nil, err
	// }
	//
	//
	// msg.GetServiceApi().

	// nodeMsg := &JrpcMessage{
	// 	cp:             cp,
	// 	serviceApi:     serviceApi,
	// 	msg:            &msg,
	// 	requestedBlock: requestedBlock,
	// }
	return nil, nil
}

func (cp *JrpcChainProxy) NewMessage(serviceApi *spectypes.ServiceApi, method string, requestedBlock int64, params []interface{}) JrpcMessage {
	nodeMsg := &JrpcMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		requestedBlock: requestedBlock,
		msg: &JsonrpcMessage{
			Version: "2.0",
			ID:      []byte("1"), //TODO:: use ids
			Method:  method,
			Params:  params,
		},
	}
	return *nodeMsg
}

func (cp *JrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
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
			log.Println("in <<< ", string(msg))

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

	app.Post("/", func(c *fiber.Ctx) error {
		log.Println("in <<< ", string(c.Body()))
		reply, err := SendRelay(ctx, cp, privKey, "", string(c.Body()))
		if err != nil {
			log.Println(err)
			return nil
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
	return
}

func (nm *JrpcMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *JrpcMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *JrpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	//
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
