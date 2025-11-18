package relaycore

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

const (
	CacheMaxCost     = 20000 // each item cost would be 1
	CacheNumCounters = 20000 // expect 2000 items
	EntryTTL         = 5 * time.Minute
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
	cache  *ristretto.Cache[string, any]
	specId string
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
	// seen block is only increasing
	if block, found := cc.GetLatestBlock(key); found && block >= blockSeen {
		return
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

func NewConsistency(specId string) Consistency {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for consistency", err)
	}
	return &ConsistencyImpl{cache: cache, specId: specId}
}
