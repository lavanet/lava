package chainproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/server/grpc/gogoreflection"
	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

type Server struct {
	cp      *GrpcChainProxy
	privKey *btcec.PrivateKey
}

func NewServer(cp *GrpcChainProxy, privKey *btcec.PrivateKey) *Server {
	return &Server{
		cp:      cp,
		privKey: privKey,
	}
}

// func (s Server) Proxy(
// 	ctx context.Context,
// 	req *proxypb.ProxyRequest,
// ) (resp *proxypb.ProxyResponse, err error) {
// 	var b []byte
// 	if b, err = grpcutil.Marshal(req.GetBody()); err != nil {
// 		log.Println("[grpcutil.Marshal]:", err.Error())
// 		return nil, utils.LavaFormatError("", err, nil)
// 	}
//
// 	log.Println("[b]:", string(b))
//
// 	path := req.GetName()
// 	log.Println("in <<< ", path)
//
// 	var reply *pairingtypes.RelayReply
// 	if reply, err = SendRelay(ctx, s.cp, s.privKey, path, string(b), ""); err != nil {
// 		log.Println("[SendRelay]:", err.Error())
// 		return nil, utils.LavaFormatError("", err, nil)
// 	}
//
// 	return handleReply(reply)
// }

type GrpcMessage struct {
	methodDesc *desc.MethodDescriptor
	formatter  grpcurl.Formatter

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
	// Check api is supported and save it in nodeMsg
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

	// var s string
	// if s, err = formatter(respMessage); err != nil {
	// 	return nil, errors.Wrap(err, "formatter()")
	// }
	// log.Println("[RESP]", s)
	//

	// TODO:
	msgFactory := dynamic.NewMessageFactoryWithDefaults()
	msg := msgFactory.NewMessage(m.methodDesc.GetOutputType())
	if err := proto.Unmarshal(m.Result, msg); err != nil {
		log.Println(errors.Wrap(err, "proto.Unmarshal()"))
		return m.Result
	}

	s, err := m.formatter(msg)
	if err != nil {
		log.Println(errors.Wrap(err, "m.formatter()"))
		return m.Result
	}
	log.Println("m.formatter(): ", s)

	return []byte(s)

	// return m.Result

	// log.Println("m.formatter(): ", s)
	// return []byte(s)

	// if err := jsonpb.UnmarshalString(string(m.Result), msg); err != nil {
	// 	log.Println(errors.Wrap(err, "jsonpb.UnmarshalString()"))
	// 	return m.Result
	// }

	// marshaler := &jsonpb.Marshaler{
	// 	OrigName:     true,
	// 	EnumsAsInts:  false,
	// 	EmitDefaults: false,
	// 	Indent:       "",
	// 	AnyResolver:  nil,
	// }
	// s, err := marshaler.MarshalToString(msg)
	// if err != nil {
	// 	log.Println(errors.Wrap(err, "marshaler.MarshalToString()"))
	// 	return m.Result
	// }
	//
	// log.Println("marshaler.MarshalToString(): ", s)
	//
	// return []byte(s)
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
		log.Println(errors.Wrap(err, "parser.ParseBlockFromReply()"))
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
	switch connectionType {
	case "grpc_server_descriptors": // TODO: make a const.
		serviceApi := &spectypes.ServiceApi{
			ComputeUnits: 10,
		}
		nodeMsg := &GrpcMessage{
			cp:             cp,
			serviceApi:     serviceApi,
			connectionType: connectionType,
		}

		return nodeMsg, nil
	default:
		// Check API is supported and save it in nodeMsg.
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
		log.Println("[nodeMsg.msg]", string(nodeMsg.msg.([]byte)))

		return nodeMsg, nil
	}
}

