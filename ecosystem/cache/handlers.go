package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/ecosystem/cache/format"
	rpcInterfaceMessages "github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/parser"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var (
	NotFoundError     = sdkerrors.New("Cache miss", 1, "cache entry for specific block and request wasn't found")                                                   // client could'nt connect to any provider.
	HashMismatchError = sdkerrors.New("Cache hit but hash mismatch", 2, "cache entry for specific block and request had a mismatching hash stored")                 // client could'nt connect to any provider.
	EntryTypeError    = sdkerrors.New("Cache hit but entry is a different object", 3, "cache entry for specific block and request had a mismatching object stored") // client could'nt connect to any provider.
)

const (
	SEP = ";"
)

type RelayerCacheServer struct {
	pairingtypes.UnimplementedRelayerCacheServer
	CacheServer *CacheServer
	cacheHits   uint64
	cacheMisses uint64
}

type CacheValue struct {
	Response         pairingtypes.RelayReply
	Hash             []byte
	OptionalMetadata []pairingtypes.Metadata
}

func (cv *CacheValue) ToCacheReply() *pairingtypes.CacheRelayReply {
	return &pairingtypes.CacheRelayReply{
		Reply:            &cv.Response,
		OptionalMetadata: cv.OptionalMetadata,
	}
}

func (cv *CacheValue) Cost() int64 {
	return int64(len(cv.Response.Data))
}

type LastestCacheStore struct {
	latestBlock          int64
	latestExpirationTime time.Time
}

func (cv *LastestCacheStore) Cost() int64 {
	return 8 + 16
}

func (s *RelayerCacheServer) GetRelay(ctx context.Context, relayCacheGet *pairingtypes.RelayCacheGet) (cacheReply *pairingtypes.CacheRelayReply, err error) {
	requestedBlock := relayCacheGet.Request.RequestBlock // save requested block

	cacheReply, err = s.getRelayInner(ctx, relayCacheGet)
	var hit bool
	if err != nil {
		s.cacheMiss(ctx, err)
	} else {
		hit = true
		s.cacheHit(ctx)
	}
	// add prometheus metrics
	s.CacheServer.CacheMetrics.AddApiSpecific(requestedBlock, relayCacheGet.ChainID, getMethodFromRequest(relayCacheGet), relayCacheGet.Request.ApiInterface, hit)
	return
}

func (s *RelayerCacheServer) getRelayInner(ctx context.Context, relayCacheGet *pairingtypes.RelayCacheGet) (*pairingtypes.CacheRelayReply, error) {
	inputFormatter, outputFormatter := format.FormatterForRelayRequestAndResponse(relayCacheGet.Request.ApiInterface)
	relayCacheGet.Request.Data = inputFormatter(relayCacheGet.Request.Data)
	requestedBlock := relayCacheGet.Request.RequestBlock
	getLatestBlock := s.getLatestBlock(relayCacheGet.ChainID, relayCacheGet.Provider)
	relayCacheGet.Request.RequestBlock = lavaprotocol.ReplaceRequestedBlock(requestedBlock, getLatestBlock)
	cacheKey := formatCacheKey(relayCacheGet.Request.ApiInterface, relayCacheGet.ChainID, relayCacheGet.Request, relayCacheGet.Provider)
	utils.LavaFormatDebug("Got Cache Get", utils.Attribute{Key: "cacheKey", Value: parser.CapStringLen(cacheKey)},
		utils.Attribute{Key: "finalized", Value: relayCacheGet.Finalized},
		utils.Attribute{Key: "requestedBlock", Value: requestedBlock},
		utils.Attribute{Key: "requestHash", Value: relayCacheGet.BlockHash},
		utils.Attribute{Key: "getLatestBlock", Value: relayCacheGet.Request.RequestBlock},
	)
	cacheVal, cache_source, found := s.findInAllCaches(relayCacheGet.Finalized, cacheKey)
	// TODO: use the information when a new block is finalized
	if !found {
		return nil, NotFoundError
	}
	if cacheVal.Hash == nil {
		// if we didn't store a hash its also always a match
		cacheVal.Response.Data = outputFormatter(cacheVal.Response.Data)
		utils.LavaFormatDebug("returning response", utils.Attribute{Key: "cache_source", Value: cache_source},
			utils.Attribute{Key: "hash", Value: "nil"},
			utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(cacheVal.Response.Data))},
		)
		return cacheVal.ToCacheReply(), nil
	}
	// entry found, now we check the hash requested and hash stored
	if bytes.Equal(cacheVal.Hash, relayCacheGet.BlockHash) {
		cacheVal.Response.Data = outputFormatter(cacheVal.Response.Data)
		utils.LavaFormatDebug("returning response", utils.Attribute{Key: "cache_source", Value: cache_source},
			utils.Attribute{Key: "hash", Value: "match"},
			utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(cacheVal.Response.Data))},
		)
		return cacheVal.ToCacheReply(), nil
	}
	// TODO: handle case where we have hash stored and it became finalized
	return nil, HashMismatchError
}

