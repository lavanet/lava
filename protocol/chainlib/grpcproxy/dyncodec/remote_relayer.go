package dyncodec

import (
	"context"

	"github.com/lavanet/lava/v2/protocol/chainlib/grpcproxy"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ProtoFileRegistry = (*RelayerRemote)(nil)

func NewRelayerRemote(relay grpcproxy.ProxyCallBack) *RelayerRemote {
	return &RelayerRemote{relay: relay}
}

type RelayerRemote struct {
	relay grpcproxy.ProxyCallBack
}

func (r RelayerRemote) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	return r.sendReq(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{
			FileByFilename: path,
		},
	})
}

func (r RelayerRemote) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	return r.sendReq(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})
}

func (r RelayerRemote) sendReq(req *grpc_reflection_v1alpha.ServerReflectionRequest) (*descriptorpb.FileDescriptorProto, error) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBytes, _, err := r.relay(context.Background(), "grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", reqBytes)
	if err != nil {
		return nil, err
	}

	respOneof := &grpc_reflection_v1alpha.ServerReflectionResponse{}
	err = proto.Unmarshal(respBytes, respOneof)
	if err != nil {
		return nil, err
	}
	return parseFileDescriptorResponse(respOneof)
}

func (r RelayerRemote) Close() error {
	return nil
}
