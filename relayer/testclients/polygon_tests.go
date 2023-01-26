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
	JSONRPC_BOR_GETAUTHOR            = `{"jsonrpc":"2.0","method":"bor_getAuthor","params":["0x1000"],"id":1}`
	JSONRPC_BOR_GETCURRENTVALIDATORS = `{"jsonrpc":"2.0","method":"bor_getCurrentValidators","id":1}`
	JSONRPC_BOR_GETSIGNERSATHASH     = `{"jsonrpc":"2.0","method":"bor_getSignersAtHash","params":["0x29fa73e3da83ddac98f527254fe37002e052725a88904bac14f03e919e1e2876"],"id":1}`
	JSONRPC_BOR_GETROOTHASH          = `{"jsonrpc":"2.0","method":"bor_getRootHash","params":[1024,1026],"id":1}`
	// [NOT SUPPORTED] JSONRPC_BOR_GETCURRENTPROPOSER   = `{"jsonrpc":"2.0","method":"bor_getCurrentProposer","id":1}`
	// [NOT SUPPORTED] JSONRPC_ETH_GETROOTHASH          = `{"jsonrpc":"2.0","method":"eth_getRootHash","params":[1024,1026],"id":1}`
)

var polygon_tests = []struct{ name, payload string }{
	{"eth_blockNumber", JSONRPC_ETH_BLOCKNUMBER},
	{"eth_getBalance", JSONRPC_ETH_GETBALANCE},
	{"eth_newFilter", JSONRPC_ETH_NEWFILTER},
	{"bor_getAuthor", JSONRPC_BOR_GETAUTHOR},
	{"bor_getCurrentValidators", JSONRPC_BOR_GETCURRENTVALIDATORS},
	{"bor_getSignersAtHash", JSONRPC_BOR_GETSIGNERSATHASH},
	{"bor_getRootHash", JSONRPC_BOR_GETROOTHASH},
	// [NOT SUPPORTED] { "bor_getCurrentProposer", JSONRPC_BOR_GETCURRENTPROPOSER },
	// [NOT SUPPORTED] { "eth_getRootHash", JSONRPC_ETH_GETROOTHASH },
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
