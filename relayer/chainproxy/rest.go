package chainproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestMessage struct {
	cp             *RestChainProxy
	serviceApi     *spectypes.ServiceApi
	path           string
	msg            []byte
	requestedBlock int64
	Result         json.RawMessage
}

type RestChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func (r *RestMessage) GetMsg() interface{} {
	return r.msg
}

func NewRestChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &RestChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *RestChainProxy) NewMessage(path string, data []byte) (*RestMessage, error) {
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

func (m RestMessage) GetParams() []interface{} {
	retArr := make([]interface{}, 0)
	retArr = append(retArr, m.msg)
	return retArr
}

func (m RestMessage) GetResult() json.RawMessage {
	return m.Result
}

func (m RestMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *RestChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum), nil)
	} else {
		nodeMsg, err = cp.NewMessage(serviceApi.Name, nil)
	}

	if err != nil {
		return "", err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return "", err
	}

	blockData, err := parser.ParseMessageResponse((nodeMsg.(*RestMessage)), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	return blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string), nil
}

func (cp *RestChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return parser.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []byte{}
	nodeMsg, err := cp.NewMessage(serviceApi.GetName(), params)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return parser.NOT_APPLICABLE, err
	}

	return blocknum, nil
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

func (nm *RestMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *RestMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *RestMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	httpClient := http.Client{
		Timeout: DefaultTimeout, // Timeout after 5 seconds
	}

	//
	// TODO: some APIs use POST!
	msgBuffer := bytes.NewBuffer(nm.msg)
	req, err := http.NewRequest(http.MethodGet, nm.cp.nodeUrl+nm.path, msgBuffer)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: body,
	}
	nm.Result = body
	return reply, nil
}
