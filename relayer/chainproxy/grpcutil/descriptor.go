package grpcutil

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/lavanet/lava/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescriptorSourceFromServer(refClient *grpcreflect.Client) DescriptorSource {
	return ServerSource{Client: refClient}
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

type ServerSource struct {
	Client *grpcreflect.Client
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

type HackPB json.RawMessage

// func NewHackPB() HackPB {
// 	return &hackPB{}
// }

func (h HackPB) Reset()         {}
func (h HackPB) String() string { return string(h) }
func (h HackPB) ProtoMessage()  {}
func (h HackPB) Marshal(v interface{}) ([]byte, error) {
	log.Println("HackPB.Marshal()")
	log.Printf("%T", v)
	return h, nil
}
func (h *HackPB) Unmarshal(dAtA []byte) error {
	log.Println("HackPB.Unmarshal()")
	log.Println("data:", string(dAtA))
	*h = dAtA
	return nil
}
func (h HackPB) Unmarshal2(data []byte, v interface{}) (err error) {
	log.Println("HackPB.Unmarshal()")
	log.Printf("%T", v)
	log.Println("data:", string(data))
	h = data
	return nil
}
func (h HackPB) Name() string {
	return "hackPB"
}

type ServiceDesc grpc.ServiceDesc

func (s *ServiceDesc) MarshalJSON() ([]byte, error) {
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
