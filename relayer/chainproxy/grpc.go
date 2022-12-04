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
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
	"github.com/lavanet/lava/relayer/chainproxy/rpcclient"
	"github.com/lavanet/lava/relayer/lavasession"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	reflectionpbo "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/runtime/protoiface"
)

const GRPC_DESCRIPTORS_REQUEST = "get_grpc_server_descriptors"

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
	nodeUrl    string
	sentry     *sentry.Sentry
	csm        *lavasession.ConsumerSessionManager
	portalLogs *PortalLogs
	test       interface{}
}

func (r *GrpcMessage) GetMsg() interface{} {
	return r.msg
}

func NewGrpcChainProxy(nodeUrl string, sentry *sentry.Sentry, csm *lavasession.ConsumerSessionManager, pLogs *PortalLogs) ChainProxy {
	nodeUrl = strings.TrimSuffix(nodeUrl, "/")
	return &GrpcChainProxy{
		nodeUrl:    nodeUrl,
		sentry:     sentry,
		csm:        csm,
		portalLogs: pLogs,
	}
}

func (cp *GrpcChainProxy) GetConsumerSessionManager() *lavasession.ConsumerSessionManager {
	return cp.csm
}

func (cp *GrpcChainProxy) NewMessage(path string, data []byte) (*GrpcMessage, error) {
	//
	// Check api is supported and save it in nodeMsg
	serviceApi, err := cp.getSupportedApi(path)
	if err != nil {
		return nil, utils.LavaFormatError("failed to get supported api in NewMessage", err, &map[string]string{"path": path})
	}

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

	_, _, _, err = nodeMsg.Send(ctx, nil)
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

	_, _, _, err = nodeMsg.Send(ctx, nil)
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

func (cp *GrpcChainProxy) PortalStart(ctx context.Context, privKey *btcec.PrivateKey, listenAddr string) {
	log.Println("gRPC PortalStart")
	reply, _, _, err := SendRelay(ctx, cp, privKey, GRPC_DESCRIPTORS_REQUEST, "", "")
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
			log.Println(fmt.Sprintf("Got request, %v", req))
			m, ok := req.(*tmservice.GetLatestBlockRequest)
			if !ok {
				return nil, utils.LavaFormatError("Not a proto message", err, nil)
			}
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

			// info ,err := proto.Marshal(req.(proto.Message))

			var relayReply *pairingtypes.RelayReply
			var nodeMsg NodeMessage
			if relayReply, _, nodeMsg, err = SendRelay(ctx, cp, privKey, "cosmos.base.tendermint.v1beta1.Service/GetLatestBlock", s, "grpc"); err != nil {
				return nil, errors.Wrap(err, "SendRelay()")
			}

			grpcMesssage := nodeMsg.(*GrpcMessage)
			if grpcMesssage.methodDesc == nil {
				return nil, utils.LavaFormatError("gRPC message method descriptor is nil.", nil, nil)
			}
			msgFactory := dynamic.NewMessageFactoryWithDefaults()
			msg := msgFactory.NewMessage(grpcMesssage.methodDesc.GetOutputType())
			err := proto.Unmarshal(relayReply.Data, msg)
			if err != nil {
				return nil, utils.LavaFormatError("Error unmarshalling proto using descriptor", err, nil)
			}
			log.Println(fmt.Sprintf("Got Response: %s", string(relayReply.Data)))
			return relayReply.Data, nil
		},
	)
	log.Println("gogoreflection.Register()")
	gogoreflection.Register(gs)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Println("net.Listen()", err.Error())
		utils.LavaFormatFatal("provider failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
	}

	utils.LavaFormatInfo("Server listening", &map[string]string{"Address": lis.Addr().String()})
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

func descriptorSourceFromServer(refClient *grpcreflect.Client) grpcutil.DescriptorSource {
	return grpcutil.ServerSource{Client: refClient}
}

