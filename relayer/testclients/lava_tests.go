package testclients

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
)

// LavaTests
func LavaTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string, s *sentry.Sentry, clientCtx client.Context) error {
	if apiInterface == "rest" {
		serverApis := s.ServerApis
		clientAdress := clientCtx.FromAddress
		mostImportantApisToTest := []string{
			"/blocks/latest",
			"/lavanet/lava/pairing/providers/LAV1",
			"/lavanet/lava/pairing/clients/LAV1",
			fmt.Sprintf("/lavanet/lava/pairing/get_pairing/LAV1/%s", clientAdress),
			fmt.Sprintf("/lavanet/lava/pairing/verify_pairing/LAV1/%s/%s/%d", clientAdress, clientAdress, 78),
			fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s", clientAdress),
			"/cosmos/gov/v1beta1/proposals",
			"/lavanet/lava/spec/spec",
			"/blocks/1"}

		for _, api := range mostImportantApisToTest {
			for i := 0; i < 100; i++ {
				reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api, "")
				if err != nil {
					log.Println(err)
					return err
				} else {
					prettyPrintReply(*reply, "LavaTestsResponse")
				}
			}
		}
		// finish with testing all other API methods that dont require parameters
		for _, api := range serverApis {
			if strings.Contains(api.Name, "/{") {
				continue
			}

			for _, api_interface := range api.ApiInterfaces {

			}

			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api.Name, "")
			if err != nil {
				log.Println(err)
				return err
			} else {
				prettyPrintReply(*reply, "LavaTestsResponse")
			}
		}
		return nil
	} else {
		log.Println(fmt.Sprintf("currently no tests for %s protocol", apiInterface))
		return nil
	}
}
