package types

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ---------------------------------------------------------------------------
// Relayer service
// ---------------------------------------------------------------------------

const Relayer_ServiceName = "lavanet.lava.pairing.Relayer"

// RelayerClient is the client API for the Relayer service.
type RelayerClient interface {
	Relay(ctx context.Context, in *RelayRequest, opts ...grpc.CallOption) (*RelayReply, error)
	RelaySubscribe(ctx context.Context, in *RelayRequest, opts ...grpc.CallOption) (Relayer_RelaySubscribeClient, error)
	Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeReply, error)
}

// RelayerServer is the server API for the Relayer service.
type RelayerServer interface {
	Relay(context.Context, *RelayRequest) (*RelayReply, error)
	RelaySubscribe(*RelayRequest, Relayer_RelaySubscribeServer) error
	Probe(context.Context, *ProbeRequest) (*ProbeReply, error)
}

// UnimplementedRelayerServer must be embedded in any RelayerServer
// implementation to satisfy the interface without implementing all methods.
type UnimplementedRelayerServer struct{}

func (UnimplementedRelayerServer) Relay(_ context.Context, _ *RelayRequest) (*RelayReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Relay not implemented")
}

func (UnimplementedRelayerServer) RelaySubscribe(_ *RelayRequest, _ Relayer_RelaySubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method RelaySubscribe not implemented")
}

func (UnimplementedRelayerServer) Probe(_ context.Context, _ *ProbeRequest) (*ProbeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Probe not implemented")
}

// Relayer_RelaySubscribeClient is the client-side streaming interface for RelaySubscribe.
type Relayer_RelaySubscribeClient interface {
	Recv() (*RelayReply, error)
	grpc.ClientStream
}

// Relayer_RelaySubscribeServer is the server-side streaming interface for RelaySubscribe.
type Relayer_RelaySubscribeServer interface {
	Send(*RelayReply) error
	grpc.ServerStream
}

// relayerClient wraps a grpc.ClientConnInterface for the Relayer service.
type relayerClient struct {
	cc grpc.ClientConnInterface
}

// NewRelayerClient creates a new RelayerClient connected via cc.
func NewRelayerClient(cc grpc.ClientConnInterface) RelayerClient {
	return &relayerClient{cc}
}

