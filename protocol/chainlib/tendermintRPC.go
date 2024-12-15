package chainlib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/protocol/parser"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type TendermintChainParser struct {
	BaseChainParser
}

// NewTendermintRpcChainParser creates a new instance of TendermintChainParser
func NewTendermintRpcChainParser() (chainParser *TendermintChainParser, err error) {
	return &TendermintChainParser{}, nil
}

func (bcp *TendermintChainParser) GetUniqueName() string {
	return "tendermint_chain_parser"
}

func (apip *TendermintChainParser) getApiCollection(connectionType, internalPath, addon string) (*spectypes.ApiCollection, error) {
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getApiCollection(connectionType, internalPath, addon)
}

func (apip *TendermintChainParser) getSupportedApi(name, connectionType string) (*ApiContainer, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	apiKey := ApiKey{Name: name, ConnectionType: connectionType}
	return apip.BaseChainParser.getSupportedApi(apiKey)
}

func (apip *TendermintChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		path := craftData.Path
		if craftData.InternalPath != "" {
			path = craftData.InternalPath
		}

		chainMessage, err := apip.ParseMsg(path, craftData.Data, craftData.ConnectionType, metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
		if err == nil {
			chainMessage.AppendHeader(metadata)
		}
		return chainMessage, err
	}

	msg := rpcInterfaceMessages.JsonrpcMessage{
		Version:     "2.0",
		ID:          []byte("1"),
		Method:      parsing.ApiName,
		Params:      nil,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata},
	}

	apiCont, err := apip.getSupportedApi(parsing.ApiName, connectionType)
	if err != nil {
		return nil, err
	}
	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, err
	}
	tenderMsg := rpcInterfaceMessages.TendermintrpcMessage{JsonrpcMessage: msg, Path: parsing.ApiName}
	return apip.newChainMessage(apiCont.api, spectypes.NOT_APPLICABLE, nil, &tenderMsg, apiCollection, false), nil
}

// ParseMsg parses message data into chain message object
func (apip *TendermintChainParser) ParseMsg(urlPath string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (ChainMessage, error) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return nil, errors.New("TendermintChainParser not defined")
	}

	// connectionType is currently only used in rest api
	// Unmarshal request
	var msgs []rpcInterfaceMessages.JsonrpcMessage
	isJsonrpc := string(data) != ""
	if isJsonrpc {
		// Fetch pointer to message and error
		var err error
		msgs, err = rpcInterfaceMessages.ParseJsonRPCMsg(data)
		if err != nil {
			return nil, err
		}
	} else {
		// assuming URI
		urlObj, err := url.Parse(urlPath)
		if err != nil {
			return nil, err
		}
		urlObj.Query()

		msg := rpcInterfaceMessages.JsonrpcMessage{
			ID:      []byte("1"),
			Version: "2.0",
			Method:  urlObj.Path,
		}
		params := make(map[string]interface{})
		queryValues := urlObj.Query()
		for key, values := range queryValues {
			params[key] = strings.Join(values, ",")
		}
		msg.Params = params
		msgs = []rpcInterfaceMessages.JsonrpcMessage{msg}
	}

	msgsLength := len(msgs)

	if msgsLength == 0 {
		return nil, errors.New("empty unmarshaled json")
	}

	var api *spectypes.Api
	var apiCollection *spectypes.ApiCollection
	var latestRequestedBlock, earliestRequestedBlock int64 = 0, spectypes.LATEST_BLOCK
	blockHashes := []string{}
	parsedDefault := true
	for idx, msg := range msgs {
		parsedInput := parser.NewParsedInput()
		// Check api is supported and save it in nodeMsg
		apiCont, err := apip.getSupportedApi(msg.Method, connectionType)
		if err != nil {
			utils.LavaFormatDebug("getSupportedApi tendermintrpc failed",
				utils.LogAttr("method", msg.Method),
				utils.LogAttr("connectionType", connectionType),
				utils.LogAttr("error", err),
			)
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
			parsedInput = parser.ParseBlockFromParams(msg, apiCont.api.BlockParsing, apiCont.api.Parsers)
			if hashes, err := parsedInput.GetBlockHashes(); err == nil {
				blockHashes = append(blockHashes, hashes...)
			}
			if !parsedInput.UsedDefaultValue {
				parsedDefault = false
			}
		} else {
			parsedBlock, err := msg.ParseBlock(overwriteReqBlock)
			parsedInput.SetBlock(parsedBlock)
			if err != nil {
				utils.LavaFormatError("failed parsing block from an overwrite header", err,
					utils.LogAttr("chain", apip.spec.Name),
					utils.LogAttr("overwriteReqBlock", overwriteReqBlock),
				)
				parsedInput.SetBlock(spectypes.NOT_APPLICABLE)
			} else {
				parsedInput.UsedDefaultValue = false
			}
		}

		parsedBlock := parsedInput.GetBlock()

		if idx == 0 {
			// on the first entry store them
			api = apiCont.api
			apiCollection = apiCollectionForMessage
			latestRequestedBlock = parsedBlock
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
		}

		if msgsLength > 1 {
			latestRequestedBlock, earliestRequestedBlock = CompareRequestedBlockInBatch(latestRequestedBlock, earliestRequestedBlock, parsedBlock)
		}
	}

	var nodeMsg *baseChainMessageContainer
	if msgsLength == 1 {
		tenderMsg := rpcInterfaceMessages.TendermintrpcMessage{JsonrpcMessage: msgs[0], Path: ""}
		if !isJsonrpc {
			tenderMsg.Path = urlPath // add path
		}
		nodeMsg = apip.newChainMessage(api, latestRequestedBlock, blockHashes, &tenderMsg, apiCollection, parsedDefault)
	} else {
		var err error
		nodeMsg, err = apip.newBatchChainMessage(api, latestRequestedBlock, earliestRequestedBlock, blockHashes, msgs, apiCollection, parsedDefault)
		if err != nil {
			return nil, err
		}
	}

	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, extensionInfo)
	return nodeMsg, apip.BaseChainParser.Validate(nodeMsg)
}

