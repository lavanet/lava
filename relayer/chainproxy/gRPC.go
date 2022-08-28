package chainproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

type GrpcMessage struct {
	cp             *GrpcChainProxy
	serviceApi     *spectypes.ServiceApi
	path           string
	msg            interface{}
	requestedBlock int64
	connectionType string
	Result         json.RawMessage
}

type GrpcChainProxy struct {
	nodeUrl string
	sentry  *sentry.Sentry
}

func (r *GrpcMessage) GetMsg() interface{} {
	return r.msg
}

func NewGrpcChainProxy(nodeUrl string, sentry *sentry.Sentry) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &GrpcChainProxy{
		nodeUrl: nodeUrl,
		sentry:  sentry,
	}
}

func (cp *GrpcChainProxy) NewMessage(path string, data []byte) (*GrpcMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, utils.LavaFormatError("failed to get supported api in NewMessage", err, &map[string]string{"path": path})
	}
	// var d interface{}
	// err = json.Unmarshal(data, &d)
	// if err != nil {
	// 	return nil, utils.LavaFormatError("failed to unmarshal gRPC NewMessage", err, &map[string]string{"data": fmt.Sprintf("%s", data)})
	// }

	nodeMsg := &GrpcMessage{
		cp:         cp,
		serviceApi: serviceApi,
		path:       path,
		msg:        data,
	}

	return nodeMsg, nil
}

func (m GrpcMessage) GetParams() interface{} {
	return m.msg
}

func (m GrpcMessage) GetResult() json.RawMessage {
	// utils.LavaFormatInfo("getting result: "+string(m.Result), nil)
	return m.Result
}

func (m GrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

func (cp *GrpcChainProxy) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCK_BY_NUM)
	if !ok {
		return "", errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	var nodeMsg NodeMessage
	var err error
	if serviceApi.GetParsing().FunctionTemplate != "" {
		nodeMsg, err = cp.ParseMsg(serviceApi.Name, []byte(fmt.Sprintf(serviceApi.GetParsing().FunctionTemplate, blockNum)), "")
	} else {
		nodeMsg, err = cp.NewMessage(serviceApi.Name, nil)
	}

	if err != nil {
		return "", err
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return "", err
	}

	blockData, err := parser.ParseMessageResponse((nodeMsg.(*GrpcMessage)), serviceApi.Parsing.ResultParsing)
	if err != nil {
		return "", err
	}

	// blockData is an interface array with the parsed result in index 0.
	// we know to expect a string result for a hash.
	return blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string), nil
}

func (cp *GrpcChainProxy) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	serviceApi, ok := cp.GetSentry().GetSpecApiByTag(spectypes.GET_BLOCKNUM)
	if !ok {
		return spectypes.NOT_APPLICABLE, errors.New(spectypes.GET_BLOCKNUM + " tag function not found")
	}

	params := make(json.RawMessage, 0)
	nodeMsg, err := cp.NewMessage(serviceApi.GetName(), params)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("new Message creation Failed at FetchLatestBlockNum", err, nil)
	}

	_, err = nodeMsg.Send(ctx)
	if err != nil {
		return spectypes.NOT_APPLICABLE, utils.LavaFormatError("Message send Failed at FetchLatestBlockNum", err, nil)
	}

	blocknum, err := parser.ParseBlockFromReply(nodeMsg, serviceApi.Parsing.ResultParsing)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}

	return blocknum, nil
}

func (cp *GrpcChainProxy) GetSentry() *sentry.Sentry {
	return cp.sentry
}

func (cp *GrpcChainProxy) Start(context.Context) error {
	return nil
}

func (cp *GrpcChainProxy) getSupportedApi(path string) (*spectypes.ServiceApi, error) {
	if api, ok := cp.sentry.MatchSpecApiByName(path); ok {
		if !api.Enabled {
			return nil, fmt.Errorf("gRPC Api is disabled %s ", path)
		}
		return &api, nil
	}
	return nil, fmt.Errorf("gRPC Api not supported %s ", path)
}

func (cp *GrpcChainProxy) ParseMsg(path string, data []byte, connectionType string) (NodeMessage, error) {
	//
	// Check api is supported an save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, utils.LavaFormatError("failed to getSupportedApi gRPC", err, nil)
	}

	nodeMsg := &GrpcMessage{
		cp:             cp,
		serviceApi:     serviceApi,
		path:           path,
		msg:            data,
		connectionType: connectionType,
	}

	return nodeMsg, nil
}

func (cp *GrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	//
	// // Setup Grpc Server
	// lis, err := net.Listen("tcp", listenAddr)
	// if err != nil {
	// 	utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
	// }
	// s := grpc.NewServer()

	// Server := &relayServer{}
	// pairingtypes.RegisterRelayerServer(s, Server)
	// reflection.Register(s)
	// utils.LavaFormatInfo("Server listening", &map[string]string{"Address": lis.Addr().String()})
	// if err := s.Serve(lis); err != nil {
	// 	utils.LavaFormatFatal("portal failed to serve", err, &map[string]string{"Address": lis.Addr().String()})
	// }
	// return

}

