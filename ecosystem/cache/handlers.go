package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/parser"
	relaytypes "github.com/lavanet/lava/v5/types/relay"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var (
	NotFoundError     = errors.New("cache entry for specific block and request wasn't found")
	HashMismatchError = errors.New("cache entry for specific block and request had a mismatching hash stored")
	EntryTypeError    = errors.New("cache entry for specific block and request had a mismatching object stored")
)

const (
	DbValueConfirmationAttempts = 5
	SEP                         = ";"
	// CompressionThreshold is the minimum data size (in bytes) before gzip is attempted.
	CompressionThreshold = 1024
)

type RelayerCacheServer struct {
	relaytypes.UnimplementedRelayerCacheServer
	CacheServer *CacheServer
	cacheHits   uint64
	cacheMisses uint64
}

type CacheValue struct {
	Response         relaytypes.RelayReply
	Hash             []byte
	OptionalMetadata []relaytypes.Metadata
	SeenBlock        int64
	IsCompressed     bool
}

func (cv *CacheValue) ToCacheReply() *relaytypes.CacheRelayReply {
	response := cv.Response
	if cv.IsCompressed && len(response.Data) > 0 {
		decompressed, err := decompressData(response.Data)
		if err != nil {
			utils.LavaFormatError("Failed to decompress cache data", err)
		} else {
			response.Data = decompressed
		}
	}
	return &relaytypes.CacheRelayReply{
		Reply:            &response,
		OptionalMetadata: cv.OptionalMetadata,
		SeenBlock:        cv.SeenBlock,
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

// compressData gzip-compresses data if it exceeds CompressionThreshold.
// Returns (compressed, true, nil) on success, (original, false, nil) if below threshold.
func compressData(data []byte) ([]byte, bool, error) {
	if len(data) < CompressionThreshold {
		return data, false, nil
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return data, false, err
	}
	if err := w.Close(); err != nil {
		return data, false, err
	}
	compressed := buf.Bytes()
	// Only use compression if it actually saves space.
	if len(compressed) >= len(data) {
		return data, false, nil
	}
	return compressed, true, nil
}

// decompressData gunzips data compressed by compressData.
func decompressData(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// replaceRequestedBlock maps special block constants (LATEST, SAFE, etc.) to latestBlock.
func replaceRequestedBlock(requestedBlock, latestBlock int64) int64 {
	switch requestedBlock {
	case spectypes.LATEST_BLOCK:
		return latestBlock
	case spectypes.SAFE_BLOCK:
		return latestBlock
	case spectypes.FINALIZED_BLOCK:
		return latestBlock
	case spectypes.PENDING_BLOCK:
		return latestBlock
	case spectypes.EARLIEST_BLOCK:
		return spectypes.NOT_APPLICABLE
	}
	return requestedBlock
}

func (s *RelayerCacheServer) getSeenBlockForSharedStateMode(chainId string, sharedStateId string) int64 {
	if sharedStateId != "" {
		id := latestBlockKey(chainId, sharedStateId)
		value, found := getNonExpiredFromCache(s.CacheServer.finalizedCache, id)
		if !found {
			utils.LavaFormatInfo("Failed fetching state from cache for this user id", utils.LogAttr("id", id))
			return 0
		}
		utils.LavaFormatInfo("getting seen block cache", utils.LogAttr("id", id), utils.LogAttr("value", value))
		if cacheValue, ok := value.(int64); ok {
			return cacheValue
		}
		utils.LavaFormatFatal("Failed converting cache result to int64", nil, utils.LogAttr("value", value))
	}
	return 0
}

func (s *RelayerCacheServer) getBlockHeightsFromHashes(chainId string, hashes []*relaytypes.BlockHashToHeight) []*relaytypes.BlockHashToHeight {
	for _, hashToHeight := range hashes {
		formattedKey := s.formatChainIdWithHashKey(chainId, hashToHeight.Hash)
		value, found := getNonExpiredFromCache(s.CacheServer.blocksHashesToHeightsCache, formattedKey)
		if found {
			if cacheValue, ok := value.(int64); ok {
				hashToHeight.Height = cacheValue
			}
		} else {
			hashToHeight.Height = spectypes.NOT_APPLICABLE
		}
	}
	return hashes
}

func (s *RelayerCacheServer) GetRelay(ctx context.Context, relayCacheGet *relaytypes.RelayCacheGet) (*relaytypes.CacheRelayReply, error) {
	cacheReply := &relaytypes.CacheRelayReply{}
	var cacheReplyTmp *relaytypes.CacheRelayReply
	var err error
	var seenBlock int64

	defer func() {
		if err != nil {
			cacheReply.Reply = nil
		}
	}()

	originalRequestedBlock := relayCacheGet.RequestedBlock
	if originalRequestedBlock < 0 {
		getLatestBlock := s.getLatestBlock(latestBlockKey(relayCacheGet.ChainId, ""))
		relayCacheGet.RequestedBlock = replaceRequestedBlock(originalRequestedBlock, getLatestBlock)
	}

	utils.LavaFormatDebug("Got Cache Get",
		utils.Attribute{Key: "request_hash", Value: string(relayCacheGet.RequestHash)},
		utils.Attribute{Key: "finalized", Value: relayCacheGet.Finalized},
		utils.Attribute{Key: "requested_block", Value: originalRequestedBlock},
		utils.Attribute{Key: "block_hash", Value: relayCacheGet.BlockHash},
		utils.Attribute{Key: "requested_block_parsed", Value: relayCacheGet.RequestedBlock},
		utils.Attribute{Key: "seen_block", Value: relayCacheGet.SeenBlock},
	)

	var blockHashes []*relaytypes.BlockHashToHeight
	if relayCacheGet.RequestedBlock >= 0 {
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(3)

		go func() {
			defer waitGroup.Done()
			cacheReplyTmp, err = s.getRelayInner(relayCacheGet)
			if cacheReplyTmp != nil {
				cacheReply = cacheReplyTmp
			}
		}()

		go func() {
			defer waitGroup.Done()
			seenBlock = s.getSeenBlockForSharedStateMode(relayCacheGet.ChainId, relayCacheGet.SharedStateId)
			if seenBlock > relayCacheGet.SeenBlock {
				relayCacheGet.SeenBlock = seenBlock
			}
		}()

		go func() {
			defer waitGroup.Done()
			blockHashes = s.getBlockHeightsFromHashes(relayCacheGet.ChainId, relayCacheGet.BlocksHashesToHeights)
		}()

		waitGroup.Wait()

		if err == nil {
			if cacheReply.SeenBlock < lavaslices.Min([]int64{relayCacheGet.SeenBlock, relayCacheGet.RequestedBlock}) {
				err = utils.LavaFormatDebug("reply seen block is smaller than our expectations",
					utils.LogAttr("cacheReply.SeenBlock", cacheReply.SeenBlock),
					utils.LogAttr("seenBlock", relayCacheGet.SeenBlock),
				)
			}
		}

		if relayCacheGet.SeenBlock > cacheReply.SeenBlock {
			cacheReply.SeenBlock = relayCacheGet.SeenBlock
		}
	} else {
		err = utils.LavaFormatDebug("Requested block is invalid",
			utils.LogAttr("requested block", relayCacheGet.RequestedBlock),
			utils.LogAttr("request_hash", string(relayCacheGet.RequestHash)),
		)
		blockHashes = s.getBlockHeightsFromHashes(relayCacheGet.ChainId, relayCacheGet.BlocksHashesToHeights)
	}

	cacheReply.BlocksHashesToHeights = blockHashes
	if blockHashes != nil {
		utils.LavaFormatDebug("block hashes:", utils.LogAttr("hashes", blockHashes))
	}

	cacheHit := cacheReply.Reply != nil
	go func() {
		cacheMetricsContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if cacheHit {
			s.cacheHit(cacheMetricsContext)
		} else {
			s.cacheMiss(cacheMetricsContext, err)
		}

		s.CacheServer.CacheMetrics.AddApiSpecific(originalRequestedBlock, relayCacheGet.ChainId, cacheHit)
	}()

	return cacheReply, nil
}

func (s *RelayerCacheServer) formatHashKey(hash []byte, parsedRequestedBlock int64) []byte {
	utils.LavaFormatDebug("formatHashKey",
		utils.Attribute{Key: "hash", Value: fmt.Sprintf("%x", hash)},
		utils.Attribute{Key: "parsedRequestedBlock", Value: parsedRequestedBlock},
	)
	hash = binary.LittleEndian.AppendUint64(hash, uint64(parsedRequestedBlock))
	return hash
}

func (s *RelayerCacheServer) formatChainIdWithHashKey(chainId, hash string) string {
	return chainId + "_" + hash
}

func (s *RelayerCacheServer) getRelayInner(relayCacheGet *relaytypes.RelayCacheGet) (*relaytypes.CacheRelayReply, error) {
	cacheKey := s.formatHashKey(relayCacheGet.RequestHash, relayCacheGet.RequestedBlock)
	cacheVal, cacheSource, found := s.findInAllCaches(relayCacheGet.Finalized, cacheKey)
	if !found {
		return nil, NotFoundError
	}
	if cacheVal.Hash == nil {
		utils.LavaFormatDebug("returning response",
			utils.Attribute{Key: "cache_source", Value: cacheSource},
			utils.Attribute{Key: "hash", Value: "nil"},
			utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(cacheVal.Response.Data))},
		)
		return cacheVal.ToCacheReply(), nil
	}
	if bytes.Equal(cacheVal.Hash, relayCacheGet.BlockHash) {
		utils.LavaFormatDebug("returning response",
			utils.Attribute{Key: "cache_source", Value: cacheSource},
			utils.Attribute{Key: "hash", Value: "match"},
			utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(cacheVal.Response.Data))},
		)
		return cacheVal.ToCacheReply(), nil
	}
	return nil, HashMismatchError
}

