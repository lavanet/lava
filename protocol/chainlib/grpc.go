package chainlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v4/protocol/chainlib/grpcproxy"
	dyncodec "github.com/lavanet/lava/v4/protocol/chainlib/grpcproxy/dyncodec"
	"github.com/lavanet/lava/v4/protocol/parser"
	protocoltypes "github.com/lavanet/lava/v4/x/protocol/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	reflectionpbo "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

const GRPCStatusCodeOnFailedMessages = 32

type GrpcNodeErrorResponse struct {
	ErrorMessage string `json:"error_message"`
	ErrorCode    uint32 `json:"error_code"`
}

type GrpcChainParser struct {
	BaseChainParser

	registry *dyncodec.Registry
	codec    *dyncodec.Codec
}

// NewGrpcChainParser creates a new instance of GrpcChainParser
func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return &GrpcChainParser{}, nil
}

func (bcp *GrpcChainParser) GetUniqueName() string {
	return "grpc_chain_parser"
}

func (apip *GrpcChainParser) getApiCollection(connectionType, internalPath, addon string) (*spectypes.ApiCollection, error) {
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	return apip.BaseChainParser.getApiCollection(connectionType, internalPath, addon)
}

func (apip *GrpcChainParser) getSupportedApi(name, connectionType string) (*ApiContainer, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("ChainParser not defined")
	}
	apiKey := ApiKey{Name: name, ConnectionType: connectionType}
	return apip.BaseChainParser.getSupportedApi(apiKey)
}

func (apip *GrpcChainParser) setupForConsumer(relayer grpcproxy.ProxyCallBack) {
	apip.registry = dyncodec.NewRegistry(dyncodec.NewRelayerRemote(relayer))
	apip.codec = dyncodec.NewCodec(apip.registry)
}

func (apip *GrpcChainParser) setupForProvider(reflectionConnection *grpc.ClientConn) error {
	remote := dyncodec.NewGRPCReflectionProtoFileRegistryFromConn(reflectionConnection)
	apip.registry = dyncodec.NewRegistry(remote)
	apip.codec = dyncodec.NewCodec(apip.registry)
	return nil
}

func (apip *GrpcChainParser) CraftMessage(parsing *spectypes.ParseDirective, connectionType string, craftData *CraftData, metadata []pairingtypes.Metadata) (ChainMessageForSend, error) {
	if craftData != nil {
		chainMessage, err := apip.ParseMsg(craftData.Path, craftData.Data, craftData.ConnectionType, metadata, extensionslib.ExtensionInfo{LatestBlock: 0})
		if err == nil {
			chainMessage.AppendHeader(metadata)
		}
		return chainMessage, err
	}

	grpcMessage := &rpcInterfaceMessages.GrpcMessage{
		Msg:         nil,
		Path:        parsing.ApiName,
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
	parsedInput := &parser.ParsedInput{}
	parsedInput.SetBlock(spectypes.NOT_APPLICABLE)
	return apip.newChainMessage(apiCont.api, parsedInput, grpcMessage, apiCollection), nil
}

// ParseMsg parses message data into chain message object
func (apip *GrpcChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (ChainMessage, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("GrpcChainParser not defined")
	}

	// Check API is supported and save it in nodeMsg.
	apiCont, err := apip.getSupportedApi(url, connectionType)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err, utils.LogAttr("url", url), utils.LogAttr("connectionType", connectionType))
	}

	apiCollection, err := apip.getApiCollection(connectionType, apiCont.collectionKey.InternalPath, apiCont.collectionKey.Addon)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getApiCollection gRPC", err)
	}

	// handle headers
	metadata, overwriteReqBlock, _ := apip.HandleHeaders(metadata, apiCollection, spectypes.Header_pass_send)

	settingHeaderDirective, _, _ := apip.GetParsingByTag(spectypes.FUNCTION_TAG_SET_LATEST_IN_METADATA)

	// Construct grpcMessage
	grpcMessage := rpcInterfaceMessages.GrpcMessage{
		Msg:         data,
		Path:        url,
		Codec:       apip.codec,
		Registry:    apip.registry,
		BaseMessage: chainproxy.BaseMessage{Headers: metadata, LatestBlockHeaderSetter: settingHeaderDirective},
	}

	// // Fetch requested block, it is used for data reliability
	// // Extract default block parser
	api := apiCont.api
	parsedInput := parser.NewParsedInput()
	if overwriteReqBlock == "" {
		parsedInput = parser.ParseBlockFromParams(grpcMessage, api.BlockParsing, api.Parsers)
	} else {
		parsedBlock, err := grpcMessage.ParseBlock(overwriteReqBlock)
		parsedInput.SetBlock(parsedBlock)
		if err != nil {
			utils.LavaFormatError("failed parsing block from an overwrite header", err,
				utils.LogAttr("chain", apip.spec.Name),
				utils.LogAttr("overwriteRequestedBlock", overwriteReqBlock),
			)
			parsedInput.SetBlock(spectypes.NOT_APPLICABLE)
		} else {
			parsedInput.UsedDefaultValue = false
		}
	}

	nodeMsg := apip.newChainMessage(apiCont.api, parsedInput, &grpcMessage, apiCollection)
	apip.BaseChainParser.ExtensionParsing(apiCollection.CollectionData.AddOn, nodeMsg, extensionInfo)
	return nodeMsg, apip.BaseChainParser.Validate(nodeMsg)
}

