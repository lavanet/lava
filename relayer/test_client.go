package relayer

import (
	context "context"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
)

const (
	JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE  = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_UNSUPPORTED     = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`
)

func sendRelay(
	ctx context.Context,
	cp chainproxy.ChainProxy,
	privKey *btcec.PrivateKey,
	req string,
) (*servicertypes.RelayReply, error) {

	//
	// Unmarshal request
	nodeMsg, err := cp.ParseMsg([]byte(req))
	if err != nil {
		return nil, err
	}

	//
	//
	reply, err := cp.GetSentry().SendRelay(ctx, func(clientSession *sentry.ClientSession) (*servicertypes.RelayReply, error) {
		clientSession.CuSum += nodeMsg.GetServiceApi().ComputeUnits

		relayRequest := &servicertypes.RelayRequest{
			Servicer:    clientSession.Client.Acc,
			Data:        []byte(req),
			SessionId:   uint64(clientSession.SessionId),
			SpecId:      uint32(cp.GetSentry().SpecId),
			CuSum:       clientSession.CuSum,
			BlockHeight: cp.GetSentry().GetBlockHeight(),
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
			return nil, fmt.Errorf("server address mismatch in reply (%s) (%s)", serverAddr.String(), clientSession.Client.Acc)
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
	sentry := sentry.NewSentry(clientCtx, specId, true, nil)
	err := sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start(ctx)
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}

	//
	// Node
	chainProxy, err := chainproxy.GetChainProxy(specId, "", 1, sentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}

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
	for i2 := 0; i2 < 30; i2++ {
		for i := 0; i < 10; i++ {
			reply, err := sendRelay(ctx, chainProxy, privKey, JSONRPC_ETH_BLOCKNUMBER)
			if err != nil {
				log.Println(err)
			} else {
				reply.Sig = nil // for nicer prints
				log.Println("reply", reply)
			}
			reply, err = sendRelay(ctx, chainProxy, privKey, JSONRPC_ETH_GETBALANCE)
			if err != nil {
				log.Println(err)
			} else {
				reply.Sig = nil // for nicer prints
				log.Println("reply", reply)
			}
		}
		time.Sleep(2 * time.Second)
	}

	//
	// Expected unsupported API:
	reply, err := sendRelay(ctx, chainProxy, privKey, JSONRPC_UNSUPPORTED)
	if err != nil {
		log.Println(err)
	} else {
		reply.Sig = nil // for nicer prints
		log.Println("reply", reply)
	}

}
