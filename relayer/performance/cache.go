package performance

import (
	"context"
	"time"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
)

type Cache struct {
	client pairingtypes.RelayerCacheClient
}

func ConnectGRPCConnectionToRelayerCacheService(ctx context.Context, addr string) (*pairingtypes.RelayerCacheClient, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
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
		return nil, err
	}
	cache := Cache{client: *relayerCacheClient}
	return &cache, nil
}

func (cache *Cache) GetEntry(ctx context.Context, request *pairingtypes.RelayRequest, apiInterface string, blockHash []byte) (reply *pairingtypes.RelayReply, err error) {
	if cache == nil {
		//TODO: try to connect again once in a while
		return nil, NotConnectedError
	}
	//TODO: handle disconnections and error types here
	return cache.client.GetRelay(ctx, &pairingtypes.RelayCacheGet{Request: request, ApiInterface: apiInterface, BlockHash: blockHash})
}

func (cache *Cache) SetEntry(ctx context.Context, request *pairingtypes.RelayRequest, apiInterface string, blockHash []byte, reply *pairingtypes.RelayReply, latest bool) {
	if cache == nil {
		//TODO: try to connect again once in a while
		return
	}
	//TODO: handle disconnections and error types here
	cache.client.SetRelay(ctx, &pairingtypes.RelayCacheSet{Request: request, ApiInterface: apiInterface, BlockHash: blockHash, Response: reply, Latest: latest})
}