func (*TendermintChainParser) newBatchChainMessage(serviceApi *spectypes.Api, requestedBlock int64, earliestRequestedBlock int64, requestedHashes []string, msgs []rpcInterfaceMessages.JsonrpcMessage, apiCollection *spectypes.ApiCollection, usedDefaultValue bool) (*baseChainMessageContainer, error) {
	batchMessage, err := rpcInterfaceMessages.NewBatchMessage(msgs)
	if err != nil {
		return nil, err
	}
	nodeMsg := &baseChainMessageContainer{
		api:                      serviceApi,
		apiCollection:            apiCollection,
		latestRequestedBlock:     requestedBlock,
		requestedBlockHashes:     requestedHashes,
		msg:                      &batchMessage,
		earliestRequestedBlock:   earliestRequestedBlock,
		resultErrorParsingMethod: rpcInterfaceMessages.CheckResponseErrorForJsonRpcBatch,
		parseDirective:           GetParseDirective(serviceApi, apiCollection),
		usedDefaultValue:         usedDefaultValue,
	}
	return nodeMsg, err
}

// overwritten because tendermintrpc doesnt use POST but an empty connecionType
func (apip *TendermintChainParser) ExtractDataFromRequest(request *http.Request) (url string, data string, connectionType string, metadata []pairingtypes.Metadata, err error) {
	url, data, _, metadata, err = apip.BaseChainParser.ExtractDataFromRequest(request)
	return url, data, "", metadata, err
}

func (*TendermintChainParser) newChainMessage(serviceApi *spectypes.Api, requestedBlock int64, requestedHashes []string, msg *rpcInterfaceMessages.TendermintrpcMessage, apiCollection *spectypes.ApiCollection, usedDefaultValue bool) *baseChainMessageContainer {
	nodeMsg := &baseChainMessageContainer{
		api:                      serviceApi,
		apiCollection:            apiCollection,
		latestRequestedBlock:     requestedBlock,
		requestedBlockHashes:     requestedHashes,
		msg:                      msg,
		resultErrorParsingMethod: msg.CheckResponseError,
		parseDirective:           GetParseDirective(serviceApi, apiCollection),
		usedDefaultValue:         usedDefaultValue,
	}
	return nodeMsg
}

