package chaintracker

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ChainTrackerServiceServer is the server API for ChainTrackerService.
type ChainTrackerServiceServer interface {
	GetLatestBlockNum(context.Context, *emptypb.Empty) (*GetLatestBlockNumResponse, error)
	GetLatestBlockData(context.Context, *LatestBlockData) (*LatestBlockDataResponse, error)
}

// UnimplementedChainTrackerServiceServer can be embedded for forward compatibility.
type UnimplementedChainTrackerServiceServer struct{}

func (*UnimplementedChainTrackerServiceServer) GetLatestBlockNum(_ context.Context, _ *emptypb.Empty) (*GetLatestBlockNumResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestBlockNum not implemented")
}

func (*UnimplementedChainTrackerServiceServer) GetLatestBlockData(_ context.Context, _ *LatestBlockData) (*LatestBlockDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLatestBlockData not implemented")
}

// RegisterChainTrackerServiceServer registers srv with the gRPC server s.
func RegisterChainTrackerServiceServer(s *grpc.Server, srv ChainTrackerServiceServer) {
	s.RegisterService(&_ChainTrackerService_serviceDesc, srv)
}

func _ChainTrackerService_GetLatestBlockNum_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChainTrackerServiceServer).GetLatestBlockNum(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chainTracker.ChainTrackerService/GetLatestBlockNum",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChainTrackerServiceServer).GetLatestBlockNum(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ChainTrackerService_GetLatestBlockData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LatestBlockData)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChainTrackerServiceServer).GetLatestBlockData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chainTracker.ChainTrackerService/GetLatestBlockData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChainTrackerServiceServer).GetLatestBlockData(ctx, req.(*LatestBlockData))
	}
	return interceptor(ctx, in, info, handler)
}

var _ChainTrackerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "chainTracker.ChainTrackerService",
	HandlerType: (*ChainTrackerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLatestBlockNum",
			Handler:    _ChainTrackerService_GetLatestBlockNum_Handler,
		},
		{
			MethodName: "GetLatestBlockData",
			Handler:    _ChainTrackerService_GetLatestBlockData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "chainTracker/chainTracker.proto",
}
