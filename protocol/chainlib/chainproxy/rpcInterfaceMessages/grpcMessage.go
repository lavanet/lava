package rpcInterfaceMessages

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-json"

	"github.com/fullstorydev/grpcurl"
	"github.com/gogo/status"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	dyncodec "github.com/lavanet/lava/v4/protocol/chainlib/grpcproxy/dyncodec"
	"github.com/lavanet/lava/v4/protocol/parser"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/sigs"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type GrpcMessage struct {
	Msg        []byte
	Path       string
	methodDesc *desc.MethodDescriptor
	formatter  grpcurl.Formatter

	Registry *dyncodec.Registry
	Codec    *dyncodec.Codec
	chainproxy.BaseMessage
}

func (gm *GrpcMessage) SubscriptionIdExtractor(reply *rpcclient.JsonrpcMessage) string {
	return ""
}

// get msg hash byte array containing all the relevant information for a unique request. (headers / api / params)
func (gm *GrpcMessage) GetRawRequestHash() ([]byte, error) {
	headers := gm.GetHeaders()
	headersByteArray, err := json.Marshal(headers)
	if err != nil {
		utils.LavaFormatError("Failed marshalling headers on jsonRpc message", err, utils.LogAttr("headers", headers))
		return []byte{}, err
	}
	pathByteArray := []byte(gm.Path)
	return sigs.HashMsg(append(append(pathByteArray, gm.Msg...), headersByteArray...)), nil
}

func (jm GrpcMessage) CheckResponseError(data []byte, httpStatusCode int) (hasError bool, errorMessage string) {
	// grpc status code different than OK or 0 is a node error.
	if httpStatusCode != 0 && httpStatusCode != http.StatusOK {
		return true, ""
	}
	return false, ""
}

// GetParams will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (gm GrpcMessage) GetParams() interface{} {
	if len(gm.Msg) > 0 {
		if gm.Msg[0] == '{' || gm.Msg[0] == '[' {
			var parsedData interface{}
			err := json.Unmarshal(gm.Msg, &parsedData)
			if err != nil {
				utils.LavaFormatError("failed to unmarshal GetParams", err)
				return nil
			}
			return parsedData
		}
	}
	parsedData, err := gm.dynamicResolve()
	if err != nil {
		utils.LavaFormatError("failed to dynamicResolve", err)
		return nil
	}
	return parsedData
}

func (gm *GrpcMessage) UpdateLatestBlockInMessage(latestBlock uint64, modifyContent bool) (success bool) {
	// return gm.SetLatestBlockWithHeader(latestBlock, modifyContent)
	// disabled due to cosmos sdk inconsistency with the headers that needs to be handled
	return false
	// when !done: we need a different setter
}

func (gm GrpcMessage) dynamicResolve() (interface{}, error) {
	md, err := gm.Registry.FindDescriptorByName(protoreflect.FullName(strings.ReplaceAll(gm.Path, "/", ".")))
	if err != nil {
		return nil, err
	}
	msg := dynamicpb.NewMessage(md.(protoreflect.MethodDescriptor).Input())
	err = gm.Codec.UnmarshalProto(gm.Msg, msg)
	if err != nil {
		return nil, err
	}
	jsonMsg, err := gm.Codec.MarshalProtoJSON(msg)
	if err != nil {
		return nil, err
	}
	var parsedData interface{}
	err = json.Unmarshal(jsonMsg, &parsedData)
	if err != nil {
		return nil, err
	}

	return parsedData, nil
}

// GetResult will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (gm GrpcMessage) GetResult() json.RawMessage {
	return nil
}

func (gm GrpcMessage) GetMethod() string {
	return gm.Path
}

func (gm GrpcMessage) GetID() json.RawMessage {
	return nil
}

func (gm GrpcMessage) GetError() *rpcclient.JsonError {
	return nil
}

func (gm GrpcMessage) NewParsableRPCInput(input json.RawMessage) (parser.RPCInput, error) {
	msgFactory := dynamic.NewMessageFactoryWithDefaults()
	if gm.methodDesc == nil {
		return nil, utils.LavaFormatError("does not have a methodDescriptor set in grpcMessage", nil)
	}
	msg := msgFactory.NewMessage(gm.methodDesc.GetOutputType())
	if err := proto.Unmarshal(input, msg); err != nil {
		return nil, utils.LavaFormatError("failed to unmarshal input", err)
	}

	formattedInput, err := gm.formatter(msg)
	if err != nil {
		return nil, utils.LavaFormatError("m.formatter(msg)", err)
	}
	return ParsableRPCInput{Result: []byte(formattedInput)}, nil
}

func (gm *GrpcMessage) SetParsingData(methodDesc *desc.MethodDescriptor, formatter grpcurl.Formatter) {
	gm.formatter = formatter
	gm.methodDesc = methodDesc
}

func (gm GrpcMessage) ParseBlock(inp string) (int64, error) {
	return parser.ParseDefaultBlockParameter(inp)
}

type ServerSource struct {
	Client *grpcreflect.Client
}

func DescriptorSourceFromServer(refClient *grpcreflect.Client) grpcurl.DescriptorSource {
	return ServerSource{Client: refClient}
}

func (ss ServerSource) ListServices() ([]string, error) {
	svcs, err := ss.Client.ListServices()
	return svcs, ReflectionSupport(err)
}

func (ss ServerSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	file, err := ss.Client.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, ReflectionSupport(err)
	}
	d := file.FindSymbol(fullyQualifiedName)
	if d == nil {
		return nil, utils.LavaFormatError("Symbol not found", fmt.Errorf("missing symbol: %s", fullyQualifiedName))
	}
	return d, nil
}

func (ss ServerSource) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	var exts []*desc.FieldDescriptor
	nums, err := ss.Client.AllExtensionNumbersForType(typeName)
	if err != nil {
		return nil, ReflectionSupport(err)
	}
	for _, fieldNum := range nums {
		ext, err := ss.Client.ResolveExtension(typeName, fieldNum)
		if err != nil {
			return nil, ReflectionSupport(err)
		}
		exts = append(exts, ext)
	}
	return exts, nil
}

func ReflectionSupport(err error) error {
	if err == nil {
		return nil
	}
	if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
		return utils.LavaFormatError("server does not support the reflection API", err)
	}
	return err
}

func ParseSymbol(svcAndMethod string) (string, string) {
	pos := strings.LastIndex(svcAndMethod, "/")
	if pos < 0 {
		pos = strings.LastIndex(svcAndMethod, ".")
		if pos < 0 {
			return "", ""
		}
	}
	return svcAndMethod[:pos], svcAndMethod[pos+1:]
}
