package relayer

import (
	context "context"
	"log"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/x/spec/types"

	grpc "google.golang.org/grpc"
)

const JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
const JSONRPC_ETH_GETBALANCE = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`

func sendRelay(ctx context.Context, clientCtx client.Context, c RelayerClient, privKey *btcec.PrivateKey, sessionId int64, req string) {
	//
	//
	relayRequest := &RelayRequest{
		Data:      []byte(req),
		SessionId: uint64(sessionId),
	}

	sig, err := signRelay(privKey, []byte(relayRequest.String()))
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	relayRequest.Sig = sig

	reply, err := c.Relay(ctx, relayRequest)
	if err != nil {
		log.Println("error: c.Relay", err)
		return
	}
	serverKey, err := recoverPubKeyFromRelayReply(reply)
	if err != nil {
		log.Println("error: recoverPubKeyFromRelayReply", err)
		return
	}

	reply.Sig = nil // for nicer prints
	log.Println("server addr", serverKey.Address(), "reply", reply)
}

func TestClient(ctx context.Context, clientCtx client.Context, queryClient types.QueryClient, addr string) {
	//
	// Set up a connection to the server.
	log.Println("TestClient connecting to", addr)

	keyName, err := getKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}

	privKey, err := getPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Client pubkey", clientKey.GetPubKey().Address())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := NewRelayerClient(conn)

	sessionId := rand.Int63()
	for i := 0; i < 10; i++ {
		sendRelay(ctx, clientCtx, c, privKey, sessionId, JSONRPC_ETH_BLOCKNUMBER)
		sendRelay(ctx, clientCtx, c, privKey, sessionId, JSONRPC_ETH_GETBALANCE)
	}

}
