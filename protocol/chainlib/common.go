package chainlib

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	common "github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/metadata"
)

const (
	ContextUserValueKeyDappID = "dappID"
	RetryListeningInterval    = 10 // seconds
	debug                     = false
)

type VerificationKey struct {
	Extension string
	Addon     string
}

type VerificationContainer struct {
	ConnectionType string
	Name           string
	ParseDirective spectypes.ParseDirective
	Value          string
	VerificationKey
}

type TaggedContainer struct {
	Parsing       *spectypes.ParseDirective
	ApiCollection *spectypes.ApiCollection
}

type ApiContainer struct {
	api           *spectypes.Api
	collectionKey CollectionKey
}

type updatableRPCInput interface {
	parser.RPCInput
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
	AppendHeader(metadata []pairingtypes.Metadata)
}

type parsedMessage struct {
	api            *spectypes.Api
	requestedBlock int64
	msg            updatableRPCInput
	apiCollection  *spectypes.ApiCollection
}

type ApiKey struct {
	Name           string
	ConnectionType string
}

type CollectionKey struct {
	ConnectionType string
	InternalPath   string
	Addon          string
}

type BaseChainProxy struct {
	ErrorHandler
	averageBlockTime time.Duration
	NodeUrl          common.NodeUrl
}

func (pm parsedMessage) AppendHeader(metadata []pairingtypes.Metadata) {
	pm.msg.AppendHeader(metadata)
}

func (pm parsedMessage) GetApi() *spectypes.Api {
	return pm.api
}

func (pm parsedMessage) GetApiCollection() *spectypes.ApiCollection {
	return pm.apiCollection
}

func (pm parsedMessage) RequestedBlock() int64 {
	return pm.requestedBlock
}

func (pm parsedMessage) GetRPCMessage() parser.RPCInput {
	return pm.msg
}

func (pm *parsedMessage) UpdateLatestBlockInMessage(latestBlock int64, modifyContent bool) (modifiedOnLatestReq bool) {
	if latestBlock <= spectypes.NOT_APPLICABLE || pm.RequestedBlock() != spectypes.LATEST_BLOCK {
		return false
	}
	success := pm.msg.UpdateLatestBlockInMessage(uint64(latestBlock), modifyContent)
	if success {
		pm.requestedBlock = latestBlock
		return true
	}
	return false
}

func extractDappIDFromFiberContext(c *fiber.Ctx) (dappID string) {
	dappID = c.Params("dappId")
	if dappID == "" {
		dappID = "NoDappID"
	}
	return dappID
}

func constructFiberCallbackWithHeaderAndParameterExtraction(callbackToBeCalled fiber.Handler, isMetricEnabled bool) fiber.Handler {
	webSocketCallback := callbackToBeCalled
	handler := func(c *fiber.Ctx) error {
		if isMetricEnabled {
			c.Locals(metrics.RefererHeaderKey, c.Get(metrics.RefererHeaderKey, ""))
		}
		return webSocketCallback(c) // uses external dappID
	}
	return handler
}

func extractDappIDFromWebsocketConnection(c *websocket.Conn) string {
	dappId := c.Params("dappId")
	if dappId == "" {
		dappId = "NoDappID"
	}
	return dappId
}

func convertToJsonError(errorMsg string) string {
	jsonResponse, err := json.Marshal(fiber.Map{
		"error": errorMsg,
	})
	if err != nil {
		return `{"error": "Failed to marshal error response to json"}`
	}

	return string(jsonResponse)
}

func addAttributeToError(key string, value string, errorMessage string) string {
	return errorMessage + fmt.Sprintf(`, "%v": "%v"`, key, value)
}

// rpc default endpoint should be websocket. otherwise return an error
func verifyRPCEndpoint(endpoint string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		utils.LavaFormatFatal("unparsable url", err, utils.Attribute{Key: "url", Value: endpoint})
	}
	switch u.Scheme {
	case "ws", "wss":
		return
	default:
		utils.LavaFormatWarning("URL scheme should be websocket (ws/wss), got: "+u.Scheme, nil)
	}
}

// rpc default endpoint should be websocket. otherwise return an error
func verifyTendermintEndpoint(endpoints []common.NodeUrl) (websocketEndpoint common.NodeUrl, httpEndpoint common.NodeUrl) {
	for _, endpoint := range endpoints {
		u, err := url.Parse(endpoint.Url)
		if err != nil {
			utils.LavaFormatFatal("unparsable url", err, utils.Attribute{Key: "url", Value: endpoint.Url})
		}
		switch u.Scheme {
		case "http", "https":
			httpEndpoint = endpoint
		case "ws", "wss":
			websocketEndpoint = endpoint
		default:
			utils.LavaFormatFatal("URL scheme should be websocket (ws/wss) or (http/https), got: "+u.Scheme, nil)
		}
	}

	if websocketEndpoint.String() == "" || httpEndpoint.String() == "" {
		utils.LavaFormatError("Tendermint Provider was not provided with both http and websocket urls. please provide both", nil,
			utils.Attribute{Key: "websocket", Value: websocketEndpoint.String()}, utils.Attribute{Key: "http", Value: httpEndpoint.String()})
		if httpEndpoint.String() != "" {
			return httpEndpoint, httpEndpoint
		} else {
			utils.LavaFormatFatal("Tendermint Provider was not provided with http url. please provide a url that starts with http/https", nil)
		}
	}
	return websocketEndpoint, httpEndpoint
}

func ListenWithRetry(app *fiber.App, address string) {
	for {
		err := app.Listen(address)
		if err != nil {
			utils.LavaFormatError("app.Listen(listenAddr)", err)
		}
		time.Sleep(RetryListeningInterval * time.Second)
	}
}

func GetListenerWithRetryGrpc(protocol string, addr string) net.Listener {
	for {
		lis, err := net.Listen(protocol, addr)
		if err == nil {
			return lis
		}
		utils.LavaFormatError("failure setting up listener, net.Listen(protocol, addr)", err, utils.Attribute{Key: "listenAddr", Value: addr})
		time.Sleep(RetryListeningInterval * time.Second)
		utils.LavaFormatWarning("Attempting connection retry", nil)
	}
}

type CraftData struct {
	Path           string
	Data           []byte
	ConnectionType string
}

func CraftChainMessage(parsing *spectypes.ParseDirective, connectionType string, chainParser ChainParser, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	return chainParser.CraftMessage(parsing, connectionType, craftData, metadata)
}

// rest request headers are formatted like map[string]string
func convertToMetadataMap(md map[string]string) []pairingtypes.Metadata {
	metadata := make([]pairingtypes.Metadata, len(md))
	indexer := 0
	for k, v := range md {
		metadata[indexer] = pairingtypes.Metadata{Name: k, Value: v}
		indexer += 1
	}
	return metadata
}

// rest response headers / grpc headers are formatted like map[string][]string
func convertToMetadataMapOfSlices(md map[string][]string) []pairingtypes.Metadata {
	metadata := make([]pairingtypes.Metadata, len(md))
	indexer := 0
	for k, v := range md {
		metadata[indexer] = pairingtypes.Metadata{Name: k, Value: v[0]}
		indexer += 1
	}
	return metadata
}

func convertRelayMetaDataToMDMetaData(md []pairingtypes.Metadata) metadata.MD {
	responseMetaData := make(metadata.MD)
	for _, v := range md {
		responseMetaData[v.Name] = append(responseMetaData[v.Name], v.Value)
	}
	return responseMetaData
}
