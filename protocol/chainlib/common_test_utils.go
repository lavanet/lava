package chainlib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/lavanet/lava/v5/utils"
	specutils "github.com/lavanet/lava/v5/utils/keeper"
)

// ---------------------------------------------------------------------------
// gRPC test service — programmatic proto descriptors for unit testing
// ---------------------------------------------------------------------------
//
// The test gRPC spec (specs/grpctest.json) references
// lavatest.v1.BlockService/GetLatestBlock. For gRPC reflection to resolve
// this service, we register a FileDescriptorProto in the global proto
// registry at init time. The mock handler uses dynamicpb messages so that
// proto marshal/unmarshal works end-to-end.

// testBlockServiceFD holds the registered file descriptor for lavatest.v1.
var testBlockServiceFD protoreflect.FileDescriptor

func init() {
	syntax := "proto3"
	fileName := "lavatest/v1/block_service.proto"
	pkgName := "lavatest.v1"
	reqName := "GetLatestBlockRequest"
	respName := "GetLatestBlockResponse"
	svcName := "BlockService"
	methodName := "GetLatestBlock"
	heightField := "height"
	inputType := ".lavatest.v1.GetLatestBlockRequest"
	outputType := ".lavatest.v1.GetLatestBlockResponse"
	fieldNum := int32(1)
	fieldType := descriptorpb.FieldDescriptorProto_TYPE_INT64
	fieldLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	fdp := &descriptorpb.FileDescriptorProto{
		Name:    &fileName,
		Package: &pkgName,
		Syntax:  &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &reqName},
			{
				Name: &respName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     &heightField,
						Number:   &fieldNum,
						Type:     &fieldType,
						Label:    &fieldLabel,
						JsonName: &heightField,
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: &svcName,
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       &methodName,
						InputType:  &inputType,
						OutputType: &outputType,
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create test proto file descriptor: %v", err))
	}
	if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
		// Already registered (e.g. multiple test binaries) — that's fine.
		if !strings.Contains(err.Error(), "already registered") {
			panic(fmt.Sprintf("failed to register test proto file descriptor: %v", err))
		}
	}
	testBlockServiceFD = fd
}

type mockResponseWriter struct {
	blockToReturn *int
}

func (mockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (mockResponseWriter) Write(in []byte) (int, error) {
	return 0, nil
}

func (mrw mockResponseWriter) WriteHeader(statusCode int) {
	*mrw.blockToReturn = statusCode
}

// mockGRPCServiceDesc is the gRPC service descriptor for the test block service.
var mockGRPCServiceDesc = grpc.ServiceDesc{
	ServiceName: "lavatest.v1.BlockService",
	HandlerType: (*mockBlockServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLatestBlock",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				reqDesc := testBlockServiceFD.Messages().ByName("GetLatestBlockRequest")
				in := dynamicpb.NewMessage(reqDesc)
				if err := dec(in); err != nil {
					return nil, err
				}
				return srv.(mockBlockServiceServer).GetLatestBlock(ctx, in)
			},
		},
	},
	Streams: []grpc.StreamDesc{},
}

type mockBlockServiceServer interface {
	GetLatestBlock(context.Context, *dynamicpb.Message) (*dynamicpb.Message, error)
}

type mockBlockServiceImpl struct {
	serverCallback http.HandlerFunc
}

func (s mockBlockServiceImpl) GetLatestBlock(ctx context.Context, _ *dynamicpb.Message) (*dynamicpb.Message, error) {
	md, exists := metadata.FromIncomingContext(ctx)
	req := &http.Request{}
	if exists {
		headers := map[string][]string{}
		for key, val := range md {
			headers[key] = val
		}
		req = &http.Request{Header: headers}
	}
	num := 5
	respWriter := mockResponseWriter{blockToReturn: &num}
	s.serverCallback(respWriter, req)

	respDesc := testBlockServiceFD.Messages().ByName("GetLatestBlockResponse")
	resp := dynamicpb.NewMessage(respDesc)
	resp.Set(respDesc.Fields().ByName("height"), protoreflect.ValueOfInt64(int64(num)))
	return resp, nil
}

func generateCombinations(arr []string) [][]string {
	if len(arr) == 0 {
		return [][]string{{}}
	}

	first, rest := arr[0], arr[1:]
	combinationsWithoutFirst := generateCombinations(rest)
	var combinationsWithFirst [][]string

	for _, c := range combinationsWithoutFirst {
		combinationsWithFirst = append(combinationsWithFirst, append(c, first))
	}

	return append(combinationsWithoutFirst, combinationsWithFirst...)
}

func genericWebSocketHandler() http.HandlerFunc {
	upGrader := websocket.Upgrader{}

	// Create a simple websocket server that mocks the node
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upGrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			panic("got error in upgrader")
		}
		defer conn.Close()

		for {
			// Read the request
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				// Don't panic here, just return to close the connection gracefully
				return
			}
			fmt.Println("got ws message", string(message), messageType)
			conn.WriteMessage(messageType, message)
			fmt.Println("writing ws message", string(message), messageType)
		}
	}
}