// SetSpec sets the spec for the TendermintChainParser
func (apip *TendermintChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the TendermintChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	internalPaths, serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceTendermintRPC)
	apip.BaseChainParser.Construct(spec, internalPaths, taggedApis, serverApis, apiCollections, headers, verifications)
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *TendermintChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the TendermintChainParser instance exists
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
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData, spec.BlocksInFinalizationProof)
func (apip *TendermintChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32) {
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

type TendermintRpcChainListener struct {
	endpoint                      *lavasession.RPCEndpoint
	relaySender                   RelaySender
	healthReporter                HealthReporter
	logger                        *metrics.RPCConsumerLogs
	refererData                   *RefererData
	consumerWsSubscriptionManager *ConsumerWSSubscriptionManager
	listeningAddress              string
	websocketConnectionLimiter    *WebsocketConnectionLimiter
}

// NewTendermintRpcChainListener creates a new instance of TendermintRpcChainListener
func NewTendermintRpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint,
	relaySender RelaySender, healthReporter HealthReporter,
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	refererData *RefererData,
	consumerWsSubscriptionManager *ConsumerWSSubscriptionManager,
) (chainListener *TendermintRpcChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &TendermintRpcChainListener{
		endpoint:                      listenEndpoint,
		relaySender:                   relaySender,
		healthReporter:                healthReporter,
		logger:                        rpcConsumerLogs,
		refererData:                   refererData,
		consumerWsSubscriptionManager: consumerWsSubscriptionManager,
		websocketConnectionLimiter:    &WebsocketConnectionLimiter{ipToNumberOfActiveConnections: make(map[string]int64)},
	}

	return chainListener
}

