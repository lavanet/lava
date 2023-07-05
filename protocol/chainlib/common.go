package chainlib

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
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

type TaggedContainer struct {
	Parsing       *spectypes.ParseDirective
	ApiCollection *spectypes.ApiCollection
}

type ApiContainer struct {
	api           *spectypes.Api
	collectionKey CollectionKey
}

type BaseChainParser struct {
	taggedApis     map[spectypes.FUNCTION_TAG]TaggedContainer
	spec           spectypes.Spec
	rwLock         sync.RWMutex
	serverApis     map[ApiKey]ApiContainer
	apiCollections map[CollectionKey]*spectypes.ApiCollection
	headers        map[ApiKey]*spectypes.Header
}

func (bcp *BaseChainParser) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) (filteredHeaders []pairingtypes.Metadata, overwriteRequestedBlock string, ignoredMetadata []pairingtypes.Metadata) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()
	if len(metadata) == 0 {
		return []pairingtypes.Metadata{}, "", []pairingtypes.Metadata{}
	}
	retMeatadata := []pairingtypes.Metadata{}
	for _, header := range metadata {
		headerName := strings.ToLower(header.Name)
		apiKey := ApiKey{Name: headerName, ConnectionType: apiCollection.CollectionData.Type}
		headerDirective, ok := bcp.headers[apiKey]
		if !ok {
			// this header is not handled
			continue
		}
		if headerDirective.Kind == headersDirection || headerDirective.Kind == spectypes.Header_pass_both {
			retMeatadata = append(retMeatadata, header)
			if headerDirective.FunctionTag == spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA {
				// this header sets the latest requested block
				overwriteRequestedBlock = header.Value
			}
		} else if headerDirective.Kind == spectypes.Header_pass_ignore {
			ignoredMetadata = append(ignoredMetadata, header)
		}
	}
	utils.LavaFormatDebug("Headers filtering", utils.Attribute{Key: "received", Value: metadata}, utils.Attribute{Key: "filtered", Value: retMeatadata})

	return retMeatadata, overwriteRequestedBlock, ignoredMetadata
}

func (bcp *BaseChainParser) Construct(spec spectypes.Spec, taggedApis map[spectypes.FUNCTION_TAG]TaggedContainer, serverApis map[ApiKey]ApiContainer, apiCollections map[CollectionKey]*spectypes.ApiCollection, headers map[ApiKey]*spectypes.Header) {
	bcp.spec = spec
	bcp.serverApis = serverApis
	bcp.taggedApis = taggedApis
	bcp.headers = headers
	bcp.apiCollections = apiCollections
}

