package dyncodec

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/v2/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewGRPCReflectionProtoFileRegistryFromConn(conn *grpc.ClientConn) *GRPCReflectionProtoFileRegistry {
	return &GRPCReflectionProtoFileRegistry{
		rpb: grpc_reflection_v1alpha.NewServerReflectionClient(conn),
	}
}

// GRPCReflectionProtoFileRegistry is a ProtoFileRegistry
// which uses grpc reflection to resolve files.
type GRPCReflectionProtoFileRegistry struct {
	rpb grpc_reflection_v1alpha.ServerReflectionClient
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileByPath(path string) (_ *descriptorpb.FileDescriptorProto, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("proto file by path: %w", err)
		}
	}()

	stream, err := g.rpb.ServerReflectionInfo(context.Background())
	if err != nil {
		return nil, err
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{FileByFilename: path},
	})
	if err != nil {
		return nil, err
	}

	recv, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	return parseFileDescriptorResponse(recv)
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (_ *descriptorpb.FileDescriptorProto, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("proto file containing symbol: %w", err)
		}
	}()
	stream, err := g.rpb.ServerReflectionInfo(context.Background())
	if err != nil {
		return nil, err
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})
	if err != nil {
		return nil, err
	}

	recv, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	return parseFileDescriptorResponse(recv)
}

func maybeFileDescriptorResponse(resp *grpc_reflection_v1alpha.ServerReflectionResponse) (*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse, error) {
	r, ok := resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		errorResponse, convertionSuccessful := resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_ErrorResponse)
		if convertionSuccessful {
			return nil, fmt.Errorf("%#v", errorResponse.ErrorResponse.ErrorMessage)
		}
		return nil, utils.LavaFormatError("Failed to convert response to ServerReflectionResponse_FileDescriptorResponse and is not an error", nil, utils.Attribute{Key: "resp.MessageResponse", Value: resp.MessageResponse})
	}
	return r, nil
}

func (g *GRPCReflectionProtoFileRegistry) Close() error { return nil }

func parseFileDescriptorResponse(recv *grpc_reflection_v1alpha.ServerReflectionResponse) (*descriptorpb.FileDescriptorProto, error) {
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
