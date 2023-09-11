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

type DB interface {
	Key() string
	Save(dbEntry *DBEntry) error
	BatchSave(dbEntries []*DBEntry) error
	FindOne(key string) ([]byte, error)
	FindAll() (map[string][]byte, error)
	Delete(key string) error
	DeletePrefix(prefix string) error
	Close() error
}

type DBEntry struct {
	Key  string
	Data []byte
	Ttl  time.Duration
}

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

type ConsumerProofEntity struct {
	ConsumerAddr string
	ConsumerKey  string
	Proof        *pairingtypes.RelaySession
}

func (rs *RewardDB) Save(cpe *ConsumerProofEntity) error {
	return rs.BatchSave([]*ConsumerProofEntity{cpe})
}

func (rs *RewardDB) BatchSave(cpes []*ConsumerProofEntity) (err error) {
	dbEntriesMap := map[string][]*DBEntry{} // Key is specId

	for _, reward := range cpes {
		key := rs.assembleKey(uint64(reward.Proof.Epoch), reward.ConsumerAddr, reward.Proof.SessionId, reward.ConsumerKey)
		buf, err := json.Marshal(reward)
		if err != nil {
			return utils.LavaFormatError("failed to encode proof: %s", err)
		}

		dbEntry := &DBEntry{
			Key:  key,
			Data: buf,
			Ttl:  rs.ttl,
		}

		dbEntriesMap[reward.Proof.SpecId] = append(dbEntriesMap[reward.Proof.SpecId], dbEntry)
	}

	for specId, rewards := range dbEntriesMap {
		db, found := rs.dbs[specId]
		if !found {
			return fmt.Errorf("reward_db: db not found for spec id: %s", specId)
		}

		err = db.BatchSave(rewards)
		if err != nil {
			return err
		}
	}

	return nil
}

// currently unused
func (rs *RewardDB) FindOne(
	epoch uint64,
	consumerAddr string,
	consumerKey string,
	sessionId uint64,
) (*pairingtypes.RelaySession, error) {
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
	return nil, fmt.Errorf("reward not found for key: %s", key)
}

func (rs *RewardDB) FindAll() (map[uint64]*EpochRewards, error) {
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
	// reading key before lock to avoid double locking.
	dbKey := db.Key()

	rs.lock.Lock()
	defer rs.lock.Unlock()

	_, found := rs.dbs[dbKey]
	if found {
		return fmt.Errorf("db already exists for key: %s", dbKey)
	}
	rs.dbs[dbKey] = db
	return nil
}

func (rs *RewardDB) DBExists(specId string) bool {
	_, found := rs.dbs[specId]
	return found
}

func (rs *RewardDB) Close() error {
	for _, db := range rs.dbs {
		err := db.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func NewRewardDB() *RewardDB {
	rdb := NewRewardDBWithTTL(DefaultRewardTTL)
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
