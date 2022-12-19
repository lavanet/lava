package testclients

import (
	"bytes"
	"log"

	"github.com/lavanet/lava/x/pairing/types"
)

const (
	JSONRPC_ETH_BLOCKNUMBER                     = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	JSONRPC_ETH_GETBALANCE                      = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`
	JSONRPC_ETH_TRACE_REPLAY_BLOCK_TRANSACTIONS = `{"jsonrpc":"2.0","method":"trace_replayBlockTransactions","params":["latest", "trace"],"id":1}`
	JSONRPC_UNSUPPORTED                         = `{"jsonrpc":"2.0","method":"eth_blahblah","params":[],"id":1}`
	JSONRPC_ETH_NEWFILTER                       = `{"jsonrpc":"2.0","method":"eth_newFilter","params":[{"fromBlock": "0x12345","toBlock": "0x23456"}],"id":73}`
	JSONRPC_ETH_GETBLOCK_FORMAT                 = `{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x", false],"id":1}`

	TERRA_BLOCKS_LATEST_URL_REST  = "/blocks/latest"
	TERRA_BLOCKS_LATEST_DATA_REST = ``
	OSMOSIS_NUM_POOLS_URL_REST    = "/osmosis/gamm/v1beta1/num_pools"
	OSMOSIS_NUM_POOLS_DATA_REST   = ``
	JSONRPC_TERRA_STATUS          = `{"jsonrpc":"2.0","method":"status","params":[],"id":1}`
	JSONRPC_TERRA_HEALTH          = `{"jsonrpc":"2.0","method":"health","params":[],"id":2}`
	URIRPC_TERRA_STATUS           = `status?`
	URIRPC_TERRA_HEALTH           = `health`
)

func prettyPrintReply(reply types.RelayReply, name string) {
	reply.Sig = nil // for nicer prints
	reply.SigBlocks = nil
	reply.FinalizedBlocksHashes = nil
	if len(reply.Data) > 200 {
		reply.Data = bytes.Join([][]byte{reply.Data[:200], []byte("...TooLong...")}, nil) // too long is ugly
	}
	log.Printf("reply %s, %s", name, reply.String())
}
