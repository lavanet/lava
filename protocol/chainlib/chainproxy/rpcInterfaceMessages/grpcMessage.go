package rpcInterfaceMessages

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fullstorydev/grpcurl"
	"github.com/gogo/status"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/relayer/parser"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc/codes"
)

type GrpcMessage struct {
	Msg  []byte
	Path string
}

// GetParams will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (cp GrpcMessage) GetParams() interface{} {
	return nil
}

// GetResult will be deprecated after we remove old client
// Currently needed because of parser.RPCInput interface
func (cp GrpcMessage) GetResult() json.RawMessage {
	return nil
}

func (cp GrpcMessage) ParseBlock(inp string) (int64, error) {
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
		return nil, utils.LavaFormatError("Symbol not found", fmt.Errorf("missing symbol: %s", fullyQualifiedName), nil)
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
		return utils.LavaFormatError("server does not support the reflection API", err, nil)
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
