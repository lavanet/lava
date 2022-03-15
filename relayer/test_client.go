package relayer

import (
	context "context"
	"log"
	"time"

	grpc "google.golang.org/grpc"
)

const JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
const JSONRPC_ETH_GETBALANCE = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`

func TestClient(ctx context.Context, addr string) {
	//
	// Set up a connection to the server.
	log.Println("TestClient connecting to", addr)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := NewRelayerClient(conn)

	//
	reply, err := c.Relay(ctx, &RelayRequest{
		Data: []byte(JSONRPC_ETH_BLOCKNUMBER),
	})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(reply)

	//
	reply, err = c.Relay(ctx, &RelayRequest{
		Data: []byte(JSONRPC_ETH_GETBALANCE),
	})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(reply)
}
