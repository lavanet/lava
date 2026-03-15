package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type relayerCacheClientStore struct {
	client       pairingtypes.RelayerCacheClient
	conn         *grpc.ClientConn // stored so we can close it on reconnect or shutdown
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

func (r *relayerCacheClientStore) connectGRPCConnectionToRelayerCacheService() (*pairingtypes.RelayerCacheClient, *grpc.ClientConn, error) {
	connectCtx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	conn, err := lavasession.ConnectGRPCClient(connectCtx, r.address, false, true, false)
	if err != nil {
		return nil, nil, err
	}

	c := pairingtypes.NewRelayerCacheClient(conn)
	return &c, conn, nil
}

func (r *relayerCacheClientStore) connectClient() error {
	relayerCacheClient, conn, err := r.connectGRPCConnectionToRelayerCacheService()
	if err == nil {
		utils.LavaFormatInfo("cache service connected successfully", utils.LogAttr("address", r.address))
		func() {
			r.lock.Lock()
			defer r.lock.Unlock()
			// Close the old connection before replacing it to prevent goroutine leaks.
			// Each *grpc.ClientConn spawns internal goroutines (reader, writer, callback serializers)
			// that only exit when conn.Close() is called.
			if r.conn != nil {
				utils.LavaFormatDebug("closing previous cache gRPC connection before replacing", utils.LogAttr("address", r.address))
				r.conn.Close()
			}
			r.client = *relayerCacheClient
			r.conn = conn
		}()

		r.reconnecting.Store(false)
		return nil // connected
	}

	utils.LavaFormatDebug("cache service connection attempt failed", utils.LogAttr("address", r.address), utils.LogAttr("error", err))
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

	utils.LavaFormatInfo("cache service reconnection loop started", utils.LogAttr("address", r.address))

	for {
		select {
		case <-r.ctx.Done():
			utils.LavaFormatInfo("cache service reconnection loop exiting (context cancelled)", utils.LogAttr("address", r.address))
			return
		case <-time.After(reconnectInterval):
			// connectClient() returns nil on success, non-nil error on failure.
			// Exit the loop on success, keep retrying on failure.
			if r.connectClient() == nil {
				utils.LavaFormatInfo("cache service reconnection succeeded, exiting reconnect loop", utils.LogAttr("address", r.address))
				return
			}
		}
	}
}

// resetOnConnectionError clears the client when a gRPC connection-level error is detected,
// allowing the next getClient() call to trigger reconnection.
func (r *relayerCacheClientStore) resetOnConnectionError(err error) {
	if err == nil {
		return
	}
	code := status.Code(err)
	if code != codes.Unavailable {
		return
	}
	utils.LavaFormatWarning("cache service connection error detected, triggering reconnection", err, utils.LogAttr("address", r.address))
	r.lock.Lock()
	r.client = nil
	r.lock.Unlock()
	go r.reconnectClient()
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
	if err != nil {
		cache.clientStore.resetOnConnectionError(err)
	}
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
	if err != nil {
		cache.clientStore.resetOnConnectionError(err)
	}
	return err
}