// Serve http server for TendermintRpcChainListener
func (apil *TendermintRpcChainListener) Serve(ctx context.Context, cmdFlags common.ConsumerCmdFlags) {
	// Guard that the TendermintChainParser instance exists
	if apil == nil {
		return
	}

	// Setup HTTP Server
	app := createAndSetupBaseAppListener(cmdFlags, apil.endpoint.HealthCheckPath, apil.healthReporter)
	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface

	app.Use("/ws", func(c *fiber.Ctx) error {
		apil.websocketConnectionLimiter.HandleFiberRateLimitFlags(c)
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	webSocketCallback := websocket.New(func(websocketConn *websocket.Conn) {
		canOpenConnection, decreaseIpConnection := apil.websocketConnectionLimiter.CanOpenConnection(websocketConn)
		defer decreaseIpConnection()
		if !canOpenConnection {
			return
		}

		rateLimitInf := websocketConn.Locals(WebSocketRateLimitHeader)
		rateLimit, assertionSuccessful := rateLimitInf.(int64)
		if !assertionSuccessful || rateLimit < 0 {
			rateLimit = 0
		}

		utils.LavaFormatDebug("tendermintrpc websocket opened", utils.LogAttr("consumerIp", websocketConn.LocalAddr().String()))
		defer utils.LavaFormatDebug("tendermintrpc websocket closed", utils.LogAttr("consumerIp", websocketConn.LocalAddr().String()))

		consumerWebsocketManager := NewConsumerWebsocketManager(ConsumerWebsocketManagerOptions{
			WebsocketConn:                 websocketConn,
			RpcConsumerLogs:               apil.logger,
			RefererMatchString:            refererMatchString,
			CmdFlags:                      cmdFlags,
			RelayMsgLogMaxChars:           relayMsgLogMaxChars,
			ChainID:                       chainID,
			ApiInterface:                  apiInterface,
			ConnectionType:                "", // We use it for the ParseMsg method, which needs to know the connection type to find the method in the spec
			RefererData:                   apil.refererData,
			RelaySender:                   apil.relaySender,
			ConsumerWsSubscriptionManager: apil.consumerWsSubscriptionManager,
			WebsocketConnectionUID:        strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10),
			headerRateLimit:               uint64(rateLimit),
		})

		consumerWebsocketManager.ListenToMessages()
	})
	websocketCallbackWithDappID := constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, apil.logger.StoreMetricData)
	app.Get("/ws", websocketCallbackWithDappID)
	app.Get("/websocket", websocketCallbackWithDappID) // catching http://HOST:PORT/1/websocket requests.

	handlerPost := func(fiberCtx *fiber.Ctx) error {
		// Set response header content-type to application/json
		fiberCtx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		startTime := time.Now()
		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()
		dappID := extractDappIDFromFiberContext(fiberCtx)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		metricsData.SetProcessingTimestampBeforeRelay(startTime)
		ctx, cancel := context.WithCancel(context.Background())
		guid := utils.GenerateUniqueIdentifier()
		ctx = utils.WithUniqueIdentifier(ctx, guid)
		defer cancel() // incase there's a problem make sure to cancel the connection
		msgSeed := strconv.FormatUint(guid, 10)
		metadataValues := fiberCtx.GetReqHeaders()
		headers := convertToMetadataMap(metadataValues)

		msg := string(fiberCtx.Body())
		logFormattedMsg := msg
		if !cmdFlags.DebugRelays {
			logFormattedMsg = utils.FormatLongString(logFormattedMsg, relayMsgLogMaxChars)
		}

		utils.LavaFormatDebug("in <<<",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("seed", msgSeed),
			utils.LogAttr("_msg", logFormattedMsg),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("headers", headers),
		)
		refererMatch := fiberCtx.Params(refererMatchString, "")
		userIp := fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP())
		relayResult, err := apil.relaySender.SendRelay(ctx, "", msg, "", dappID, userIp, metricsData, headers)
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
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "POST", fiberCtx.Request().URI().String(), msg, errMasking, msgSeed, time.Since(startTime), err)

			// Set status to internal error
			if relayResult.GetStatusCode() != 0 {
				fiberCtx.Status(relayResult.StatusCode)
			} else {
				fiberCtx.Status(fiber.StatusInternalServerError)
			}

			// Construct json response
			response := rpcInterfaceMessages.ConvertToTendermintError(errMasking, fiberCtx.Body())
			// Return error json response
			return addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "POST", fiberCtx.Request().URI().String(), msg, string(reply.Data), msgSeed, time.Since(startTime), nil)
		if relayResult.GetStatusCode() != 0 {
			fiberCtx.Status(relayResult.StatusCode)
		}
		response := string(reply.Data)
		// Return json response
		err = addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		apil.logger.AddMetricForProcessingLatencyAfterProvider(metricsData, chainID, apiInterface)
		return err
	}

	handlerGet := func(fiberCtx *fiber.Ctx) error {
		// Set response header content-type to application/json
		fiberCtx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()
		startTime := time.Now()
		query := "?" + string(fiberCtx.Request().URI().QueryString())
		path := fiberCtx.Params("*")
		dappID := extractDappIDFromFiberContext(fiberCtx)
		ctx, cancel := context.WithCancel(context.Background())
		guid := utils.GenerateUniqueIdentifier()
		ctx = utils.WithUniqueIdentifier(ctx, guid)
		defer cancel() // incase there's a problem make sure to cancel the connection
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		metricsData.SetProcessingTimestampBeforeRelay(startTime)
		metadataValues := fiberCtx.GetReqHeaders()
		headers := convertToMetadataMap(metadataValues)
		utils.LavaFormatDebug("urirpc in <<<",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("_msg", path),
			utils.LogAttr("dappID", dappID),
			utils.LogAttr("headers", headers),
		)
		userIp := fiberCtx.Get(common.IP_FORWARDING_HEADER_NAME, fiberCtx.IP())
		refererMatch := fiberCtx.Params(refererMatchString, "")
		relayResult, err := apil.relaySender.SendRelay(ctx, path+query, "", "", dappID, userIp, metricsData, headers)
		if refererMatch != "" && apil.refererData != nil && err == nil {
			go apil.refererData.SendReferer(refererMatch, chainID, path, userIp, metadataValues, nil)
		}
		msgSeed := strconv.FormatUint(guid, 10)
		reply := relayResult.GetReply()
		go apil.logger.AddMetricForHttp(metricsData, err, fiberCtx.GetReqHeaders())
		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "GET", fiberCtx.Request().URI().String(), "", errMasking, msgSeed, time.Since(startTime), err)

			// Set status to internal error
			if relayResult.GetStatusCode() != 0 {
				fiberCtx.Status(relayResult.StatusCode)
			} else {
				fiberCtx.Status(fiber.StatusInternalServerError)
			}

			if string(fiberCtx.Body()) != "" {
				errMasking = addAttributeToError("recommendation", "For jsonRPC use POST", errMasking)
			}

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return addHeadersAndSendString(fiberCtx, reply.GetMetadata(), response)
		}
		response := string(reply.Data)
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "GET", fiberCtx.Request().URI().String(), "", response, msgSeed, time.Since(startTime), nil)
		if relayResult.GetStatusCode() != 0 {
			fiberCtx.Status(relayResult.StatusCode)
		}
		// Return json response
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
		app.Get("/"+apil.refererData.Marker+":"+refererMatchString+"/*", handlerGet)
	}

	app.Post("/*", handlerPost)
	app.Get("/*", handlerGet)
	//
	// Go
	addrChannel := make(chan string)
	addrChannelSafe := common.NewSafeChannelSender(ctx, addrChannel)
	go func() {
		addr := <-addrChannel
		apil.listeningAddress = addr
	}()

	ListenWithRetry(app, apil.endpoint.NetworkAddress, addrChannelSafe)
}

func (apil *TendermintRpcChainListener) GetListeningAddress() string {
	return apil.listeningAddress
}

