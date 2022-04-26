package relayer

import (
	context "context"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
)

const (
	JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE  = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_UNSUPPORTED     = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`

	TERRA_BLOCKS_LATEST_URL_REST  = "/blocks/latest"
	TERRA_BLOCKS_LATEST_DATA_REST = ``
)

func ethTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey) {
	//
	// Call a few times and print results
	for i2 := 0; i2 < 30; i2++ {
		for i := 0; i < 10; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_BLOCKNUMBER)
			if err != nil {
				log.Println(err)
			} else {
				reply.Sig = nil // for nicer prints
				log.Println("reply JSONRPC_ETH_BLOCKNUMBER", reply)
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_GETBALANCE)
			if err != nil {
				log.Println(err)
			} else {
				reply.Sig = nil // for nicer prints
				log.Println("reply JSONRPC_ETH_GETBALANCE", reply)
			}
		}
		time.Sleep(2 * time.Second)
	}

	//
	// Expected unsupported API:
	reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_UNSUPPORTED)
	if err != nil {
		log.Println(err)
	} else {
		reply.Sig = nil // for nicer prints
		log.Println("reply", reply)
	}
}

func terraTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey) {
	for i := 0; i < 10; i++ {
		reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST)
		if err != nil {
			log.Println(err)
		} else {
			reply.Sig = nil // for nicer prints
			log.Println("reply", reply)
		}
	}
}

func TestClient(
	ctx context.Context,
	clientCtx client.Context,
	chainID string,
	apiInterface string,
) {
	//
	// Start sentry
	sentry := sentry.NewSentry(clientCtx, chainID, true, nil, apiInterface)
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
	chainProxy, err := chainproxy.GetChainProxy("", 1, sentry)
	if err != nil {
		log.Fatalln("error: GetChainProxy", err)
	}

	//
	// Set up a connection to the server.
	log.Println("TestClient connecting")

	keyName, err := sigs.GetKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}

	privKey, err := sigs.GetPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Client pubkey", clientKey.GetPubKey().Address())

	//
	// Run tests
	switch chainID {
	case "ETH1", "ETH4":
		ethTests(ctx, chainProxy, privKey)
	case "COS1":
		terraTests(ctx, chainProxy, privKey)
	}
}
