package chainlib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/metrics"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type RestChainParser struct {
	BaseChainParser
}

// NewRestChainParser creates a new instance of RestChainParser
func NewRestChainParser() (chainParser *RestChainParser, err error) {
	return &RestChainParser{}, nil
}

func (apip *RestChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		// chain fetcher sends the replaced request inside data
		chainMessage, err := apip.ParseMsg(string(craftData.Data), nil, craftData.ConnectionType, metadata, 0)
		chainMessage.AppendHeader(metadata)
		return chainMessage, err
	}

	restMessage := &rpcInterfaceMessages.RestMessage{
		Msg:         nil,
		Path:        parsing.ApiName,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata},
	}

	apiCont, err := apip.getSupportedApi(parsing.ApiName, connectionType)
	if err != nil {
		return nil, err
	}
	api := apiCont.api
	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}
	return apip.newChainMessage(api, spectypes.NOT_APPLICABLE, restMessage, apiCollection), nil
}

// ParseMsg parses message data into chain message object
func (apip *RestChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, latestBlock uint64) (ChainMessage, error) {
	// Guard that the RestChainParser instance exists
	if apip == nil {
		return nil, errors.New("RestChainParser not defined")
	}

	// Check api is supported and save it in nodeMsg
	apiCont, err := apip.getSupportedApi(url, connectionType)
	if err != nil {
		return nil, err
	}

	// Extract default block parser
	blockParser := apiCont.api.BlockParsing

	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}
	metadata, overwriteReqBlock, _ := apip.HandleHeaders(metadata, apiCollection, spectypes.Header_pass_send)

	settingHeaderDirective, _, _ := apip.GetParsingByTag(spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA)
	// Construct restMessage
	restMessage := rpcInterfaceMessages.RestMessage{
		Msg:         data,
		Path:        url,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata},
	}
	if connectionType == http.MethodGet {
		// support for optional params, our listener puts them inside Msg data
		restMessage = rpcInterfaceMessages.RestMessage{
			Msg:         nil,
			Path:        url + string(data),
			BaseMessage: chainproxy.BaseMessage{Headers: metadata, LatestBlockHeaderSetter: settingHeaderDirective},
		}
	}
	// add spec path to rest message so we can extract the requested block.
	restMessage.SpecPath = apiCont.api.Name
	var requestedBlock int64
	if overwriteReqBlock == "" {
		// Fetch requested block, it is used for data reliability
		requestedBlock, err = parser.ParseBlockFromParams(restMessage, blockParser)
		if err != nil {
			return nil, utils.LavaFormatError("ParseBlockFromParams failed parsing block", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "blockParsing", Value: apiCont.api.BlockParsing})
		}
	} else {
		requestedBlock, err = restMessage.ParseBlock(overwriteReqBlock)
		if err != nil {
			return nil, utils.LavaFormatError("failed parsing block from an overwrite header", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "overwriteReqBlock", Value: overwriteReqBlock})
		}
	}

	nodeMsg := apip.newChainMessage(apiCont.api, requestedBlock, &restMessage, apiCollection)
	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, latestBlock)
	return nodeMsg, nil
}

func (*RestChainParser) newChainMessage(serviceApi *spectypes.Api, requestBlock int64, restMessage *rpcInterfaceMessages.RestMessage, apiCollection *spectypes.ApiCollection) *parsedMessage {
	nodeMsg := &parsedMessage{
		api:                  serviceApi,
		apiCollection:        apiCollection,
		msg:                  restMessage,
		latestRequestedBlock: requestBlock,
	}
	return nodeMsg
}

func (apip *RestChainParser) getApiCollection(connectionType, internalPath, addon string) (*spectypes.ApiCollection, error) {
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getApiCollection(connectionType, internalPath, addon)
}

// overwrites the base class match for a supported api
func (apip *RestChainParser) getSupportedApi(name, connectionType string) (*ApiContainer, error) {
	// Guard that the RestChainParser instance exists
	if apip == nil {
		return nil, errors.New("RestChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server apiCont by name
	apiCont, ok := matchSpecApiByName(name, connectionType, apip.serverApis)

	// Return an error if spec does not exist
	if !ok {
		return nil, errors.New("rest api not supported " + name)
	}
	api := apiCont.api

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
	}

	return apiCont, nil
}

// SetSpec sets the spec for the TendermintChainParser
func (apip *RestChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceRest)
	apip.BaseChainParser.Construct(spec, taggedApis, serverApis, apiCollections, headers, verifications)
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *RestChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the RestChainParser instance exists
	if apip == nil {
		return false, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Return enabled and data reliability threshold from spec
	return apip.spec.DataReliabilityEnabled, apip.spec.GetReliabilityThreshold()
}

// ChainBlockStats returns block stats from spec
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData)
func (apip *RestChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return 0, 0, 0, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Convert average block time from int64 -> time.Duration
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// Return values
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData, apip.spec.BlocksInFinalizationProof
}

type RestChainListener struct {
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *metrics.RPCConsumerLogs
}

