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
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/performance"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
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
	connectionType string
}

type RestChainProxy struct {
	nodeUrl    string
	sentry     *sentry.Sentry
	csm        *lavasession.ConsumerSessionManager
	portalLogs *PortalLogs
	cache      *performance.Cache
}

func (r *RestMessage) GetMsg() interface{} {
	return r.msg
}

func NewRestChainProxy(nodeUrl string, sentry *sentry.Sentry, csm *lavasession.ConsumerSessionManager, pLogs *PortalLogs) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &RestChainProxy{
		nodeUrl:    nodeUrl,
		sentry:     sentry,
		csm:        csm,
		portalLogs: pLogs,
	}
}

func (cp *RestChainProxy) GetConsumerSessionManager() *lavasession.ConsumerSessionManager {
	return cp.csm
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

func (m RestMessage) GetParams() interface{} {
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

func (cp *RestChainProxy) SetCache(cache *performance.Cache) {
	cp.cache = cache
}
func (cp *RestChainProxy) GetCache() *performance.Cache {
	return cp.cache
}

func (cp *RestChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum), nil, http.MethodGet)
	} else {
		nodeMsg, err = cp.NewMessage(serviceApi.Name, nil)
	}

	if err != nil {
		return "", err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return "", utils.LavaFormatError("Error On Send FetchBlockHashByNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
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
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []byte{}
	nodeMsg, err := cp.NewMessage(serviceApi.GetName(), params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	_, _, _, err = nodeMsg.Send(ctx, nil)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Error On Send FetchLatestBlockNum", err, &map[string]string{"nodeUrl": cp.nodeUrl})
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Failed To Parse FetchLatestBlockNum", err, &map[string]string{
			"nodeUrl":  cp.nodeUrl,
			"Method":   nodeMsg.path,
			"Response": string(nodeMsg.Result),
		})
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

func (cp *RestChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, err
	}
	// data contains the query string
	nodeMsg := &RestMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		path:           path,
		msg:            data,
		connectionType: connectionType, // POST,GET etc..
	}

	return nodeMsg, nil
}

func (cp *RestChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	// Catch Post
	app.Post("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("rest-http")

		path := "/" + c.Params("*")

		// TODO: handle contentType, in case its not application/json currently we set it to application/json in the Send() method
		// contentType := string(c.Context().Request.Header.ContentType())
		dappID := ExtractDappIDFromFiberContext(c)
		//TODO: fix msgSeed and print it here
		utils.LavaFormatInfo("in <<<", &map[string]string{"path": path, "dappID": dappID})
		requestBody := string(c.Body())
		reply, _, err := SendRelay(ctx, cp, privKey, path, requestBody, http.MethodPost, dappID)
		if err != nil {
			msgSeed := cp.portalLogs.GetUniqueGuidResponseForError(err)
			cp.portalLogs.LogRequestAndResponse("http in/out", true, http.MethodPost, path, requestBody, "", msgSeed, err)
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information:" %s}`, msgSeed))
		}
		responseBody := string(reply.Data)
		cp.portalLogs.LogRequestAndResponse("http in/out", false, http.MethodPost, path, requestBody, responseBody, "", nil)
		return c.SendString(responseBody)
	})

	//
	// Catch the others
	app.Use("/:dappId/*", func(c *fiber.Ctx) error {
		cp.portalLogs.LogStartTransaction("rest-http")

		query := "?" + string(c.Request().URI().QueryString())
		path := "/" + c.Params("*")
		dappID := ""
		if len(c.Route().Params) > 1 {
			dappID = c.Route().Params[1]
			dappID = strings.Replace(dappID, "*", "", -1)
		}
		utils.LavaFormatInfo("in <<<", &map[string]string{"path": path, "dappID": dappID})
		reply, _, err := SendRelay(ctx, cp, privKey, path, query, http.MethodGet, dappID)
		if err != nil {
			msgSeed := cp.portalLogs.GetUniqueGuidResponseForError(err)
			cp.portalLogs.LogRequestAndResponse("http in/out", true, http.MethodGet, path, "", "", msgSeed, err)
			return c.SendString(fmt.Sprintf(`{"error": "unsupported api","more_information": %s}`, msgSeed))
		}
		responseBody := string(reply.Data)
		cp.portalLogs.LogRequestAndResponse("http in/out", false, http.MethodGet, path, "", responseBody, "", nil)
		return c.SendString(responseBody)
	})
	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
}

func (nm *RestMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *RestMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *RestMessage) Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil, nil)
	}
	httpClient := http.Client{
		Timeout: time.Duration(nm.serviceApi.ComputeUnits * uint64(time.Second)), // Timeout after 5 seconds
	}

	var connectionTypeSlected string = http.MethodGet
	// if ConnectionType is default value or empty we will choose http.MethodGet otherwise choosing the header type provided
	if nm.connectionType != "" {
		connectionTypeSlected = nm.connectionType
	}

	msgBuffer := bytes.NewBuffer(nm.msg)
	url := nm.cp.nodeUrl + nm.path
	// Only get calls uses query params the rest uses the body
	if connectionTypeSlected == http.MethodGet {
		url += string(nm.msg)
	}
	req, err := http.NewRequest(connectionTypeSlected, url, msgBuffer)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	// setting the content-type to be application/json instead of Go's defult http.DefaultClient
	if connectionTypeSlected == "POST" || connectionTypeSlected == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := httpClient.Do(req)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data: body,
	}
	nm.Result = body

	return reply, "", nil, nil
}
