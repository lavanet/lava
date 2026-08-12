package relaycore

import (
	"math"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

const (
	CacheMaxCost     = 20000 // each item cost would be 1
	CacheNumCounters = 20000 // expect 2000 items
	EntryTTL         = 5 * time.Minute

	// A seen block entry lives at most EntryTTL, so the largest advance a healthy
	// chain can legitimately show between two updates is the number of blocks it
	// produces in that window. seenBlockAdvanceSafetyFactor multiplies that bound so
	// bursty block production, clock skew and provider spread never trip the guard —
	// it is meant to reject values that are impossible, not merely surprising.
	seenBlockAdvanceSafetyFactor = 4
	// minSeenBlockAdvance keeps the bound usable on chains that produce less than one
	// block per EntryTTL, where the computed bound truncates to zero and would reject
	// every advance. It is deliberately small: a blanket allowance large enough for a
	// fast chain would leave a slow one — BTC produces a block every 10 minutes —
	// accepting jumps of days' worth of blocks, which is exactly what this guard is
	// supposed to catch.
	minSeenBlockAdvance = 16
)

// Consistency interface for managing block consistency
type Consistency interface {
	SetSeenBlock(blockSeen int64, userData common.UserData)
	GetSeenBlock(userData common.UserData) (int64, bool)
	SetSeenBlockFromKey(blockSeen int64, key string)
	Key(userData common.UserData) string
}

// ConsistencyImpl is the default implementation of Consistency
type ConsistencyImpl struct {
	cache            *ristretto.Cache[string, any]
	specId           string
	averageBlockTime time.Duration
}

// maxPlausibleSeenBlockAdvance is the largest jump in seen block this chain could
// legitimately produce within the lifetime of a cache entry. Anything beyond it did
// not come from the chain advancing.
//
// A zero or negative average block time means the chain's cadence is unknown, so the
// guard fails open rather than gating on a number it cannot justify.
func (cc *ConsistencyImpl) maxPlausibleSeenBlockAdvance() int64 {
	if cc.averageBlockTime <= 0 {
		return math.MaxInt64
	}
	blocksWithinTTL := int64(EntryTTL / cc.averageBlockTime)
	if bound := blocksWithinTTL * seenBlockAdvanceSafetyFactor; bound > minSeenBlockAdvance {
		return bound
	}
	return minSeenBlockAdvance
}

func (cc *ConsistencyImpl) SetLatestBlock(key string, block int64) {
	// we keep consistency data for 5 minutes
	// if in that time no new block was updated we will remove seen data and let providers return what they have
	cc.cache.SetWithTTL(key, block, 1, EntryTTL)
}

func (cc *ConsistencyImpl) GetLatestBlock(key string) (block int64, found bool) {
	storedVal, found := cc.cache.Get(key)
	if found {
		var ok bool
		block, ok = storedVal.(int64)
		if !ok {
			utils.LavaFormatError("failed to cast block from cache", nil,
				utils.Attribute{Key: "storedVal", Value: storedVal},
				utils.Attribute{Key: "specId", Value: cc.specId},
			)
		}
	}
	return block, found
}

func (cc *ConsistencyImpl) Key(userData common.UserData) string {
	return userData.DappId + "__" + userData.ConsumerIp
}

// used on subscription, where we already have the dapp key stored, but we don't keep the dappId and ip separately
func (cc *ConsistencyImpl) SetSeenBlockFromKey(blockSeen int64, key string) {
	if cc == nil {
		return
	}
	block, found := cc.GetLatestBlock(key)
	if !found {
		// Cache writes are applied asynchronously, so a value written moments ago may
		// not be readable yet. Flush once and look again before concluding the entry is
		// genuinely cold: relay responses are handled back-to-back, and treating a
		// not-yet-visible baseline as absent would skip the guard for precisely the
		// rapid successive updates it exists to police.
		cc.cache.Wait()
		block, found = cc.GetLatestBlock(key)
	}
	if found {
		// seen block is only increasing
		if block >= blockSeen {
			return
		}
		// The seen block is taken from whatever a provider reports as its latest block
		// and is then sent to every other provider, which bail when they cannot reach
		// it. A single provider reporting a value from outside this chain's numeric
		// domain — a different unit, a bug, or a deliberately inflated number — would
		// otherwise pin the seen block out of reach and make every honest provider look
		// hopelessly behind, for the lifetime of the entry.
		if advance := blockSeen - block; advance > cc.maxPlausibleSeenBlockAdvance() {
			utils.LavaFormatWarning("discarding implausible seen block advance", nil,
				utils.LogAttr("specId", cc.specId),
				utils.LogAttr("storedSeenBlock", block),
				utils.LogAttr("reportedSeenBlock", blockSeen),
				utils.LogAttr("advance", advance),
				utils.LogAttr("maxPlausibleAdvance", cc.maxPlausibleSeenBlockAdvance()),
				utils.LogAttr("averageBlockTime", cc.averageBlockTime),
			)
			return
		}
	}
	cc.SetLatestBlock(key, blockSeen)
}

func (cc *ConsistencyImpl) SetSeenBlock(blockSeen int64, userData common.UserData) {
	if cc == nil {
		return
	}
	if userData.DappId == "" {
		return
	}
	key := cc.Key(userData)
	cc.SetSeenBlockFromKey(blockSeen, key)
}

func (cc *ConsistencyImpl) GetSeenBlock(userData common.UserData) (int64, bool) {
	if cc == nil {
		return 0, false
	}
	return cc.GetLatestBlock(cc.Key(userData))
}

// NewConsistency builds the per-chain seen block store. averageBlockTime comes from
// the spec and bounds how far the seen block may advance at once; pass 0 when it is
// not known, which disables that guard.
func NewConsistency(specId string, averageBlockTime time.Duration) Consistency {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for consistency", err)
	}
	return &ConsistencyImpl{cache: cache, specId: specId, averageBlockTime: averageBlockTime}
}
