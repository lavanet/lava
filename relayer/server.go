package relayer

import (
	context "context"
	"log"
	"net"

	grpc "google.golang.org/grpc"
)

type relayServer struct {
	UnimplementedRelayerServer
}

func (s *relayServer) Relay(ctx context.Context, in *RelayRequest) (*RelayReply, error) {
	log.Println("server got Relay")

	reply := RelayReply{
		Data: []byte("hello"),
	}
	return &reply, nil
}

func Server(ctx context.Context, listenAddr string) {
	log.Println("Server starting", listenAddr)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	go func() {
		select {
		case <-ctx.Done():
			log.Println("server ctx.Done")
			s.Stop()
		}
	}()

	Server := &relayServer{}
	RegisterRelayerServer(s, Server)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
