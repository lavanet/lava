package relayer

import (
	context "context"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"

	grpc "google.golang.org/grpc"
)

const JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
const JSONRPC_ETH_GETBALANCE = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`

func sendRelay(ctx context.Context, clientCtx client.Context, c RelayerClient, req string, privKey *btcec.PrivateKey) {
	//
	//
	relayRequest := &RelayRequest{
		Data: []byte(req),
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

	log.Println("server addr", serverKey.Address(), "reply", reply)
}

func TestClient(ctx context.Context, addr string, clientCtx client.Context) {
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

	sendRelay(ctx, clientCtx, c, JSONRPC_ETH_BLOCKNUMBER, privKey)
	sendRelay(ctx, clientCtx, c, JSONRPC_ETH_GETBALANCE, privKey)
}
