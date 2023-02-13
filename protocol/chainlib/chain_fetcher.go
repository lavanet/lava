package chainlib

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
)

const (
	TendermintStatusQuery = "status"
)

type ChainFetcher struct {
	chainProxy ChainProxy
}

func (cf *ChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (cf *ChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func NewChainFetcher(ctx context.Context, chainProxy ChainProxy /*here needs some more params, maybe chainParser*/) *ChainFetcher {
	// save here the information needed to fetch the latest block and it's hash
	cf := &ChainFetcher{chainProxy: chainProxy}
	return cf
}

type LavaChainFetcher struct {
	clientCtx client.Context
}

func (lcf *LavaChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	resultStatus, err := lcf.clientCtx.Client.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resultStatus.SyncInfo.LatestBlockHeight, nil
}

func (lcf *LavaChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	resultStatus, err := lcf.clientCtx.Client.Status(ctx)
	if err != nil {
		return "", err
	}
	return resultStatus.SyncInfo.LatestBlockHash.String(), nil
}

func NewLavaChainFetcher(ctx context.Context, clientCtx client.Context) *LavaChainFetcher {
	lcf := &LavaChainFetcher{clientCtx: clientCtx}
	return lcf
}
