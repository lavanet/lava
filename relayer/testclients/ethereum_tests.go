package testclients

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func EthTests(ctx context.Context, chainID string, rpcURL string) error {
	log.Println("Starting " + chainID + " Tests")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("client dial error %s", err.Error())
	}
	// eth_blockNumber
	latestBlockNumberUint, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("eth_blockNumber error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_blockNumber")

	// put in a loop for cases that a block have no tx because
	var latestBlock *types.Block
	var latestBlockNumber *big.Int
	var latestBlockTxs types.Transactions
	for {
		// eth_getBlockByNumber
		latestBlockNumber = big.NewInt(int64(latestBlockNumberUint))
		latestBlock, err = client.BlockByNumber(ctx, latestBlockNumber)
		if err != nil {
			return fmt.Errorf("eth_getBlockByNumber error %s", err.Error())
		}
		latestBlockTxs = latestBlock.Transactions()

		if len(latestBlockTxs) == 0 {
			latestBlockNumberUint -= 1
			continue
		}
		break
	}
	log.Println("reply JSONRPC_eth_getBlockByNumber")

	// eth_gasPrice
	_, err = client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("eth_gasPrice error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_gasPrice")

	if chainID != "FTM250" {
		// eth_getBlockByHash
		_, err = client.BlockByHash(ctx, latestBlock.Hash())
		if err != nil {
			return fmt.Errorf("eth_getBlockByHash error %s", err.Error())
		}
		log.Println("reply JSONRPC_eth_getBlockByHash")
	}

	targetTx := latestBlockTxs[0]

	// eth_getTransactionByHash
	targetTx, _, err = client.TransactionByHash(ctx, targetTx.Hash())
	if err != nil {
		return fmt.Errorf("eth_getTransactionByHash error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_getTransactionByHash")

	// eth_getTransactionReceipt
	_, err = client.TransactionReceipt(ctx, targetTx.Hash())
	if err != nil {
		return fmt.Errorf("eth_getTransactionReceipt error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_getTransactionReceipt")

	targetTxMsg, _ := targetTx.AsMessage(types.LatestSignerForChainID(targetTx.ChainId()), nil)

	// eth_getBalance
	_, err = client.BalanceAt(ctx, targetTxMsg.From(), nil)
	if err != nil {
		return fmt.Errorf("eth_getBalance error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_getBalance")

	// eth_getStorageAt
	_, err = client.StorageAt(ctx, *targetTx.To(), common.HexToHash("00"), nil)
	if err != nil {
		return fmt.Errorf("eth_getStorageAt error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_getStorageAt")

	if chainID != "FTM250" {
		// eth_getTransactionCount
		_, err = client.TransactionCount(ctx, latestBlock.Hash())
		if err != nil {
			return fmt.Errorf("eth_getTransactionCount error %s", err.Error())
		}
		log.Println("reply JSONRPC_eth_getTransactionCount")
	}
	// eth_getCode
	_, err = client.CodeAt(ctx, *targetTx.To(), nil)
	if err != nil {
		return fmt.Errorf("eth_getCode error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_getCode")

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
	if err != nil && !strings.Contains(err.Error(), "execution reverted") {
		return fmt.Errorf("eth_call error %s", err.Error())
	}
	log.Println("reply JSONRPC_eth_call")

	if chainID != "GTH1" {
		// eth_estimateGas
		_, err = client.EstimateGas(ctx, callMsg)
		if err != nil && !strings.Contains(err.Error(), "execution reverted") {
			return fmt.Errorf("eth_estimateGas error %s", err.Error())
		}
		log.Println("reply JSONRPC_eth_estimateGas")
	}

	return nil
}