func (cp *GrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	// service := grpcdynamic2.NewService("t.Query")
	// service.RegisterUnaryMethod("Params", new(reqT), new(resT), func(ctx context.Context, in interface{}) (interface{}, error) {
	// 	log.Println("[RPC]")
	// 	req := in.(*reqT)
	// 	log.Println("[REQ]:", req)
	// 	return &resT{Pong: req.Ping}, nil
	// })
	// srv := grpcdynamic2.NewServer([]*grpcdynamic2.Service{service})
	// reflection.Register(srv)

	log.Println("gRPC PortalStart")
	reply, err := SendRelay(ctx, cp, privKey, "", "", "grpc_server_descriptors") // TODO: Make a const.
	if err != nil {
		log.Println(errors.Wrap(err, "SendRelay()"))
		utils.LavaFormatFatal("SendRelay()", err, nil)
		return
	}

	var serviceDescSlice []*grpc.ServiceDesc
	if err = json.Unmarshal(reply.GetData(), &serviceDescSlice); err != nil {
		log.Println(errors.Wrap(err, "json.Unmarshal()"))
		utils.LavaFormatFatal("json.Unmarshal()", err, nil)
		return
	}

	gs := grpcutil.NewServerFromServiceDescSlice(
		serviceDescSlice,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			log.Println("")
			m, ok := req.(*tmservice.GetLatestBlockRequest)
			if !ok {
				// TODO: return error
				return nil, utils.LavaFormatError("Not a proto message", err, nil)
			}

			// m := &tmservice.GetBlockByHeightRequest{Height: 5}

			// var b []byte
			// if b, err = m.Marshal(); err != nil {
			// 	log.Println(errors.Wrap(err, "m.Marshal()"))
			// 	return nil, utils.LavaFormatError("grpcutil.Marshal()", err, nil)
			// }

			marshaler := jsonpb.Marshaler{
				OrigName:     true,
				EnumsAsInts:  false,
				EmitDefaults: false,
				Indent:       "",
			}
			var s string
			if s, err = marshaler.MarshalToString(m); err != nil {
				log.Println(errors.Wrap(err, "marshaler.MarshalToString()"))
				return nil, utils.LavaFormatError("marshaler.MarshalToString()", err, nil)
			}

			log.Println("[HANDLER]: SendRelay")

			var relayReply *pairingtypes.RelayReply
			if relayReply, err = SendRelay(ctx, cp, privKey, "cosmos.base.tendermint.v1beta1.Service/GetLatestBlock", s, "grpc"); err != nil {
				return nil, errors.Wrap(err, "SendRelay()")
			}

			// var getBlockByHeightResponse tmservice.GetBlockByHeightResponse
			// if err = getBlockByHeightResponse.Unmarshal(relayReply.Data); err != nil {
			// 	log.Println("[ERR]: getBlockByHeightResponse.Unmarshal()", err.Error())
			// 	return nil, errors.Wrap(err, "getBlockByHeightResponse.Unmarshal()")
			// }
			// if err = jsonpb.UnmarshalString(string(relayReply.Data), &getBlockByHeightResponse); err != nil {
			// 	log.Println("[ERR]: jsonpb.UnmarshalString()", err.Error())
			// 	return nil, errors.Wrap(err, "jsonpb.UnmarshalString()")
			// }

			// log.Println("[relayReply.String()]", getBlockByHeightResponse.String())
			return relayReply, nil

			// m, ok := req.(proto.Message)
			// if !ok {
			// 	// TODO: return error
			// 	return nil, utils.LavaFormatError("Not a proto message", err, nil)
			// }
			//
			// var b []byte
			// if b, err = grpcutil.Marshal(m); err != nil {
			// 	return nil, utils.LavaFormatError("grpcutil.Marshal()", err, nil)
			// }
			//
			// log.Println("[HANDLER]: SendRelay")
			// return SendRelay(ctx, cp, privKey, "", string(b), "")
		},
	)

	// gs := grpc.NewServer()
	// gs, err := cp.GetSentry().ProxyGRPC(ctx)
	// if err != nil {
	// 	utils.LavaFormatFatal("cp.GetSentry().ProxyGRPC()", err, nil)
	// }

	// proxypb.RegisterProxyServiceServer(gs, NewServer(cp, privKey))
	log.Println("gogoreflection.Register()")
	gogoreflection.Register(gs)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Println("net.Listen()", err.Error())
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
	}

	_ = utils.LavaFormatInfo("Server listening", &map[string]string{"Address": lis.Addr().String()})

	log.Println("[ADDR]:", listenAddr)
	if err = gs.Serve(lis); err != nil {
		utils.LavaFormatFatal("portal failed to serve", err, &map[string]string{"Address": lis.Addr().String()})
	}
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

type WrapPBI interface {
	proto.Message
	Unmarshal(data []byte) (err error)
}

type WrapPB struct{}

func (w *WrapPB) Reset() {
	// TODO implement me
	panic("implement me")
}

func (w *WrapPB) String() string {
	// TODO implement me
	panic("implement me")
}

func (w *WrapPB) ProtoMessage() {
	// TODO implement me
	panic("implement me")
}

func NewWrapPB() WrapPBI {
	return &WrapPB{}
}

func (w *WrapPB) Unmarshal(data []byte) (err error) {
	return nil
}

