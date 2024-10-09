package performance

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/lavanet/lava/v3/utils"
	pairingtypes "github.com/lavanet/lava/v3/x/pairing/types"
)

type Cache struct {
	client       pairingtypes.RelayerCacheClient
	address      string
	serviceCtx   context.Context
	reconnecting atomic.Bool
}

const (
	ReconnectInterval = 5 * time.Second
)

func ConnectGRPCConnectionToRelayerCacheService(ctx context.Context, addr string) (*pairingtypes.RelayerCacheClient, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := lavasession.ConnectGRPCClient(connectCtx, addr, false, true, false)
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerCacheClient(conn)
	return &c, nil
}

func InitCache(ctx context.Context, addr string) (*Cache, error) {
	relayerCacheClient, err := ConnectGRPCConnectionToRelayerCacheService(ctx, addr)
	if err != nil {
		cache := &Cache{client: nil, address: addr, serviceCtx: ctx}
		go cache.reconnectLoop()
		return cache, err
	}
	cache := Cache{client: *relayerCacheClient, address: addr, serviceCtx: ctx}
	return &cache, nil
}

func (cache *Cache) reconnectLoop() {
	// This is a simple atomic operation to ensure that only one goroutine is reconnecting at a time.
	// reconnecting.CompareAndSwap(false, true):
	// if reconnecting == false {
	// 	reconnecting = true
	// 	return true -> reconnect
	// }
	// return false -> already reconnecting
	if !cache.reconnecting.CompareAndSwap(false, true) {
		return
	}

	for {
		select {
		case <-cache.serviceCtx.Done():
			return
		case <-time.After(ReconnectInterval):
			relayerCacheClient, err := ConnectGRPCConnectionToRelayerCacheService(cache.serviceCtx, cache.address)
			if err == nil {
				utils.LavaFormatInfo("Connection to cache service restored", utils.LogAttr("address", cache.address))
				cache.client = *relayerCacheClient
				cache.reconnecting.Store(false)
				return // connected
			} else {
				utils.LavaFormatDebug("Failed to connect to cache service", utils.LogAttr("address", cache.address), utils.LogAttr("error", err))
			}
		}
	}
}

func (cache *Cache) reconnectIfNeeded(err error) {
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "connection refused")) {
		utils.LavaFormatDebug("Cache connection failed, reconnecting", utils.LogAttr("address", cache.address), utils.LogAttr("error", err))
		go cache.reconnectLoop()
	}
}

func (cache *Cache) GetEntry(ctx context.Context, relayCacheGet *pairingtypes.RelayCacheGet) (reply *pairingtypes.CacheRelayReply, err error) {
	if cache == nil {
		return nil, NotInitializedError
	}

	if cache.client == nil {
		go cache.reconnectLoop()
		return nil, NotConnectedError.Wrapf("No client connected to address: %s", cache.address)
	}

	reply, err = cache.client.GetRelay(ctx, relayCacheGet)
	cache.reconnectIfNeeded(err)
	return reply, err
}

func (cache *Cache) CacheActive() bool {
	return cache != nil && cache.client != nil
}

func (cache *Cache) SetEntry(ctx context.Context, cacheSet *pairingtypes.RelayCacheSet) error {
	if cache == nil {
		return NotInitializedError
	}

	if cache.client == nil {
		go cache.reconnectLoop()
		return NotConnectedError.Wrapf("No client connected to address: %s", cache.address)
	}

	_, err := cache.client.SetRelay(ctx, cacheSet)
	cache.reconnectIfNeeded(err)
	return err
}
