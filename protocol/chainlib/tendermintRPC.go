package chainlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type TendermintChainParser struct {
	BaseChainParser
}

// NewTendermintRpcChainParser creates a new instance of TendermintChainParser
func NewTendermintRpcChainParser() (chainParser *TendermintChainParser, err error) {
	return &TendermintChainParser{}, nil
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
	return apip.BaseChainParser.getSupportedApi(name, connectionType)
}

func (apip *TendermintChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		chainMessage, err := apip.ParseMsg("", craftData.Data, craftData.ConnectionType, metadata, 0)
		chainMessage.AppendHeader(metadata)
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
	return apip.newChainMessage(apiCont.api, spectypes.NOT_APPLICABLE, &tenderMsg, apiCollection), nil
}

// ParseMsg parses message data into chain message object
func (apip *TendermintChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, latestBlock uint64) (ChainMessage, error) {
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
		var parsedMethod string
		idx := strings.Index(url, "?")
		if idx == -1 {
			parsedMethod = url
		} else {
			parsedMethod = url[0:idx]
		}

		msg := rpcInterfaceMessages.JsonrpcMessage{
			ID:      []byte("1"),
			Version: "2.0",
			Method:  parsedMethod,
		}
		if strings.Contains(url[idx+1:], "=") {
			params := make(map[string]interface{})
			rawParams := strings.Split(url[idx+1:], "&") // list with structure ['height=0x500',...]
			for _, param := range rawParams {
				splitParam := strings.Split(param, "=")
				if len(splitParam) != 2 {
					return nil, utils.LavaFormatError("Cannot parse query params", nil, utils.Attribute{Key: "params", Value: param})
				}
				params[splitParam[0]] = splitParam[1]
			}
			msg.Params = params
		} else {
			msg.Params = make(map[string]interface{}, 0)
		}
		msgs = []rpcInterfaceMessages.JsonrpcMessage{msg}
	}
	if len(msgs) == 0 {
		return nil, errors.New("empty unmarshaled json")
	}

	var api *spectypes.Api
	var apiCollection *spectypes.ApiCollection
	var latestRequestedBlock, earliestRequestedBlock int64 = 0, 0
	for idx, msg := range msgs {
		var requestedBlockForMessage int64
		// Check api is supported and save it in nodeMsg
		apiCont, err := apip.getSupportedApi(msg.Method, connectionType)
		if err != nil {
			return nil, utils.LavaFormatError("getSupportedApi jsonrpc failed", err, utils.Attribute{Key: "method", Value: msg.Method})
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
			requestedBlockForMessage, err = parser.ParseBlockFromParams(msg, apiCont.api.BlockParsing)
			if err != nil {
				return nil, utils.LavaFormatError("ParseBlockFromParams failed parsing block", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "blockParsing", Value: apiCont.api.BlockParsing})
			}
		} else {
			requestedBlockForMessage, err = msg.ParseBlock(overwriteReqBlock)
			if err != nil {
				return nil, utils.LavaFormatError("failed parsing block from an overwrite header", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "overwriteReqBlock", Value: overwriteReqBlock})
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

	var nodeMsg *parsedMessage
	if len(msgs) == 1 {
		tenderMsg := rpcInterfaceMessages.TendermintrpcMessage{JsonrpcMessage: msgs[0], Path: ""}
		if !isJsonrpc {
			tenderMsg.Path = url // add path
		}
		nodeMsg = apip.newChainMessage(api, latestRequestedBlock, &tenderMsg, apiCollection)
	} else {
		var err error
		nodeMsg, err = apip.newBatchChainMessage(api, latestRequestedBlock, earliestRequestedBlock, msgs, apiCollection)
		if err != nil {
			return nil, err
		}
	}

	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, latestBlock)
	return nodeMsg, nil
}

func (*TendermintChainParser) newBatchChainMessage(serviceApi *spectypes.Api, requestedBlock int64, earliestRequestedBlock int64, msgs []rpcInterfaceMessages.JsonrpcMessage, apiCollection *spectypes.ApiCollection) (*parsedMessage, error) {
	batchMessage, err := rpcInterfaceMessages.NewBatchMessage(msgs)
	if err != nil {
		return nil, err
	}
	nodeMsg := &parsedMessage{
		api:                    serviceApi,
		apiCollection:          apiCollection,
		latestRequestedBlock:   requestedBlock,
		msg:                    &batchMessage,
		earliestRequestedBlock: earliestRequestedBlock,
	}
	return nodeMsg, err
}

