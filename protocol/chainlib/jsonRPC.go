package chainlib

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v2/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/protocol/metrics"
	"github.com/lavanet/lava/v2/protocol/parser"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const SEP = "&"

type JsonRPCChainParser struct {
	BaseChainParser
}

// NewJrpcChainParser creates a new instance of JsonRPCChainParser
func NewJrpcChainParser() (chainParser *JsonRPCChainParser, err error) {
	return &JsonRPCChainParser{}, nil
}

func (bcp *JsonRPCChainParser) GetUniqueName() string {
	return "jsonrpc_chain_parser"
}

func (apip *JsonRPCChainParser) getApiCollection(connectionType, internalPath, addon string) (*spectypes.ApiCollection, error) {
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getApiCollection(connectionType, internalPath, addon)
}

func (apip *JsonRPCChainParser) getSupportedApi(name, connectionType string, internalPath string) (*ApiContainer, error) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	apiKey := ApiKey{Name: name, ConnectionType: connectionType, InternalPath: internalPath}
	return apip.BaseChainParser.getSupportedApi(apiKey)
}

func (apip *JsonRPCChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		chainMessage, err := apip.ParseMsg("", craftData.Data, craftData.ConnectionType, metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
		if err == nil {
			chainMessage.AppendHeader(metadata)
		}
		return chainMessage, err
	}

	msg := &rpcInterfaceMessages.JsonrpcMessage{
		Version:     "2.0",
		ID:          []byte("1"),
		Method:      parsing.ApiName,
		Params:      nil,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata},
	}
	apiCont, err := apip.getSupportedApi(parsing.ApiName, connectionType, "")
	if err != nil {
		return nil, err
	}
	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}
	return apip.newChainMessage(apiCont.api, spectypes.NOT_APPLICABLE, msg, apiCollection), nil
}

