package dyncodec

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v2/protocol/chainlib/grpcproxy"
	"github.com/lavanet/lava/v2/protocol/chainlib/grpcproxy/testproto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestRemotes(t *testing.T) {
	// create in memory grpc server with reflection
	grpcSrv := grpc.NewServer()
	reflection.Register(grpcSrv)
	conn := testproto.InMemoryClientConn(t, grpcSrv)

	testRemote := func(remote ProtoFileRegistry) {
		defer remote.Close()
		registry := NewRegistry(remote)
		codec := NewCodec(registry)
		desc, err := registry.FindDescriptorByName("grpc.reflection.v1alpha.ServerReflectionRequest")
		require.NoError(t, err)
		// assert dynamic protobuf works correctly
		convertedDesc, ok := desc.(protoreflect.MessageDescriptor)
		require.True(t, ok)
		msg := dynamicpb.NewMessage(convertedDesc)
		msg.Set(msg.Descriptor().Fields().ByName("host"), protoreflect.ValueOfString("test"))

		// proto marshalling must be equal
		reflectionProtoBytes, err := codec.MarshalProto(msg)
		require.NoError(t, err)
		concreteProtoBytes, err := proto.Marshal(&grpc_reflection_v1alpha.ServerReflectionRequest{Host: "test"})
		require.NoError(t, err)
		require.Equal(t, concreteProtoBytes, reflectionProtoBytes)

		// json marshalling must be equal too
		reflectionJSONBytes, err := codec.MarshalProtoJSON(msg)
		require.NoError(t, err)
		concreteJSONBytes, err := protojson.Marshal(&grpc_reflection_v1alpha.ServerReflectionRequest{Host: "test"})
		require.NoError(t, err)
		require.Equal(t, concreteJSONBytes, reflectionJSONBytes)

		// assert proto file by path works correctly
		protoFile, err := registry.FindFileByPath("google/protobuf/descriptor.proto")
		require.NoError(t, err)
		require.Equal(t,
			(&descriptorpb.DescriptorProto{}).ProtoReflect().Descriptor().ParentFile().FullName(),
			protoFile.FullName(),
		)

		// do some registry safety asserions
		mType, err := registry.FindMessageByName("grpc.reflection.v1alpha.ServerReflectionRequest")
		require.NoError(t, err)
		msg.Reset()
		require.Equal(t, mType.New(), msg)

		mType2, err := registry.FindMessageByURL("type.googleapis.com/grpc.reflection.v1alpha.ServerReflectionRequest")
		require.NoError(t, err)
		require.Equal(t, mType.New(), mType2.New())
	}

	t.Run("test grpc remote", func(t *testing.T) {
		remote := NewGRPCReflectionProtoFileRegistryFromConn(conn)
		testRemote(remote)
	})

	t.Run("test relayer remote", func(t *testing.T) {
		remote := NewRelayerRemote(func(ctx context.Context, method string, req []byte) ([]byte, metadata.MD, error) {
			var resp []byte
			err := conn.Invoke(ctx, method, req, &resp, grpc.CustomCodecCallOption{Codec: grpcproxy.RawBytesCodec{}})
			return resp, make(metadata.MD), err
		})
		testRemote(remote)
	})
}
