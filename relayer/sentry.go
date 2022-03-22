package relayer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
)

type Sentry struct {
	rpcClient           rpcclient.Client
	specQueryClient     spectypes.QueryClient
	servicerQueryClient servicertypes.QueryClient
	specId              uint64
	txs                 <-chan ctypes.ResultEvent

	blockHeight int64

	specMu     sync.RWMutex
	specHash   []byte
	serverSpec spectypes.Spec
	serverApis map[string]spectypes.ServiceApi
}

func (s *Sentry) getSpec(ctx context.Context) error {
	//
	// TODO: decide if it's fatal to not have spec (probably!)
	spec, err := s.specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
		Id: s.specId,
	})
	if err != nil {
		return err
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(spec.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.specHash, hash) {
		return nil
	}
	s.specHash = hash

	//
	// Update
	log.Println("new spec found; updating spec!")
	serverApis := map[string]spectypes.ServiceApi{}
	for _, api := range spec.Spec.Apis {
		serverApis[api.Name] = api
	}
	s.specMu.Lock()
	defer s.specMu.Unlock()
	s.serverSpec = spec.Spec
	s.serverApis = serverApis

	return nil
}

func (s *Sentry) Init(ctx context.Context) error {
	//
	// New client
	err := s.rpcClient.Start()
	if err != nil {
		return err
	}

	//
	// Listen to new block events
	query := "tm.event = 'NewBlock'"
	txs, err := s.rpcClient.Subscribe(ctx, "test-client", query)
	if err != nil {
		return err
	}
	s.txs = txs

	//
	// Get spec for the first time
	err = s.getSpec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Sentry) Start(ctx context.Context) {
	for e := range s.txs {
		switch data := e.Data.(type) {
		case tenderminttypes.EventDataNewBlock:

			//
			// Update block
			s.SetBlockHeight(data.Block.Height)
			fmt.Printf("Block %s - Height: %d \n", hex.EncodeToString(data.Block.Hash()), data.Block.Height)

			//
			// Update specs
			err := s.getSpec(ctx)
			if err != nil {
				log.Println("error: getSpec", err)
			}
		}
	}
}

func (s *Sentry) GetSpecName() string {
	return s.serverSpec.Name
}

func (s *Sentry) GetSpecApiByName(name string) (spectypes.ServiceApi, bool) {
	s.specMu.RLock()
	defer s.specMu.RUnlock()

	val, ok := s.serverApis[name]
	return val, ok
}

func (s *Sentry) GetBlockHeight() int64 {
	return atomic.LoadInt64(&s.blockHeight)
}

func (s *Sentry) SetBlockHeight(blockHeight int64) {
	atomic.StoreInt64(&s.blockHeight, blockHeight)
}

func NewSentry(
	rpcClient rpcclient.Client,
	specQueryClient spectypes.QueryClient,
	servicerQueryClient servicertypes.QueryClient,
	specId uint64,
) *Sentry {
	return &Sentry{
		rpcClient:           rpcClient,
		specQueryClient:     specQueryClient,
		servicerQueryClient: servicerQueryClient,
		specId:              specId,
	}
}