func (nm *GrpcMessage) invokeV2(ctx context.Context, conn *grpc.ClientConn) (b []byte, err error) {
	cl := grpcreflect.NewClient(ctx, reflectionpb.NewServerReflectionClient(conn))
	descriptorSource := descriptorSourceFromServer(cl)
	svc, methodName := parseSymbol(nm.path)

	var descriptor desc.Descriptor
	if descriptor, err = descriptorSource.FindSymbol(svc); err != nil {
		return nil, errors.Wrap(err, "descriptorSource.FindSymbol()")
	}

	serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, errors.Wrap(err, "must be a '*desc.ServiceDescriptor'")
	}

	methodDescriptor := serviceDescriptor.FindMethodByName(methodName)
	nm.methodDesc = methodDescriptor
	msgFactory := dynamic.NewMessageFactoryWithDefaults()
	stub := grpcdynamic.NewStubWithMessageFactory(conn, msgFactory)
	_ = stub

	var reader io.Reader
	switch v := nm.msg.(type) {
	case []byte:
		// log.Println("[switch]: []byte")
		// log.Println("[DEBUG]: reader", string(v))
		reader = bytes.NewReader(v)
	case string:
		// log.Println("[switch]: string")
		// log.Println("[DEBUG]: reader", v)
		reader = strings.NewReader(v)
	case proto.Message:
		// log.Println("[switch]: proto.Message")
		marshaler := jsonpb.Marshaler{
			OrigName:     true,
			EnumsAsInts:  false,
			EmitDefaults: false,
			Indent:       "",
		}

		var s string
		if s, err = marshaler.MarshalToString(v); err != nil {
			return nil, errors.Wrap(err, "marshaler.MarshalToString()")
		}

		// log.Println("[DEBUG]: reader", s)
		reader = strings.NewReader(s)
	default:
		log.Println("[switch]: default")
		reader = strings.NewReader("")
	}

	var requestParser grpcurl.RequestParser
	var formatter grpcurl.Formatter
	if requestParser, formatter, err = grpcurl.RequestParserAndFormatter(
		grpcurl.FormatJSON,
		descriptorSource,
		reader,
		grpcurl.FormatOptions{
			EmitJSONDefaultFields: false,
			IncludeTextSeparator:  false,
			AllowUnknownFields:    true,
		},
	); err != nil {
		return nil, errors.Wrap(err, "grpcurl.RequestParserAndFormatter()")
	}
	_ = requestParser
	nm.formatter = formatter

	var reqBytes []byte
	if reqBytes, err = ioutil.ReadAll(reader); err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadAll()")
	}

	log.Println("[REQ TYPE]", methodDescriptor.GetInputType())
	log.Println("[RESP TYPE]", methodDescriptor.GetOutputType())

	msg := msgFactory.NewMessage(methodDescriptor.GetInputType())
	if len(reqBytes) != 0 {
		if err = jsonpb.UnmarshalString(string(reqBytes), msg); err != nil {
			return nil, errors.Wrap(err, "gjsonpb.UnmarshalString()")
		}
		// log.Println("[MSG 2]", msg)
		// if err = requestParser.Next(msg); err != nil {
		// 	return nil, errors.Wrap(err, "requestParser.Next()")
		// }
	}

	var resp hackPB
	if err = conn.Invoke(
		ctx,
		fmt.Sprintf("/%s/%s", methodDescriptor.GetService().GetFullyQualifiedName(), methodDescriptor.GetName()),
		msg,
		&resp,
	); err != nil {
		return nil, errors.Wrap(err, "conn.Invoke()")
	}

	// log.Println("json.RawMessage:", resp.String())
	return resp, nil
	// return []byte(resp.String()), nil

	// var respMessage proto.Message
	// if respMessage, err = stub.InvokeRpc(ctx, methodDescriptor, msg); err != nil {
	// 	return nil, errors.Wrap(err, "stub.InvokeRpc()")
	// }

	// if b, err = proto.Marshal(respMessage); err != nil {
	// 	return nil, errors.Wrap(err, "proto.Marshal()")
	// }
	//
	// return b, nil

	// extensionRegistry := dynamic.NewExtensionRegistryWithDefaults()
	// if err = extensionRegistry.AddExtension(methodDescriptor.GetOutputType().GetFields()...); err != nil {
	// 	return nil, errors.Wrap(err, "extensionRegistry.AddExtension()")
	// }

	// var dynamicMessage *dynamic.Message
	// if dynamicMessage, err = dynamic.AsDynamicMessage(respMessage); err != nil {
	// 	return nil, errors.Wrap(err, "dynamic.AsDynamicMessage()")
	// }
	// if b, err = dynamicMessage.Marshal(); err != nil {
	// 	return nil, errors.Wrap(err, "dynamicMessage.Marshal()")
	// }
	// log.Println("[RESP]", string(b))
	//
	// return b, nil

	// log.Println("[respMessage]", respMessage.String())

	// var s string
	// if s, err = formatter(respMessage); err != nil {
	// 	return nil, errors.Wrap(err, "formatter()")
	// }
	// log.Println("[RESP]", s)
	//
	// return []byte(s), nil
}

