package relayer

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type Sentry struct {
	rpcClient   rpcclient.Client
	txs         <-chan ctypes.ResultEvent
	mu          sync.RWMutex
	blockHeight int64
}

func (s *Sentry) Init(ctx context.Context) error {
	err := s.rpcClient.Start()
	if err != nil {
		return err
	}

	query := "tm.event = 'NewBlock'"
	txs, err := s.rpcClient.Subscribe(ctx, "test-client", query)
	if err != nil {
		return err
	}
	s.txs = txs
	return nil
}

func (s *Sentry) Start() {
	for e := range s.txs {
		switch data := e.Data.(type) {
		case types.EventDataNewBlock:
			s.SetBlockHeight(data.Block.Height)
			fmt.Printf("Block %s - Height: %d \n", hex.EncodeToString(data.Block.Hash()), data.Block.Height)
		}
	}
}

func (s *Sentry) GetBlockHeight() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.blockHeight
}

func (s *Sentry) SetBlockHeight(blockHeight int64) {
	s.mu.Lock()
	s.blockHeight = blockHeight
	s.mu.Unlock()
}

func NewSentry(rpcClient rpcclient.Client) *Sentry {
	return &Sentry{rpcClient: rpcClient}
}
