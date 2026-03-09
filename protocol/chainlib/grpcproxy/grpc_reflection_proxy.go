package grpcproxy

import (
	"context"
	"io"

	"github.com/lavanet/lava/v5/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

// ReflectionProxyCallback is a function that creates a reflection client for proxying.
// It should return a grpc.ClientConn that can be used to create a ServerReflectionClient.
// The caller is responsible for returning the connection when done (via the cleanup function).
type ReflectionProxyCallback func(ctx context.Context) (*grpc.ClientConn, func(), error)

// ReflectionProxyService implements the gRPC reflection service by proxying
// requests to an upstream gRPC server. This enables tools like grpcurl to work
// with the smart router in Direct RPC mode.
type ReflectionProxyService struct {
	reflectionpb.UnimplementedServerReflectionServer
	getConnection ReflectionProxyCallback
}

// NewReflectionProxyService creates a new reflection proxy service.
// The callback should return a gRPC connection to the upstream server.
func NewReflectionProxyService(getConnection ReflectionProxyCallback) *ReflectionProxyService {
	return &ReflectionProxyService{
		getConnection: getConnection,
	}
}

// ServerReflectionInfo implements the bidirectional streaming reflection API.
// It proxies requests to the upstream gRPC server and returns responses.
func (s *ReflectionProxyService) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	if s.getConnection == nil {
		return status.Error(codes.Unavailable, "reflection proxy not configured")
	}

	ctx := stream.Context()

	// Get connection to upstream server
	conn, cleanup, err := s.getConnection(ctx)
	if err != nil {
		utils.LavaFormatWarning("reflection proxy: failed to get upstream connection", err)
		return status.Errorf(codes.Unavailable, "failed to connect to upstream: %v", err)
	}
	defer cleanup()

	// Create reflection client for upstream
	upstreamClient := reflectionpb.NewServerReflectionClient(conn)
	upstreamStream, err := upstreamClient.ServerReflectionInfo(ctx)
	if err != nil {
		utils.LavaFormatWarning("reflection proxy: failed to create upstream stream", err)
		return status.Errorf(codes.Unavailable, "failed to create upstream reflection stream: %v", err)
	}

	// Create error channels for bidirectional streaming
	errChan := make(chan error, 2)

	// Forward requests from client to upstream
	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				// Client finished sending - close the send side of upstream
				upstreamStream.CloseSend()
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			if err := upstreamStream.Send(req); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Forward responses from upstream to client
	go func() {
		for {
			resp, err := upstreamStream.Recv()
			if err == io.EOF {
				errChan <- nil // Normal completion
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			if err := stream.Send(resp); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for completion or error
	err = <-errChan
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// RegisterReflectionProxy registers the reflection proxy service on a gRPC server.
// This should be called before the server starts serving.
func RegisterReflectionProxy(server *grpc.Server, getConnection ReflectionProxyCallback) {
	if getConnection == nil {
		utils.LavaFormatWarning("reflection proxy: no connection callback provided, skipping registration", nil)
		return
	}

	service := NewReflectionProxyService(getConnection)
	reflectionpb.RegisterServerReflectionServer(server, service)
	utils.LavaFormatDebug("reflection proxy: registered on gRPC server")
}