func (s *RelayerCacheServer) SetRelay(ctx context.Context, relayCacheSet *pairingtypes.RelayCacheSet) (*emptypb.Empty, error) {
	if relayCacheSet.Request.RequestBlock < 0 {
		return nil, utils.LavaFormatError("invalid relay cache set data, request block is negative", nil, utils.Attribute{Key: "requestBlock", Value: relayCacheSet.Request.RequestBlock})
	}
	// TODO: make this non-blocking
	inputFormatter, _ := format.FormatterForRelayRequestAndResponse(relayCacheSet.Request.ApiInterface)
	relayCacheSet.Request.Data = inputFormatter(relayCacheSet.Request.Data) // so we can find the entry regardless of id

	cacheKey := formatCacheKey(relayCacheSet.Request.ApiInterface, relayCacheSet.ChainID, relayCacheSet.Request, relayCacheSet.Provider)
	cacheValue := formatCacheValue(relayCacheSet.Response, relayCacheSet.BlockHash, relayCacheSet.Finalized, relayCacheSet.OptionalMetadata)
	utils.LavaFormatDebug("Got Cache Set", utils.Attribute{Key: "cacheKey", Value: parser.CapStringLen(cacheKey)},
		utils.Attribute{Key: "finalized", Value: fmt.Sprintf("%t", relayCacheSet.Finalized)},
		utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(relayCacheSet.Response.Data))},
		utils.Attribute{Key: "requestHash", Value: string(relayCacheSet.BlockHash)})
	// finalized entries can stay there
	if relayCacheSet.Finalized {
		cache := s.CacheServer.finalizedCache
		cache.SetWithTTL(cacheKey, cacheValue, cacheValue.Cost(), s.CacheServer.ExpirationFinalized)
	} else {
		cache := s.CacheServer.tempCache
		cache.SetWithTTL(cacheKey, cacheValue, cacheValue.Cost(), s.getExpirationForChain(relayCacheSet.ChainID, relayCacheSet.BlockHash))
	}
	s.setLatestBlock(relayCacheSet.ChainID, relayCacheSet.Provider, relayCacheSet.Request.RequestBlock)
	return &emptypb.Empty{}, nil
}

func (s *RelayerCacheServer) Health(ctx context.Context, req *emptypb.Empty) (*pairingtypes.CacheUsage, error) {
	cacheHits := atomic.LoadUint64(&s.cacheHits)
	cacheMisses := atomic.LoadUint64(&s.cacheMisses)
	return &pairingtypes.CacheUsage{CacheHits: cacheHits, CacheMisses: cacheMisses}, nil
}

func (s *RelayerCacheServer) cacheHit(ctx context.Context) {
	atomic.AddUint64(&s.cacheHits, 1)
	s.PrintCacheStats(ctx, "[+] cache hit")
}

func (s *RelayerCacheServer) cacheMiss(ctx context.Context, errPrint error) {
	atomic.AddUint64(&s.cacheMisses, 1)
	s.PrintCacheStats(ctx, "[-] cache miss, error:"+errPrint.Error())
}

func (s *RelayerCacheServer) PrintCacheStats(ctx context.Context, desc string) {
	health, err := s.Health(ctx, nil)
	if err != nil {
		_ = utils.LavaFormatError("Failed to get health response", err)
	}
	_ = utils.LavaFormatDebug(desc,
		utils.Attribute{Key: "misses", Value: strconv.FormatUint(health.CacheMisses, 10)},
		utils.Attribute{Key: "hits", Value: strconv.FormatUint(health.CacheHits, 10)},
	)
}

func (s *RelayerCacheServer) getLatestBlockInner(chainID string, providerAddr string) (latestBlock int64, expirationTime time.Time) {
	value, found := getNonExpiredFromCache(s.CacheServer.finalizedCache, latestBlockKey(chainID, providerAddr))
	if !found {
		return spectypes.NOT_APPLICABLE, time.Time{}
	}
	if cacheValue, ok := value.(LastestCacheStore); ok {
		return cacheValue.latestBlock, cacheValue.latestExpirationTime
	}
	utils.LavaFormatError("latestBlock value is not a LastestCacheStore", EntryTypeError, utils.Attribute{Key: "value", Value: fmt.Sprintf("%+v", value)})
	return spectypes.NOT_APPLICABLE, time.Time{}
}

func (s *RelayerCacheServer) getLatestBlock(chainID string, providerAddr string) int64 {
	latestBlock, expirationTime := s.getLatestBlockInner(chainID, providerAddr)
	if latestBlock != spectypes.NOT_APPLICABLE && expirationTime.After(time.Now()) {
		return latestBlock
	}
	return spectypes.NOT_APPLICABLE
}

