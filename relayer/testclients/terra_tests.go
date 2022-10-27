package testclients

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
)

func TerraTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) error {
	errors := []string{}
	if apiInterface == "rest" {
		for i := 0; i < 10; i++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST, http.MethodGet)
			if err != nil {
				log.Println("1:" + err.Error())
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 10; i++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS, http.MethodGet)
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH, http.MethodGet)
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "", http.MethodGet)
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
				log.Println("reply URIRPC_TERRA_STATUS", reply)
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "", http.MethodGet)
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println("ERROR: not supported apiInterface: ", apiInterface)
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil

}