func (*GrpcChainParser) newChainMessage(api *spectypes.Api, parsedInput *parser.ParsedInput, grpcMessage *rpcInterfaceMessages.GrpcMessage, apiCollection *spectypes.ApiCollection) *baseChainMessageContainer {
	requestedBlock := parsedInput.GetBlock()
	requestedHashes, _ := parsedInput.GetBlockHashes()
	nodeMsg := &baseChainMessageContainer{
		api:                      api,
		msg:                      grpcMessage, // setting the grpc message as a pointer so we can set descriptors for parsing
		latestRequestedBlock:     requestedBlock,
		requestedBlockHashes:     requestedHashes,
		apiCollection:            apiCollection,
		resultErrorParsingMethod: grpcMessage.CheckResponseError,
		parseDirective:           GetParseDirective(api, apiCollection),
		usedDefaultValue:         parsedInput.UsedDefaultValue,
	}
	return nodeMsg
}

// SetSpec sets the spec for the GrpcChainParser
func (apip *GrpcChainParser) SetSpec(spec spectypes.Spec) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return
	}

	// Add a read-write lock to ensure thread safety
	apip.rwLock.Lock()
	defer apip.rwLock.Unlock()

	// extract server and tagged apis from spec
	internalPaths, serverApis, taggedApis, apiCollections, headers, verifications := getServiceApis(spec, spectypes.APIInterfaceGrpc)
	apip.BaseChainParser.Construct(spec, internalPaths, taggedApis, serverApis, apiCollections, headers, verifications)
}

