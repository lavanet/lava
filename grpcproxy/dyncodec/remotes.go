package dyncodec

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"sync"
)

func NewGRPCReflectionProtoFileRegistry(grpcEndpoint string) (*GRPCReflectionProtoFileRegistry, error) {
	conn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCReflectionProtoFileRegistry{
		rpb:    grpc_reflection_v1alpha.NewServerReflectionClient(conn),
		once:   new(sync.Once),
		stream: nil,
	}, nil
}

// GRPCReflectionProtoFileRegistry is a ProtoFileRegistry
// which uses grpc reflection to resolve files.
type GRPCReflectionProtoFileRegistry struct {
	rpb    grpc_reflection_v1alpha.ServerReflectionClient
	once   *sync.Once
	stream grpc_reflection_v1alpha.ServerReflection_ServerReflectionInfoClient
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	err := g.init()
	if err != nil {
		return nil, err
	}

	err = g.stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{
			FileByFilename: path,
		}})
	if err != nil {
		return nil, err
	}

	recv, err := g.stream.Recv()
	if err != nil {
		return nil, err
	}

	resp, err := maybeFileDescriptorResponse(recv)
	if err != nil {
		return nil, err
	}
	fdRawBytes := resp.FileDescriptorResponse.FileDescriptorProto[0]
	fdPb := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(fdRawBytes, fdPb)
	if err != nil {
		return nil, err
	}

	return fdPb, nil
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	err := g.init()
	if err != nil {
		return nil, err
	}

	err = g.stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})

	if err != nil {
		return nil, err
	}

	recv, err := g.stream.Recv()
	if err != nil {
		return nil, err
	}

	resp, err := maybeFileDescriptorResponse(recv)
	if err != nil {
		return nil, err
	}
	fdRawBytes := resp.FileDescriptorResponse.FileDescriptorProto[0]
	fdPb := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(fdRawBytes, fdPb)
	if err != nil {
		return nil, err
	}

	return fdPb, nil
}

func maybeFileDescriptorResponse(resp *grpc_reflection_v1alpha.ServerReflectionResponse) (*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse, error) {
	r, ok := resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return nil, fmt.Errorf("%#v", resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_ErrorResponse).ErrorResponse.ErrorMessage)
	}
	return r, nil
}

func (g *GRPCReflectionProtoFileRegistry) init() (err error) {
	g.once.Do(func() {
		g.stream, err = g.rpb.ServerReflectionInfo(context.Background())
	})
	return err
}

func (g *GRPCReflectionProtoFileRegistry) Close() error {
	return g.stream.CloseSend()
}
