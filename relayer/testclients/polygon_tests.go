package testclients

import (
	"context"
	"net/http"
	"time"

	"github.com/lavanet/lava/utils"

	"github.com/btcsuite/btcd/btcec"
	"github.com/lavanet/lava/relayer/chainproxy"
)

const (
	// JSONRPC_ETH_BLOCKNUMBER       (defined in relayer/testclients/test_utils.go)
	// JSONRPC_ETH_GETBALANCE        (defined in relayer/testclients/test_utils.go)
	// JSONRPC_ETH_NEWFILTER         (defined in relayer/testclients/test_utils.go)
)

var polygon_tests = []struct{ name, payload string }{
	{"eth_blockNumber", JSONRPC_ETH_BLOCKNUMBER},
	{"eth_getBalance", JSONRPC_ETH_GETBALANCE},
	{"eth_newFilter", JSONRPC_ETH_NEWFILTER},
}

func PolygonTests(ctx context.Context, chainID string, rpcURL string, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting "+chainID+" Tests", nil)

	for start := time.Now(); time.Since(start) < testDuration; {
		for j := 0; j < 10; j++ {
			for _, t := range polygon_tests {
				reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, rpcURL, t.payload, http.MethodGet, "polygon_test")
				if err != nil {
					return utils.LavaFormatError("error "+t.name, err, nil)
				}
				prettyPrintReply(*reply, "JSONRPC_"+t.name)
			}
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
