package relayer

import (
	"bytes"
	context "context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/pflag"
)

const (
	JSONRPC_ETH_BLOCKNUMBER                     = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE                      = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_ETH_TRACE_REPLAY_BLOCK_TRANSACTIONS = `{"jsonrpc":"2.0","method":"trace_replayBlockTransactions","params":["latest", "trace"],"id":1}`
	JSONRPC_UNSUPPORTED                         = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`
	JSONRPC_ETH_NEWFILTER                       = `{"jsonrpc":"2.0","method":"eth_newFilter","params":[{"fromBlock": "0x12345","toBlock": "0x23456"}],"id":73}`
	JSONRPC_ETH_GETBLOCK_FORMAT                 = `{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x", false],"id":1}`

	TERRA_BLOCKS_LATEST_URL_REST  = "/blocks/latest"
	TERRA_BLOCKS_LATEST_DATA_REST = ``
	OSMOSIS_NUM_POOLS_URL_REST    = "/osmosis/gamm/v1beta1/num_pools"
	OSMOSIS_NUM_POOLS_DATA_REST   = ``
	JSONRPC_TERRA_STATUS          = `{"jsonrpc":"2.0","method":"status","params":[],"id":1}`
	JSONRPC_TERRA_HEALTH          = `{"jsonrpc":"2.0","method":"health","params":[],"id":2}`
	URIRPC_TERRA_STATUS           = `status?`
	URIRPC_TERRA_HEALTH           = `health`
)

func prettyPrintReply(reply types.RelayReply, name string) {
	reply.Sig = nil // for nicer prints
	reply.SigBlocks = nil
	reply.FinalizedBlocksHashes = nil
	if len(reply.Data) > 200 {
		reply.Data = bytes.Join([][]byte{reply.Data[:200], []byte("...TooLong...")}, nil) //too long is ugly
	}
	log.Printf("reply %s, %s", name, reply.String())
}

func ethTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey) {
	fmt.Println("starting Eth Tests")
	//
	// Call a few times and print results
	for i2 := 0; i2 < 30; i2++ {
		var blockNumReply *types.RelayReply
		blockNumReply = nil
		for i := 0; i < 10; i++ {

			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_BLOCKNUMBER)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_BLOCKNUMBER")
				blockNumReply = reply
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_GETBALANCE)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_GETBALANCE")
			}
		}
		time.Sleep(1 * time.Second)
		//reliability testing

		if blockNumReply == nil {
			continue
		}

		var msg chainproxy.JsonrpcMessage
		err := json.Unmarshal(blockNumReply.GetData(), &msg)
		if err != nil {
			log.Println("Unmarshal error: " + err.Error())
		}
		latestBlockstr, err := strconv.Unquote(string(msg.Result))
		if err != nil {
			log.Println("unquote Unmarshal error: " + err.Error())
		}
		latestBlock, err := strconv.ParseInt(latestBlockstr, 0, 64)
		if err != nil {
			log.Println("blockNum Unmarshal error: " + err.Error())
		}
		for i := 0; i < 10; i++ {
			request_data := fmt.Sprintf(JSONRPC_ETH_GETBLOCK_FORMAT, latestBlock-7)
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", request_data)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_GETBLOCKBYNUMBER")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_NEWFILTER)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_NEWFILTER")
			}
		}
		time.Sleep(1 * time.Second)
	}

	//
	// Expected unsupported API:
	reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_UNSUPPORTED)
	if err != nil {
		log.Println(err)
	} else {
		prettyPrintReply(*reply, "unsupported")
	}
}

func terraTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) {
	if apiInterface == "rest" {
		for i := 0; i < 10; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST)
			if err != nil {
				log.Println("1:" + err.Error())
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 10; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "")
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
				log.Println("reply URIRPC_TERRA_STATUS", reply)
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "")
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println("ERROR: not supported apiInterface: ", apiInterface)
	}

}

func osmosisTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) {
	if apiInterface == "rest" {
		for i := 0; i < 100; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST)
			if err != nil {
				log.Println("1:" + err.Error())
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, OSMOSIS_NUM_POOLS_URL_REST, OSMOSIS_NUM_POOLS_DATA_REST)
			if err != nil {
				log.Println("1:" + err.Error())
			} else {
				prettyPrintReply(*reply, "OSMOSIS_NUM_POOLS_URL_REST")
			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 100; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "")
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "")
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println("ERROR: not supported apiInterface: ", apiInterface)
	}

}

func TestClient(
	ctx context.Context,
	clientCtx client.Context,
	chainID string,
	apiInterface string,
	flagSet *pflag.FlagSet,
) {
	// Every client must preseed
	rand.Seed(time.Now().UnixNano())

	//
	sk, _, err := utils.GetOrCreateVRFKey(clientCtx)
	if err != nil {
		log.Fatalln("error: GetOrCreateVRFKey", err)
	}
	// Start sentry
	sentry := sentry.NewSentry(clientCtx, chainID, true, nil, apiInterface, sk, flagSet, 0)
	err = sentry.Init(ctx)
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
		terraTests(ctx, chainProxy, privKey, apiInterface)
	case "COS3":
		osmosisTests(ctx, chainProxy, privKey, apiInterface)
	}

	log.Printf("%s Client test  complete \n", chainID)
}
