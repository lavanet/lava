package chainproxy

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/relayer/sentry"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestMessage struct {
	cp         *RestChainProxy
	serviceApi *spectypes.ServiceApi
	path       string
	msg        []byte
}

type RestChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func NewRestChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &RestChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *RestChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *RestChainProxy) Start(context.Context) error {
	return nil
}

func (cp *RestChainProxy) getSupportedApi(path string) (*spectypes.ServiceApi, error) {
	path = strings.SplitN(path, "?", 2)[0]
	if api, ok := cp.sentry.MatchSpecApiByName(path); ok {
		if !api.Enabled {
			return nil, fmt.Errorf("REST Api is disabled %s ", path)
		}
		return &api, nil
	}
	return nil, fmt.Errorf("REST Api not supported %s ", path)
}

func (cp *RestChainProxy) ParseMsg(path string, data []byte) (NodeMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, err
	}
	nodeMsg := &RestMessage{
		cp:         cp,
		serviceApi: serviceApi,
		path:       path,
		msg:        data,
	}

	return nodeMsg, nil
}

func (cp *RestChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	//
	// Catch all
	app.Use("/:dappId/*", func(c *fiber.Ctx) error {
		path := "/" + c.Params("*")

		log.Println("in <<< ", path)
		reply, err := SendRelay(ctx, cp, privKey, path, "")
		if err != nil {
			log.Println(err)
			//
			// TODO: better errors
			return c.SendString(`{"error": "unsupported api"}`)
		}

		log.Println("out >>> len", len(string(reply.Data)))
		return c.SendString(string(reply.Data))
	})

	//
	// TODO: add POST support
	//app.Post("/", func(c *fiber.Ctx) error {})

	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
	return
}

func (nm *RestMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *RestMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	httpClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	//
	// TODO: some APIs use POST!
	msgBuffer := bytes.NewBuffer(nm.msg)
	req, err := http.NewRequest(http.MethodGet, nm.cp.nodeUrl+nm.path, msgBuffer)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: body,
	}
	return reply, nil
}
