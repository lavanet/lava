package chainproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/relayer/sentry"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type TendemintRpcMessage struct {
	JrpcMessage
	cp *tendermintRpcChainProxy
}

type tendermintRpcChainProxy struct {
	//embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
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

func (cp *tendermintRpcChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	//
	// Unmarshal request
	var msg jsonrpcMessage
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

		msg = jsonrpcMessage{Method: parsedMethod} //other parameters don't matter
	}
	//
	// Check api is supported and save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(msg.Method)
	if err != nil {
		return nil, err
	}
	nodeMsg := &TendemintRpcMessage{
		JrpcMessage: JrpcMessage{serviceApi: serviceApi,
			msg: &msg},
		cp: cp,
	}
	return nodeMsg, nil
}

func (cp *tendermintRpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use("/:dappId/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/:dappId/ws", websocket.New(func(c *websocket.Conn) {
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

	app.Post("/:dappId/", func(c *fiber.Ctx) error {
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
	replyMsg := jsonrpcMessage{
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
