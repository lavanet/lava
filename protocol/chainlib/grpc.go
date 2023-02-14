package chainlib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/relayer/metrics"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
	reflectionpbo "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type GrpcChainParser struct {
	spec       spectypes.Spec
	rwLock     sync.RWMutex
	serverApis map[string]spectypes.ServiceApi
	taggedApis map[string]spectypes.ServiceApi
}

// NewGrpcChainParser creates a new instance of GrpcChainParser
func NewGrpcChainParser() (chainParser *GrpcChainParser, err error) {
	return &GrpcChainParser{}, nil
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
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err, nil)
	}

	var apiInterface *spectypes.ApiInterface = nil
	for i := range serviceApi.ApiInterfaces {
		if serviceApi.ApiInterfaces[i].Type == connectionType {
			apiInterface = &serviceApi.ApiInterfaces[i]
			break
		}
	}
	if apiInterface == nil {
		return nil, fmt.Errorf("could not find the interface %s in the service %s", connectionType, serviceApi.Name)
	}

	// Construct grpcMessage
	grpcMessage := chainproxy.GrpcMessage{
		Msg:  data,
		Path: url,
	}

	// TODO why we don't have requested block here?
	nodeMsg := &parsedMessage{
		serviceApi:   serviceApi,
		apiInterface: apiInterface,
		msg:          grpcMessage,
	}
	return nodeMsg, nil
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
	api, ok := matchSpecApiByName(name, apip.serverApis)

	// Return an error if spec does not exist
	if !ok {
		return nil, errors.New("JRPC api not supported")
	}

	// Return an error if api is disabled
	if !api.Enabled {
		return nil, errors.New("api is disabled")
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
	apip.taggedApis = taggedApis
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
	logger      *common.RPCConsumerLogs
}

func NewGrpcChainListener(ctx context.Context, listenEndpoint *lavasession.RPCEndpoint, relaySender RelaySender, rpcConsumerLogs *common.RPCConsumerLogs) (chainListener *GrpcChainListener) {
	// Create a new instance of GrpcChainListener
	chainListener = &GrpcChainListener{
		listenEndpoint,
		relaySender,
		rpcConsumerLogs,
	}

	return chainListener
}

// Serve http server for GrpcChainListener
func (apil *GrpcChainListener) Serve(ctx context.Context) {
	// Guard that the GrpcChainListener instance exists
	if apil == nil {
		return
	}

	utils.LavaFormatInfo("gRPC PortalStart", nil)

	lis, err := net.Listen("tcp", apil.endpoint.NetworkAddress)
	if err != nil {
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": apil.endpoint.NetworkAddress})
	}
	apiInterface := apil.endpoint.ApiInterface
	sendRelayCallback := func(ctx context.Context, method string, reqBody []byte) ([]byte, error) {
		msgSeed := apil.logger.GetMessageSeed()
		utils.LavaFormatInfo("GRPC Got Relay: "+method, nil)
		var relayReply *pairingtypes.RelayReply
		metricsData := metrics.NewRelayAnalytics("NoDappID", apil.endpoint.ChainID, apiInterface)
		if relayReply, _, err = apil.relaySender.SendRelay(ctx, method, string(reqBody), "", "NoDappID", metricsData); err != nil {
			go apil.logger.AddMetric(metricsData, err != nil)
			errMasking := apil.logger.GetUniqueGuidResponseForError(err, msgSeed)
			apil.logger.LogRequestAndResponse("http in/out", true, method, string(reqBody), "", errMasking, msgSeed, err)
			return nil, utils.LavaFormatError("Failed to SendRelay", fmt.Errorf(errMasking), nil)
		}
		apil.logger.LogRequestAndResponse("http in/out", false, method, string(reqBody), "", "", msgSeed, nil)
		return relayReply.Data, nil
	}

	_, httpServer, err := thirdparty.RegisterServer(apil.endpoint.ChainID, sendRelayCallback)
	if err != nil {
		utils.LavaFormatFatal("provider failure RegisterServer", err, &map[string]string{"listenAddr": apil.endpoint.NetworkAddress})
	}

	utils.LavaFormatInfo("Server listening", &map[string]string{"Address": lis.Addr().String()})

	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Portal failed to serve", err, &map[string]string{"Address": lis.Addr().String(), "ChainID": apil.endpoint.ChainID})
	}
}

type GrpcChainProxy struct {
	BaseChainProxy
	conn *chainproxy.GRPCConnector
}

