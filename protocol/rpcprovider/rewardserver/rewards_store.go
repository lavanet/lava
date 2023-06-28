package rewardserver

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const keySeparator = "."

type RewardStore struct {
	store map[string]*RewardEntity
	lock  sync.RWMutex
}

type RewardEntity struct {
	epoch        uint64
	consumerAddr string
	consumerKey  string
	sessionId    uint64
	proof        *pairingtypes.RelaySession
}

func (rs *RewardStore) Save(ctx context.Context, consumerAddr string, consumerKey string, proof *pairingtypes.RelaySession) (bool, error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	key := assembleKey(uint64(proof.Epoch), consumerAddr, consumerKey, proof.SessionId)

	rs.store[key] = &RewardEntity{
		epoch:        uint64(proof.Epoch),
		consumerAddr: consumerAddr,
		consumerKey:  consumerKey,
		sessionId:    proof.SessionId,
		proof:        proof,
	}

	return true, nil
}

func (rs *RewardStore) FindOne(
	ctx context.Context,
	epoch uint64,
	consumerAddr string,
	consumerKey string,
	sessionId uint64,
) (*pairingtypes.RelaySession, error) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	key := assembleKey(epoch, consumerAddr, consumerKey, sessionId)
	re, ok := rs.store[key]
	if !ok {
		return nil, utils.LavaFormatDebug("reward not found")
	}
	return re.proof, nil
}

func (rs *RewardStore) FindAll(ctx context.Context) (map[uint64]*EpochRewards, error) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	result := make(map[uint64]*EpochRewards)
	for key, rewards := range rs.store {
		epoch, consumerAddr, consumerKey, sessionId := keyParts(key)

		epochRewards, ok := result[epoch]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{sessionId: rewards.proof}
			consumerRewards := map[string]*ConsumerRewards{consumerKey: &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs}}
			result[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewards}
			continue
		}

		consumerRewards, ok := epochRewards.consumerRewards[consumerKey]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{sessionId: rewards.proof}
			epochRewards.consumerRewards[consumerKey] = &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs}
			continue
		}

		_, ok = consumerRewards.proofs[sessionId]
		if !ok {
			consumerRewards.proofs[sessionId] = rewards.proof
			continue
		}
	}
	return result, nil
}

func (rs *RewardStore) DeleteClaimedRewards(ctx context.Context, claimedRewards []*pairingtypes.RelaySession) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	for key, re := range rs.store {
		for _, claimedReward := range claimedRewards {
			consumerAddr, err := sigs.ExtractSignerAddress(claimedReward)
			if err != nil {
				continue
			}

			if re.epoch == uint64(claimedReward.Epoch) &&
				re.consumerAddr == consumerAddr.String() &&
				re.sessionId == claimedReward.SessionId {
				delete(rs.store, key)
			}
		}
	}
	return nil
}

func NewRewardStore() *RewardStore {
	return &RewardStore{
		store: make(map[string]*RewardEntity),
		lock:  sync.RWMutex{},
	}
}

func assembleKey(epoch uint64, consumerAddr, consumerKey string, sessionId uint64) string {
	keyParts := []string{
		strconv.FormatUint(epoch, 10),
		consumerAddr,
		consumerKey,
		strconv.FormatUint(sessionId, 10),
	}

	return strings.Join(keyParts, keySeparator)
}

func keyParts(key string) (epoch uint64, consumerAddr string, consumerKey string, sessionId uint64) {
	parts := strings.Split(key, keySeparator)

	epoch, _ = strconv.ParseUint(parts[0], 10, 64)
	consumerAddr = parts[1]
	consumerKey = parts[2]
	sessionId, _ = strconv.ParseUint(parts[3], 10, 64)

	return
}
