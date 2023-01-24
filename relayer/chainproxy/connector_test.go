package chainproxy

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	grpcListener = "localhost:48353"
)

func createGRPCServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", grpcListener)
	require.Nil(t, err)
	s := grpc.NewServer()
	go s.Serve(lis) // serve in a different thread
	return s
}

func TestHappyFlow(t *testing.T) {
	s := createGRPCServer(t) // create a grpcServer so we can connect to its endpoint and validate everything works.
	defer s.Stop()           // stop the server when finished.
	// ctx := context.Background()

}
