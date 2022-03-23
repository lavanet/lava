package relayer

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE  = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_UNSUPPORTED     = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`
)

func sendRelay(
	ctx context.Context,
	sentry *Sentry,
	privKey *btcec.PrivateKey,
	specId uint64,
	req string,
	blockHeight int64,
) (*RelayReply, error) {

	//
	// Unmarshal request
	var msg jsonrpcMessage
	err := json.Unmarshal([]byte(req), &msg)
	if err != nil {
		return nil, err
	}
	serviceApi, err := getSupportedApi(msg.Method, g_sentry)
	if err != nil {
		return nil, err
	}

	//
	//
	reply, err := sentry.SendRelay(ctx, func(clientSession *ClientSession) (*RelayReply, error) {
		clientSession.CuSum += serviceApi.ComputeUnits

		relayRequest := &RelayRequest{
			Data:        []byte(req),
			SessionId:   uint64(clientSession.SessionId),
			SpecId:      uint32(specId),
			CuSum:       clientSession.CuSum,
			BlockHeight: blockHeight,
		}

		sig, err := signRelay(privKey, []byte(relayRequest.String()))
		if err != nil {
			return nil, err
		}
		relayRequest.Sig = sig

		c := *clientSession.Client.Client
		reply, err := c.Relay(ctx, relayRequest)
		if err != nil {
			return nil, err
		}
		serverKey, err := recoverPubKeyFromRelayReply(reply)
		if err != nil {
			return nil, err
		}
		serverAddr, err := sdk.AccAddressFromHex(serverKey.Address().String())
		if err != nil {
			return nil, err
		}
		if serverAddr.String() != clientSession.Client.Acc {
			return nil, errors.New("server address mismatch in reply")
		}

		return reply, nil
	})

	return reply, err
}

func TestClient(
	ctx context.Context,
	clientCtx client.Context,
	specId uint64,
) {
	//
	// Start sentry
	sentry := NewSentry(clientCtx, specId, true)
	err := sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start(ctx)
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}
	g_sentry = sentry
	g_serverSpecId = specId

	//
	// Set up a connection to the server.
	log.Println("TestClient connecting")

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

	//
	// Call a few times and print results
	for i := 0; i < 10; i++ {
		reply, err := sendRelay(ctx, sentry, privKey, specId, JSONRPC_ETH_BLOCKNUMBER, sentry.GetBlockHeight())
		if err != nil {
			log.Println(err)
		} else {
			reply.Sig = nil // for nicer prints
			log.Println("reply", reply)
		}
		reply, err = sendRelay(ctx, sentry, privKey, specId, JSONRPC_ETH_GETBALANCE, sentry.GetBlockHeight())
		if err != nil {
			log.Println(err)
		} else {
			reply.Sig = nil // for nicer prints
			log.Println("reply", reply)
		}
	}

	//
	// Expected unsupported API:
	reply, err := sendRelay(ctx, sentry, privKey, specId, JSONRPC_UNSUPPORTED, sentry.GetBlockHeight())
	if err != nil {
		log.Println(err)
	} else {
		reply.Sig = nil // for nicer prints
		log.Println("reply", reply)
	}

}