// this func parses message data into chain message object
func (apip *JsonRPCChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (ChainMessage, error) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return nil, errors.New("JsonRPCChainParser not defined")
	}

	// connectionType is currently only used in rest API.
	// Unmarshal request
	msgs, err := rpcInterfaceMessages.ParseJsonRPCMsg(data)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, errors.New("empty unmarshaled json")
	}
	var api *spectypes.Api
	var apiCollection *spectypes.ApiCollection
	var latestRequestedBlock, earliestRequestedBlock int64 = 0, 0
	for idx, msg := range msgs {
		var requestedBlockForMessage int64
		internalPath := ""
		if apip.isValidInternalPath(url) {
			internalPath = url
		}
		// Check api is supported and save it in nodeMsg
		apiCont, err := apip.getSupportedApi(msg.Method, connectionType, internalPath)
		if err != nil {
			utils.LavaFormatDebug("getSupportedApi jsonrpc failed", utils.LogAttr("method", msg.Method), utils.LogAttr("error", err))
			return nil, err
		}

		apiCollectionForMessage, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
		if err != nil {
			return nil, fmt.Errorf("could not find the interface %s in the service %s, %w", connectionType, apiCont.api.Name, err)
		}

		metadata, overwriteReqBlock, _ := apip.HandleHeaders(metadata, apiCollectionForMessage, spectypes.Header_pass_send)
		settingHeaderDirective, _, _ := apip.GetParsingByTag(spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA)
		msg.BaseMessage = chainproxy.BaseMessage{Headers: metadata, LatestBlockHeaderSetter: settingHeaderDirective}

		if overwriteReqBlock == "" {
			// Fetch requested block, it is used for data reliability
			requestedBlockForMessage, err = parser.ParseBlockFromParams(msg, apiCont.api.BlockParsing, apiCont.api.Parsers)
			if err != nil {
				utils.LavaFormatError("ParseBlockFromParams failed parsing block", err,
					utils.LogAttr("chain", apip.spec.Name),
					utils.LogAttr("blockParsing", apiCont.api.BlockParsing),
					utils.LogAttr("apiName", apiCont.api.Name),
					utils.LogAttr("connectionType", "jsonrpc"),
				)
				requestedBlockForMessage = spectypes.NOT_APPLICABLE
			}
		} else {
			requestedBlockForMessage, err = msg.ParseBlock(overwriteReqBlock)
			if err != nil {
				utils.LavaFormatError("failed parsing block from an overwrite header", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "overwriteReqBlock", Value: overwriteReqBlock})
				requestedBlockForMessage = spectypes.NOT_APPLICABLE
			}
		}
		if idx == 0 {
			// on the first entry store them
			api = apiCont.api
			apiCollection = apiCollectionForMessage
			latestRequestedBlock = requestedBlockForMessage
		} else {
			// on next entries we need to compare to existing data
			if api == nil {
				utils.LavaFormatFatal("invalid parsing, api is nil", nil)
			}
			// on a batch request we need to do the following:
			// 1. create a new api object, since it's not a single one
			// 2. we need to add the compute units
			// 3. we need to set the requested block to be the latest of them all or not_applicable
			// 4. we need to take the most comprehensive apiCollection (addon)
			// 5. take the strictest category
			category := api.GetCategory()
			category = category.Combine(apiCont.api.GetCategory())
			if apiCollectionForMessage.CollectionData.AddOn != "" && apiCollectionForMessage.CollectionData.AddOn != apiCollection.CollectionData.AddOn {
				if apiCollection.CollectionData.AddOn != "" {
					return nil, utils.LavaFormatError("unable to parse batch request with api from multiple addons", nil,
						utils.Attribute{Key: "first addon", Value: apiCollection.CollectionData.AddOn},
						utils.Attribute{Key: "second addon", Value: apiCollectionForMessage.CollectionData.AddOn})
				}
				apiCollection = apiCollectionForMessage // overwrite apiColleciton to take the addon
			}
			api = &spectypes.Api{
				Enabled:           api.Enabled && apiCont.api.Enabled,
				Name:              api.Name + SEP + apiCont.api.Name,
				ComputeUnits:      api.ComputeUnits + apiCont.api.ComputeUnits,
				ExtraComputeUnits: api.ExtraComputeUnits + apiCont.api.ExtraComputeUnits,
				Category:          category,
				BlockParsing: spectypes.BlockParser{
					ParserArg:    []string{},
					ParserFunc:   spectypes.PARSER_FUNC_EMPTY,
					DefaultValue: "",
					Encoding:     "",
				},
			}
			latestRequestedBlock, earliestRequestedBlock = CompareRequestedBlockInBatch(latestRequestedBlock, requestedBlockForMessage)
		}
	}
	var nodeMsg *baseChainMessageContainer
	if len(msgs) == 1 {
		nodeMsg = apip.newChainMessage(api, latestRequestedBlock, &msgs[0], apiCollection)
	} else {
		nodeMsg, err = apip.newBatchChainMessage(api, latestRequestedBlock, earliestRequestedBlock, msgs, apiCollection)
		if err != nil {
			return nil, err
		}
	}
	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, extensionInfo)
	return nodeMsg, apip.BaseChainParser.Validate(nodeMsg)
}

func (*JsonRPCChainParser) newBatchChainMessage(serviceApi *spectypes.Api, requestedBlock int64, earliestRequestedBlock int64, msgs []rpcInterfaceMessages.JsonrpcMessage, apiCollection *spectypes.ApiCollection) (*baseChainMessageContainer, error) {
	batchMessage, err := rpcInterfaceMessages.NewBatchMessage(msgs)
	if err != nil {
		return nil, err
	}
	nodeMsg := &baseChainMessageContainer{
		api:                      serviceApi,
		apiCollection:            apiCollection,
		latestRequestedBlock:     requestedBlock,
		msg:                      &batchMessage,
		earliestRequestedBlock:   earliestRequestedBlock,
		resultErrorParsingMethod: rpcInterfaceMessages.CheckResponseErrorForJsonRpcBatch,
	}
	return nodeMsg, err
}

func (*JsonRPCChainParser) newChainMessage(serviceApi *spectypes.Api, requestedBlock int64, msg *rpcInterfaceMessages.JsonrpcMessage, apiCollection *spectypes.ApiCollection) *baseChainMessageContainer {
	nodeMsg := &baseChainMessageContainer{
		api:                      serviceApi,
		apiCollection:            apiCollection,
		latestRequestedBlock:     requestedBlock,
		msg:                      msg,
		resultErrorParsingMethod: msg.CheckResponseError,
	}
	return nodeMsg
}

// SetSpec sets the spec for the JsonRPCChainParser
func (apip *JsonRPCChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	internalPaths, serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceJsonRPC)
	apip.BaseChainParser.Construct(spec, internalPaths, taggedApis, serverApis, apiCollections, headers, verifications, apip.BaseChainParser.extensionParser)
}