func (*TendermintChainParser) newChainMessage(serviceApi *spectypes.Api, requestedBlock int64, msg *rpcInterfaceMessages.TendermintrpcMessage, apiCollection *spectypes.ApiCollection) *parsedMessage {
	nodeMsg := &parsedMessage{
		api:                  serviceApi,
		apiCollection:        apiCollection,
		latestRequestedBlock: requestedBlock,
		msg:                  msg,
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
	serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceTendermintRPC)
	apip.BaseChainParser.Construct(spec, taggedApis, serverApis, apiCollections, headers, verifications)
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
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *metrics.RPCConsumerLogs
}

// NewTendermintRpcChainListener creates a new instance of TendermintRpcChainListener
func NewTendermintRpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *metrics.RPCConsumerLogs) (chainListener *TendermintRpcChainListener) {
	// Create a new instance of JsonRPCChainListener
	chainListener = &TendermintRpcChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

// Serve http server for TendermintRpcChainListener
func (apil *TendermintRpcChainListener) Serve(ctx context.Context) {
	// Guard that the TendermintChainParser instance exists
	if apil == nil {
		return
	}

	// Setup HTTP Server
	app := fiber.New(fiber.Config{})
	chainID := apil.endpoint.ChainID
	apiInterface := apil.endpoint.ApiInterface

	app.Use(favicon.New())

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	webSocketCallback := websocket.New(func(c *websocket.Conn) {
		var (
			mt  int
			msg []byte
			err error
		)
		msgSeed := apil.logger.GetMessageSeed()
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				break
			}
			dappID, ok := c.Locals("dappId").(string)
			if !ok {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, nil, msgSeed, []byte("Unable to extract dappID"), spectypes.APIInterfaceJsonRPC)
			}

			ctx, cancel := context.WithCancel(context.Background())
			ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
			defer cancel() // incase there's a problem make sure to cancel the connection
			utils.LavaFormatInfo("ws in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: msg}, utils.Attribute{Key: "dappID", Value: dappID})

			metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
			reply, replyServer, err := apil.relaySender.SendRelay(ctx, "", string(msg), "", dappID, metricsData, nil)
			go apil.logger.AddMetricForWebSocket(metricsData, err, c)
			if err != nil {
				apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
				continue
			}
			// If subscribe the first reply would contain the RPC ID that can be used for disconnect.
			if replyServer != nil {
				var reply pairingtypes.RelayReply
				err = (*replyServer).RecvMsg(&reply) // this reply contains the RPC ID
				if err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}

				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				for {
					err = (*replyServer).RecvMsg(&reply)
					if err != nil {
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						break
					}

					// If portal cant write to the client
					if err = c.WriteMessage(mt, reply.Data); err != nil {
						cancel()
						apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
						// break
					}
					apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
				}
			} else {
				if err = c.WriteMessage(mt, reply.Data); err != nil {
					apil.logger.AnalyzeWebSocketErrorAndWriteMessage(c, mt, err, msgSeed, msg, "tendermint")
					continue
				}
				apil.logger.LogRequestAndResponse("tendermint ws", false, "ws", c.LocalAddr().String(), string(msg), string(reply.Data), msgSeed, nil)
			}
		}
	})
	websocketCallbackWithDappID := constructFiberCallbackWithHeaderAndParameterExtraction(webSocketCallback, apil.logger.StoreMetricData)
	app.Get("/ws", websocketCallbackWithDappID)
	app.Get("/websocket", websocketCallbackWithDappID) // catching http://HOST:PORT/1/websocket requests.

	app.Post("/*", func(c *fiber.Ctx) error {
		// Set response header content-type to application/json
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()
		msgSeed := apil.logger.GetMessageSeed()
		dappID := extractDappIDFromFiberContext(c)
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection

		utils.LavaFormatInfo("in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: c.Body()}, utils.Attribute{Key: "dappID", Value: dappID})
		reply, _, err := apil.relaySender.SendRelay(ctx, "", string(c.Body()), "", dappID, metricsData, nil)
		go apil.logger.AddMetricForHttp(metricsData, err, c.GetReqHeaders())

		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "POST", c.Request().URI().String(), string(c.Body()), errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			// Construct json response
			response := rpcInterfaceMessages.ConvertToTendermintError(errMasking, c.Body())

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "POST", c.Request().URI().String(), string(c.Body()), string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})

	app.Get("/*", func(c *fiber.Ctx) error {
		// Set response header content-type to application/json
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

		endTx := apil.logger.LogStartTransaction("tendermint-WebSocket")
		defer endTx()

		query := "?" + string(c.Request().URI().QueryString())
		path := c.Params("*")
		dappID := extractDappIDFromFiberContext(c)
		msgSeed := apil.logger.GetMessageSeed()
		ctx, cancel := context.WithCancel(context.Background())
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		defer cancel() // incase there's a problem make sure to cancel the connection
		utils.LavaFormatInfo("urirpc in <<<", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "seed", Value: msgSeed}, utils.Attribute{Key: "msg", Value: path}, utils.Attribute{Key: "dappID", Value: dappID})
		metricsData := metrics.NewRelayAnalytics(dappID, chainID, apiInterface)
		reply, _, err := apil.relaySender.SendRelay(ctx, path+query, "", "", dappID, metricsData, nil)
		go apil.logger.AddMetricForHttp(metricsData, err, c.GetReqHeaders())

		if err != nil {
			// Get unique GUID response
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)

			// Log request and response
			apil.logger.LogRequestAndResponse("tendermint http in/out", true, "GET", c.Request().URI().String(), "", errMasking, msgSeed, err)

			// Set status to internal error
			c.Status(fiber.StatusInternalServerError)

			if string(c.Body()) != "" {
				errMasking = addAttributeToError("recommendation", "For jsonRPC use POST", errMasking)
			}

			// Construct json response
			response := convertToJsonError(errMasking)

			// Return error json response
			return c.SendString(response)
		}
		// Log request and response
		apil.logger.LogRequestAndResponse("tendermint http in/out", false, "GET", c.Request().URI().String(), "", string(reply.Data), msgSeed, nil)

		// Return json response
		return c.SendString(string(reply.Data))
	})
	//
	// Go
	ListenWithRetry(app, apil.endpoint.NetworkAddress)
}

