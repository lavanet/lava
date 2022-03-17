package relayer

import (
	context "context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	grpc "google.golang.org/grpc"
)

const (
	JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE  = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_UNSUPPORTED     = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`
)

var (
	g_clientApis map[string]types.ServiceApi
	g_cu_sum     uint64
)

func sendRelay(
	ctx context.Context,
	clientCtx client.Context,
	c RelayerClient,
	privKey *btcec.PrivateKey,
	specId int,
	sessionId int64,
	req string,
	blockHeight int64,
) (*RelayReply, *secp256k1.PubKey, error) {

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal([]byte(req), &msg)
	if err != nil {
		return nil, nil, err
	}
	serviceApi, err := getSupportedApi(msg.Method, g_clientApis)
	if err != nil {
		return nil, nil, err
	}
	g_cu_sum += serviceApi.ComputeUnits

	//
	//
	relayRequest := &RelayRequest{
		Data:        []byte(req),
		SessionId:   uint64(sessionId),
		SpecId:      uint32(specId),
		CuSum:       g_cu_sum,
		BlockHeight: blockHeight,
	}

	sig, err := signRelay(privKey, []byte(relayRequest.String()))
	if err != nil {
		return nil, nil, err
	}
	relayRequest.Sig = sig

	reply, err := c.Relay(ctx, relayRequest)
	if err != nil {
		return nil, nil, err
	}
	serverKey, err := recoverPubKeyFromRelayReply(reply)
	if err != nil {
		return nil, nil, err
	}

	return reply, &serverKey, nil
}

func TestClient(ctx context.Context, clientCtx client.Context, queryClient types.QueryClient, addr string, specId int) {
	//
	// Get specs
	_, clientApis, err := getSpec(ctx, queryClient, specId)
	if err != nil {
		log.Fatalln("error: getSpec", err)
	}
	log.Println(clientApis, specId)
	g_clientApis = clientApis

	//
	// Start sentry
	sentry := NewSentry(clientCtx.Client)
	err = sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start()
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}

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

	connectCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := NewRelayerClient(conn)
	sessionId := rand.Int63()

	//
	// Call a few times and print results
	for i := 0; i < 10; i++ {
		reply, serverKey, err := sendRelay(ctx, clientCtx, c, privKey, specId, sessionId, JSONRPC_ETH_BLOCKNUMBER, sentry.GetBlockHeight())
		if err != nil {
			log.Println(err)
		} else {
			reply.Sig = nil // for nicer prints
			log.Println("server addr", serverKey.Address(), "reply", reply)
		}
		reply, serverKey, err = sendRelay(ctx, clientCtx, c, privKey, specId, sessionId, JSONRPC_ETH_GETBALANCE, sentry.GetBlockHeight())
		if err != nil {
			log.Println(err)
		} else {
			reply.Sig = nil // for nicer prints
			log.Println("server addr", serverKey.Address(), "reply", reply)
		}
	}

	//
	// Expected unsupported API:
	reply, serverKey, err := sendRelay(ctx, clientCtx, c, privKey, specId, sessionId, JSONRPC_UNSUPPORTED, sentry.GetBlockHeight())
	if err != nil {
		log.Println(err)
	} else {
		reply.Sig = nil // for nicer prints
		log.Println("server addr", serverKey.Address(), "reply", reply)
	}

}
