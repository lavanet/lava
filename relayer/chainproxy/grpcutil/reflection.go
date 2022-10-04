package grpcutil

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewProxyServer(
	ctx context.Context,
	addr string,
	whitelistMethodDescriptorProtoNameSlice map[string]struct{},
) (server *grpc.Server, err error) {
	server = grpc.NewServer()
	var serverReflectionInfoClient reflection.ServerReflection_ServerReflectionInfoClient
	if serverReflectionInfoClient, err = reflection.NewServerReflectionClient(MustDial(ctx, addr)).ServerReflectionInfo(ctx); err != nil {
		return nil, errors.Wrap(err, "reflection.NewServerReflectionClient().ServerReflectionInfo()")
	}

	if err = serverReflectionInfoClient.Send(&reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}); err != nil {
		if errors.Is(err, io.EOF) {
			_, err = serverReflectionInfoClient.Recv()
		}

		return nil, errors.Wrap(err, "serverReflectionInfoClient.Send()")
	}

	var serverReflectionResp *reflection.ServerReflectionResponse
	if serverReflectionResp, err = serverReflectionInfoClient.Recv(); err != nil {
		return nil, errors.Wrap(err, "serverReflectionInfoClient.Recv()")
	}

	if errResponse := serverReflectionResp.GetErrorResponse(); errResponse != nil {
		return nil, errors.Wrap(
			status.Error(codes.Code(errResponse.GetErrorCode()), errResponse.GetErrorMessage()),
			"serverReflectionInfoClient.GetErrorResponse()",
		)
	}

	for _, service := range serverReflectionResp.GetListServicesResponse().GetService() {
		if err = serverReflectionInfoClient.Send(&reflection.ServerReflectionRequest{
			MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: service.GetName(),
			},
		}); err != nil {
			if errors.Is(err, io.EOF) {
				_, err = serverReflectionInfoClient.Recv()
			}

			return nil, errors.Wrap(err, "serverReflectionInfoClient.Send()")
		}

		if serverReflectionResp, err = serverReflectionInfoClient.Recv(); err != nil {
			return nil, errors.Wrap(err, "serverReflectionInfoClient.Recv()")
		}

		if errResponse := serverReflectionResp.GetErrorResponse(); errResponse != nil {
			return nil, errors.Wrap(
				status.Error(codes.Code(errResponse.GetErrorCode()), errResponse.GetErrorMessage()),
				"serverReflectionResp.GetErrorResponse()",
			)
		}

		for _, fileDescriptorProto := range serverReflectionResp.GetFileDescriptorResponse().GetFileDescriptorProto() {
			var pbFileDescriptorProto descriptorpb.FileDescriptorProto
			if err = proto.Unmarshal(fileDescriptorProto, &pbFileDescriptorProto); err != nil {
				return nil, err
			}

			nameToDescriptorProto := make(map[string]*descriptorpb.DescriptorProto, len(pbFileDescriptorProto.GetMessageType()))
			for _, descriptorProto := range pbFileDescriptorProto.GetMessageType() {
				nameToDescriptorProto[descriptorProto.GetName()] = descriptorProto
			}
			for _, serviceDescriptorProtoSlice := range pbFileDescriptorProto.GetService() {
				serviceName := fmt.Sprintf("%s.%s", pbFileDescriptorProto.GetPackage(), serviceDescriptorProtoSlice.GetName())
				methods := make([]grpc.MethodDesc, 0, len(serviceDescriptorProtoSlice.GetMethod()))
				for _, methodDescriptorProto := range serviceDescriptorProtoSlice.GetMethod() {
					stringSlice := strings.Split(methodDescriptorProto.GetInputType(), ".")
					if len(stringSlice) == 0 {
						continue
					}

					messageName := stringSlice[len(stringSlice)-1]
					descriptorProto, ok := nameToDescriptorProto[messageName]
					if !ok {
						continue
					}

					message := descriptorProto.ProtoReflect().New().Interface()
					methodName := methodDescriptorProto.GetName()
					if _, ok = whitelistMethodDescriptorProtoNameSlice[methodName]; !ok {
						continue
					}

					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						// proto.Unmarshal()
						// TODO: Create handler based on grpc.MethodDesc.
						log.Println(req)
						return nil, nil
					}
					methods = append(methods, grpc.MethodDesc{
						MethodName: methodName,
						Handler: func(
							srv interface{},
							ctx context.Context,
							dec func(interface{}) error,
							interceptor grpc.UnaryServerInterceptor,
						) (interface{}, error) {
							log.Println("Handler")

							req := message
							if err := dec(req); err != nil {
								log.Println(err.Error())
								return nil, err
							}
							if interceptor != nil {
								unaryServerInfo := &grpc.UnaryServerInfo{
									Server:     srv,
									FullMethod: fmt.Sprintf("/%s/%s", serviceName, methodName),
								}
								return interceptor(ctx, req, unaryServerInfo, handler)
							}

							return handler(ctx, req)
						},
					})
				}

				type emptyI interface{}
				server.RegisterService(&grpc.ServiceDesc{
					ServiceName: serviceName,
					HandlerType: (*emptyI)(nil),
					Methods:     methods,
					Streams:     nil,
					Metadata:    "api/v1/service.proto", // TODO: Change.
				}, struct{}{})
			}
		}
	}

	return server, nil
}