type tendermintRpcChainProxy struct {
	// embedding the jrpc chain proxy because the only diff is on parse message
	JrpcChainProxy
	httpNodeUrl   common.NodeUrl
	httpConnector *chainproxy.Connector
}

func NewtendermintRpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, chainParser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := chainParser.ChainBlockStats()
	websocketUrl, httpUrl := verifyTendermintEndpoint(rpcProviderEndpoint.NodeUrls)
	cp := &tendermintRpcChainProxy{
		JrpcChainProxy: JrpcChainProxy{BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime, NodeUrl: websocketUrl, ErrorHandler: &TendermintRPCErrorHandler{}}, conn: map[string]*chainproxy.Connector{}},
		httpNodeUrl:    httpUrl,
		httpConnector:  nil,
	}
	cp.addHttpConnector(ctx, nConns, httpUrl)
	return cp, cp.start(ctx, nConns, websocketUrl, nil)
}

func (cp *tendermintRpcChainProxy) addHttpConnector(ctx context.Context, nConns uint, nodeUrl common.NodeUrl) error {
	conn, err := chainproxy.NewConnector(ctx, nConns, nodeUrl)
	if err != nil {
		return err
	}
	cp.httpConnector = conn
	if cp.httpConnector == nil {
		return errors.New("g_conn == nil")
	}
	return nil
}

func (cp *tendermintRpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
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

func (cp *tendermintRpcChainProxy) SendURI(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// check if the input channel is not nil
	if ch != nil {
		// return an error if the channel is not nil
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on Tendermint URI", nil)
	}

	// create a new http client with a timeout set by the getTimePerCu function
	httpClient := http.Client{
		Timeout: common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits),
	}

	// construct the url by concatenating the node url with the path variable
	url := cp.httpNodeUrl.Url + "/" + nodeMessage.Path

	// create context
	relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits)
	// check if this API is hanging (waiting for block confirmation)
	if chainMessage.GetApi().Category.HangingApi {
		relayTimeout += cp.averageBlockTime
	}
	connectCtx, cancel := cp.httpNodeUrl.LowerContextTimeout(ctx, relayTimeout)
	defer cancel()

	// create a new http request
	req, err := http.NewRequestWithContext(connectCtx, http.MethodGet, cp.httpNodeUrl.AuthConfig.AddAuthPath(url), nil)
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

	cp.httpNodeUrl.SetAuthHeaders(ctx, req.Header.Set)

	cp.httpNodeUrl.SetIpForwardingIfNecessary(ctx, req.Header.Set)
	// send the http request and get the response
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}

	// close the response body
	if res.Body != nil {
		defer res.Body.Close()
	}

	err = cp.HandleStatusError(res.StatusCode)
	if err != nil {
		return nil, "", nil, err
	}

	// read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct with the response body as the data
	reply := &pairingtypes.RelayReply{
		Data: body,
	}

	// checking if rest reply data is in json format
	err = cp.HandleJSONFormatError(reply.Data)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Tendermint reply is neither a JSON object nor a JSON array of objects", nil, utils.Attribute{Key: "reply.Data", Value: string(reply.Data)})
	}

	return reply, "", nil, nil
}