func (s *RelayerCacheServer) setLatestBlock(chainID string, providerAddr string, latestBlock int64) {
	existingLatest, _ := s.getLatestBlockInner(chainID, providerAddr) // we need to bypass the expirationTimeCheck

	if existingLatest <= latestBlock { // equal refreshes latest if it expired
		// we are setting this with a futuristic invalidation time, we still want the entry in cache to protect us from putting a lower last block
		cacheStore := LastestCacheStore{latestBlock: latestBlock, latestExpirationTime: time.Now().Add(DefaultExpirationForNonFinalized)}
		utils.LavaFormatDebug("setting latest block", utils.Attribute{Key: "providerAddr", Value: providerAddr}, utils.Attribute{Key: "chainID", Value: chainID}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
		s.CacheServer.finalizedCache.Set(latestBlockKey(chainID, providerAddr), cacheStore, cacheStore.Cost()) // no expiration time
	}
}

func (s *RelayerCacheServer) getExpirationForChain(chainID string, blockHash []byte) time.Duration {
	if blockHash != nil {
		// this means that this entry has a block hash, so we don't have to delete it quickly
		return s.CacheServer.ExpirationFinalized
	}
	// if there is no block hash, for non finalized we cant know if there was a fork, so we have to delete it as soon as we have new data
	// with the assumption new data should arrive by the arrival of a new block (average block time)
	return s.CacheServer.ExpirationForChain(chainID)
}

func getNonExpiredFromCache(c *ristretto.Cache, key string) (value interface{}, found bool) {
	value, found = c.Get(key)
	if found {
		return value, true
	}
	return nil, false
}

func (s *RelayerCacheServer) findInAllCaches(finalized bool, cacheKey string) (retVal CacheValue, cacheSource string, found bool) {
	inner := func(finalized bool, cacheKey string) (interface{}, string, bool) {
		if finalized {
			cache := s.CacheServer.finalizedCache
			value, found := getNonExpiredFromCache(cache, cacheKey)
			if found {
				return value, "finalized_cache", true
			}
			// if a key is finalized still doesn't mean it wasn't set when unfinalized
			cache = s.CacheServer.tempCache
			value, found = getNonExpiredFromCache(cache, cacheKey)
			if found {
				return value, "temp_cache", true
			}
		} else {
			// if something isn't finalized now it was never finalized, but sometimes when we don't have information we try to get a non finalized entry when in fact its finalized
			cache := s.CacheServer.tempCache
			value, found := getNonExpiredFromCache(cache, cacheKey)
			if found {
				return value, "temp_cache", true
			}
			cache = s.CacheServer.finalizedCache
			value, found = getNonExpiredFromCache(cache, cacheKey)
			if found {
				return value, "finalized_cache", true
			}
		}

		return nil, "", false
	}

	value, cacheSource, found := inner(finalized, cacheKey)
	if !found {
		return CacheValue{}, "", false
	}
	if cacheVal, ok := value.(CacheValue); ok {
		return cacheVal, cacheSource, true
	}
	utils.LavaFormatError("entry in cache was not a CacheValue", EntryTypeError, utils.Attribute{Key: "entry", Value: fmt.Sprintf("%+v", value)})
	return CacheValue{}, "", false
}

func formatCacheKey(apiInterface string, chainID string, request *pairingtypes.RelayPrivateData, provider string) string {
	return chainID + SEP + usedFieldsFromRequest(request, provider)
}

func usedFieldsFromRequest(request *pairingtypes.RelayPrivateData, provider string) string {
	// used fields:
	// RelayData except for salt: because it defines the query
	// Provider: because we want to keep coherence between calls, assuming different providers can return different forks, useful for cache in rpcconsumer
	request.Salt = nil
	relayDataStr := request.String()
	return relayDataStr + SEP + provider
}

func formatCacheValue(response *pairingtypes.RelayReply, hash []byte, finalized bool, optionalMetadata []pairingtypes.Metadata) CacheValue {
	response.Sig = []byte{} // make sure we return a signed value, as the output was modified by our outputParser
	if !finalized {
		// hash value is only used on non finalized entries to check for forks
		return CacheValue{
			Response:         *response,
			Hash:             hash,
			OptionalMetadata: optionalMetadata,
		}
	}
	// no need to store the hash value for finalized entries
	return CacheValue{
		Response:         *response,
		Hash:             nil,
		OptionalMetadata: optionalMetadata,
	}
}

func latestBlockKey(chainID string, providerAddr string) string {
	// because we want to support coherence in providers
	return chainID + providerAddr
}

func getMethodFromRequest(relayCacheGet *pairingtypes.RelayCacheGet) string {
	if relayCacheGet.Request.ApiUrl != "" {
		return relayCacheGet.Request.ApiUrl
	}
	var msg rpcInterfaceMessages.JsonrpcMessage
	err := json.Unmarshal(relayCacheGet.Request.Data, &msg)
	if err != nil {
		return "failed_parsing_method"
	}
	return msg.Method
}