func (s *RelayerCacheServer) performInt64WriteWithValidationAndRetry(
	getBlockCallback func() int64,
	setBlockCallback func(),
	newInfo int64,
) {
	existingInfo := getBlockCallback()
	if existingInfo <= newInfo {
		setBlockCallback()
		go func() {
			for i := 0; i < DbValueConfirmationAttempts; i++ {
				time.Sleep(time.Millisecond)
				currentInfo := getBlockCallback()
				if currentInfo > newInfo {
					return
				}
				if currentInfo < newInfo {
					setBlockCallback()
				}
			}
		}()
	}
}

func (s *RelayerCacheServer) setSeenBlockOnSharedStateMode(chainId, sharedStateId string, seenBlock int64) {
	if sharedStateId == "" {
		return
	}
	key := latestBlockKey(chainId, sharedStateId)
	set := func() {
		s.CacheServer.finalizedCache.SetWithTTL(key, seenBlock, 0, s.CacheServer.ExpirationFinalized)
	}
	get := func() int64 {
		return s.getSeenBlockForSharedStateMode(chainId, sharedStateId)
	}
	s.performInt64WriteWithValidationAndRetry(get, set, seenBlock)
}

func (s *RelayerCacheServer) setBlocksHashesToHeights(chainId string, blocksHashesToHeights []*relaytypes.BlockHashToHeight) {
	for _, hashToHeight := range blocksHashesToHeights {
		if hashToHeight.Height >= 0 {
			formattedKey := s.formatChainIdWithHashKey(chainId, hashToHeight.Hash)
			s.CacheServer.blocksHashesToHeightsCache.SetWithTTL(formattedKey, hashToHeight.Height, 1, s.CacheServer.ExpirationBlocksHashesToHeights)
		}
	}
}