func NewGrpcChainProxy(ctx context.Context, nConns uint, rpcProviderEndpoint *lavasession.RPCProviderEndpoint, averageBlockTime time.Duration) (ChainProxy, error) {
	if len(rpcProviderEndpoint.NodeUrl) == 0 {
		return nil, utils.LavaFormatError("rpcProviderEndpoint.NodeUrl list is empty missing node url", nil, &map[string]string{"chainID": rpcProviderEndpoint.ChainID, "ApiInterface": rpcProviderEndpoint.ApiInterface})
	}
	cp := &GrpcChainProxy{
		BaseChainProxy: BaseChainProxy{averageBlockTime: averageBlockTime},
	}
	cp.conn = chainproxy.NewGRPCConnector(ctx, nConns, strings.TrimSuffix(rpcProviderEndpoint.NodeUrl[0], "/"))
	if cp.conn == nil {
		return nil, utils.LavaFormatError("g_conn == nil", nil, nil)
	}
	return cp, nil
}

func (cp *GrpcChainProxy) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage ChainMessage) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil, nil)
	}
	conn, err := cp.conn.GetRpc(ctx, true)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("grpc get connection failed ", err, nil)
	}
	defer cp.conn.ReturnRpc(conn)

	rpcInputMessage := chainMessage.GetRPCMessage()
	nodeMessage, ok := rpcInputMessage.(chainproxy.GrpcMessage)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("invalid message type in jsonrpc failed to cast RPCInput from chainMessage", nil, &map[string]string{"rpcMessage": fmt.Sprintf("%+v", rpcInputMessage)})
	}
	relayTimeout := LocalNodeTimePerCu(chainMessage.GetServiceApi().ComputeUnits)
	// check if this API is hanging (waiting for block confirmation)
	if chainMessage.GetInterface().Category.HangingApi {
		relayTimeout += cp.averageBlockTime
	}
	connectCtx, cancel := context.WithTimeout(ctx, relayTimeout)
	defer cancel()

	cl := grpcreflect.NewClient(ctx, reflectionpbo.NewServerReflectionClient(conn)) // TODO: improve functionality, this is reading descriptors every send
	descriptorSource := chainproxy.DescriptorSourceFromServer(cl)
	svc, methodName := chainproxy.ParseSymbol(nodeMessage.Path)
	var descriptor desc.Descriptor
	if descriptor, err = descriptorSource.FindSymbol(svc); err != nil {
		return nil, "", nil, utils.LavaFormatError("descriptorSource.FindSymbol", err, nil)
	}

	serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, "", nil, utils.LavaFormatError("serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)", err, &map[string]string{"descriptor": fmt.Sprintf("%v", descriptor)})
	}
	methodDescriptor := serviceDescriptor.FindMethodByName(methodName)
	if methodDescriptor == nil {
		return nil, "", nil, utils.LavaFormatError("serviceDescriptor.FindMethodByName returned nil", err, &map[string]string{"methodName": methodName})
	}
	msgFactory := dynamic.NewMessageFactoryWithDefaults()

	var reader io.Reader
	msg := msgFactory.NewMessage(methodDescriptor.GetInputType())
	formatMessage := false
	if len(nodeMessage.Msg) > 0 {
		reader = bytes.NewReader(nodeMessage.Msg)
		formatMessage = true
	}

	// nodeMessage.MethodDesc = methodDescriptor // TODO: this is useful for parsing the response
	// rp, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, descriptorSource, reader, grpcurl.FormatOptions{
	rp, _, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, descriptorSource, reader, grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  false,
		AllowUnknownFields:    true,
	})
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Failed to create formatter", err, nil)
	}
	// nm.formatter = formatter
	if formatMessage {
		err = rp.Next(msg)
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("rp.Next(msg) Failed", err, nil)
		}
	}

	response := msgFactory.NewMessage(methodDescriptor.GetOutputType())
	err = grpc.Invoke(connectCtx, nodeMessage.Path, msg, response, conn)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("Invoke Failed", err, &map[string]string{"Method": nodeMessage.Path, "msg": string(nodeMessage.Msg)})
	}

	var respBytes []byte
	respBytes, err = proto.Marshal(response)
	if err != nil {
		return nil, "", nil, utils.LavaFormatError("proto.Marshal(response) Failed", err, nil)
	}

	reply := &pairingtypes.RelayReply{
		Data: respBytes,
	}
	return reply, "", nil, nil
}
