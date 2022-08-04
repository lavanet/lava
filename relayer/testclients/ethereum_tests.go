package testclients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/x/pairing/types"
)

func EthTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey) error {
	errors := []string{}
	fmt.Println("starting Eth Tests")
	//
	// Call a few times and print results
	for i2 := 0; i2 < 30; i2++ {
		var blockNumReply *types.RelayReply
		blockNumReply = nil
		for i := 0; i < 10; i++ {

			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_BLOCKNUMBER, "")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_BLOCKNUMBER")
				blockNumReply = reply
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_GETBALANCE, "")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
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
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		latestBlockstr, err := strconv.Unquote(string(msg.Result))
		if err != nil {
			log.Println("unquote Unmarshal error: " + err.Error())
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		latestBlock, err := strconv.ParseInt(latestBlockstr, 0, 64)
		if err != nil {
			log.Println("blockNum Unmarshal error: " + err.Error())
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		for i := 0; i < 10; i++ {
			request_data := fmt.Sprintf(JSONRPC_ETH_GETBLOCK_FORMAT, latestBlock-7)
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", request_data, "")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_GETBLOCKBYNUMBER")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_ETH_NEWFILTER, "")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_ETH_NEWFILTER")
			}
		}
		time.Sleep(1 * time.Second)
	}

	//
	// Expected unsupported API:
	reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_UNSUPPORTED, "")
	if err != nil {
		log.Println(err)
		errors = append(errors, fmt.Sprintf("%s", err))
	} else {
		prettyPrintReply(*reply, "unsupported")
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}