func (s *RelayerCacheServer) SetRelay(ctx context.Context, relayCacheSet *relaytypes.RelayCacheSet) (*emptypb.Empty, error) {
	if relayCacheSet.RequestedBlock < 0 {
		return nil, utils.LavaFormatError("invalid relay cache set data, request block is negative", nil, utils.Attribute{Key: "requestBlock", Value: relayCacheSet.RequestedBlock})
	}
	latestKnownBlock := int64(math.Max(float64(relayCacheSet.Response.LatestBlock), float64(relayCacheSet.SeenBlock)))

	cacheKey := s.formatHashKey(relayCacheSet.RequestHash, relayCacheSet.RequestedBlock)
	cacheValue := formatCacheValue(relayCacheSet.Response, relayCacheSet.BlockHash, relayCacheSet.Finalized, relayCacheSet.OptionalMetadata, latestKnownBlock)
	utils.LavaFormatDebug("Got Cache Set",
		utils.Attribute{Key: "cacheKey", Value: string(cacheKey)},
		utils.Attribute{Key: "finalized", Value: fmt.Sprintf("%t", relayCacheSet.Finalized)},
		utils.Attribute{Key: "requested_block", Value: relayCacheSet.RequestedBlock},
		utils.Attribute{Key: "response_data", Value: parser.CapStringLen(string(relayCacheSet.Response.Data))},
		utils.Attribute{Key: "requestHash", Value: string(relayCacheSet.BlockHash)},
		utils.Attribute{Key: "latestKnownBlock", Value: latestKnownBlock},
		utils.Attribute{Key: "IsNodeError", Value: relayCacheSet.IsNodeError},
		utils.Attribute{Key: "BlocksHashesToHeights", Value: relayCacheSet.BlocksHashesToHeights},
	)

	if relayCacheSet.Finalized {
		cache := s.CacheServer.finalizedCache
		if relayCacheSet.IsNodeError {
			nodeErrorExpiration := lavaslices.Min([]time.Duration{time.Duration(relayCacheSet.AverageBlockTime), s.CacheServer.ExpirationNodeErrors})
			cache.SetWithTTL(string(cacheKey), cacheValue, cacheValue.Cost(), nodeErrorExpiration)
		} else {
			cache.SetWithTTL(string(cacheKey), cacheValue, cacheValue.Cost(), s.CacheServer.ExpirationFinalized)
		}
	} else {
		cache := s.CacheServer.tempCache
		cache.SetWithTTL(string(cacheKey), cacheValue, cacheValue.Cost(), s.getExpirationForChain(time.Duration(relayCacheSet.AverageBlockTime), relayCacheSet.BlockHash))
	}

	s.setSeenBlockOnSharedStateMode(relayCacheSet.ChainId, relayCacheSet.SharedStateId, latestKnownBlock)
	s.setLatestBlock(latestBlockKey(relayCacheSet.ChainId, ""), latestKnownBlock)
	s.setBlocksHashesToHeights(relayCacheSet.ChainId, relayCacheSet.BlocksHashesToHeights)
	return &emptypb.Empty{}, nil
}