func (nm *GrpcMessage) RequestedBlock() int64 {
	return nm.requestedBlock
}

func (nm *GrpcMessage) GetServiceApi() *spectypes.ServiceApi {
	return nm.serviceApi
}

func descriptorSourceFromServer(refClient *grpcreflect.Client) DescriptorSource {
	return serverSource{client: refClient}
}

type DescriptorSource interface {
	// ListServices returns a list of fully-qualified service names. It will be all services in a set of
	// descriptor files or the set of all services exposed by a gRPC server.
	ListServices() ([]string, error)
	// FindSymbol returns a descriptor for the given fully-qualified symbol name.
	FindSymbol(fullyQualifiedName string) (desc.Descriptor, error)
	// AllExtensionsForType returns all known extension fields that extend the given message type name.
	AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error)
}

type serverSource struct {
	client *grpcreflect.Client
}

func (ss serverSource) ListServices() ([]string, error) {
	svcs, err := ss.client.ListServices()
	return svcs, reflectionSupport(err)
}

func (ss serverSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	file, err := ss.client.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, reflectionSupport(err)
	}
	d := file.FindSymbol(fullyQualifiedName)
	if d == nil {
		return nil, utils.LavaFormatError("Symbol not found", fmt.Errorf("missing symbol: %s", fullyQualifiedName), nil)
	}
	return d, nil
}

func (ss serverSource) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	var exts []*desc.FieldDescriptor
	nums, err := ss.client.AllExtensionNumbersForType(typeName)
	if err != nil {
		return nil, reflectionSupport(err)
	}
	for _, fieldNum := range nums {
		ext, err := ss.client.ResolveExtension(typeName, fieldNum)
		if err != nil {
			return nil, reflectionSupport(err)
		}
		exts = append(exts, ext)
	}
	return exts, nil
}

func reflectionSupport(err error) error {
	if err == nil {
		return nil
	}
	if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
		return utils.LavaFormatError("server does not support the reflection API", err, nil)
	}
	return err
}

func parseSymbol(svcAndMethod string) (string, string) {
	pos := strings.LastIndex(svcAndMethod, "/")
	if pos < 0 {
		pos = strings.LastIndex(svcAndMethod, ".")
		if pos < 0 {
			return "", ""
		}
	}
	return svcAndMethod[:pos], svcAndMethod[pos+1:]
}

func (nm *GrpcMessage) invoke(ctx context.Context, conn *grpc.ClientConn) ([]byte, error) {
	client := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	source := descriptorSourceFromServer(client)
	svc, mth := parseSymbol(nm.path)
	dsc, err := source.FindSymbol(svc)
	if err != nil {
		return nil, utils.LavaFormatError("failed to find symbol", err, &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
	}
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, utils.LavaFormatError("failed to expose", fmt.Errorf("server does not expose service %q", svc), &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
	}
	mtd := sd.FindMethodByName(mth)
	if mtd == nil {
		return nil, utils.LavaFormatError("failed to expose", fmt.Errorf("service %q does not include a method named %q", svc, mth), &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
	}

	msgFactory := dynamic.NewMessageFactoryWithDefaults()
	req := msgFactory.NewMessage(mtd.GetInputType())
	stub := grpcdynamic.NewStubWithMessageFactory(conn, msgFactory)

	var in io.Reader
	input := ""
	switch inputContent := nm.msg.(type) {
	case []byte:
		input = string(inputContent)
	case string:
		input = inputContent
	default:
		utils.LavaFormatInfo(fmt.Sprintf("msg type %T is unknown. continuing with empty message parameters", nm.msg), &map[string]string{"msg": fmt.Sprintf("%v", nm.msg)})
	}
	in = strings.NewReader(input)

	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: false,
		IncludeTextSeparator:  false,
		AllowUnknownFields:    true,
	}
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.Format("json"), source, in, options)
	if err != nil {
		fmt.Printf("%s", err)
	}
	h := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}
	rff := rf.Next
	err = rff(req)
	if err != nil {
		fmt.Printf("%s", err)
	}

	msg, err := stub.InvokeRpc(ctx, mtd, req)

	if respStr, err := h.Formatter(msg); err != nil {
		return nil, utils.LavaFormatError("failed to format response from chain", err, &map[string]string{"msg": fmt.Sprintf("%v", msg)})

	} else {
		return []byte(respStr), nil
	}
}

func (nm *GrpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {

	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Todo, maybe create a list of grpc clients in connector.go instead of creating one and closing it each time?
	conn, err := grpc.DialContext(connectCtx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}
	defer conn.Close()

	res, err := nm.invoke(ctx, conn)
	if err != nil {
		nm.Result = []byte(fmt.Sprintf("%s", err))
		return nil, utils.LavaFormatError("Sending the gRPC message failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
	}
	nm.Result = res

	reply := &pairingtypes.RelayReply{
		Data: res,
	}
	nm.Result = res
	return reply, nil
}
