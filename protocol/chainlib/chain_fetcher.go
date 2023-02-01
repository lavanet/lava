package chainlib

import (
	"context"
	"fmt"
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