type tendermintRpcChainProxy struct {
	// embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
	httpClient *http.Client
}

func NewtendermintRpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()

	validateEndpoints(rpcProviderEndpoint.NodeUrls, spectypes.APIInterfaceTendermintRPC)

	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	cp := &tendermintRpcChainProxy{
		JrpcChainProxy: JrpcChainProxy{
			BaseChainProxy: BaseChainProxy{
				averageBlockTime: averageBlockTime,
				NodeUrl:          nodeUrl,
				ErrorHandler:     &TendermintRPCErrorHandler{},
				ChainID:          rpcProviderEndpoint.ChainID,
			},
			conn: nil,
		},
	}

	return cp, cp.start(ctx, nConns, nodeUrl)
}

func (cp *tendermintRpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.TendermintrpcMessage)
	if !ok {
		batchMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.JsonrpcBatchMessage)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("invalid message type in tendermintrpc failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage}, utils.Attribute{Key: "ptrCast", Value: ok})
		}
		if ch != nil {
			return nil, "", nil, utils.LavaFormatError("does not support subscribe in a batch", nil)
		}
		reply, err := cp.sendBatchMessage(ctx, batchMessage, chainMessage)
		return reply, "", nil, err
	}
	if nodeMessage.Path != "" {
		return cp.SendURI(ctx, nodeMessage, ch, chainMessage)
	}

	// Else do RPC call
	return cp.SendRPC(ctx, nodeMessage, ch, chainMessage)
}

func (cp *tendermintRpcChainProxy) SendURI(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// check if the input channel is not nil
	if ch != nil {
		// return an error if the channel is not nil
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on Tendermint URI", nil)
	}

	if cp.httpClient == nil {
		cp.httpClient = &http.Client{
			Timeout: 5 * time.Minute, // we are doing a timeout by request
		}
	}

	httpClient := cp.httpClient

	// appending hashed url
	grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeAddressHash, cp.conn.GetUrlHash()))

	// construct the url by concatenating the node url with the path variable
	url := cp.NodeUrl.Url + "/" + nodeMessage.Path

	// set context with timeout
	connectCtx, cancel := cp.CapTimeoutForSend(ctx, chainMessage)
	defer cancel()

	// create a new http request
	req, err := http.NewRequestWithContext(connectCtx, http.MethodGet, cp.NodeUrl.AuthConfig.AddAuthPath(url), nil)
	if err != nil {
		// Validate if the error is related to the provider connection to the node or it is a valid error
		// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
		if parsedError := cp.HandleNodeError(ctx, err); parsedError != nil {
			return nil, "", nil, parsedError
		}
		// TODO: if not parsed return error as reply or as error?
		return nil, "", nil, err
	}

	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			req.Header.Set(metadata.Name, metadata.Value)
		}
	}

	cp.NodeUrl.SetAuthHeaders(ctx, req.Header.Set)

	cp.NodeUrl.SetIpForwardingIfNecessary(ctx, req.Header.Set)
	// send the http request and get the response
	res, err := httpClient.Do(req)
	if res != nil {
		// resp can be non nil on error
		grpc.SetTrailer(ctx, metadata.Pairs(common.StatusCodeMetadataKey, strconv.Itoa(res.StatusCode))) // we ignore this error here since this code can be triggered not from grpc
	}
	if err != nil {
		return nil, "", nil, err
	}
	// close the response body
	if res.Body != nil {
		defer res.Body.Close()
	}

	err = cp.HandleStatusError(res.StatusCode, nodeMessage.GetDisableErrorHandling())
	if err != nil {
		return nil, "", nil, utils.LavaFormatWarning("Received invalid status code", nil, utils.Attribute{Key: "Status Code", Value: res.StatusCode}, utils.Attribute{Key: "chainID", Value: cp.BaseChainProxy.ChainID}, utils.Attribute{Key: "apiName", Value: chainMessage.GetApi().Name})
	}

	// read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct with the response body as the data
	reply := &RelayReplyWrapper{
		StatusCode: http.StatusOK, // status code is used only for rest at the moment
		RelayReply: &pairingtypes.RelayReply{
			Data: body,
		},
	}

	// checking if rest reply data is in json format
	err = cp.HandleJSONFormatError(reply.RelayReply.Data)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Tendermint reply is neither a JSON object nor a JSON array of objects", nil, utils.Attribute{Key: "reply.Data", Value: string(reply.RelayReply.Data)})
	}

	return reply, "", nil, nil
}

