package grpcutil

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewServerFromServiceDescSlice(
	serviceDescSlice []*grpc.ServiceDesc,
	handler grpc.UnaryHandler,
) (server *grpc.Server) {
	server = grpc.NewServer()
	for _, serviceDesc := range serviceDescSlice {
		if serviceDesc.ServiceName == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}

		methodSlice := make([]grpc.MethodDesc, 0, len(serviceDesc.Methods))
		for _, method := range serviceDesc.Methods {
			log.Println("[METHOD]:", fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.MethodName))
			method.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				log.Println("Handler")

				req := new(tmservice.GetLatestBlockRequest)
				if err := dec(req); err != nil {
					log.Println(err.Error())
					return nil, err
				}
				if interceptor != nil {
					unaryServerInfo := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.MethodName),
					}
					return interceptor(ctx, req, unaryServerInfo, handler)
				}

				return handler(ctx, req)
			}

			methodSlice = append(methodSlice, method)
		}

		type emptyI interface{}
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: serviceDesc.ServiceName,
			HandlerType: (*emptyI)(nil),
			Methods:     methodSlice,
			// Streams:     serviceDesc.Streams,
			Metadata: serviceDesc.Metadata,
		}, struct{}{})
	}

	return server
}

func GetServiceDescSlice(
	ctx context.Context,
	conn *grpc.ClientConn,
) (serviceDescSlice []*grpc.ServiceDesc, err error) {
	var serverReflectionInfoClient reflectionpb.ServerReflection_ServerReflectionInfoClient
	if serverReflectionInfoClient, err = reflectionpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx); err != nil {
		return nil, errors.Wrap(err, "reflection.NewServerReflectionClient().ServerReflectionInfo()")
	}

	if err = serverReflectionInfoClient.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}); err != nil {
		if errors.Is(err, io.EOF) {
			_, err = serverReflectionInfoClient.Recv()
		}

		return nil, errors.Wrap(err, "serverReflectionInfoClient.Send()")
	}

	var serverReflectionResp *reflectionpb.ServerReflectionResponse
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
		if err = serverReflectionInfoClient.Send(&reflectionpb.ServerReflectionRequest{
			MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{
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

					methodName := methodDescriptorProto.GetName()

					methods = append(methods, grpc.MethodDesc{
						MethodName: methodName,
						Handler:    nil,
					})
				}

				type emptyI interface{}
				serviceDescSlice = append(serviceDescSlice, &grpc.ServiceDesc{
					ServiceName: serviceName,
					HandlerType: (*emptyI)(nil),
					Methods:     methods,
					Streams:     nil,
					Metadata:    "api/v1/service.proto", // TODO: Change.
				})
			}
		}
	}

	return serviceDescSlice, nil
}

func GetFileDescriptorSlice(
	ctx context.Context,
	conn *grpc.ClientConn,
) (fileDescriptorSlice []*desc.FileDescriptor, err error) {
	cl := grpcreflect.NewClient(ctx, reflectionpb.NewServerReflectionClient(conn))
	descriptorSource := DescriptorSourceFromServer(cl)
	return grpcurl.GetAllFiles(descriptorSource)
}