// DataReliabilityParams returns data reliability params from spec (spec.enabled and spec.dataReliabilityThreshold)
func (apip *GrpcChainParser) DataReliabilityParams() (enabled bool, dataReliabilityThreshold uint32) {
	// Guard that the GrpcChainParser instance exists
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
func (apip *GrpcChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32) {
	// Guard that the GrpcChainParser instance exists
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

type GrpcChainListener struct {
	endpoint         *lavasession.RPCEndpoint
	relaySender      RelaySender
	logger           *metrics.RPCConsumerLogs
	chainParser      *GrpcChainParser
	healthReporter   HealthReporter
	refererData      *RefererData
	listeningAddress string
}

func NewGrpcChainListener(
	ctx context.Context,
	listenEndpoint *lavasession.RPCEndpoint,
	relaySender RelaySender,
	healthReporter HealthReporter,
	rpcConsumerLogs *metrics.RPCConsumerLogs,
	chainParser ChainParser,
	refererData *RefererData,
) (chainListener *GrpcChainListener) {
	// Create a new instance of GrpcChainListener
	chainListener = &GrpcChainListener{
		endpoint:       listenEndpoint,
		relaySender:    relaySender,
		logger:         rpcConsumerLogs,
		chainParser:    chainParser.(*GrpcChainParser),
		healthReporter: healthReporter,
		refererData:    refererData,
	}
	return chainListener
}

// Serve http server for GrpcChainListener
func (apil *GrpcChainListener) Serve(ctx context.Context, cmdFlags common.ConsumerCmdFlags) {
	// Guard that the GrpcChainListener instance exists
	if apil == nil {
		return
	}

	lis := GetListenerWithRetryGrpc("tcp", apil.endpoint.NetworkAddress)
	apil.listeningAddress = lis.Addr().String()
	apiInterface := apil.endpoint.ApiInterface
	sendRelayCallback := func(ctx context.Context, method string, reqBody []byte) ([]byte, metadata.MD, error) {
		if method == "grpc.reflection.v1.ServerReflection/ServerReflectionInfo" {
			return nil, nil, status.Error(codes.Unimplemented, "v1 reflection currently not supported by cosmos-sdk")
		}

		guid := utils.GenerateUniqueIdentifier()
		ctx = utils.WithUniqueIdentifier(ctx, guid)
		msgSeed := strconv.FormatUint(guid, 10)
		metadataValues, _ := metadata.FromIncomingContext(ctx)
		startTime := time.Now()
		// Extract dappID from grpc header
		dappID := extractDappIDFromGrpcHeader(metadataValues)

		grpcHeaders := convertToMetadataMapOfSlices(metadataValues)
		utils.LavaFormatDebug("in <<< GRPC Relay ",
			utils.LogAttr("GUID", ctx),
			utils.LogAttr("_method", method),
			utils.LogAttr("headers", grpcHeaders),
		)
		metricsData := metrics.NewRelayAnalytics(dappID, apil.endpoint.ChainID, apiInterface)
		metricsData.SetProcessingTimestampBeforeRelay(startTime)
		consumerIp := common.GetIpFromGrpcContext(ctx)
		relayResult, err := apil.relaySender.SendRelay(ctx, method, string(reqBody), "", dappID, consumerIp, metricsData, grpcHeaders)
		relayReply := relayResult.GetReply()
		go apil.logger.AddMetricForGrpc(metricsData, err, &metadataValues)

		if err != nil {
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)
			apil.logger.LogRequestAndResponse("grpc in/out", true, method, string(reqBody), "", errMasking, msgSeed, time.Since(startTime), err)
			return nil, nil, utils.LavaFormatError("Failed to SendRelay", fmt.Errorf("%s", errMasking))
		}
		apil.logger.LogRequestAndResponse("grpc in/out", false, method, string(reqBody), "", "", msgSeed, time.Since(startTime), nil)
		apil.logger.AddMetricForProcessingLatencyAfterProvider(metricsData, apil.endpoint.ChainID, apiInterface)

		// try checking for node errors.
		nodeError := &GrpcNodeErrorResponse{}
		unMarshalingError := json.Unmarshal(relayReply.Data, nodeError)
		metadataToReply := relayReply.Metadata
		if unMarshalingError == nil {
			return nil, convertRelayMetaDataToMDMetaData(metadataToReply), status.Error(codes.Code(nodeError.ErrorCode), nodeError.ErrorMessage)
		}
		return relayReply.Data, convertRelayMetaDataToMDMetaData(metadataToReply), nil
	}

	_, httpServer, err := grpcproxy.NewGRPCProxy(sendRelayCallback, apil.endpoint.HealthCheckPath, cmdFlags, apil.healthReporter)
	if err != nil {
		utils.LavaFormatFatal("provider failure RegisterServer", err, utils.Attribute{Key: "listenAddr", Value: apil.endpoint.NetworkAddress})
	}

	// setup chain parser
	apil.chainParser.setupForConsumer(sendRelayCallback)

	utils.LavaFormatInfo("Server listening", utils.Attribute{Key: "Address", Value: lis.Addr()})

	var serveExecutor func() error
	if apil.endpoint.TLSEnabled {
		utils.LavaFormatInfo("Running with self signed TLS certificate")
		var certificateErr error
		httpServer.TLSConfig, certificateErr = lavasession.GetSelfSignedConfig()
		if certificateErr != nil {
			utils.LavaFormatFatal("failed getting a self signed certificate", certificateErr)
		}
		serveExecutor = func() error { return httpServer.ServeTLS(lis, "", "") }
	} else {
		utils.LavaFormatInfo("Running with disabled TLS configuration")
		serveExecutor = func() error { return httpServer.Serve(lis) }
	}

	fmt.Printf(`
 ┌───────────────────────────────────────────────────┐ 
 │               Lava's Grpc Server                  │ 
 │               %s│ 
 │               Lavap Version: %s│ 
 └───────────────────────────────────────────────────┘

`, truncateAndPadString(apil.endpoint.NetworkAddress, 36), truncateAndPadString(protocoltypes.DefaultVersion.ConsumerTarget, 21))
	if err := serveExecutor(); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Portal failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr()}, utils.Attribute{Key: "ChainID", Value: apil.endpoint.ChainID})
	}
}