func (s *RelayerCacheServer) Health(ctx context.Context, req *emptypb.Empty) (*relaytypes.CacheUsage, error) {
	return &relaytypes.CacheUsage{}, nil
}

func (s *RelayerCacheServer) cacheHit(ctx context.Context) {
	atomic.AddUint64(&s.cacheHits, 1)
	s.PrintCacheStats(ctx, "[+] cache hit")
}

func (s *RelayerCacheServer) cacheMiss(ctx context.Context, errPrint error) {
	atomic.AddUint64(&s.cacheMisses, 1)
	errMsg := "nil"
	if errPrint != nil {
		errMsg = errPrint.Error()
	}
	s.PrintCacheStats(ctx, "[-] cache miss, error:"+errMsg)
}

func (s *RelayerCacheServer) PrintCacheStats(ctx context.Context, desc string) {
	hits := atomic.LoadUint64(&s.cacheHits)
	misses := atomic.LoadUint64(&s.cacheMisses)
	_ = utils.LavaFormatDebug(desc,
		utils.Attribute{Key: "misses", Value: strconv.FormatUint(misses, 10)},
		utils.Attribute{Key: "hits", Value: strconv.FormatUint(hits, 10)},
	)
}

func (s *RelayerCacheServer) getLatestBlockInner(key string) (latestBlock int64, expirationTime time.Time) {
	value, found := getNonExpiredFromCache(s.CacheServer.finalizedCache, key)
	if !found {
		return spectypes.NOT_APPLICABLE, time.Time{}
	}
	if cacheValue, ok := value.(LastestCacheStore); ok {
		return cacheValue.latestBlock, cacheValue.latestExpirationTime
	}
	utils.LavaFormatError("latestBlock value is not a LastestCacheStore", EntryTypeError, utils.Attribute{Key: "value", Value: fmt.Sprintf("%+v", value)})
	return spectypes.NOT_APPLICABLE, time.Time{}
}