// CreateMockSpec returns a minimal spectypes.Spec suitable for unit tests.
func CreateMockSpec() spectypes.Spec {
	specName := "mockspec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.ReliabilityThreshold = 4294967295
	spec.BlockDistanceForFinalizedData = 0
	spec.DataReliabilityEnabled = true
	spec.ApiCollections = []*spectypes.ApiCollection{
		{
			Enabled: true,
			CollectionData: spectypes.CollectionData{
				ApiInterface: "stub",
				Type:         "GET",
			},
			Apis: []*spectypes.Api{
				{
					Name:         specName + "API",
					ComputeUnits: 100,
					Enabled:      true,
				},
			},
		},
	}
	spec.Shares = 1
	return spec
}

// generates a chain parser, a chain fetcher messages based on it
// apiInterface can either be an ApiInterface string as in spectypes.ApiInterfaceXXX or a number for an index in the apiCollections
func CreateChainLibMocks(
	ctx context.Context,
	specIndex string,
	apiInterface string,
	httpServerCallback http.HandlerFunc,
	wsServerCallback http.HandlerFunc,
	getToTopMostPath string,
	services []string,
) (cpar ChainParser, crout ChainRouter, cfetc chaintracker.ChainFetcher, closeServer func(), endpointRet *lavasession.RPCProviderEndpoint, errRet error) {
	utils.SetGlobalLoggingLevel("debug")
	// Create a cancellable context for the connector to prevent leaks
	connectorCtx, cancelConnector := context.WithCancel(ctx)

	closeServer = func() {
		cancelConnector()
	}

	// Load specs from all directories so cross-directory imports (e.g. LAV1 -> COSMOSSDK) resolve.
	spec, err := specutils.GetSpecFromLocalDirs([]string{
		getToTopMostPath + "specs/",
	}, specIndex)
	if err != nil {
		cancelConnector()
		return nil, nil, nil, nil, nil, err
	}
	index, err := strconv.Atoi(apiInterface)
	if err == nil && index < len(spec.ApiCollections) {
		apiInterface = spec.ApiCollections[index].CollectionData.ApiInterface
	}
	chainParser, err := NewChainParser(apiInterface)
	if err != nil {
		cancelConnector()
		return nil, nil, nil, nil, nil, err
	}
	var chainRouter ChainRouter
	chainParser.SetSpec(spec)
	endpoint := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{},
		ChainID:        specIndex,
		ApiInterface:   apiInterface,
		Geolocation:    1,
		NodeUrls:       []common.NodeUrl{},
	}
	addons, extensions, err := chainParser.SeparateAddonsExtensions(context.Background(), services)
	if err != nil {
		cancelConnector()
		return nil, nil, nil, nil, nil, err
	}

	if httpServerCallback == nil {
		httpServerCallback = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	}

	if wsServerCallback == nil {
		wsServerCallback = genericWebSocketHandler()
	}

	if apiInterface == spectypes.APIInterfaceGrpc {
		// Start a new gRPC server using the buffered connection
		grpcServer := grpc.NewServer()
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: addons})
		allCombinations := generateCombinations(extensions)
		for _, extensionsList := range allCombinations {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: lis.Addr().String(), Addons: append(addons, extensionsList...)})
		}
		go func() {
			service := mockBlockServiceImpl{serverCallback: httpServerCallback}
			grpcServer.RegisterService(&mockGRPCServiceDesc, service)
			reflection.Register(grpcServer)
			// Serve requests on the buffered connection
			if err := grpcServer.Serve(lis); err != nil {
				return
			}
		}()
		time.Sleep(10 * time.Millisecond)
		chainRouter, err = GetChainRouter(connectorCtx, 1, endpoint, chainParser)
		if err != nil {
			grpcServer.Stop()
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
		closeServer = func() {
			grpcServer.Stop()
			cancelConnector()
		}
	} else {
		var mockWebSocketServer *httptest.Server
		var wsUrl string
		mockWebSocketServer = httptest.NewServer(wsServerCallback)
		wsUrl = "ws" + strings.TrimPrefix(mockWebSocketServer.URL, "http")
		mockHttpServer := httptest.NewServer(httpServerCallback)
		closeServer = func() {
			mockHttpServer.Close()
			mockWebSocketServer.Close()
			cancelConnector()
		}
		endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: addons})
		// Only add WebSocket URLs for API interfaces that support WebSocket (JsonRPC and TendermintRPC)
		// REST API only supports HTTP/HTTPS, not WebSocket
		supportsWebSocket := apiInterface == spectypes.APIInterfaceJsonRPC || apiInterface == spectypes.APIInterfaceTendermintRPC
		if len(extensions) > 0 {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: mockHttpServer.URL, Addons: extensions})
			if supportsWebSocket {
				endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: extensions})
			}
		}
		if supportsWebSocket {
			endpoint.NodeUrls = append(endpoint.NodeUrls, common.NodeUrl{Url: wsUrl, Addons: nil})
		}
		chainRouter, err = GetChainRouter(connectorCtx, 1, endpoint, chainParser)
		if err != nil {
			mockHttpServer.Close()
			mockWebSocketServer.Close()
			cancelConnector()
			return nil, nil, nil, closeServer, nil, err
		}
	}
	chainFetcher := NewChainFetcher(ctx, &ChainFetcherOptions{chainRouter, chainParser, endpoint, nil})
	return chainParser, chainRouter, chainFetcher, closeServer, endpoint, err
}
