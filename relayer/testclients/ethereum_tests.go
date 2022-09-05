package testclients

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lavanet/lava/utils"
)

func EthTests(ctx context.Context, chainID string, rpcURL string, testDuration time.Duration) error {
	utils.LavaFormatInfo("Starting "+chainID+" Tests", nil)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return utils.LavaFormatError("error client dial", err, nil)
	}
	for start := time.Now(); time.Since(start) < testDuration; {
		// eth_blockNumber
		latestBlockNumberUint, err := client.BlockNumber(ctx)
		if err != nil {
			return utils.LavaFormatError("error eth_blockNumber", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_blockNumber", &map[string]string{"blockNumber": fmt.Sprintf("%d", latestBlockNumberUint)})

		// put in a loop for cases that a block have no tx because
		var latestBlock *types.Block
		var latestBlockNumber *big.Int
		var latestBlockTxs types.Transactions
		for {
			// eth_getBlockByNumber
			latestBlockNumber = big.NewInt(int64(latestBlockNumberUint))
			latestBlock, err = client.BlockByNumber(ctx, latestBlockNumber)
			if err != nil {
				return utils.LavaFormatError("error eth_getBlockByNumber", err, nil)
			}
			latestBlockTxs = latestBlock.Transactions()

			if len(latestBlockTxs) == 0 {
				latestBlockNumberUint -= 1
				continue
			}
			break
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getBlockByNumber", nil)

		// eth_gasPrice
		_, err = client.SuggestGasPrice(ctx)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_gasPrice", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_gasPrice", nil)

		if chainID != "FTM250" {
			// eth_getBlockByHash
			_, err = client.BlockByHash(ctx, latestBlock.Hash())
			if err != nil && !strings.Contains(err.Error(), "rpc error") {
				return utils.LavaFormatError("error eth_getBlockByHash", err, nil)
			}
			utils.LavaFormatInfo("reply JSONRPC_eth_getBlockByHash", nil)
		}

		targetTx := latestBlockTxs[0]

		// eth_getTransactionByHash
		targetTx, _, err = client.TransactionByHash(ctx, targetTx.Hash())
		if err != nil {
			return utils.LavaFormatError("error eth_getTransactionByHash", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getTransactionByHash", nil)

		// eth_getTransactionReceipt
		_, err = client.TransactionReceipt(ctx, targetTx.Hash())
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_getTransactionReceipt", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getTransactionReceipt", nil)

		targetTxMsg, _ := targetTx.AsMessage(types.LatestSignerForChainID(targetTx.ChainId()), nil)

		// eth_getBalance
		_, err = client.BalanceAt(ctx, targetTxMsg.From(), nil)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_getBalance", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getBalance", nil)

		// eth_getStorageAt
		_, err = client.StorageAt(ctx, *targetTx.To(), common.HexToHash("00"), nil)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_getStorageAt", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getStorageAt", nil)

		if chainID != "FTM250" {
			// eth_getTransactionCount
			_, err = client.TransactionCount(ctx, latestBlock.Hash())
			if err != nil && !strings.Contains(err.Error(), "rpc error") {
				return utils.LavaFormatError("error eth_getTransactionCount", err, nil)
			}
			utils.LavaFormatInfo("reply JSONRPC_eth_getTransactionCount", nil)
		}
		// eth_getCode
		_, err = client.CodeAt(ctx, *targetTx.To(), nil)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_getCode", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_getCode", nil)

		previousBlock := big.NewInt(int64(latestBlockNumberUint - 1))

		callMsg := ethereum.CallMsg{
			From:       targetTxMsg.From(),
			To:         targetTxMsg.To(),
			Gas:        targetTxMsg.Gas(),
			GasPrice:   targetTxMsg.GasPrice(),
			GasFeeCap:  targetTxMsg.GasFeeCap(),
			GasTipCap:  targetTxMsg.GasTipCap(),
			Value:      targetTxMsg.Value(),
			Data:       targetTxMsg.Data(),
			AccessList: targetTxMsg.AccessList(),
		}

		// eth_call
		_, err = client.CallContract(ctx, callMsg, previousBlock)
		if err != nil && !strings.Contains(err.Error(), "rpc error") {
			return utils.LavaFormatError("error eth_call", err, nil)
		}
		utils.LavaFormatInfo("reply JSONRPC_eth_call", nil)

		if chainID != "GTH1" {
			// eth_estimateGas
			_, err = client.EstimateGas(ctx, callMsg)
			if err != nil && !strings.Contains(err.Error(), "rpc error") {
				return utils.LavaFormatError("error eth_estimateGas", err, nil)
			}
			utils.LavaFormatInfo("reply JSONRPC_eth_estimateGas", nil)
		}
	}
	return nil
}