func (apil *GrpcChainListener) GetListeningAddress() string {
	return apil.listeningAddress
}

type GrpcChainProxy struct {
	BaseChainProxy
	conn             grpcConnectorInterface
	descriptorsCache *common.SafeSyncMap[string, *desc.MethodDescriptor]
}
type grpcConnectorInterface interface {
	Close()
	GetRpc(ctx context.Context, block bool) (*grpc.ClientConn, error)
	ReturnRpc(rpc *grpc.ClientConn)
}

func NewGrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint lavasession.RPCProviderEndpoint, parser ChainParser) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	_, averageBlockTime, _, _ := parser.ChainBlockStats()
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	nodeUrl.Url = strings.TrimSuffix(nodeUrl.Url, "/") // remove suffix if exists
	conn, err := chainproxy.NewGRPCConnector(ctx, nConns, nodeUrl)
	if err != nil {
		return nil, err
	}
	return newGrpcChainProxy(ctx, averageBlockTime, parser, conn, rpcProviderEndpoint)
}

func newGrpcChainProxy(ctx context.Context, averageBlockTime time.Duration, parser ChainParser, conn grpcConnectorInterface, rpcProviderEndpoint lavasession.RPCProviderEndpoint) (ChainProxy, error) {
	cp := &GrpcChainProxy{
		BaseChainProxy:   BaseChainProxy{averageBlockTime: averageBlockTime, ErrorHandler: &GRPCErrorHandler{}, ChainID: rpcProviderEndpoint.ChainID, HashedNodeUrl: chainproxy.HashURL(rpcProviderEndpoint.NodeUrls[0].Url)},
		descriptorsCache: &common.SafeSyncMap[string, *desc.MethodDescriptor]{},
	}
	cp.conn = conn
	if cp.conn == nil {
		return nil, utils.LavaFormatError("g_conn == nil", nil)
	}

	reflectionConnection, err := conn.GetRpc(context.Background(), true)
	if err != nil {
		return nil, utils.LavaFormatError("reflectionConnection Error", err)
	}
	// this connection is kept open so it needs to be closed on teardown
	go func() {
		<-ctx.Done()
		utils.LavaFormatInfo("tearing down reflection connection, context done")
		conn.ReturnRpc(reflectionConnection)
	}()

	err = parser.(*GrpcChainParser).setupForProvider(reflectionConnection)
	if err != nil {
		return nil, fmt.Errorf("grpc chain proxy: failed to setup parser: %w", err)
	}
	return cp, nil
}

