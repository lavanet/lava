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

func (j *JrpcMessage) GetMsg() interface{} {
	return j.msg
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

func (cp *JrpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []interface{}{}
	nodeMsg, err := cp.NewMessage(&serviceApi, serviceApi.GetName(), spectypes.LATEST_BLOCK, params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg.msg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
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
		nodeMsg, err = cp.NewMessage(&serviceApi, serviceApi.GetName(), spectypes.LATEST_BLOCK, params)
	}

	if err != nil {
		return "", err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return "", err
	}
	// log.Println("%s", reply)

	blockData, err := parser.ParseMessageResponse((nodeMsg.GetMsg().(*JsonrpcMessage)), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	return blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string), nil
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

	return nil, errors.New("api not supported")
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

func (cp *JrpcChainProxy) NewMessage(serviceApi *spectypes.ServiceApi, method string, requestedBlock int64, params []interface{}) (*JrpcMessage, error) {
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
			ID:      []byte("1"), //TODO:: use ids
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
				utils.LavaFormatInfo("read error received", &map[string]string{"err": err.Error()})
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				break
			}
			utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(msg)})

			reply, err := SendRelay(ctx, cp, privKey, "", string(msg), "")
			if err != nil {
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				log.Println(err)
				break
			}

			if err = c.WriteMessage(mt, reply.Data); err != nil {
				c.WriteMessage(mt, []byte("Error Received: "+err.Error()))
				utils.LavaFormatInfo("write error received", &map[string]string{"err": err.Error()})
				break
			}
			utils.LavaFormatInfo("out >>>", &map[string]string{"seed": msgSeed, "reply": string(reply.Data)})
		}
	})

	app.Get("/ws/:dappId", webSocketCallback)
	app.Get("/:dappId/websocket", webSocketCallback) // catching http://ip:port/1/websocket requests.

	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		msgSeed := strconv.Itoa(rand.Intn(10000000000))
		utils.LavaFormatInfo("in <<<", &map[string]string{"seed": msgSeed, "msg": string(c.Body())})
		reply, err := SendRelay(ctx, cp, privKey, "", string(c.Body()), "")
		if err != nil {
			log.Println(err)
			return c.SendString(fmt.Sprintf(`{"error": {"code":-32000,"message":"%s"}}`, err.Error()))
		}

		utils.LavaFormatInfo("out >>>", &map[string]string{"seed": msgSeed, "reply": string(reply.Data)})
		return c.SendString(string(reply.Data))
	})

	//
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

// the error check here would only wrap errors not from the rpc
func (nm *JrpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	// Call our node
	var result JsonrpcMessage
	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	err = rpc.CallContext(connectCtx, &result, nm.msg.Method, nm.msg.Params)

	var replyMsg JsonrpcMessage
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
		return nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: data,
	}
	return reply, nil
}

// TODO Identify if API is tendermint or ethereum
func (nm *JrpcMessage) SendSubscribe(ctx context.Context, ch chan interface{}) (*rpcclient.ClientSubscription, *pairingtypes.RelayReply, error) {
	// Get node
	rpc, err := nm.cp.conn.GetRpc(true)
	if err != nil {
		return nil, nil, err
	}
	defer nm.cp.conn.ReturnRpc(rpc)

	// TODO Websockets for tendermint
	// subscribe is used for tendermint based chains
	// if nm.msg.Method == "subscribe" {
	// }

	// Need to do this since some networks (Fantom) have a subscribe method under a different namespace.
	method := strings.Split(nm.msg.Method, "_")
	if len(method) != 2 {
		return nil, nil, utils.LavaFormatError("Invalid Method "+nm.msg.Method, nil, nil)
	}
	namespace := method[0]

	var result JsonrpcMessage
	sub, err := rpc.Subscribe(context.Background(), &result, namespace, ch, nm.msg.Params...)

	var replyMsg JsonrpcMessage
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
		return nil, nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: data,
	}

	return sub, reply, err

}
