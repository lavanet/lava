package chainlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/lavanet/lava/protocol/chainlib/grpcproxy"

	"google.golang.org/grpc/metadata"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	reflectionpbo "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type GrpcChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	BaseChainParser
}

// NewGrpcChainParser creates a new instance of GrpcChainParser
func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return &GrpcChainParser{}, nil
}

func (apip *GrpcChainParser) CraftMessage(serviceApi spectypes.ServiceApi, craftData *CraftData) (ChainMessageForSend, error) {
	if craftData != nil {
		return apip.ParseMsg(craftData.Path, craftData.Data, craftData.ConnectionType)
	}

	grpcMessage := &rpcInterfaceMessages.GrpcMessage{
		Msg:  nil,
		Path: serviceApi.GetName(),
	}
	return apip.newChainMessage(&serviceApi, &serviceApi.ApiInterfaces[0], spectypes.NOT_APPLICABLE, grpcMessage), nil
}

// ParseMsg parses message data into chain message object
func (apip *GrpcChainParser) ParseMsg(url string, data []byte, connectionType string) (ChainMessage, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("GrpcChainParser not defined")
	}

	// Check API is supported and save it in nodeMsg.
	serviceApi, err := apip.getSupportedApi(url)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err)
	}

	apiInterface := GetApiInterfaceFromServiceApi(serviceApi, connectionType)
	if apiInterface == nil {
		return nil, fmt.Errorf("could not find the interface %s in the service %s", connectionType, serviceApi.Name)
	}

	// Construct grpcMessage
	grpcMessage := rpcInterfaceMessages.GrpcMessage{
		Msg:  data,
		Path: url,
	}

	// // Fetch requested block, it is used for data reliability
	// // Extract default block parser
	// blockParser := serviceApi.BlockParsing
	// requestedBlock, err := parser.ParseBlockFromParams(grpcMessage, blockParser)
	// if err != nil {
	// 	return nil, utils.LavaFormatError("ParseBlockFromParams failed parsing block", err, utils.Attribute{Key: "chain", Value: apip.spec.Name}, utils.Attribute{Key: "blockParsing", Value: serviceApi.BlockParsing})
	// }

	nodeMsg := apip.newChainMessage(serviceApi, apiInterface, spectypes.NOT_APPLICABLE, &grpcMessage)
	return nodeMsg, nil
}

func (*GrpcChainParser) newChainMessage(serviceApi *spectypes.ServiceApi, apiInterface *spectypes.ApiInterface, requestedBlock int64, grpcMessage *rpcInterfaceMessages.GrpcMessage) *parsedMessage {
	nodeMsg := &parsedMessage{
		serviceApi:     serviceApi,
		apiInterface:   apiInterface,
		msg:            grpcMessage, // setting the grpc message as a pointer so we can set descriptors for parsing
		requestedBlock: requestedBlock,
	}
	return nodeMsg
}

// getSupportedApi fetches service api from spec by name
func (apip *GrpcChainParser) getSupportedApi(name string) (*spectypes.ServiceApi, error) {
	// Guard that the GrpcChainParser instance exists
	if apip == nil {
		return nil, errors.New("GrpcChainParser not defined")
	}

	// Acquire read lock
	apip.rwLock.RLock()
	defer apip.rwLock.RUnlock()

	// Fetch server api by name
	api, ok := apip.serverApis[name]

	// Return an error if spec does not exist
	if !ok {
		return nil, utils.LavaFormatError("GRPC api not supported", nil, utils.Attribute{Key: "name", Value: name})
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, utils.LavaFormatError("GRPC api is disabled", nil, utils.Attribute{Key: "name", Value: name})
	}

	return &api, nil
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
	serverApis, taggedApis := getServiceApis(spec, spectypes.APIInterfaceGrpc)

	// Set the spec field of the JsonRPCChainParser object
	apip.spec = spec
	apip.serverApis = serverApis
	apip.BaseChainParser.SetTaggedApis(taggedApis)
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
	return apip.spec.Enabled, apip.spec.GetReliabilityThreshold()
}

// ChainBlockStats returns block stats from spec
// (spec.AllowedBlockLagForQosSync, spec.AverageBlockTime, spec.BlockDistanceForFinalizedData)
func (apip *GrpcChainParser) ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData uint32, blocksInFinalizationProof uint32) {
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
	endpoint    *lavasession.RPCEndpoint
	relaySender RelaySender
	logger      *metrics.RPCConsumerLogs
}

func NewGrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *metrics.RPCConsumerLogs) (chainListener *GrpcChainListener) {
	// Create a new instance of GrpcChainListener
	chainListener = &GrpcChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

func convertToMetadata(md map[string][]string) []pairingtypes.Metadata {
	var metadata []pairingtypes.Metadata
	for k, v := range md {
		for _, val := range v {
			metadata = append(metadata, pairingtypes.Metadata{Name: k, Value: val})
		}
	}
	return metadata
}

// Serve http server for GrpcChainListener
func (apil *GrpcChainListener) Serve(ctx context.Context) {
	// Guard that the GrpcChainListener instance exists
	if apil == nil {
		return
	}

	utils.LavaFormatInfo("gRPC PortalStart")

	lis := GetListenerWithRetryGrpc("tcp", apil.endpoint.NetworkAddress)
	apiInterface := apil.endpoint.ApiInterface
	sendRelayCallback := func(ctx context.Context, method string, reqBody []byte) ([]byte, error) {
		ctx = utils.WithUniqueIdentifier(ctx, utils.GenerateUniqueIdentifier())
		msgSeed := apil.logger.GetMessageSeed()

		// grpc header
		metadataValues, _ := metadata.FromIncomingContext(ctx)
		fmt.Println("metadataValues: ", metadataValues)
		blockHeader := metadataValues.Get("x-cosmos-block-height") // ToDo: change it into a header variable instead of hardcoded key
		fmt.Println("blockHeader: ", blockHeader)
		convertedMd := convertToMetadata(metadataValues)
		fmt.Println("convertedMd: ", convertedMd)

		//
		utils.LavaFormatInfo("GRPC Got Relay ", utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "method", Value: method})
		var relayReply *pairingtypes.RelayReply
		metricsData := metrics.NewRelayAnalytics("NoDappID", apil.endpoint.ChainID, apiInterface)
		relayReply, _, err := apil.relaySender.SendRelay(ctx, method, string(reqBody), "", "NoDappID", metricsData, convertedMd)
		fmt.Println("relayReply.Data: ", relayReply.Data)
		go apil.logger.AddMetricForGrpc(metricsData, err, &metadataValues)

		if err != nil {
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)
			apil.logger.LogRequestAndResponse("http in/out", true, method, string(reqBody), "", errMasking, msgSeed, err)
			return nil, utils.LavaFormatError("Failed to SendRelay", fmt.Errorf(errMasking))
		}
		apil.logger.LogRequestAndResponse("http in/out", false, method, string(reqBody), "", "", msgSeed, nil)
		return relayReply.Data, nil
	}

	_, httpServer, err := grpcproxy.NewGRPCProxy(sendRelayCallback)
	if err != nil {
		utils.LavaFormatFatal("provider failure RegisterServer", err, utils.Attribute{Key: "listenAddr", Value: apil.endpoint.NetworkAddress})
	}

	utils.LavaFormatInfo("Server listening", utils.Attribute{Key: "Address", Value: lis.Addr()})

	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Portal failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr()}, utils.Attribute{Key: "ChainID", Value: apil.endpoint.ChainID})
	}
}

type GrpcChainProxy struct {
	BaseChainProxy
	conn *chainproxy.GRPCConnector
}

func NewGrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, averageBlockTime time.Duration) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrls) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, utils.Attribute{Key: "chainID", Value: rpcProviderEndpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: rpcProviderEndpoint.ApiInterface})
	}
	cp := &GrpcChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime},
	}
	nodeUrl := rpcProviderEndpoint.NodeUrls[0]
	nodeUrl.Url = strings.TrimSuffix(nodeUrl.Url, "/") // remove suffix if exists
	conn, err := chainproxy.NewGRPCConnector(ctx, nConns, nodeUrl)
	if err != nil {
		return nil, err
	}
	cp.conn = conn
	if cp.conn == nil {
		return nil, utils.LavaFormatError("g_conn == nil", nil)
	}
	return cp, nil
}

