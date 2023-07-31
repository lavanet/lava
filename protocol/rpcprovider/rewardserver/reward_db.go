package rewardserver

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
)

const keySeparator = "."

type RewardDB struct {
	lock sync.RWMutex
	dbs  map[string]DB // key is spec id
	ttl  time.Duration
}

type RewardEntity struct {
	Epoch        uint64
	ConsumerAddr string
	ConsumerKey  string
	SessionId    uint64
	Proof        *pairingtypes.RelaySession
}

func (rs *RewardDB) Save(consumerAddr string, consumerKey string, proof *pairingtypes.RelaySession) (*pairingtypes.RelaySession, bool, error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	key := rs.assembleKey(uint64(proof.Epoch), consumerAddr, proof.SessionId, consumerKey)

	re := &RewardEntity{
		Epoch:        uint64(proof.Epoch),
		ConsumerAddr: consumerAddr,
		ConsumerKey:  consumerKey,
		SessionId:    proof.SessionId,
		Proof:        proof,
	}

	prevReward, err := rs.findOne(key)
	if err != nil {
		saved, err := rs.save(key, re)
		return proof, saved, err
	}

	if prevReward.Proof.CuSum < proof.CuSum {
		if prevReward.Proof.Badge != nil && proof.Badge == nil {
			proof.Badge = prevReward.Proof.Badge
		}

		saved, err := rs.save(key, re)
		return proof, saved, err
	}

	return prevReward.Proof, false, nil
}

func (rs *RewardDB) save(key string, re *RewardEntity) (bool, error) {
	buf, err := json.Marshal(re)
	if err != nil {
		return false, utils.LavaFormatError("failed to encode proof: %s", err)
	}

	db, found := rs.dbs[re.Proof.SpecId]
	if !found {
		return false, fmt.Errorf("reward_db: db not found for spec id: %s", re.Proof.SpecId)
	}

	err = db.Save(key, buf, rs.ttl)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (rs *RewardDB) FindOne(
	epoch uint64,
	consumerAddr string,
	consumerKey string,
	sessionId uint64,
) (*pairingtypes.RelaySession, error) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	key := rs.assembleKey(epoch, consumerAddr, sessionId, consumerKey)

	re, err := rs.findOne(key)
	if err != nil {
		return nil, err
	}

	return re.Proof, nil
}

func (rs *RewardDB) findOne(key string) (*RewardEntity, error) {
	for _, db := range rs.dbs {
		reward, err := db.FindOne(key)
		// if not found, continue to next db
		if err != nil {
			continue
		}

		if reward != nil {
			re := RewardEntity{}
			err := json.Unmarshal(reward, &re)
			if err != nil {
				utils.LavaFormatError("failed to decode proof: %s", err)
				return nil, err
			}

			return &re, nil
		}
	}

	return nil, utils.LavaFormatDebug("reward not found")
}

func (rs *RewardDB) FindAll() (map[uint64]*EpochRewards, error) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	rawRewards := make(map[string]*RewardEntity)
	for _, db := range rs.dbs {
		raw, err := db.FindAll()
		if err != nil {
			return nil, err
		}

		for key, reward := range raw {
			re := RewardEntity{}
			err := json.Unmarshal(reward, &re)
			if err != nil {
				utils.LavaFormatError("failed to decode proof: %s", err)
				continue
			}

			rawRewards[key] = &re
		}
	}

	result := make(map[uint64]*EpochRewards)
	for _, reward := range rawRewards {
		epochRewards, ok := result[reward.Epoch]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{reward.SessionId: reward.Proof}
			consumerRewards := map[string]*ConsumerRewards{reward.ConsumerKey: {epoch: reward.Epoch, consumer: reward.ConsumerAddr, proofs: proofs}}
			result[reward.Epoch] = &EpochRewards{epoch: reward.Epoch, consumerRewards: consumerRewards}
			continue
		}

		consumerRewards, ok := epochRewards.consumerRewards[reward.ConsumerKey]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{reward.SessionId: reward.Proof}
			epochRewards.consumerRewards[reward.ConsumerKey] = &ConsumerRewards{epoch: reward.Epoch, consumer: reward.ConsumerAddr, proofs: proofs}
			continue
		}

		_, ok = consumerRewards.proofs[reward.SessionId]
		if !ok {
			consumerRewards.proofs[reward.SessionId] = reward.Proof
			continue
		}
	}

	return result, nil
}

func (rs *RewardDB) DeleteClaimedRewards(claimedRewards []*pairingtypes.RelaySession) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	var deletedPrefixes []string
	for _, claimedReward := range claimedRewards {
		consumer, err := sigs.ExtractSignerAddress(claimedReward)
		if err != nil {
			utils.LavaFormatError("failed to extract consumer address: %s", err)
			continue
		}

		prefix := rs.assembleKey(uint64(claimedReward.Epoch), consumer.String(), claimedReward.SessionId, "")
		if slices.Contains(deletedPrefixes, prefix) {
			continue
		}

		err = rs.deletePrefix(prefix)
		if err != nil {
			utils.LavaFormatError("failed to delete rewards: %s", err)
			continue
		}

		deletedPrefixes = append(deletedPrefixes, prefix)
	}

	return nil
}

func (rs *RewardDB) DeleteEpochRewards(epoch uint64) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	prefix := strconv.FormatUint(epoch, 10)
	return rs.deletePrefix(prefix)
}

func (rs *RewardDB) deletePrefix(prefix string) error {
	for _, db := range rs.dbs {
		err := db.DeletePrefix(prefix)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rs *RewardDB) AddDB(db DB) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	_, found := rs.dbs[db.Key()]
	if found {
		return fmt.Errorf("db already exists for key: %s", db.Key())
	}

	rs.dbs[db.Key()] = db

	return nil
}

func (rs *RewardDB) GetDB(providerAddr string, specId string, shardId uint) (DB, bool) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	db, found := rs.dbs[specId]
	return db, found
}

func (rs *RewardDB) Close() error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	for _, db := range rs.dbs {
		err := db.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func NewRewardDB(db DB) *RewardDB {
	rdb := NewRewardDBWithTTL(DefaultRewardTTL)
	rdb.AddDB(db)
	return rdb
}

func NewRewardDBWithTTL(ttl time.Duration) *RewardDB {
	return &RewardDB{
		dbs: map[string]DB{},
		ttl: ttl,
	}
}

func (rs *RewardDB) assembleKey(epoch uint64, consumerAddr string, sessionId uint64, consumerKey string) string {
	keyParts := []string{
		strconv.FormatUint(epoch, 10),
		consumerAddr,
		strconv.FormatUint(sessionId, 10),
	}

	if consumerKey != "" {
		keyParts = append(keyParts, consumerKey)
	}

	return strings.Join(keyParts, keySeparator)
}
