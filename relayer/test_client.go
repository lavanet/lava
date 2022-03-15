package relayer

import (
	context "context"
	"log"
	"time"

	grpc "google.golang.org/grpc"
)

func TestClient(ctx context.Context, addr string) {
	//
	// Set up a connection to the server.
	log.Println("TestClient connecting to", addr)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := NewRelayerClient(conn)
	reply, err := c.Relay(ctx, &RelayRequest{})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(reply)
}
