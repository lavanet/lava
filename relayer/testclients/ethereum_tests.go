package testclients

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
)

func EthTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey) error {
	errors := []string{}
	fmt.Println("starting Eth Tests")

	for _, ethApi := range ethApis {
		reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", ethApi.req, "")
		if err != nil {
			if ethApi.name == "eth_unsupported" {
				log.Println("reply JSON_eth_unsupported")
			} else {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			}
		} else {
			if strings.Compare(ethApi.res, string(reply.Data)) == 0 {
				prettyPrintReply(*reply, "JSONRPC_"+ethApi.name)
			} else {
				log.Println("ERROR API CHECK FAILED", ethApi.name, "RESPONSE MISMATCH")
				log.Println(ethApi.req, string(reply.Data), ethApi.res)
				errors = append(errors, fmt.Sprintf("%s RESPONSE MISMATCH %s", ethApi.name, string(reply.Data)))
			}
		}
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}