// type HackPB interface {
// 	encoding.Codec
// 	proto.Message
// }

type hackPB json.RawMessage

// func NewHackPB() HackPB {
// 	return &hackPB{}
// }

func (h hackPB) Reset()         {}
func (h hackPB) String() string { return string(h) }
func (h hackPB) ProtoMessage()  {}
func (h hackPB) Marshal(v interface{}) ([]byte, error) {
	log.Println("HackPB.Marshal()")
	log.Printf("%T", v)
	return h, nil
}
func (h *hackPB) Unmarshal(dAtA []byte) error {
	log.Println("HackPB.Unmarshal()")
	log.Println("data:", string(dAtA))
	*h = dAtA
	return nil
}
func (h hackPB) Unmarshal2(data []byte, v interface{}) (err error) {
	log.Println("HackPB.Unmarshal()")
	log.Printf("%T", v)
	log.Println("data:", string(data))
	h = data
	return nil
}
func (h hackPB) Name() string {
	return "hackPB"
}

// func (nm *GrpcMessage) invoke(ctx context.Context, conn *grpc.ClientConn) ([]byte, error) {
// 	client := grpcreflect.NewClient(ctx, reflectionpb.NewServerReflectionClient(conn))
// 	source := descriptorSourceFromServer(client)
// 	svc, mth := parseSymbol(nm.path)
// 	dsc, err := source.FindSymbol(svc)
// 	if err != nil {
// 		return nil, utils.LavaFormatError("failed to find symbol", err, &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
// 	}
// 	sd, ok := dsc.(*desc.ServiceDescriptor)
// 	if !ok {
// 		return nil, utils.LavaFormatError("failed to expose", fmt.Errorf("server does not expose service %q", svc), &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
// 	}
// 	mtd := sd.FindMethodByName(mth)
// 	if mtd == nil {
// 		return nil, utils.LavaFormatError("failed to expose", fmt.Errorf("service %q does not include a method named %q", svc, mth), &map[string]string{"symbol": nm.path, "svc": svc, "method": mth})
// 	}
//
// 	msgFactory := dynamic.NewMessageFactoryWithDefaults()
// 	req := msgFactory.NewMessage(mtd.GetInputType())
// 	stub := grpcdynamic.NewStubWithMessageFactory(conn, msgFactory)
//
// 	// m, ok := nm.msg.(proto.Message)
//
// 	var in io.Reader
// 	input := ""
// 	switch inputContent := nm.msg.(type) {
// 	case []byte:
// 		input = string(inputContent)
// 	case string:
// 		input = inputContent
// 	// case proto.Message:
// 	// 	log.Println("[INFO] proto.Message")
// 	// 	var b []byte
// 	// 	if b, err = grpcutil.Marshal(inputContent); err != nil {
// 	// 		log.Println(errors.Wrap(err, "grpcutil.Marshal()"))
// 	// 		return nil, utils.LavaFormatError("grpcutil.Marshal()", err, nil)
// 	// 	}
// 	//
// 	// 	input = string(b)
// 	default:
// 		utils.LavaFormatInfo(fmt.Sprintf("msg type %T is unknown. continuing with empty message parameters", nm.msg), &map[string]string{"msg": fmt.Sprintf("%v", nm.msg)})
// 	}
//
// 	log.Println("[INFO] input:", input)
// 	log.Println("[INFO] path:", nm.path)
//
// 	in = strings.NewReader(input)
//
// 	options := grpcurl.FormatOptions{
// 		EmitJSONDefaultFields: false,
// 		IncludeTextSeparator:  false,
// 		AllowUnknownFields:    true,
// 	}
// 	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, source, in, options)
// 	if err != nil {
// 		log.Println("grpcurl.RequestParserAndFormatter()", err.Error())
// 		return nil, utils.LavaFormatError("grpcurl.RequestParserAndFormatter()", err, nil)
// 	}
// 	h := &grpcurl.DefaultEventHandler{
// 		Out:            os.Stdout,
// 		Formatter:      formatter,
// 		VerbosityLevel: 0,
// 	}
//
// 	rff := rf.Next
// 	err = rff(req)
// 	if err != nil {
// 		log.Println(errors.Wrap(err, "rff()"))
// 		return nil, utils.LavaFormatError("rff()", err, nil)
// 	}
//
// 	log.Println("[REQ]:", req.String())
// 	log.Println("[MTD]:", mtd)
// 	msg, err := stub.InvokeRpc(ctx, mtd, req)
// 	if err != nil {
// 		log.Println("[stub.InvokeRpc]:", err.Error())
// 		return nil, utils.LavaFormatError("stub.InvokeRpc()", err, nil)
// 	}
// 	log.Println("[stub.InvokeRpc]: no error")
//
// 	respStr, err := h.Formatter(msg)
// 	if err != nil {
// 		log.Println("[MSG]:", msg.String())
// 		log.Println("[gRPC Formatter error]:", err.Error())
// 		return nil, utils.LavaFormatError("failed to format response from chain", err, &map[string]string{"msg": fmt.Sprintf("%v", msg)})
// 	}
//
// 	return []byte(respStr), nil
// }
func (nm *GrpcMessage) Send(ctx context.Context) (*pairingtypes.RelayReply, error) {
	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Todo, maybe create a list of grpc clients in connector.go instead of creating one and closing it each time?
	conn, err := grpc.DialContext(connectCtx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}
	defer conn.Close()

	var respBytes []byte
	switch nm.connectionType {
	case "grpc_server_descriptors": // TODO: Make a const.
		// var fileDescriptorSlice []*desc.FileDescriptor
		// if fileDescriptorSlice, err = grpcutil.GetFileDescriptorSlice(ctx, conn); err != nil {
		// 	nm.Result = []byte(fmt.Sprintf("%s", err))
		// 	return nil, utils.LavaFormatError("Receiving the gRPC service descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		// }
		//
		// var b []byte
		// if b, err = json.Marshal(fileDescriptorSlice); err != nil {
		// 	log.Println("[ERROR!]:", err.Error())
		// 	nm.Result = []byte(fmt.Sprintf("%s", err))
		// 	return nil, utils.LavaFormatError("Marshalling the gRPC file descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		// }
		//
		// log.Println("[OK] json.Marshal():", string(b))

		var grpcServiceDescSlice []*grpc.ServiceDesc
		if grpcServiceDescSlice, err = grpcutil.GetServiceDescSlice(ctx, conn); err != nil {
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, utils.LavaFormatError("Receiving the gRPC service descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}

		serviceDescSlice := make([]serviceDesc, 0, len(grpcServiceDescSlice))
		for _, grpcServiceDesc := range grpcServiceDescSlice {
			serviceDescSlice = append(serviceDescSlice, serviceDesc(*grpcServiceDesc))
		}

		var b []byte
		if b, err = json.Marshal(serviceDescSlice); err != nil {
			log.Println("[ERROR!]:", err.Error())
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, utils.LavaFormatError("Marshalling the gRPC service descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}

		respBytes = b
	default:
		if respBytes, err = nm.invokeV2(ctx, conn); err != nil {
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, utils.LavaFormatError("Sending the gRPC message failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}
	}

	nm.Result = respBytes
	reply := &pairingtypes.RelayReply{
		Data: respBytes,
	}

	return reply, nil
}

type serviceDesc grpc.ServiceDesc

func (s *serviceDesc) MarshalJSON() ([]byte, error) {
	type Method struct {
		MethodName string
	}

	type ServiceDesc struct {
		ServiceName string
		Methods     []Method
	}

	methods := make([]Method, 0, len(s.Methods))
	for _, method := range s.Methods {
		methods = append(methods, Method{MethodName: method.MethodName})
	}

	return json.Marshal(ServiceDesc{
		ServiceName: s.ServiceName,
		Methods:     methods,
	})
}

// func handleReply(reply *pairingtypes.RelayReply) (resp *proxypb.ProxyResponse, err error) {
// 	log.Println("out >>> len", len(reply.GetData()))
//
// 	var pbAny *anypb.Any
// 	if pbAny, err = getAnyByInterface(reply.GetData()); err != nil {
// 		log.Println("[getAnyByInterface]:", err.Error())
// 		return nil, utils.LavaFormatError("", err, nil)
// 	}
//
// 	return &proxypb.ProxyResponse{Body: pbAny}, nil
// }
//
// func getAnyByInterface(v interface{}) (pbAny *anypb.Any, err error) {
// 	var pbValue *structpb.Value
// 	if pbValue, err = structpb.NewValue(v); err != nil {
// 		return nil, err
// 	}
//
// 	return anypb.New(pbValue)
// }