func (cp *GrpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend) (relayReply *RelayReplyWrapper, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on grpc", nil, utils.Attribute{Key: "GUID", Value: ctx})
	}
	conn, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("grpc get connection failed ", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	defer cp.conn.ReturnRpc(conn)

	// appending hashed url
	grpc.SetTrailer(ctx, metadata.Pairs(RPCProviderNodeAddressHash, cp.BaseChainProxy.HashedNodeUrl))

	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.GrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in grpc failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage})
	}

	metadataMap := make(map[string]string, 0)
	for _, metaData := range nodeMessage.GetHeaders() {
		metadataMap[metaData.Name] = metaData.Value
	}

	if len(metadataMap) > 0 {
		md := metadata.New(metadataMap)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	cl := grpcreflect.NewClient(ctx, reflectionpbo.NewServerReflectionClient(conn))
	descriptorSource := rpcInterfaceMessages.DescriptorSourceFromServer(cl)
	svc, methodName := rpcInterfaceMessages.ParseSymbol(nodeMessage.Path)

	// Check if we have method descriptor already cached.
	// The reason we do Load and then Store here, instead of LoadOrStore:
	// On the worst case scenario, where 2 threads are accessing the map at the same time, the same descriptor will be stored twice.
	// It is better than the alternative, which is always creating the descriptor, since the outcome is the same.
	methodDescriptor, found, _ := cp.descriptorsCache.Load(methodName)
	if !found { // method descriptor not cached yet, need to fetch it and add to cache
		var descriptor desc.Descriptor
		if descriptor, err = descriptorSource.FindSymbol(svc); err != nil {
			return nil, "", nil, utils.LavaFormatError("descriptorSource.FindSymbol", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
		serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
		if !ok {
			return nil, "", nil, utils.LavaFormatError("serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "descriptor", Value: descriptor})
		}
		methodDescriptor = serviceDescriptor.FindMethodByName(methodName)
		if methodDescriptor == nil {
			return nil, "", nil, utils.LavaFormatError("serviceDescriptor.FindMethodByName returned nil", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "methodName", Value: methodName})
		}

		// add the descriptor to the chainProxy cache
		cp.descriptorsCache.Store(methodName, methodDescriptor)
	}

	msgFactory := dynamic.NewMessageFactoryWithDefaults()

	var reader io.Reader
	msg := msgFactory.NewMessage(methodDescriptor.GetInputType())
	formatMessage := false
	if len(nodeMessage.Msg) > 0 {
		// guess if json or binary
		if nodeMessage.Msg[0] != '{' && nodeMessage.Msg[0] != '[' {
			msgLocal := msgFactory.NewMessage(methodDescriptor.GetInputType())
			err = proto.Unmarshal(nodeMessage.Msg, msgLocal)
			if err != nil {
				return nil, "", nil, utils.LavaFormatError("Failed to unmarshal proto.Unmarshal(nodeMessage.Msg, msgLocal)", err)
			}
			jsonBytes, err := marshalJSON(msgLocal)
			if err != nil {
				return nil, "", nil, utils.LavaFormatError("Failed to unmarshal marshalJSON(msgLocal)", err)
			}
			reader = bytes.NewReader(jsonBytes)
		} else {
			reader = bytes.NewReader(nodeMessage.Msg)
		}
		formatMessage = true
	}

	rp, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, descriptorSource, reader, grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  false,
		AllowUnknownFields:    true,
	})
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Failed to create formatter", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	// used when parsing the grpc result
	nodeMessage.SetParsingData(methodDescriptor, formatter)

	if formatMessage {
		err = rp.Next(msg)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("rp.Next(msg) Failed", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}

	utils.LavaFormatTrace("provider sending node message",
		utils.LogAttr("_method", nodeMessage.Path),
		utils.LogAttr("headers", metadataMap),
		utils.LogAttr("apiInterface", "grpc"),
	)

	var respHeaders metadata.MD
	response := msgFactory.NewMessage(methodDescriptor.GetOutputType())
	connectCtx, cancel := cp.CapTimeoutForSend(ctx, chainMessage)
	defer cancel()
	err = conn.Invoke(connectCtx, "/"+nodeMessage.Path, msg, response, grpc.Header(&respHeaders))
	if err != nil {
		// Validate if the error is related to the provider connection to the node or it is a valid error
		// in case the error is valid (e.g. bad input parameters) the error will return in the form of a valid error reply
		if parsedError := cp.HandleNodeError(ctx, err); parsedError != nil {
			return nil, "", nil, parsedError
		}
		// return the node's error back to the client as the error type is a invalid request which is cu deductible
		respBytes, statusCode, handlingError := parseGrpcNodeErrorToReply(ctx, err)
		if handlingError != nil {
			return nil, "", nil, handlingError
		}
		// set status code for user header
		grpc.SetTrailer(ctx, metadata.Pairs(common.StatusCodeMetadataKey, strconv.Itoa(int(statusCode)))) // we ignore this error here since this code can be triggered not from grpc
		reply := &RelayReplyWrapper{
			StatusCode: int(statusCode),
			RelayReply: &pairingtypes.RelayReply{
				Data:     respBytes,
				Metadata: convertToMetadataMapOfSlices(respHeaders),
			},
		}
		return reply, "", nil, nil
	}

	var respBytes []byte
	respBytes, err = proto.Marshal(response)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("proto.Marshal(response) Failed", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	// set response status code
	validResponseStatus := http.StatusOK
	grpc.SetTrailer(ctx, metadata.Pairs(common.StatusCodeMetadataKey, strconv.Itoa(validResponseStatus))) // we ignore this error here since this code can be triggered not from grpc
	// create reply wrapper
	reply := &RelayReplyWrapper{
		StatusCode: validResponseStatus, // status code is used only for rest at the moment
		RelayReply: &pairingtypes.RelayReply{
			Data:     respBytes,
			Metadata: convertToMetadataMapOfSlices(respHeaders),
		},
	}
	return reply, "", nil, nil
}

// This method assumes that the error is due to misuse of the request arguments, meaning the user would like to get
// the response from the server to fix the request arguments. this method will make sure the user will get the response
// from the node in the same format as expected.
func parseGrpcNodeErrorToReply(ctx context.Context, err error) ([]byte, uint32, error) {
	var respBytes []byte
	var marshalingError error
	var errorCode uint32 = GRPCStatusCodeOnFailedMessages
	// try fetching status code from error or otherwise use the GRPCStatusCodeOnFailedMessages
	if statusError, ok := status.FromError(err); ok {
		errorCode = uint32(statusError.Code())
		respBytes, marshalingError = json.Marshal(&GrpcNodeErrorResponse{ErrorMessage: statusError.Message(), ErrorCode: errorCode})
		if marshalingError != nil {
			return nil, errorCode, utils.LavaFormatError("json.Marshal(&GrpcNodeErrorResponse Failed 1", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	} else {
		respBytes, marshalingError = json.Marshal(&GrpcNodeErrorResponse{ErrorMessage: err.Error(), ErrorCode: errorCode})
		if marshalingError != nil {
			return nil, errorCode, utils.LavaFormatError("json.Marshal(&GrpcNodeErrorResponse Failed 2", err, utils.Attribute{Key: "GUID", Value: ctx})
		}
	}
	return respBytes, errorCode, nil
}

func marshalJSON(msg proto.Message) ([]byte, error) {
	if dyn, ok := msg.(*dynamic.Message); ok {
		return dyn.MarshalJSON()
	}
	buf := new(bytes.Buffer)
	err := (&jsonpb.Marshaler{}).Marshal(buf, msg)
	return buf.Bytes(), err
}