func (apip *JsonRPCChainParser) GetInternalPaths() map[string]struct{} {
	internalPaths := map[string]struct{}{}
	for _, apiCollection := range apip.apiCollections {
		internalPaths[apiCollection.CollectionData.InternalPath] = struct{}{}
	}
	return internalPaths
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *JsonRPCChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the JsonRPCChainParser instance exists
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
func (apip *JsonRPCChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32) {
	// Guard that the JsonRPCChainParser instance exists
	if apip == nil {
		return 0, 0, 0, 0
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Convert average block time from int64 -> time.Duration
	averageBlockTime = time.Duration(apip.spec.AverageBlockTime) * time.Millisecond

	// Return allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData from spec
	return apip.spec.AllowedBlockLagForQosSync, averageBlockTime, apip.spec.BlockDistanceForFinalizedData, apip.spec.BlocksInFinalizationProof
}

type JsonRPCChainListener struct {
	endpoint       *lavasession.RPCEndpoint
	relaySender    RelaySender
	healthReporter HealthReporter
	logger         *metrics.RPCConsumerLogs
	refererData    *RefererData
}

// NewJrpcChainListener creates a new instance of JsonRPCChainListener
func NewJrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	relaySender RelaySender, healthReporter HealthReporter,
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	refererData *RefererData,
) (chainListener *JsonRPCChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &JsonRPCChainListener{
		listenEndpoint,
		relaySender,
		healthReporter,
		rpcConsumerLogs,
		refererData,
	}

	return chainListener
}

// Serve http server for JsonRPCChainListener
func (apil *JsonRPCChainListener) Serve(ctx context.Context, cmdFlags common.ConsumerCmdFlags) {
	// Guard that the JsonRPCChainListener instance exists
	if apil == nil {
		return
	}
	test_mode := common.IsTestMode(ctx)
	// Setup HTTP Server
	app := createAndSetupBaseAppListener(cmdFlags, apil.endpoint.HealthCheckPath, apil.healthReporter)

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface

	webSocketCallback := websocket.New(func(websockConn *websocket.Conn) {
		var (
			messageType int
			msg         []byte
			err         error
		)
		startTime := time.Now()
		msgSeed := apil.logger.GetMessageSeed()
		for {
			if messageType, msg, err = websockConn.ReadMessage(); err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
				break
			}
			dappID, ok := websockConn.Locals("dapp-id").(string)
			if !ok {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, nil, msgSeed, []byte("Unable to extract dappID"), spectypes.APIInterfaceJsonRPC, time.Since(startTime))
			}
			refererMatch, ok := websockConn.Locals(refererMatchString).(string)
			ctx, cancel := context.WithCancel(context.Background())
			guid := utils.GenerateUniqueIdentifier()
			ctx = utils.WithUniqueIdentifier(ctx, guid)
			msgSeed = strconv.FormatUint(guid, 10)
			defer cancel() // incase there's a problem make sure to cancel the connection

			logFormattedMsg := string(msg)
			if !cmdFlags.DebugRelays {
				logFormattedMsg = utils.FormatLongString(logFormattedMsg, relayMsgLogMaxChars)
			}

			utils.LavaFormatDebug("ws in <<<",
				utils.LogAttr("seed", msgSeed),
				utils.LogAttr("GUID", ctx),
				utils.LogAttr("msg", logFormattedMsg),
				utils.LogAttr("dappID", dappID),
			)
			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			relayResult, err := apil.relaySender.SendRelay(ctx, "", string(msg), http.MethodPost, dappID, websockConn.RemoteAddr().String(), metricsData, nil)
			if ok && refererMatch != "" && apil.refererData != nil && err == nil {
				go apil.refererData.SendReferer(refererMatch, chainID, string(msg), websockConn.RemoteAddr().String(), nil, websockConn)
			}
			reply := relayResult.GetReply()
			replyServer := relayResult.GetReplyServer()
			go apil.logger.AddMetricForWebSocket(metricsData, err, websockConn)
			if err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
					continue
				}

				if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, time.Since(startTime), nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
						break
					}

					// If portal cant write to the client
					if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
						cancel()
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
						// break
					}

					apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, time.Since(startTime), nil)
				}
			} else {
				if err = websockConn.WriteMessage(messageType, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(websockConn, messageType, err, msgSeed, msg, spectypes.APIInterfaceJsonRPC, time.Since(startTime))
					continue
				}
				apil.logger.LogRequestAndResponse("jsonrpc ws msg", false, "ws", websockConn.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, time.Since(startTime), nil)
			}
		}
	})
	websocketCallbackWithDappID := constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, apil.logger.StoreMetricData)
	app.Get("/ws", websocketCallbackWithDappID)
	app.Get("/websocket", websocketCallbackWithDappID) // catching http://HOST:PORT/1/websocket requests.

	handlerPost := func(fiberCtx *fiber.Ctx) error {
		// Set response header content-type to application/json
		fiberCtx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		startTime := time.Now()
		endTx := apil.logger.LogStartTransaction("jsonRpc-http post")
		defer endTx()
		dappID := extractDappIDFromFiberContext(fiberCtx)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		metricsData.SetProcessingTimestampBeforeRelay(startTime)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		guid := utils.GenerateUniqueIdentifier()
		ctx = utils.WithUniqueIdentifier(ctx, guid)
		msgSeed := strconv.FormatUint(guid, 10)
		if test_mode {
			apil.logger.LogTestMode(fiberCtx)
		}

		userIp := fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP())
		metadataValues := fiberCtx.GetReqHeaders()
		headers := convertToMetadataMap(metadataValues)

		msg := string(fiberCtx.Body())
		logFormattedMsg := msg
		if !cmdFlags.DebugRelays {
			logFormattedMsg = utils.FormatLongString(logFormattedMsg, relayMsgLogMaxChars)
		}

		path := "/" + fiberCtx.Params("*")
		utils.LavaFormatDebug("in <<<",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("path", path),
			utils.LogAttr("seed", msgSeed),
			utils.LogAttr("_msg", logFormattedMsg),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("headers", headers),
		)
		refererMatch := fiberCtx.Params(refererMatchString, "")
		relayResult, err := apil.relaySender.SendRelay(ctx, path, msg, http.MethodPost, dappID, userIp, metricsData, headers)
		if refererMatch != "" && apil.refererData != nil && err == nil {
			go apil.refererData.SendReferer(refererMatch, chainID, msg, userIp, metadataValues, nil)
		}
		reply := relayResult.GetReply()
		go apil.logger.AddMetricForHttp(metricsData, err, fiberCtx.GetReqHeaders())
		if err != nil {
			if common.APINotSupportedError.Is(err) {
				return fiberCtx.Status(fiber.StatusOK).JSON(common.JsonRpcMethodNotFoundError)
			}

			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("jsonrpc http", true, "POST", fiberCtx.Request().URI().String(), msg, errMasking, msgSeed, time.Since(startTime), err)

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
		response := string(reply.Data)
		// Log request and response
		apil.logger.LogRequestAndResponse("jsonrpc http",
			false,
			"POST",
			fiberCtx.Request().URI().String(),
			msg,
			response,
			msgSeed,
			time.Since(startTime),
			nil,
		)
		if relayResult.GetStatusCode() != 0 {
			fiberCtx.Status(relayResult.StatusCode)
		}
		// Return json response and add metric for after provider processing
		err = addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		apil.logger.AddMetricForProcessingLatencyAfterProvider(metricsData, chainID, apiInterface)
		return err
	}
	if apil.refererData != nil && apil.refererData.Marker != "" {
		app.Use("/"+apil.refererData.Marker+":"+refererMatchString+"/ws", func(c *fiber.Ctx) error {
			if websocket.IsWebSocketUpgrade(c) {
				c.Locals("allowed", true)
				return c.Next()
			}
			return fiber.ErrUpgradeRequired
		})
		websocketCallbackWithDappIDAndReferer := constructFiberCallbackWithHeaderAndParameterExtractionAndReferer(webSocketCallback, apil.logger.StoreMetricData)
		app.Get("/"+apil.refererData.Marker+":"+refererMatchString+"/ws", websocketCallbackWithDappIDAndReferer)
		app.Get("/"+apil.refererData.Marker+":"+refererMatchString+"/websocket", websocketCallbackWithDappIDAndReferer)
		app.Post("/"+apil.refererData.Marker+":"+refererMatchString+"/*", handlerPost)
	}
	app.Post("/*", handlerPost)
	// Go
	ListenWithRetry(app, apil.endpoint.NetworkAddress)
}

