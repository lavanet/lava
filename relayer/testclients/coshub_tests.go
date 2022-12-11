package testclients

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
)

// CosmoshubTests
func CosmoshubTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string, s *sentry.Sentry, clientCtx client.Context) error {
	errors := []string{}
	if apiInterface == "rest" {

		log.Println("starting run important apis")
		clientAdress := clientCtx.FromAddress
		mostImportantApisToTest := map[string][]string{http.MethodGet: {
			"/blocks/latest",
			fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s", clientAdress),
			"/cosmos/gov/v1beta1/proposals",
			"/blocks/latest",
			"/cosmos/bank/v1beta1/balances/osmo1500hy75krs9e8t50aav6fahk8sxhajn9ctp40qwvvn8tcprkk6wszun4a5",
			"/cosmos/gov/v1beta1/proposals"},
			http.MethodPost: {},
		}

		for httpMethod, api := range mostImportantApisToTest {
			for _, api_value := range api {
				for i := 0; i < 100; i++ {
					reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api_value, "", httpMethod, "coshub_test")
					if err != nil {
						log.Println(err)
						errors = append(errors, fmt.Sprintf("%s", err))
					} else {
						prettyPrintReply(*reply, "CosmoshubTestsResonse")
					}
				}
			}
		}

		log.Println("continuing to other spec apis")
		// finish with testing all other API methods that dont require parameters
		allSpecNames, err := s.GetAllSpecNames(ctx)
		if err != nil {
			log.Println(err)
			errors = append(errors, fmt.Sprintf("%s", err))
		}
		for apiName, apiInterfaceList := range allSpecNames {
			if strings.Contains(apiName, "/{") {
				continue
			}

			for _, api_interface := range apiInterfaceList {
				httpMethod := http.MethodGet
				if api_interface.Type == "POST" {
					httpMethod = http.MethodPost
					// for now we dont want to run the post apis in this test
					continue
				}
				log.Println(fmt.Sprintf("%s", apiName))
				reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, apiName, "", httpMethod, "coshub_test")
				if err != nil {
					log.Println(err)
					errors = append(errors, fmt.Sprintf("%s", err))
				} else {
					prettyPrintReply(*reply, "CosmoshubTestsResonse")
				}

			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 100; i++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS, http.MethodGet, "coshub_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH, http.MethodGet, "coshub_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "", http.MethodGet, "coshub_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_STATUS")
			}
			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "", http.MethodGet, "coshub_test")
			if err != nil {
				log.Println(err)
				errors = append(errors, fmt.Sprintf("%s", err))
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println(fmt.Sprintf("currently no tests for %s protocol", apiInterface))
		return nil
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}
