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

// LavaTests
func LavaTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string, s *sentry.Sentry, clientCtx client.Context) error {
	errors := []string{}
	if apiInterface == restString {
		log.Println("starting run important apis")
		clientAdress := clientCtx.FromAddress
		mostImportantApisToTest := map[string][]string{
			http.MethodGet: {
				"/blocks/latest",
				"/lavanet/lava/pairing/providers/LAV1",
				"/lavanet/lava/pairing/clients/LAV1",
				fmt.Sprintf("/lavanet/lava/pairing/get_pairing/LAV1/%s", clientAdress),
				// fmt.Sprintf("/lavanet/lava/pairing/verify_pairing/LAV1/%s/%s/%d", clientAdress, clientAdress, 78), // verify pairing needs more work. as block is changed every iterations
				fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s", clientAdress),
				"/cosmos/gov/v1beta1/proposals",
				"/lavanet/lava/spec/spec",
				"/blocks/1",
			},
			http.MethodPost: {},
		}

		for httpMethod, api := range mostImportantApisToTest {
			for _, api_value := range api {
				for i := 0; i < 100; i++ {
					reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api_value, "", httpMethod, "lava_test")
					if err != nil {
						log.Println(err)
						errors = append(errors, fmt.Sprintf("%s", err))
					} else {
						prettyPrintReply(*reply, "LavaTestsResponse")
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
				if api_interface.Type == postString {
					httpMethod = http.MethodPost
					// for now we dont want to run the post apis in this test
					continue
				}
				log.Println(fmt.Sprintf("%s", apiName))
				reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, apiName, "", httpMethod, "lava_test")
				if err != nil {
					log.Println(err)
					errors = append(errors, fmt.Sprintf("%s", err))
				} else {
					prettyPrintReply(*reply, "LavaTestsResponse")
				}
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
