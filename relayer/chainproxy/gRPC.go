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
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
)

type GrpcMessage struct {
	cp             *GrpcChainProxy
	serviceApi     *spectypes.ServiceApi
	path           string
	msg            interface{}
	requestedBlock int64
	connectionType string
	Result         json.RawMessage
}

type GrpcChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func (r *GrpcMessage) GetMsg() interface{} {
	return r.msg
}

func NewGrpcChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &GrpcChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *GrpcChainProxy) NewMessage(path string, data []byte) (*GrpcMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, err
	}
	var d interface{}
	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal gRPC NewMessage %s", err)
	}

	nodeMsg := &GrpcMessage{
		cp:         cp,
		serviceApi: serviceApi,
		path:       path,
		msg:        d,
	}

	return nodeMsg, nil
}

func (m GrpcMessage) GetParams() interface{} {
	return m.msg
}

func (m GrpcMessage) GetResult() json.RawMessage {
	return m.Result
}

func (m GrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *GrpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg(serviceApi.Name, []byte(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum)), "")
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

	blockData, err := parser.ParseMessageResponse((nodeMsg.(*GrpcMessage)), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	return blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string), nil
}

func (cp *GrpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := []byte{}
	nodeMsg, err := cp.NewMessage(serviceApi.GetName(), params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("new Message creation Failed at FetchLatestBlockNum", err, nil)
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Message send Failed at FetchLatestBlockNum", err, nil)
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	return blocknum, nil
}

func (cp *GrpcChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *GrpcChainProxy) Start(context.Context) error {
	return nil
}

func (cp *GrpcChainProxy) getSupportedApi(path string) (*spectypes.ServiceApi, error) {
	if api, ok := cp.sentry.MatchSpecApiByName(path); ok {
		if !api.Enabled {
			return nil, fmt.Errorf("gRPC Api is disabled %s ", path)
		}
		return &api, nil
	}
	return nil, fmt.Errorf("gRPC Api not supported %s ", path)
}

func (cp *GrpcChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err, nil)
	}

	var d interface{}
	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, utils.LavaFormatError("failed to unmarshal gRPC ParseMsg", err, nil)
	}

	nodeMsg := &GrpcMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		path:           path,
		msg:            d,
		connectionType: connectionType,
	}

	return nodeMsg, nil
}

func (cp *GrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// Setup Server
	app := fiber.New(fiber.Config{})

	// Catch all
	app.Use("/:dappId/*", func(c *fiber.Ctx) error {
		path := "/" + c.Params("*")
		log.Println("in <<< ", path)
		return c.SendString("got message")
	})
	//
	// Go
	err := app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
	return
}

func (nm *GrpcMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *GrpcMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func (nm *GrpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {

	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Todo, maybe create a list of grpc clients in connector.go instead of creating one and closing it each time?
	conn, err := grpc.DialContext(connectCtx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}
	defer conn.Close()
	var result json.RawMessage
	err = conn.Invoke(connectCtx, nm.path, nm.msg, &result)

	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, utils.LavaFormatError("Sending the gRPC message failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
	}
	nm.Result = result

	reply := &pairingtypes.RelayReply{
		Data: result,
	}
	nm.Result = result
	return reply, nil
}