// SendRPC sends Tendermint HTTP or WebSockets call
func (cp *tendermintRpcChainProxy) SendRPC(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get rpc connection from the connection pool
	var rpc *rpcclient.Client
	rpc, err = cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, err
	}
	// return the rpc connection to the websocket pool after the function completes
	defer cp.conn.ReturnRpc(rpc)

	// appending hashed url
	grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeAddressHash, cp.conn.GetUrlHash()))

	// create variables for the rpc message and reply message
	var rpcMessage *rpcclient.JsonrpcMessage
	var sub *rpcclient.ClientSubscription
	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			rpc.SetHeader(metadata.Name, metadata.Value)
			// clear this header upon function completion so it doesn't last in the next usage from the rpc pool
			defer rpc.SetHeader(metadata.Name, "")
		}
	}
	// If ch is not nil do subscription
	var nodeErr error
	if ch != nil {
		// subscribe to the rpc call if the channel is not nil
		utils.LavaFormatTrace("Sending subscription",
			utils.LogAttr("chainID", cp.BaseChainProxy.ChainID),
			utils.LogAttr("apiName", chainMessage.GetApi().Name),
			utils.LogAttr("nodeMessage.ID", nodeMessage.ID),
			utils.LogAttr("nodeMessage.Method", nodeMessage.Method),
			utils.LogAttr("nodeMessage.Params", nodeMessage.Params),
		)
		sub, rpcMessage, nodeErr = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		// set context with timeout
		connectCtx, cancel := cp.CapTimeoutForSend(ctx, chainMessage)
		defer cancel()

		cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)
		// perform the rpc call
		rpcMessage, nodeErr = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params, false, nodeMessage.GetDisableErrorHandling())
		if nodeErr != nil {
			if common.StatusCodeError504.Is(nodeErr) || common.StatusCodeError429.Is(nodeErr) || common.StatusCodeErrorStrict.Is(nodeErr) {
				return nil, "", nil, utils.LavaFormatWarning("Received invalid status code", nodeErr, utils.Attribute{Key: "chainID", Value: cp.BaseChainProxy.ChainID}, utils.Attribute{Key: "apiName", Value: chainMessage.GetApi().Name})
			}
			// Validate if the error is related to the provider connection to the node or it is a valid error
			// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
			if parsedError := cp.HandleNodeError(ctx, nodeErr); parsedError != nil {
				return nil, "", nil, parsedError
			}
		}
	}

	var replyMsg *rpcInterfaceMessages.RPCResponse
	// the error check here would only wrap errors not from the rpc
	if nodeErr != nil {
		rpcMessage = TryRecoverNodeErrorFromClientError(nodeErr)
		if rpcMessage == nil {
			utils.LavaFormatDebug("got error from node", utils.LogAttr("GUID", ctx), utils.LogAttr("nodeErr", nodeErr))
			return nil, "", nil, nodeErr
		}
	}

	replyMsg, err = rpcInterfaceMessages.ConvertTendermintMsg(rpcMessage)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("tendermintRPC error", err)
	}

	// if we didn't get a node error.
	if replyMsg.Error == nil {
		// validate result is valid
		responseIsNilValidationError := ValidateNilResponse(string(replyMsg.Result))
		if responseIsNilValidationError != nil {
			return nil, "", nil, responseIsNilValidationError
		}
	}
	err = cp.ValidateRequestAndResponseIds(nodeMessage.ID, rpcMessage.ID)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("tendermintRPC ID mismatch error", err,
			utils.Attribute{Key: "GUID", Value: ctx},
			utils.Attribute{Key: "requestId", Value: nodeMessage.ID},
			utils.Attribute{Key: "responseId", Value: rpcMessage.ID},
		)
	}

	// marshal the jsonrpc message to json
	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct
	reply := &RelayReplyWrapper{
		StatusCode: http.StatusOK, // status code is used only for rest at the moment
		RelayReply: &pairingtypes.RelayReply{
			Data: data,
		},
	}

	if ch != nil {
		// get the params for the rpc call
		params := nodeMessage.Params

		paramsMap, ok := params.(map[string]interface{})
		if !ok {
			return nil, "", nil, utils.LavaFormatError("unknown params type on tendermint subscribe", nil)
		}
		subscriptionID, ok = paramsMap["query"].(string)
		if !ok {
			utils.LavaFormatTrace("could not get subscriptionID from query params", utils.LogAttr("params", params))
			// This is probably because of a misuse, therefore the provider will return a node error to the user as the subscription failed
			subscriptionID = ""
		}
	}

	return reply, subscriptionID, sub, err
}