type JrpcChainProxy struct {
	BaseChainProxy
	conn map[string]*chainproxy.Connector
}

func NewJrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	cp := &JrpcChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: nodeUrl, ErrorHandler: &JsonRPCErrorHandler{}, ChainID: rpcProviderEndpoint.ChainID},
		conn:           map[string]*chainproxy.Connector{},
	}
	verifyRPCEndpoint(nodeUrl.Url)
	internalPaths := map[string]struct{}{}
	jsonRPCChainParser, ok := chainParser.(*JsonRPCChainParser)
	if ok {
		internalPaths = jsonRPCChainParser.GetInternalPaths()
	}
	internalPathsLength := len(internalPaths)
	if internalPathsLength > 0 && internalPathsLength == len(rpcProviderEndpoint.NodeUrls) {
		return cp, cp.startWithSpecificInternalPaths(ctx, nConns, rpcProviderEndpoint.NodeUrls, internalPaths)
	} else if internalPathsLength > 0 && len(rpcProviderEndpoint.NodeUrls) > 1 {
		// provider provided specific endpoints but not enough to fill all requirements
		return nil, utils.LavaFormatError("Internal Paths specified but not all paths provided", nil, utils.Attribute{Key: "required", Value: internalPaths}, utils.Attribute{Key: "provided", Value: rpcProviderEndpoint.NodeUrls})
	}
	return cp, cp.start(ctx, nConns, nodeUrl, internalPaths)
}