func (nm *GrpcMessage) initInvoke(ctx context.Context) (string, protoiface.MessageV1, error) {
	conn, err := grpc.DialContext(ctx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return "", nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}
	defer conn.Close()
	cl := grpcreflect.NewClient(ctx, reflectionpbo.NewServerReflectionClient(conn))
	descriptorSource := descriptorSourceFromServer(cl)
	svc, methodName := grpcutil.ParseSymbol(nm.path)

	var descriptor desc.Descriptor
	if descriptor, err = descriptorSource.FindSymbol(svc); err != nil {
		return "", nil, errors.Wrap(err, "descriptorSource.FindSymbol()")
	}

	serviceDescriptor, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return "", nil, errors.Wrap(err, "must be a '*desc.ServiceDescriptor'")
	}

	methodDescriptor := serviceDescriptor.FindMethodByName(methodName)
	nm.methodDesc = methodDescriptor
	msgFactory := dynamic.NewMessageFactoryWithDefaults()

	var reader io.Reader
	switch v := nm.msg.(type) {
	case []byte:
		log.Println("[switch]: []byte")
		log.Println("[DEBUG]: reader", string(v))
		reader = bytes.NewReader(v)
	case string:
		log.Println("[switch]: string")
		log.Println("[DEBUG]: reader", v)
		reader = strings.NewReader(v)
	case proto.Message:
		log.Println("[switch]: proto.Message")
		marshaler := jsonpb.Marshaler{
			OrigName:     true,
			EnumsAsInts:  false,
			EmitDefaults: false,
			Indent:       "",
		}

		var s string
		if s, err = marshaler.MarshalToString(v); err != nil {
			return "", nil, errors.Wrap(err, "marshaler.MarshalToString()")
		}

		log.Println("[DEBUG]: reader", s)
		reader = strings.NewReader(s)
	default:
		return "", nil, utils.LavaFormatError("Unsupported type for gRPC msg", nil, &map[string]string{"type": fmt.Sprintf("%s", v)})
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
		return "", nil, errors.Wrap(err, "grpcurl.RequestParserAndFormatter()")
	}
	_ = requestParser
	nm.formatter = formatter

	var reqBytes []byte
	if reqBytes, err = ioutil.ReadAll(reader); err != nil {
		return "", nil, errors.Wrap(err, "ioutil.ReadAll()")
	}

	log.Println("[REQ TYPE]", methodDescriptor.GetInputType())
	log.Println("[RESP TYPE]", methodDescriptor.GetOutputType())

	msg := msgFactory.NewMessage(methodDescriptor.GetInputType())
	if len(reqBytes) != 0 {
		if err = jsonpb.UnmarshalString(string(reqBytes), msg); err != nil {
			return "", nil, errors.Wrap(err, "gjsonpb.UnmarshalString()")
		}
	}
	return fmt.Sprintf("%s/%s", methodDescriptor.GetService().GetFullyQualifiedName(), methodDescriptor.GetName()), msg, nil
}

func (nm *GrpcMessage) invokeV2(ctx context.Context) (b []byte, err error) {
	method, msg, err := nm.initInvoke(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("initInvoke failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}

	conn, err := grpc.DialContext(ctx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
	}
	defer conn.Close()

	var resp []byte
	if err = conn.Invoke(
		ctx,
		method,
		msg,
		&resp,
	); err != nil {
		return nil, errors.Wrap(err, "conn.Invoke()")
	}
	return resp, nil
}

func (nm *GrpcMessage) Send(ctx context.Context, ch chan interface{}) (relayReply *pairingtypes.RelayReply, subscriptionID string, relayReplyServer *rpcclient.ClientSubscription, err error) {
	if ch != nil {
		return nil, "", nil, utils.LavaFormatError("Subscribe is not allowed on rest", nil, nil)
	}

	connectCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Todo, maybe create a list of grpc clients in connector.go instead of creating one and closing it each time?

	var respBytes []byte
	switch nm.path {
	case GRPC_DESCRIPTORS_REQUEST:
		conn, err := grpc.DialContext(connectCtx, nm.cp.nodeUrl, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			return nil, "", nil, utils.LavaFormatError("dialing the gRPC client failed.", err, &map[string]string{"addr": nm.cp.nodeUrl})
		}
		defer conn.Close()
		var grpcServiceDescSlice []*grpc.ServiceDesc
		if grpcServiceDescSlice, err = grpcutil.GetServiceDescSlice(ctx, conn); err != nil {
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, "", nil, utils.LavaFormatError("Receiving the gRPC service descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}

		serviceDescSlice := make([]grpcutil.ServiceDesc, 0, len(grpcServiceDescSlice))
		for _, grpcServiceDesc := range grpcServiceDescSlice {
			serviceDescSlice = append(serviceDescSlice, grpcutil.ServiceDesc(*grpcServiceDesc))
		}

		var b []byte
		if b, err = json.Marshal(serviceDescSlice); err != nil {
			log.Println("[ERROR!]:", err.Error())
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, "", nil, utils.LavaFormatError("Marshalling the gRPC service descriptors failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}

		respBytes = b
	default:
		if respBytes, err = nm.invokeV2(ctx); err != nil {
			nm.Result = []byte(fmt.Sprintf("%s", err))
			return nil, "", nil, utils.LavaFormatError("Sending the gRPC message failed.", err, &map[string]string{"msg": fmt.Sprintf("%s", nm.msg), "addr": nm.cp.nodeUrl, "path": nm.path})
		}
	}

	nm.Result = respBytes
	reply := &pairingtypes.RelayReply{
		Data: respBytes,
	}

	return reply, "", nil, nil
}
