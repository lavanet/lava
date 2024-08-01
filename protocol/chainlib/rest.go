package chainlib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/parser"
	"github.com/lavanet/lava/v2/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"

	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/metrics"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type RestChainParser struct {
	BaseChainParser
}

// NewRestChainParser creates a new instance of RestChainParser
func NewRestChainParser() (chainParser *RestChainParser, err error) {
	return &RestChainParser{}, nil
}

func (bcp *RestChainParser) GetUniqueName() string {
	return "rest_chain_parser"
}

func (apip *RestChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		var data []byte = nil
		urlPath := string(craftData.Data)
		if craftData.ConnectionType == http.MethodPost {
			// on post we need to send the data provided in the templace with the api as method
			data = craftData.Data
			urlPath = craftData.Path
		}
		// chain fetcher sends the replaced request inside data
		chainMessage, err := apip.ParseMsg(urlPath, data, craftData.ConnectionType, metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
		if err == nil {
			chainMessage.AppendHeader(metadata)
		}
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
func (apip *RestChainParser) ParseMsg(urlPath string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (ChainMessage, error) {
	// Guard that the RestChainParser instance exists
	if apip == nil {
		return nil, errors.New("RestChainParser not defined")
	}
	urlObj, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	urlWithNoQuery := urlObj.Path

	// Check api is supported and save it in nodeMsg
	apiCont, err := apip.getSupportedApi(urlWithNoQuery, connectionType)
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
		Path:        urlPath,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata, LatestBlockHeaderSetter: settingHeaderDirective},
	}
	// add spec path to rest message so we can extract the requested block.
	restMessage.SpecPath = apiCont.api.Name
	var requestedBlock int64
	if overwriteReqBlock == "" {
		// Fetch requested block, it is used for data reliability
		requestedBlock, err = parser.ParseBlockFromParams(restMessage, blockParser, apiCont.api.Parsers)
		if err != nil {
			utils.LavaFormatError("ParseBlockFromParams failed parsing block", err,
				utils.LogAttr("chain", apip.spec.Name),
				utils.LogAttr("blockParsing", apiCont.api.BlockParsing),
				utils.LogAttr("apiName", apiCont.api.Name),
				utils.LogAttr("connectionType", "rest"),
			)
			requestedBlock = spectypes.NOT_APPLICABLE
		}
	} else {
		requestedBlock, err = restMessage.ParseBlock(overwriteReqBlock)
		if err != nil {
			utils.LavaFormatError("failed parsing block from an overwrite header", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "overwriteReqBlock", Value: overwriteReqBlock})
			requestedBlock = spectypes.NOT_APPLICABLE
		}
	}

	nodeMsg := apip.newChainMessage(apiCont.api, requestedBlock, &restMessage, apiCollection)
	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, extensionInfo)
	return nodeMsg, apip.BaseChainParser.Validate(nodeMsg)
}

func (*RestChainParser) newChainMessage(serviceApi *spectypes.Api, requestBlock int64, restMessage *rpcInterfaceMessages.RestMessage, apiCollection *spectypes.ApiCollection) *baseChainMessageContainer {
	nodeMsg := &baseChainMessageContainer{
		api:                      serviceApi,
		apiCollection:            apiCollection,
		msg:                      restMessage,
		latestRequestedBlock:     requestBlock,
		resultErrorParsingMethod: restMessage.CheckResponseError,
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
		return nil, utils.LavaFormatWarning("rest api not supported", common.APINotSupportedError,
			utils.LogAttr("name", name),
			utils.LogAttr("connectionType", connectionType),
		)
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
	internalPaths, serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceRest)
	apip.BaseChainParser.Construct(spec, internalPaths, taggedApis, serverApis, apiCollections, headers, verifications, apip.BaseChainParser.extensionParser)
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
	endpoint       *lavasession.RPCEndpoint
	relaySender    RelaySender
	healthReporter HealthReporter
	logger         *metrics.RPCConsumerLogs
	refererData    *RefererData
}

// NewRestChainListener creates a new instance of RestChainListener
func NewRestChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	relaySender RelaySender, healthReporter HealthReporter,
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	refererData *RefererData,
) (chainListener *RestChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &RestChainListener{
		listenEndpoint,
		relaySender,
		healthReporter,
		rpcConsumerLogs,
		refererData,
	}

	return chainListener
}

