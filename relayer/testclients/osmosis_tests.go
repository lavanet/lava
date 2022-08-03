package testclients

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
)

func OsmosisTests(ctx context.Context, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, apiInterface string) error {
	if apiInterface == "rest" {
		// most important api test
		mostImportantApisToTest := map[string][]string{
			http.MethodGet: {
				// cosmos apis
				"/cosmos/bank/v1beta1/balances/osmo1500hy75krs9e8t50aav6fahk8sxhajn9ctp40qwvvn8tcprkk6wszun4a5",
				"/cosmos/gov/v1beta1/proposals",
				"/blocks/latest",
				"/blocks/1",
				// osmosis apis
				"/osmosis/gamm/v1beta1/pools",
				"/osmosis/epochs/v1beta1/epochs",
				"/osmosis/pool-incentives/v1beta1/incentivized_pools",
				fmt.Sprintf("/osmosis/incentives/v1beta1/gauge_by_id/%d", 5411374),
				"/osmosis/superfluid/v1beta1/all_assets",
				"/osmosis/pool-incentives/v1beta1/distr_info",
				"/osmosis/mint/v1beta1/epoch_provisions",
			},
			http.MethodPost: {},
		}

		for httpMethod, api := range mostImportantApisToTest {
			for _, api_value := range api {
				for i := 0; i < 20; i++ {
					reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, api_value, "", httpMethod)
					if err != nil {
						log.Println(err)
						return err
					} else {
						prettyPrintReply(*reply, "OsmosisTestsResponse")
					}
				}
			}
		}

		// other osmosis tests
		for i := 0; i < 100; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, TERRA_BLOCKS_LATEST_URL_REST, TERRA_BLOCKS_LATEST_DATA_REST, http.MethodGet)
			if err != nil {
				log.Println("1:" + err.Error())
				return err
			} else {
				prettyPrintReply(*reply, "TERRA_BLOCKS_LATEST_URL_REST")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, OSMOSIS_NUM_POOLS_URL_REST, OSMOSIS_NUM_POOLS_DATA_REST, http.MethodGet)
			if err != nil {
				log.Println("1:" + err.Error())
				return err
			} else {
				prettyPrintReply(*reply, "OSMOSIS_NUM_POOLS_URL_REST")
			}
		}
	} else if apiInterface == "tendermintrpc" {
		for i := 0; i < 100; i++ {
			reply, err := chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_STATUS, http.MethodGet)
			if err != nil {
				log.Println(err)
				return err
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, "", JSONRPC_TERRA_HEALTH, http.MethodGet)
			if err != nil {
				log.Println(err)
				return err
			} else {
				prettyPrintReply(*reply, "JSONRPC_TERRA_HEALTH")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_STATUS, "", http.MethodGet)
			if err != nil {
				log.Println(err)
				return err
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_STATUS")
			}
			reply, err = chainproxy.SendRelay(ctx, chainProxy, privKey, URIRPC_TERRA_HEALTH, "", http.MethodGet)
			if err != nil {
				log.Println(err)
				return err
			} else {
				prettyPrintReply(*reply, "URIRPC_TERRA_HEALTH")
			}
		}
	} else {
		log.Println("ERROR: not supported apiInterface: ", apiInterface)
		return nil
	}
	return nil
}