func (s *RelayerCacheServer) getLatestBlock(key string) int64 {
	latestBlock, expirationTime := s.getLatestBlockInner(key)
	if latestBlock != spectypes.NOT_APPLICABLE && expirationTime.After(time.Now()) {
		return latestBlock
	}
	return spectypes.NOT_APPLICABLE
}

func (s *RelayerCacheServer) setLatestBlock(key string, latestBlock int64) {
	cacheStore := LastestCacheStore{latestBlock: latestBlock, latestExpirationTime: time.Now().Add(DefaultExpirationForNonFinalized)}
	utils.LavaFormatDebug("setting latest block", utils.Attribute{Key: "key", Value: key}, utils.Attribute{Key: "latestBlock", Value: latestBlock})
	set := func() {
		s.CacheServer.finalizedCache.Set(key, cacheStore, cacheStore.Cost())
	}
	get := func() int64 {
		existingLatest, _ := s.getLatestBlockInner(key)
		return existingLatest
	}
	s.performInt64WriteWithValidationAndRetry(get, set, latestBlock)
}

func (s *RelayerCacheServer) getExpirationForChain(averageBlockTimeForChain time.Duration, blockHash []byte) time.Duration {
	if blockHash != nil {
		return s.CacheServer.ExpirationFinalized
	}
	return s.CacheServer.ExpirationForChain(averageBlockTimeForChain)
}

func getNonExpiredFromCache(c *ristretto.Cache[string, any], key string) (value interface{}, found bool) {
	value, found = c.Get(key)
	if found {
		return value, true
	}
	return nil, false
}

func (s *RelayerCacheServer) findInAllCaches(finalized bool, cacheKey []byte) (retVal CacheValue, cacheSource string, found bool) {
	inner := func(finalized bool, cacheKey []byte) (interface{}, string, bool) {
		if finalized {
			cache := s.CacheServer.finalizedCache
			value, found := getNonExpiredFromCache(cache, string(cacheKey))
			if found {
				return value, "finalized_cache", true
			}
			cache = s.CacheServer.tempCache
			value, found = getNonExpiredFromCache(cache, string(cacheKey))
			if found {
				return value, "temp_cache", true
			}
		} else {
			cache := s.CacheServer.tempCache
			value, found := getNonExpiredFromCache(cache, string(cacheKey))
			if found {
				return value, "temp_cache", true
			}
			cache = s.CacheServer.finalizedCache
			value, found = getNonExpiredFromCache(cache, string(cacheKey))
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

func formatCacheValue(response *relaytypes.RelayReply, hash []byte, finalized bool, optionalMetadata []relaytypes.Metadata, seenBlock int64) CacheValue {
	response.Sig = []byte{}

	compressed, isCompressed, err := compressData(response.Data)
	if err != nil {
		utils.LavaFormatWarning("Failed to compress cache data, storing uncompressed", err)
	} else if isCompressed {
		response.Data = compressed
	}

	if !finalized {
		return CacheValue{
			Response:         *response,
			Hash:             hash,
			OptionalMetadata: optionalMetadata,
			SeenBlock:        seenBlock,
			IsCompressed:     isCompressed,
		}
	}
	return CacheValue{
		Response:         *response,
		Hash:             nil,
		OptionalMetadata: optionalMetadata,
		SeenBlock:        seenBlock,
		IsCompressed:     isCompressed,
	}
}

func latestBlockKey(chainID string, uniqueId string) string {
	return chainID + "_" + uniqueId
}