// SendRPC sends Tendermint HTTP or WebSockets call
func (cp *tendermintRpcChainProxy) SendRPC(ctx context.Context, nodeMessage *rpcInterfaceMessages.TendermintrpcMessage, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	// Get rpc connection from the connection pool
	var rpc *rpcclient.Client
	if ch != nil {
		internalPath := chainMessage.GetApiCollection().CollectionData.InternalPath
		rpc, err = cp.conn[internalPath].GetRpc(ctx, true)
		if err != nil {
			return nil, "", nil, err
		}
		// return the rpc connection to the websocket pool after the function completes
		defer cp.conn[internalPath].ReturnRpc(rpc)
	} else {
		rpc, err = cp.httpConnector.GetRpc(ctx, true)
		if err != nil {
			return nil, "", nil, err
		}
		// return the rpc connection to the http pool after the function completes
		defer cp.httpConnector.ReturnRpc(rpc)
	}

	// create variables for the rpc message and reply message
	var rpcMessage *rpcclient.JsonrpcMessage
	var replyMessage *rpcInterfaceMessages.RPCResponse
	var sub *rpcclient.ClientSubscription
	if len(nodeMessage.GetHeaders()) > 0 {
		for _, metadata := range nodeMessage.GetHeaders() {
			rpc.SetHeader(metadata.Name, metadata.Value)
			// clear this header upon function completion so it doesn't last in the next usage from the rpc pool
			defer rpc.SetHeader(metadata.Name, "")
		}
	}
	// If ch is not nil do subscription
	if ch != nil {
		// subscribe to the rpc call if the channel is not nil
		sub, rpcMessage, err = rpc.Subscribe(context.Background(), nodeMessage.ID, nodeMessage.Method, ch, nodeMessage.Params)
	} else {
		// create a context with a timeout set by the LocalNodeTimePerCu function
		relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetApi().ComputeUnits)
		// check if this API is hanging (waiting for block confirmation)
		if chainMessage.GetApi().Category.HangingApi {
			relayTimeout += cp.averageBlockTime
		}
		cp.NodeUrl.SetIpForwardingIfNecessary(ctx, rpc.SetHeader)

		connectCtx, cancel := cp.NodeUrl.LowerContextTimeout(ctx, relayTimeout)
		defer cancel()
		// perform the rpc call
		rpcMessage, err = rpc.CallContext(connectCtx, nodeMessage.ID, nodeMessage.Method, nodeMessage.Params, false)
		if err != nil {
			// Validate if the error is related to the provider connection to the node or it is a valid error
			// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
			if parsedError := cp.HandleNodeError(ctx, err); parsedError != nil {
				return nil, "", nil, parsedError
			}
		}
	}

	var replyMsg *rpcInterfaceMessages.RPCResponse
	// the error check here would only wrap errors not from the rpc
	if err != nil {
		utils.LavaFormatDebug("received an error from SendNodeMsg", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "error", Value: err})
		return nil, "", nil, err
	} else {
		replyMessage, err = rpcInterfaceMessages.ConvertTendermintMsg(rpcMessage)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermingRPC error", err)
		}

		replyMsg = replyMessage

		err := cp.ValidateRequestAndResponseIds(nodeMessage.ID, rpcMessage.ID)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("tendermintRPC ID mismatch error", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}

	// marshal the jsonrpc message to json
	data, err := json.Marshal(replyMsg)
	if err != nil {
		return nil, "", nil, err
	}

	// create a new relay reply struct
	reply := &pairingtypes.RelayReply{
		Data: data,
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
			return nil, "", nil, utils.LavaFormatError("unknown subscriptionID type on tendermint subscribe", nil)
		}
	}

	return reply, subscriptionID, sub, err
}