// Serve http server for RestChainListener
func (apil *RestChainListener) Serve(ctx context.Context, cmdFlags common.ConsumerCmdFlags) {
	// Guard that the RestChainListener instance exists
	if apil == nil {
		return
	}

	// Setup HTTP Server
	app := createAndSetupBaseAppListener(cmdFlags, apil.endpoint.HealthCheckPath, apil.healthReporter)

	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface
	// Catch Post
	handlerPost := func(fiberCtx *fiber.Ctx) error {
		// Set response header content-type to application/json
		fiberCtx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		startTime := time.Now()
		endTx := apil.logger.LogStartTransaction("rest-http")
		defer endTx()

		msgSeed := apil.logger.GetMessageSeed()
		query := "?" + string(fiberCtx.Request().URI().QueryString())
		path := "/" + fiberCtx.Params("*")

		metadataValues := fiberCtx.GetReqHeaders()
		restHeaders := convertToMetadataMap(metadataValues)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection
		guid, found := utils.GetUniqueIdentifier(ctx)
		if found {
			msgSeed = strconv.FormatUint(guid, 10)
		}
		// TODO: handle contentType, in case its not application/json currently we set it to application/json in the Send() method
		// contentType := string(c.Context().Request.Header.ContentType())
		dappID := extractDappIDFromFiberContext(fiberCtx)
		analytics := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		utils.LavaFormatDebug("in <<<",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("_path", path),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("msgSeed", msgSeed),
			utils.LogAttr("headers", restHeaders),
		)
		analytics.SetProcessingTimestampBeforeRelay(startTime)
		userIp := fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP())
		refererMatch := fiberCtx.Params(refererMatchString, "")
		requestBody := string(fiberCtx.Body())
		relayResult, err := apil.relaySender.SendRelay(ctx, path+query, requestBody, http.MethodPost, dappID, userIp, analytics, restHeaders)
		if refererMatch != "" && apil.refererData != nil && err == nil {
			go apil.refererData.SendReferer(refererMatch, chainID, requestBody, userIp, metadataValues, nil)
		}
		reply := relayResult.GetReply()
		go apil.logger.AddMetricForHttp(analytics, err, fiberCtx.GetReqHeaders())
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, http.MethodPost, path, requestBody, errMasking, msgSeed, time.Since(startTime), err)

			// Set status to internal error\
			if relayResult.GetStatusCode() != 0 {
				fiberCtx.Status(relayResult.StatusCode)
			} else {
				fiberCtx.Status(fiber.StatusInternalServerError)
			}

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodPost, path, requestBody, string(reply.Data), msgSeed, time.Since(startTime), nil)
		if relayResult.GetStatusCode() != 0 {
			fiberCtx.Status(relayResult.StatusCode)
		}
		// Return json response and add metric for after provider processing
		err = addHeadersAndSendString(fiberCtx, reply.GetMetadata(), string(reply.Data))
		apil.logger.AddMetricForProcessingLatencyAfterProvider(analytics, chainID, apiInterface)
		return err
	}

	handlerUse := func(fiberCtx *fiber.Ctx) error {
		// Set response header content-type to application/json
		fiberCtx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		startTime := time.Now()
		endTx := apil.logger.LogStartTransaction("rest-http")
		defer endTx()
		msgSeed := apil.logger.GetMessageSeed()

		query := "?" + string(fiberCtx.Request().URI().QueryString())
		path := "/" + fiberCtx.Params("*")
		dappID := extractDappIDFromFiberContext(fiberCtx)
		analytics := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		analytics.SetProcessingTimestampBeforeRelay(startTime)

		metadataValues := fiberCtx.GetReqHeaders()
		restHeaders := convertToMetadataMap(metadataValues)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		guid, found := utils.GetUniqueIdentifier(ctx)
		if found {
			msgSeed = strconv.FormatUint(guid, 10)
		}
		defer cancel() // incase there's a problem make sure to cancel the connection
		utils.LavaFormatDebug("in <<<",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("_path", path),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("msgSeed", msgSeed),
			utils.LogAttr("headers", restHeaders),
		)
		userIp := fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP())
		refererMatch := fiberCtx.Params(refererMatchString, "")
		relayResult, err := apil.relaySender.SendRelay(ctx, path+query, "", fiberCtx.Method(), dappID, fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP()), analytics, restHeaders)
		if refererMatch != "" && apil.refererData != nil && err == nil {
			go apil.refererData.SendReferer(refererMatch, chainID, path, userIp, metadataValues, nil)
		}
		reply := relayResult.GetReply()
		go apil.logger.AddMetricForHttp(analytics, err, fiberCtx.GetReqHeaders())
		if err != nil {
			if common.APINotSupportedError.Is(err) {
				return common.CreateRestMethodNotFoundError(fiberCtx, chainID)
			}

			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("http in/out", true, fiberCtx.Method(), path, "", errMasking, msgSeed, time.Since(startTime), err)

			// Set status to internal error
			if relayResult.GetStatusCode() != 0 {
				fiberCtx.Status(relayResult.StatusCode)
			} else {
				fiberCtx.Status(fiber.StatusInternalServerError)
			}

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		}
		if relayResult.GetStatusCode() != 0 {
			fiberCtx.Status(relayResult.StatusCode)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("http in/out", false, http.MethodGet, path, "", string(reply.Data), msgSeed, time.Since(startTime), nil)

		// Return json response
		err = addHeadersAndSendString(fiberCtx, reply.GetMetadata(), string(reply.Data))
		apil.logger.AddMetricForProcessingLatencyAfterProvider(analytics, chainID, apiInterface)
		return err
	}

	if apil.refererData != nil && apil.refererData.Marker != "" {
		app.Post("/"+apil.refererData.Marker+":"+refererMatchString+"/*", handlerPost)
		app.Use("/"+apil.refererData.Marker+":"+refererMatchString+"/*", handlerUse)
	}

	app.Post("/*", handlerPost)
	// Catch the others
	app.Use("/*", handlerUse)

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
	httpClient *http.Client
}

func NewRestChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	nodeUrl.Url = strings.TrimSuffix(rpcProviderEndpoint.NodeUrls[0].Url, "/")
	rcp := &RestChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: rpcProviderEndpoint.NodeUrls[0], HashedNodeUrl: chainproxy.HashURL(nodeUrl.Url), ErrorHandler: &RestErrorHandler{}, ChainID: rpcProviderEndpoint.ChainID},
	}
	return rcp, nil
}

func (rcp *RestChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil)
	}
	if rcp.httpClient == nil {
		rcp.httpClient = &http.Client{
			Timeout: 5 * time.Minute, // we are doing a timeout by request
		}
	}
	httpClient := rcp.httpClient

	// appending hashed url
	grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeAddressHash, rcp.BaseChainProxy.HashedNodeUrl))

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
	urlPath := rcp.NodeUrl.Url + nodeMessage.Path

	// set context with timeout
	connectCtx, cancel := rcp.CapTimeoutForSend(ctx, chainMessage)
	defer cancel()

	req, err := http.NewRequestWithContext(connectCtx, connectionTypeSlected, rcp.NodeUrl.AuthConfig.AddAuthPath(urlPath), msgBuffer)
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
			utils.Attribute{Key: "_method", Value: nodeMessage.Path},
			utils.Attribute{Key: "headers", Value: req.Header},
			utils.Attribute{Key: "apiInterface", Value: "rest"},
		)
	}
	res, err := httpClient.Do(req)
	if res != nil {
		// resp can be non nil on error
		grpc.SetTrailer(ctx, metadata.Pairs(common.StatusCodeMetadataKey, strconv.Itoa(res.StatusCode))) // we ignore this error here since this code can be triggered not from grpc
	}
	if err != nil {
		// Validate if the error is related to the provider connection to the node or it is a valid error
		// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
		if parsedError := rcp.HandleNodeError(ctx, err); parsedError != nil {
			return nil, "", nil, parsedError
		}
		// always return a lava error in this case
		return nil, "", nil, err
	}
	// here we received a response that can be an error response with Code >300 or code < 200
	if res.Body != nil {
		defer res.Body.Close()
	}

	err = rcp.HandleStatusError(res.StatusCode, nodeMessage.GetDisableErrorHandling())
	if err != nil {
		return nil, "", nil, utils.LavaFormatWarning("Received invalid status code", nil, utils.Attribute{Key: "Status Code", Value: res.StatusCode}, utils.Attribute{Key: "chainID", Value: rcp.BaseChainProxy.ChainID}, utils.Attribute{Key: "apiName", Value: chainMessage.GetApi().Name})
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &RelayReplyWrapper{
		StatusCode: res.StatusCode,
		RelayReply: &pairingtypes.RelayReply{
			Data:     body,
			Metadata: convertToMetadataMapOfSlices(res.Header),
		},
	}

	// checking if rest reply data is in json format
	err = rcp.HandleJSONFormatError(reply.RelayReply.Data)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Rest reply is neither a JSON object nor a JSON array of objects", nil, utils.Attribute{Key: "reply.Data", Value: string(reply.RelayReply.Data)})
	}

	return reply, "", nil, nil
}
