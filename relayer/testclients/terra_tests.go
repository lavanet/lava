package testclients

import (
	"context"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
)

func TerraTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) {
	if apiInterface == "rest" {
		for i := 0; i < 10; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST, http.MethodGet)
			if err != nil {
				log.Println("1:" + err.Error())
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 10; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS, http.MethodGet)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH, http.MethodGet)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "", http.MethodGet)
			if err != nil {
				log.Println(err)
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
				log.Println("reply URIRPC_TERRA_STATUS", reply)
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "", http.MethodGet)
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
