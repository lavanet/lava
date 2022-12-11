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

func JunoTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) error {
	errors := []string{}
	if apiInterface == "rest" {
		// most important api test
		mostImportantApisToTest := map[string][]string{
			http.MethodGet: {
				// cosmos apis
				"/cosmos/gov/v1beta1/proposals",
				"/blocks/latest",
				"/blocks/1",
				// juno specific apis
				"/cosmwasm/wasm/v1/code",
				"/cosmwasm/wasm/v1/codes/pinned",
			},
			http.MethodPost: {},
		}

		for httpMethod, api := range mostImportantApisToTest {
			for _, api_value := range api {
				for i := 0; i < 20; i++ {
					reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api_value, "", httpMethod, "juno_test")
					if err != nil {
						log.Println(err)
						errors = append(errors, fmt.Sprintf("%s", err))
					} else {
						prettyPrintReply(*reply, "JunoTestsResponse")
					}
				}
			}
		}

		// other juno tests
		for i := 0; i < 100; i++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST, http.MethodGet, "juno_test")
			if err != nil {
				log.Println("1:" + err.Error())
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, OSMOSIS_NUM_POOLS_DATA_REST, http.MethodGet, "juno_test")
			if err != nil {
				log.Println("1:" + err.Error())
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_STATUS")
			}
		}

	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 100; i++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS, http.MethodGet, "juno_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH, http.MethodGet, "juno_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "", http.MethodGet, "juno_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_STATUS")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "", http.MethodGet, "juno_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println("ERROR: not supported apiInterface: ", apiInterface)
		return nil
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}
