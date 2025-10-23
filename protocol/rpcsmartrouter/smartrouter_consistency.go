package rpcsmartrouter

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
)

// this class handles seen block values in requests
const (
	CacheMaxCost     = 20000 // each item cost would be 1
	CacheNumCounters = 20000 // expect 2000 items
	EntryTTL         = 5 * time.Minute
)

type SmartRouterConsistency struct {
	cache  *ristretto.Cache[string, any]
	specId string
}

func (cc *SmartRouterConsistency) setLatestBlock(key string, block int64) {
	// we keep consistency data for 5 minutes
	// if in that time no new block was updated we will remove seen data and let providers return what they have
	cc.cache.SetWithTTL(key, block, 1, EntryTTL)
}

func (cc *SmartRouterConsistency) getLatestBlock(key string) (block int64, found bool) {
	storedVal, found := cc.cache.Get(key)
	if found {
		var ok bool
		block, ok = storedVal.(int64)
		if !ok {
			utils.LavaFormatFatal("invalid usage of cache", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
	} else {
		// no data
		block = 0
	}
	return block, found
}

func (cc *SmartRouterConsistency) Key(userData common.UserData) string {
	return userData.DappId + "__" + userData.ConsumerIp
}

// used on subscription, where we already have the dapp key stored, but we don't keep the dappId and ip separately
func (cc *SmartRouterConsistency) SetSeenBlockFromKey(blockSeen int64, key string) {
	if cc == nil {
		return
	}
	block, _ := cc.getLatestBlock(key)
	if block < blockSeen {
		cc.setLatestBlock(key, blockSeen)
	}
}

func (cc *SmartRouterConsistency) SetSeenBlock(blockSeen int64, userData common.UserData) {
	if cc == nil {
		return
	}
	if blockSeen == chaintracker.DummyChainTrackerLatestBlock {
		return
	}
	key := cc.Key(userData)
	cc.SetSeenBlockFromKey(blockSeen, key)
}

func (cc *SmartRouterConsistency) GetSeenBlock(userData common.UserData) (int64, bool) {
	if cc == nil {
		return 0, false
	}
	return cc.getLatestBlock(cc.Key(userData))
}

func NewSmartRouterConsistency(specId string) *SmartRouterConsistency {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for consumer consistency", err)
	}
	return &SmartRouterConsistency{cache: cache, specId: specId}
}