func (cp *JrpcChainProxy) startWithSpecificInternalPaths(ctx context.Context, nConns uint, nodeUrls []common.NodeUrl, internalPaths map[string]struct{}) error {
	for _, url := range nodeUrls {
		_, ok := internalPaths[url.InternalPath]
		if !ok {
			return utils.LavaFormatError("url.InternalPath was not found in internalPaths", nil, utils.Attribute{Key: "internalPaths", Value: internalPaths}, utils.Attribute{Key: "url.InternalPath", Value: url.InternalPath})
		}
		utils.LavaFormatDebug("connecting", utils.Attribute{Key: "url", Value: url.String()})
		conn, err := chainproxy.NewConnector(ctx, nConns, url)
		if err != nil {
			return err
		}
		cp.conn[url.InternalPath] = conn
	}
	if len(cp.conn) != len(internalPaths) {
		return utils.LavaFormatError("missing connectors for a chain with internal paths", nil, utils.Attribute{Key: "internalPaths", Value: internalPaths}, utils.Attribute{Key: "nodeUrls", Value: nodeUrls})
	}
	return nil
}

func (cp *JrpcChainProxy) start(ctx context.Context, nConns uint, nodeUrl common.NodeUrl, internalPaths map[string]struct{}) error {
	if len(internalPaths) == 0 {
		internalPaths = map[string]struct{}{"": {}} // add default path
	}
	basePath := nodeUrl.Url
	for path := range internalPaths {
		nodeUrl.Url = basePath + path
		conn, err := chainproxy.NewConnector(ctx, nConns, nodeUrl)
		if err != nil {
			return err
		}
		cp.conn[path] = conn
		if cp.conn == nil {
			return errors.New("g_conn == nil")
		}
	}
	return nil
}

func (cp *JrpcChainProxy) sendBatchMessage(ctx context.Context, nodeMessage *rpcInterfaceMessages.JsonrpcBatchMessage, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, err error) {
	internalPath := chainMessage.GetApiCollection().CollectionData.InternalPath
	rpc, err := cp.conn[internalPath].GetRpc(ctx, true)
	if err != nil {
		return nil, err
	}
	defer cp.conn[internalPath].ReturnRpc(rpc)
	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			rpc.SetHeader(metadata.Name, metadata.Value)
			// clear this header upon function completion so it doesn't last in the next usage from the rpc pool
			defer rpc.SetHeader(metadata.Name, "")
		}
	}
	// set context with timeout
	connectCtx, cancel := cp.CapTimeoutForSend(ctx, chainMessage)
	defer cancel()

	cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)
	batch := nodeMessage.GetBatch()
	err = rpc.BatchCallContext(connectCtx, batch, nodeMessage.GetDisableErrorHandling())
	if err != nil {
		// Validate if the error is related to the provider connection to the node or it is a valid error
		// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
		if parsedError := cp.HandleNodeError(ctx, err); parsedError != nil {
			return nil, parsedError
		}
		return nil, err
	}
	replyMsgs := make([]rpcInterfaceMessages.JsonrpcMessage, len(batch))
	for idx, element := range batch {
		// convert them because batch elements can't be marshaled back to the user, they are missing tags and flieds
		replyMsgs[idx], err = rpcInterfaceMessages.ConvertBatchElement(element)
		if err != nil {
			return nil, err
		}
	}
	retData, err := json.Marshal(replyMsgs)
	if err != nil {
		return nil, err
	}
	reply := &RelayReplyWrapper{
		StatusCode: http.StatusOK, // status code is used only for rest at the moment

		RelayReply: &pairingtypes.RelayReply{
			Data: retData,
		},
	}
	return reply, nil
}

