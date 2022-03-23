package relayer

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/client"
	servicertypes "github.com/lavanet/lava/x/servicer/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tenderminttypes "github.com/tendermint/tendermint/types"
	grpc "google.golang.org/grpc"
)

type ClientSession struct {
	CuSum     uint64
	SessionId int64
	Client    *RelayerClientWrapper
	Lock      sync.Mutex
}

type RelayerClientWrapper struct {
	Client *RelayerClient
	Acc    string
	Addr   string

	SessionsLock sync.Mutex
	Sessions     map[int64]*ClientSession
}

type Sentry struct {
	rpcClient           rpcclient.Client
	specQueryClient     spectypes.QueryClient
	servicerQueryClient servicertypes.QueryClient
	specId              uint64
	txs                 <-chan ctypes.ResultEvent
	isUser              bool
	acc                 string // account address (bech32)

	//
	// Block storage (atomic)
	blockHeight int64

	//
	// Spec storage (rw mutex)
	specMu     sync.RWMutex
	specHash   []byte
	serverSpec spectypes.Spec
	serverApis map[string]spectypes.ServiceApi

	// (client only)
	// Pairing storage (rw mutex)
	pairingMu   sync.RWMutex
	pairingHash []byte
	pairing     []*RelayerClientWrapper
}

func (s *Sentry) getPairing(ctx context.Context) error {
	//
	// sentry for server module does not need a pairing
	if !s.isUser {
		return nil
	}

	//
	// Get
	res, err := s.servicerQueryClient.GetPairing(ctx, &servicertypes.QueryGetPairingRequest{
		SpecName: s.GetSpecName(),
		UserAddr: s.acc,
	})
	if err != nil {
		return err
	}
	servicers := res.GetServicers()
	if servicers == nil || len(servicers.Staked) == 0 {
		return errors.New("no servicers found")
	}

	//
	// Check if updated
	hash := tendermintcrypto.Sha256([]byte(res.String())) // TODO: we use cheaper algo for speed
	if bytes.Equal(s.pairingHash, hash) {
		return nil
	}
	s.pairingHash = hash

	//
	// Set
	pairing := []*RelayerClientWrapper{}
	for _, servicer := range servicers.Staked {
		//
		// Sanity
		servicerAddrs := servicer.GetOperatorAddresses()
		if servicerAddrs == nil || len(servicerAddrs) == 0 {
			log.Println("servicerAddrs == nil || len(servicerAddrs) == 0")
			continue
		}

		//
		// TODO: decide how to use multiple addresses from the same operator
		pairing = append(pairing, &RelayerClientWrapper{
			Acc:      servicer.Index,
			Addr:     servicerAddrs[0],
			Sessions: map[int64]*ClientSession{},
		})
	}
	s.pairingMu.Lock()
	defer s.pairingMu.Unlock()
	s.pairing = pairing
	log.Println("update pairing list!", pairing)

	return nil
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

	//
	// Get pairing for the first time
	err = s.getPairing(ctx)
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

			//
			// Update pairing
			err = s.getPairing(ctx)
			if err != nil {
				log.Println("error: getPairing", err)
			}
		}
	}
}

func (s *Sentry) connectRawClient(ctx context.Context, addr string) (*RelayerClient, error) {

	/*connectCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()*/
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := NewRelayerClient(conn)
	return &c, nil
}

func (s *Sentry) _findPairing(ctx context.Context) (*RelayerClientWrapper, error) {
	s.pairingMu.RLock()
	defer s.pairingMu.RUnlock()

	if len(s.pairing) == 0 {
		return nil, errors.New("no pairings available")
	}

	//
	// TODO: this should be weighetd
	wrap := s.pairing[rand.Intn(len(s.pairing))]

	//
	// TODO: we should retry with another addr
	conn, err := s.connectRawClient(ctx, wrap.Addr)
	if err != nil {
		return nil, err
	}

	wrap.Client = conn
	return wrap, nil
}

func (s *Sentry) SendRelay(
	ctx context.Context,
	cb func(clientSession *ClientSession) (*RelayReply, error),
) (*RelayReply, error) {

	//
	// Get pairing
	wrap, err := s._findPairing(ctx)
	if err != nil {
		return nil, err
	}

	//
	// Get or create session and lock it
	clientSession := func() *ClientSession {
		wrap.SessionsLock.Lock()
		defer wrap.SessionsLock.Unlock()

		for _, session := range wrap.Sessions {
			if session.Lock.TryLock() {
				return session
			}
		}

		clientSession := &ClientSession{
			SessionId: rand.Int63(),
			Client:    wrap,
		}
		clientSession.Lock.Lock()
		wrap.Sessions[clientSession.SessionId] = clientSession

		return clientSession
	}()

	//
	// call user

	reply, err := cb(clientSession)
	// TODO: check servicer pubkey is correct!

	go func() {
		//Unlock client session when done &
		defer clientSession.Lock.Unlock()

		//
		// Check we're not stale
		s.pairingMu.RLock()
		defer s.pairingMu.RUnlock()
	}()

	return reply, err
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
	clientCtx client.Context,
	specId uint64,
	isUser bool,
) *Sentry {
	rpcClient := clientCtx.Client
	specQueryClient := spectypes.NewQueryClient(clientCtx)
	servicerQueryClient := servicertypes.NewQueryClient(clientCtx)
	acc := clientCtx.GetFromAddress().String()

	return &Sentry{
		rpcClient:           rpcClient,
		specQueryClient:     specQueryClient,
		servicerQueryClient: servicerQueryClient,
		specId:              specId,
		isUser:              isUser,
		acc:                 acc,
	}
}