func (cp *GrpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessageForSend, request *pairingtypes.RelayRequest) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on grpc", nil, utils.Attribute{Key: "GUID", Value: ctx})
	}
	conn, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("grpc get connection failed ", err, utils.Attribute{Key: "GUID", Value: ctx})
	}
	defer cp.conn.ReturnRpc(conn)

	rpcInputMessage := chainMessage.GetRPCMessage()
	fmt.Println("rpcInputMessage: ", rpcInputMessage)
	nodeMessage, ok := rpcInputMessage.(*rpcInterfaceMessages.GrpcMessage)
	fmt.Println("nodeMessage: ", nodeMessage)
	fmt.Println("nodeMessage.Msg: ", nodeMessage.Msg)
	fmt.Println("nodeMessage.Path: ", nodeMessage.Path)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in grpc failed to cast RPCInput from chainMessage", nil, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "rpcMessage", Value: rpcInputMessage})
	}
	relayTimeout := common.LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits)
	// check if this API is hanging (waiting for block confirmation)
	if chainMessage.GetInterface().Category.HangingApi {
		relayTimeout += cp.averageBlockTime
	}
	connectCtx, cancel := cp.NodeUrl.LowerContextTimeout(ctx, relayTimeout)
	defer cancel()

	// TODO: improve functionality, this is reading descriptors every send
	// improvement would be caching the descriptors, instead of fetching them.
	cl := grpcreflect.NewClient(ctx, reflectionpbo.NewServerReflectionClient(conn))
	descriptorSource := rpcInterfaceMessages.DescriptorSourceFromServer(cl)
	svc, methodName := rpcInterfaceMessages.ParseSymbol(nodeMessage.Path)
	var descriptor desc.Descriptor
	if descriptor, err = descriptorSource.FindSymbol(svc); err != nil {
		return nil, "", nil, utils.LavaFormatError("descriptorSource.FindSymbol", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "descriptor", Value: descriptor})
	}
	methodDescriptor := serviceDescriptor.FindMethodByName(methodName)
	fmt.Println("methodDescriptor: ", methodDescriptor)
	if methodDescriptor == nil {
		return nil, "", nil, utils.LavaFormatError("serviceDescriptor.FindMethodByName returned nil", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "methodName", Value: methodName})
	}
	msgFactory := dynamic.NewMessageFactoryWithDefaults()

	var reader io.Reader
	msg := msgFactory.NewMessage(methodDescriptor.GetInputType())
	fmt.Println("msg: ", msg)
	formatMessage := false
	if len(nodeMessage.Msg) > 0 {
		// guess if json or binary
		if nodeMessage.Msg[0] != '{' && nodeMessage.Msg[0] != '[' {
			msgLocal := msgFactory.NewMessage(methodDescriptor.GetInputType())
			err = proto.Unmarshal(nodeMessage.Msg, msgLocal)
			if err != nil {
				return nil, "", nil, err
			}
			jsonBytes, err := marshalJSON(msgLocal)
			if err != nil {
				return nil, "", nil, err
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

	// md := metadata.New(map[string]string{"x-cosmos-block-height": "2"})
	// ctxNew := metadata.NewOutgoingContext(connectCtx, md)

	// fmt.Println("ctxNew: ", ctxNew)
	// fmt.Println("Metadata: ", md)
	mdp, ok := metadata.FromOutgoingContext(connectCtx)
	if ok {
		blockHeight := mdp.Get("x-cosmos-block-height")
		fmt.Println("AHAHAHAH Block Height:", blockHeight)
	}

	response := msgFactory.NewMessage(methodDescriptor.GetOutputType())
	fmt.Println("/nodeMessage.Path: ", "/"+nodeMessage.Path)
	// err = conn.Invoke(connectCtx, "/"+nodeMessage.Path, msg, response)
	err = conn.Invoke(connectCtx, "/"+nodeMessage.Path, msg, response)
	if err != nil {
		if connectCtx.Err() == context.DeadlineExceeded {
			// Not an rpc error, return provider error without disclosing the endpoint address
			return nil, "", nil, utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
		}
		return nil, "", nil, utils.LavaFormatError("Invoke Failed", err, utils.Attribute{Key: "GUID", Value: ctx}, utils.Attribute{Key: "Method", Value: nodeMessage.Path}, utils.Attribute{Key: "msg", Value: nodeMessage.Msg})
	}

	var respBytes []byte
	respBytes, err = proto.Marshal(response)
	fmt.Println("respBytes: ", respBytes)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("proto.Marshal(response) Failed", err, utils.Attribute{Key: "GUID", Value: ctx})
	}

	reply := &pairingtypes.RelayReply{
		Data: respBytes,
	}
	fmt.Println("reply: ", reply)

	return reply, "", nil, nil
}

func marshalJSON(msg proto.Message) ([]byte, error) {
	if dyn, ok := msg.(*dynamic.Message); ok {
		return dyn.MarshalJSON()
	}
	buf := new(bytes.Buffer)
	err := (&jsonpb.Marshaler{}).Marshal(buf, msg)
	return buf.Bytes(), err
}