func (cp *JrpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.JsonrpcMessage)
	if !ok {
		// this could be a batch message
		batchMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.JsonrpcBatchMessage)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast JsonrpcMessage or JsonrpcBatchMessage from chainMessage", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage})
		}
		if ch != nil {
			return nil, "", nil, utils.LavaFormatError("does not support subscribe in a batch", nil)
		}
		reply, err := cp.sendBatchMessage(ctx, batchMessage, chainMessage)
		return reply, "", nil, err
	}
	internalPath := chainMessage.GetApiCollection().CollectionData.InternalPath
	rpc, err := cp.conn[internalPath].GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	defer cp.conn[internalPath].ReturnRpc(rpc)

	// appending hashed url
	grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeAddressHash, cp.conn[internalPath].GetUrlHash()))

	// Call our node
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *rpcInterfaceMessages.JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	// support setting headers
	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			rpc.SetHeader(metadata.Name, metadata.Value)
			// clear this header upon function completion so it doesn't last in the next usage from the rpc pool
			defer rpc.SetHeader(metadata.Name, "")
		}
	}
	if ch != nil {
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		// we use the minimum timeout between the two, spec or context. to prevent the provider from hanging
		// we don't use the context alone so the provider won't be hanging forever by an attack
		connectCtx, cancel := cp.CapTimeoutForSend(ctx, chainMessage)
		defer cancel()

		cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params, true, nodeMessage.GetDisableErrorHandling())
		if err != nil {
			// here we are getting an error for every code that is not 200-300
			if common.StatusCodeError504.Is(err) || common.StatusCodeError429.Is(err) || common.StatusCodeErrorStrict.Is(err) {
				return nil, "", nil, utils.LavaFormatWarning("Received invalid status code", err, utils.Attribute{Key: "chainID", Value: cp.BaseChainProxy.ChainID}, utils.Attribute{Key: "apiName", Value: chainMessage.GetApi().Name})
			}
			// Validate if the error is related to the provider connection to the node or it is a valid error
			// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
			if parsedError := cp.HandleNodeError(ctx, err); parsedError != nil {
				return nil, "", nil, parsedError
			}
		}
	}

	var replyMsg rpcInterfaceMessages.JsonrpcMessage
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		utils.LavaFormatDebug("received an error from SendNodeMsg", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err})
		return nil, "", nil, err
	} else {
		replyMessage, err = rpcInterfaceMessages.ConvertJsonRPCMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC error", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
		// validate result is valid
		if replyMessage.Error == nil {
			responseIsNilValidationError := ValidateNilResponse(string(replyMessage.Result))
			if responseIsNilValidationError != nil {
				return nil, "", nil, responseIsNilValidationError
			}
		}

		replyMsg = *replyMessage
		err := cp.ValidateRequestAndResponseIds(nodeMessage.ID, replyMessage.ID)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("jsonRPC ID mismatch error", err,
				utils.Attribute{Key: "GUID", Value: ctx},
				utils.Attribute{Key: "requestId", Value: nodeMessage.ID},
				utils.Attribute{Key: "responseId", Value: rpcMessage.ID},
			)
		}
	}

	retData, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	reply := &RelayReplyWrapper{
		StatusCode: http.StatusOK, // status code is used only for rest at the moment

		RelayReply: &pairingtypes.RelayReply{
			Data: retData,
		},
	}

	if ch != nil {
		subscriptionID, err = strconv.Unquote(string(replyMsg.Result))
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("Subscription failed", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}

	return reply, subscriptionID, sub, err
}