func (bcp *BaseChainParser) GetParsingByTag(tag spectypes.FUNCTION_TAG) (parsing *spectypes.ParseDirective, collectionData *spectypes.CollectionData, existed bool) {
	bcp.rwLock.RLock()
	defer bcp.rwLock.RUnlock()

	val, ok := bcp.taggedApis[tag]
	if !ok {
		return nil, nil, false
	}
	return val.Parsing, &val.ApiCollection.CollectionData, ok
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getSupportedApi(name string, connectionType string) (*ApiContainer, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	apiCont, ok := apip.serverApis[ApiKey{
		Name:           name,
		ConnectionType: connectionType,
	}]

	// Return an error if spec does not exist
	if !ok {
		return nil, utils.LavaFormatError("api not supported", nil, utils.Attribute{Key: "name", Value: name}, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	// Return an error if api is disabled
	if !apiCont.api.Enabled {
		return nil, utils.LavaFormatError("api is disabled", nil, utils.Attribute{Key: "name", Value: name}, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	return &apiCont, nil
}

// getSupportedApi fetches service api from spec by name
func (apip *BaseChainParser) getApiCollection(connectionType string, internalPath string, addon string) (*spectypes.ApiCollection, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := apip.apiCollections[CollectionKey{
		ConnectionType: connectionType,
		InternalPath:   internalPath,
		Addon:          addon,
	}]

	// Return an error if spec does not exist
	if !ok {
		return nil, utils.LavaFormatError("api not supported", nil, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, utils.LavaFormatError("api is disabled", nil, utils.Attribute{Key: "connectionType", Value: connectionType})
	}

	return api, nil
}

type updatableRPCInput interface {
	parser.RPCInput
	UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool)
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
	ErrorHandler     ErrorHandler
	averageBlockTime time.Duration
	NodeUrl          common.NodeUrl
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

func getServiceApis(spec spectypes.Spec, rpcInterface string) (retServerApis map[ApiKey]ApiContainer, retTaggedApis map[spectypes.FUNCTION_TAG]TaggedContainer, retApiCollections map[CollectionKey]*spectypes.ApiCollection, retHeaders map[ApiKey]*spectypes.Header) {
	serverApis := map[ApiKey]ApiContainer{}
	taggedApis := map[spectypes.FUNCTION_TAG]TaggedContainer{}
	headers := map[ApiKey]*spectypes.Header{}
	apiCollections := map[CollectionKey]*spectypes.ApiCollection{}
	if spec.Enabled {
		for _, apiCollection := range spec.ApiCollections {
			if !apiCollection.Enabled {
				continue
			}
			if apiCollection.CollectionData.ApiInterface != rpcInterface {
				continue
			}
			collectionKey := CollectionKey{ConnectionType: apiCollection.CollectionData.Type}
			for _, parsing := range apiCollection.ParseDirectives {
				taggedApis[parsing.FunctionTag] = TaggedContainer{
					Parsing:       parsing,
					ApiCollection: apiCollection,
				}
			}

			for _, api := range apiCollection.Apis {
				if !api.Enabled {
					continue
				}
				//
				// TODO: find a better spot for this (more optimized, precompile regex, etc)
				if rpcInterface == spectypes.APIInterfaceRest {
					re := regexp.MustCompile(`{[^}]+}`)
					processedName := string(re.ReplaceAll([]byte(api.Name), []byte("replace-me-with-regex")))
					processedName = regexp.QuoteMeta(processedName)
					processedName = strings.ReplaceAll(processedName, "replace-me-with-regex", `[^\/\s]+`)
					serverApis[ApiKey{
						Name:           processedName,
						ConnectionType: collectionKey.ConnectionType,
					}] = ApiContainer{
						api:           api,
						collectionKey: collectionKey,
					}
				} else {
					serverApis[ApiKey{
						Name:           api.Name,
						ConnectionType: collectionKey.ConnectionType,
					}] = ApiContainer{
						api:           api,
						collectionKey: collectionKey,
					}
				}
			}
			for _, header := range apiCollection.Headers {
				headers[ApiKey{
					Name:           header.Name,
					ConnectionType: collectionKey.ConnectionType,
				}] = header
			}
			apiCollections[collectionKey] = apiCollection
		}
	}
	return serverApis, taggedApis, apiCollections, headers
}

// matchSpecApiByName returns service api which match given name
func matchSpecApiByName(name string, connectionType string, serverApis map[ApiKey]ApiContainer) (*ApiContainer, bool) {
	// TODO: make it faster and better by not doing a regex instead using a better algorithm
	for apiName, api := range serverApis {
		re, err := regexp.Compile("^" + apiName.Name + "$")
		if err != nil {
			utils.LavaFormatError("regex Compile api", err, utils.Attribute{Key: "apiName", Value: apiName})
			continue
		}
		if re.MatchString(name) && apiName.ConnectionType == connectionType {
			return &api, true
		}
	}
	return nil, false
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

func CraftChainMessage(parsing *spectypes.ParseDirective, connectionType string, chainParser ChainParser, craftData *CraftData) (ChainMessageForSend, error) {
	return chainParser.CraftMessage(parsing, connectionType, craftData)
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