// NewRestChainListener creates a new instance of RestChainListener
func NewRestChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *metrics.RPCConsumerLogs) (chainListener *RestChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &RestChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

// Serve http server for RestChainListener
func (apil *RestChainListener) Serve(ctx context.Context) {
	// Guard that the RestChainListener instance exists
	if apil == nil {
		return
	}

	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use(favicon.New())

	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface
	// Catch Post
	app.Post("/*", func(c *fiber.Ctx) error {
		// Set response header content-type to application/json
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		endTx := apil.logger.LogStartTransaction("rest-http")
		defer endTx()

		msgSeed := apil.logger.GetMessageSeed()

		path := "/" + c.Params("*")

		metadataValues := c.GetReqHeaders()
		restHeaders := convertToMetadataMap(metadataValues)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection

		// TODO: handle contentType, in case its not application/json currently we set it to application/json in the Send() method
		// contentType := string(c.Context().Request.Header.ContentType())
		dappID := extractDappIDFromFiberContext(c)
		analytics := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		utils.LavaFormatInfo("in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "path", Value: path}, utils.Attribute{Key: "dappID", Value: dappID}, utils.Attribute{Key: "msgSeed", Value: msgSeed})
		requestBody := string(c.Body())
		reply, _, err := apil.relaySender.SendRelay(ctx, path, requestBody, http.MethodPost, dappID, analytics, restHeaders)
		go apil.logger.AddMetricForHttp(analytics, err, c.GetReqHeaders())

		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, http.MethodPost, path, requestBody, errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return addHeadersAndSendString(c, reply.GetMetadata(), response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodPost, path, requestBody, string(reply.Data), msgSeed, nil)

		// Return json response
		return addHeadersAndSendString(c, reply.GetMetadata(), string(reply.Data))
	})

	// Catch the others
	app.Use("/*", func(c *fiber.Ctx) error {
		// Set response header content-type to application/json
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		endTx := apil.logger.LogStartTransaction("rest-http")
		defer endTx()
		msgSeed := apil.logger.GetMessageSeed()

		query := "?" + string(c.Request().URI().QueryString())
		path := "/" + c.Params("*")
		dappID := extractDappIDFromFiberContext(c)
		analytics := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)

		metadataValues := c.GetReqHeaders()
		restHeaders := convertToMetadataMap(metadataValues)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection
		utils.LavaFormatInfo("in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "path", Value: path}, utils.Attribute{Key: "dappID", Value: dappID}, utils.Attribute{Key: "msgSeed", Value: msgSeed})

		reply, _, err := apil.relaySender.SendRelay(ctx, path, query, http.MethodGet, dappID, analytics, restHeaders)
		go apil.logger.AddMetricForHttp(analytics, err, c.GetReqHeaders())
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, http.MethodGet, path, "", errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return addHeadersAndSendString(c, reply.GetMetadata(), response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodGet, path, "", string(reply.Data), msgSeed, nil)

		// Return json response
		return addHeadersAndSendString(c, reply.GetMetadata(), string(reply.Data))
	})

	// Go
	ListenWithRetry(app, apil.endpoint.NetworkAddress)
}

func addHeadersAndSendString(c *fiber.Ctx, metaData []pairingtypes.Metadata, data string) error {
	for _, value := range metaData {
		c.Set(value.Name, value.Value)
	}

	return c.SendString(data)
}

type RestChainProxy struct {
	BaseChainProxy
}

func NewRestChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	nodeUrl.Url = strings.TrimSuffix(rpcProviderEndpoint.NodeUrls[0].Url, "/")
	rcp := &RestChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: rpcProviderEndpoint.NodeUrls[0], ErrorHandler: &RestErrorHandler{}},
	}
	return rcp, nil
}

func (rcp *RestChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil)
	}
	httpClient := http.Client{
		Timeout: common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits),
	}

	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.RestMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in rest, failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage})
	}
	var connectionTypeSlected string = http.MethodGet
	// if ConnectionType is default value or empty we will choose http.MethodGet otherwise choosing the header type provided
	if chainMessage.GetApiCollection().CollectionData.Type != "" {
		connectionTypeSlected = chainMessage.GetApiCollection().CollectionData.Type
	}

	msgBuffer := bytes.NewBuffer(nodeMessage.Msg)
	url := rcp.NodeUrl.Url + nodeMessage.Path

	relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits)
	// check if this API is hanging (waiting for block confirmation)
	if chainMessage.GetApi().Category.HangingApi {
		relayTimeout += rcp.averageBlockTime
	}

	connectCtx, cancel := rcp.NodeUrl.LowerContextTimeout(ctx, relayTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(connectCtx, connectionTypeSlected, rcp.NodeUrl.AuthConfig.AddAuthPath(url), msgBuffer)
	if err != nil {
		return nil, "", nil, err
	}

	// setting the content-type to be application/json instead of Go's defult http.DefaultClient
	if connectionTypeSlected == http.MethodPost || connectionTypeSlected == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}

	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			req.Header.Set(metadata.Name, metadata.Value)
		}
	}
	rcp.NodeUrl.SetAuthHeaders(ctx, req.Header.Set)
	rcp.NodeUrl.SetIpForwardingIfNecessary(ctx, req.Header.Set)

	if debug {
		utils.LavaFormatDebug("provider sending node message",
			utils.Attribute{Key: "method", Value: nodeMessage.Path},
			utils.Attribute{Key: "headers", Value: req.Header},
			utils.Attribute{Key: "apiInterface", Value: "rest"},
		)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		// Validate if the error is related to the provider connection to the node or it is a valid error
		// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
		if parsedError := rcp.HandleNodeError(ctx, err); parsedError != nil {
			return nil, "", nil, parsedError
		}
		return nil, "", nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	err = rcp.HandleStatusError(res.StatusCode)
	if err != nil {
		return nil, "", nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &pairingtypes.RelayReply{
		Data:     body,
		Metadata: convertToMetadataMapOfSlices(res.Header),
	}

	// checking if rest reply data is in json format
	err = rcp.HandleJSONFormatError(reply.Data)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Rest reply is neither a JSON object nor a JSON array of objects", nil, utils.Attribute{Key: "reply.Data", Value: string(reply.Data)})
	}

	return reply, "", nil, nil
}
