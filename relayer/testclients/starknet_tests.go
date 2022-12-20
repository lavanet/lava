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
	JSONRPC_STRK_BLOCKNUMBER        = `{"jsonrpc":"2.0","method":"starknet_blockNumber","params":[],"id":1}`
	JSONRPC_STRK_BLOCKHASHANDNUMBER = `{"jsonrpc":"2.0","method":"starknet_blockHashAndNumber","params":[],"id":1}`
)

func StarknetTests(ctx context.Context, chainID string, rpcURL string, chainProxy chainproxy.ChainProxy, privKey *btcec.PrivateKey, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting "+chainID+" Tests", nil)

	for start := time.Now(); time.Since(start) < testDuration; {
		for j := 0; j < 10; j++ {
			reply, _, err := chainproxy.SendRelay(ctx, chainProxy, privKey, rpcURL, JSONRPC_STRK_BLOCKNUMBER, http.MethodGet, "starknet_test")
			if err != nil {
				return utils.LavaFormatError("error starknet_blockNumber", err, nil)
			}
			prettyPrintReply(*reply, "JSONRPC_STRK_BLOCKNUMBER")

			reply, _, err = chainproxy.SendRelay(ctx, chainProxy, privKey, rpcURL, JSONRPC_STRK_BLOCKHASHANDNUMBER, http.MethodGet, "starknet_test")
			if err != nil {
				return utils.LavaFormatError("error starknet_blockHashAndNumber", err, nil)
			}
			prettyPrintReply(*reply, "JSONRPC_STRK_BLOCKHASHANDNUMBER")
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