func (c *relayerClient) Relay(ctx context.Context, in *RelayRequest, opts ...grpc.CallOption) (*RelayReply, error) {
	out := new(RelayReply)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.Relayer/Relay", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *relayerClient) RelaySubscribe(ctx context.Context, in *RelayRequest, opts ...grpc.CallOption) (Relayer_RelaySubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Relayer_serviceDesc.Streams[0], "/lavanet.lava.pairing.Relayer/RelaySubscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &relayerRelaySubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type relayerRelaySubscribeClient struct {
	grpc.ClientStream
}

func (x *relayerRelaySubscribeClient) Recv() (*RelayReply, error) {
	m := new(RelayReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *relayerClient) Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeReply, error) {
	out := new(ProbeReply)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.Relayer/Probe", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RegisterRelayerServer registers srv with the gRPC server s.
func RegisterRelayerServer(s *grpc.Server, srv RelayerServer) {
	s.RegisterService(&_Relayer_serviceDesc, srv)
}

func _Relayer_Relay_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RelayerServer).Relay(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.Relayer/Relay",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RelayerServer).Relay(ctx, req.(*RelayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Relayer_Probe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RelayerServer).Probe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.Relayer/Probe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RelayerServer).Probe(ctx, req.(*ProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Relayer_RelaySubscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(RelayRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RelayerServer).RelaySubscribe(m, &relayerRelaySubscribeServer{stream})
}

type relayerRelaySubscribeServer struct {
	grpc.ServerStream
}

func (x *relayerRelaySubscribeServer) Send(m *RelayReply) error {
	return x.ServerStream.SendMsg(m)
}

var _Relayer_serviceDesc = grpc.ServiceDesc{
	ServiceName: "lavanet.lava.pairing.Relayer",
	HandlerType: (*RelayerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Relay",
			Handler:    _Relayer_Relay_Handler,
		},
		{
			MethodName: "Probe",
			Handler:    _Relayer_Probe_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RelaySubscribe",
			Handler:       _Relayer_RelaySubscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "relay.proto",
}

// ---------------------------------------------------------------------------
// RelayerCache service
// ---------------------------------------------------------------------------

const RelayerCache_ServiceName = "lavanet.lava.pairing.RelayerCache"

// RelayerCacheClient is the client API for the RelayerCache service.
type RelayerCacheClient interface {
	GetRelay(ctx context.Context, in *RelayCacheGet, opts ...grpc.CallOption) (*CacheRelayReply, error)
	SetRelay(ctx context.Context, in *RelayCacheSet, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CacheUsage, error)
}

// RelayerCacheServer is the server API for the RelayerCache service.
type RelayerCacheServer interface {
	GetRelay(context.Context, *RelayCacheGet) (*CacheRelayReply, error)
	SetRelay(context.Context, *RelayCacheSet) (*emptypb.Empty, error)
	Health(context.Context, *emptypb.Empty) (*CacheUsage, error)
}

// UnimplementedRelayerCacheServer must be embedded to satisfy RelayerCacheServer
// without implementing every method.
type UnimplementedRelayerCacheServer struct{}

func (UnimplementedRelayerCacheServer) GetRelay(_ context.Context, _ *RelayCacheGet) (*CacheRelayReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRelay not implemented")
}

func (UnimplementedRelayerCacheServer) SetRelay(_ context.Context, _ *RelayCacheSet) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetRelay not implemented")
}

func (UnimplementedRelayerCacheServer) Health(_ context.Context, _ *emptypb.Empty) (*CacheUsage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}

// relayerCacheClient wraps a grpc.ClientConnInterface for the RelayerCache service.
type relayerCacheClient struct {
	cc grpc.ClientConnInterface
}

// NewRelayerCacheClient creates a new RelayerCacheClient connected via cc.
func NewRelayerCacheClient(cc grpc.ClientConnInterface) RelayerCacheClient {
	return &relayerCacheClient{cc}
}

func (c *relayerCacheClient) GetRelay(ctx context.Context, in *RelayCacheGet, opts ...grpc.CallOption) (*CacheRelayReply, error) {
	out := new(CacheRelayReply)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.RelayerCache/GetRelay", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *relayerCacheClient) SetRelay(ctx context.Context, in *RelayCacheSet, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.RelayerCache/SetRelay", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *relayerCacheClient) Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CacheUsage, error) {
	out := new(CacheUsage)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.RelayerCache/Health", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RegisterRelayerCacheServer registers srv with the gRPC server s.
func RegisterRelayerCacheServer(s *grpc.Server, srv RelayerCacheServer) {
	s.RegisterService(&_RelayerCache_serviceDesc, srv)
}

func _RelayerCache_GetRelay_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelayCacheGet)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RelayerCacheServer).GetRelay(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.RelayerCache/GetRelay",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RelayerCacheServer).GetRelay(ctx, req.(*RelayCacheGet))
	}
	return interceptor(ctx, in, info, handler)
}

func _RelayerCache_SetRelay_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RelayCacheSet)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RelayerCacheServer).SetRelay(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.RelayerCache/SetRelay",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RelayerCacheServer).SetRelay(ctx, req.(*RelayCacheSet))
	}
	return interceptor(ctx, in, info, handler)
}

func _RelayerCache_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RelayerCacheServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.RelayerCache/Health",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RelayerCacheServer).Health(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _RelayerCache_serviceDesc = grpc.ServiceDesc{
	ServiceName: "lavanet.lava.pairing.RelayerCache",
	HandlerType: (*RelayerCacheServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRelay",
			Handler:    _RelayerCache_GetRelay_Handler,
		},
		{
			MethodName: "SetRelay",
			Handler:    _RelayerCache_SetRelay_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _RelayerCache_Health_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "relayercache.proto",
}
