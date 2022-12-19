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

const (
	restString       string = "rest"
	tendermintString string = "tendermintrpc"
)

// AptosTests
func AptosTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string, s *sentry.Sentry, clientCtx client.Context) error {
	errors := []string{}
	log.Println("Aptos test")
	if apiInterface == restString {
		log.Println("starting run important apis")
		// clientAdress := clientCtx.FromAddress
		version := "2406784"
		account := "0x10f9091d233fd38b9d774bc641ed71ea7d3a21a0254fdfa9556901e9fad6a533"

		mostImportantApisToTest := map[string][]string{
			http.MethodGet: {
				"/blocks/by_height/5",
				"/blocks/by_version/" + version,
				"/accounts/" + account,
				"/accounts/" + account + "/resources",
				"/accounts/" + account + "/modules",
			},
			http.MethodPost: {},
		}

		for httpMethod, api := range mostImportantApisToTest {
			for _, api_value := range api {
				for i := 0; i < 100; i++ {
					reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api_value, "", httpMethod, "aptos_test")
					if err != nil {
						log.Println(err)
						errors = append(errors, fmt.Sprintf("%s", err))
					} else {
						log.Printf("LavaTestsResponse: %v\n", reply)
						// prettyPrintReply(*reply, "LavaTestsResponse")
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
				if api_interface.Type == http.MethodPost {
					// for now we dont want to run the post apis in this test
					continue
				}
				log.Printf("%s", apiName)
				reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, apiName, "", http.MethodGet, "aptos_test")
				if err != nil {
					log.Println(err)
					errors = append(errors, fmt.Sprintf("%s", err))
				} else {
					prettyPrintReply(*reply, "LavaTestsResponse")
				}
			}
		}
	} else {
		log.Printf("currently no tests for %s protocol", apiInterface)
		return nil
	}

	// if we had any errors we return them here
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, ",\n"))
	}

	return nil
}
