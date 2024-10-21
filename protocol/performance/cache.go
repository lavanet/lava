package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

type relayerCacheClientStore struct {
	client       pairingtypes.RelayerCacheClient
	lock         sync.RWMutex
	ctx          context.Context
	address      string
	reconnecting atomic.Bool
}

const (
	reconnectInterval = 5 * time.Second
)

func newRelayerCacheClientStore(ctx context.Context, address string) (*relayerCacheClientStore, error) {
	clientStore := &relayerCacheClientStore{
		client:  nil,
		ctx:     ctx,
		address: address,
	}
	return clientStore, clientStore.connectClient()
}

func (r *relayerCacheClientStore) getClient() pairingtypes.RelayerCacheClient {
	if r == nil {
		return nil
	}

	r.lock.RLock()
	defer r.lock.RUnlock()

	if r.client == nil {
		go r.reconnectClient()
	}

	return r.client // might be nil
}

func (r *relayerCacheClientStore) connectGRPCConnectionToRelayerCacheService() (*pairingtypes.RelayerCacheClient, error) {
	connectCtx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	conn, err := lavasession.ConnectGRPCClient(connectCtx, r.address, false, true, false)
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerCacheClient(conn)
	return &c, nil
}

func (r *relayerCacheClientStore) connectClient() error {
	relayerCacheClient, err := r.connectGRPCConnectionToRelayerCacheService()
	if err == nil {
		utils.LavaFormatInfo("Connected to cache service", utils.LogAttr("address", r.address))
		func() {
			r.lock.Lock()
			defer r.lock.Unlock()
			r.client = *relayerCacheClient
		}()

		r.reconnecting.Store(false)
		return nil // connected
	}

	utils.LavaFormatDebug("Failed to connect to cache service", utils.LogAttr("address", r.address), utils.LogAttr("error", err))
	return err
}

func (r *relayerCacheClientStore) reconnectClient() {
	// This is a simple atomic operation to ensure that only one goroutine is reconnecting at a time.
	// reconnecting.CompareAndSwap(false, true):
	// if reconnecting == false {
	// 	reconnecting = true
	// 	return true -> reconnect
	// }
	// return false -> already reconnecting
	if !r.reconnecting.CompareAndSwap(false, true) {
		return
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-time.After(reconnectInterval):
			if r.connectClient() != nil {
				return
			}
		}
	}
}

type Cache struct {
	clientStore *relayerCacheClientStore
	address     string
	serviceCtx  context.Context
}

func InitCache(ctx context.Context, addr string) (*Cache, error) {
	clientStore, err := newRelayerCacheClientStore(ctx, addr)
	return &Cache{
		clientStore: clientStore,
		address:     addr,
		serviceCtx:  ctx,
	}, err
}

func (cache *Cache) GetEntry(ctx context.Context, relayCacheGet *pairingtypes.RelayCacheGet) (reply *pairingtypes.CacheRelayReply, err error) {
	if cache == nil {
		return nil, NotInitializedError
	}

	client := cache.clientStore.getClient()
	if client == nil {
		return nil, NotConnectedError
	}

	reply, err = client.GetRelay(ctx, relayCacheGet)
	return reply, err
}

func (cache *Cache) CacheActive() bool {
	return cache != nil && cache.clientStore.getClient() != nil
}

func (cache *Cache) SetEntry(ctx context.Context, cacheSet *pairingtypes.RelayCacheSet) error {
	if cache == nil {
		return NotInitializedError
	}

	client := cache.clientStore.getClient()
	if client == nil {
		return NotConnectedError
	}

	_, err := client.SetRelay(ctx, cacheSet)
	return err
}
