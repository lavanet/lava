package rewardserver

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
)

const keySeparator = "."

type RewardDB struct {
	providerAddr string
	specId       string
	db           DB
	ttl          time.Duration
}

type RewardEntity struct {
	ProviderAddr string
	SpecId       string
	Epoch        uint64
	ConsumerAddr string
	ConsumerKey  string
	SessionId    uint64
	Proof        *pairingtypes.RelaySession
}

func (rs *RewardDB) Save(consumerAddr string, consumerKey string, proof *pairingtypes.RelaySession) (bool, error) {
	key := rs.assembleKey(uint64(proof.Epoch), consumerAddr, proof.SessionId, consumerKey)

	re := &RewardEntity{
		ProviderAddr: rs.providerAddr,
		SpecId:       rs.specId,
		Epoch:        uint64(proof.Epoch),
		ConsumerAddr: consumerAddr,
		ConsumerKey:  consumerKey,
		SessionId:    proof.SessionId,
		Proof:        proof,
	}

	buf, err := json.Marshal(re)
	if err != nil {
		return false, utils.LavaFormatError("failed to encode proof: %s", err)
	}

	err = rs.db.Save(key, buf, rs.ttl)
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
	key := rs.assembleKey(epoch, consumerAddr, sessionId, consumerKey)

	rawReward, err := rs.db.FindOne(key)
	if err != nil {
		return nil, utils.LavaFormatDebug("reward not found")
	}

	var re RewardEntity
	err = json.Unmarshal(rawReward, &re)
	if err != nil {
		return nil, utils.LavaFormatError("failed to decode proof: %s", err)
	}

	return re.Proof, nil
}

func (rs *RewardDB) FindAll() (map[uint64]*EpochRewards, error) {
	rawRewards, err := rs.db.FindAll(rs.providerAddr, rs.specId)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]*EpochRewards, len(rawRewards))
	for _, rewards := range rawRewards {
		re := RewardEntity{}
		err := json.Unmarshal(rewards, &re)
		if err != nil {
			utils.LavaFormatError("failed to decode proof: %s", err)
			continue
		}

		epochRewards, ok := result[re.Epoch]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{re.SessionId: re.Proof}
			consumerRewards := map[string]*ConsumerRewards{re.ConsumerKey: {epoch: re.Epoch, consumer: re.ConsumerAddr, proofs: proofs}}
			result[re.Epoch] = &EpochRewards{epoch: re.Epoch, consumerRewards: consumerRewards}
			continue
		}

		consumerRewards, ok := epochRewards.consumerRewards[re.ConsumerKey]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{re.SessionId: re.Proof}
			epochRewards.consumerRewards[re.ConsumerKey] = &ConsumerRewards{epoch: re.Epoch, consumer: re.ConsumerAddr, proofs: proofs}
			continue
		}

		_, ok = consumerRewards.proofs[re.SessionId]
		if !ok {
			consumerRewards.proofs[re.SessionId] = re.Proof
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

		err = rs.db.DeletePrefix(prefix)
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
	return rs.db.DeletePrefix(prefix)
}

func NewRewardDB(providerAddr string, specId string, db DB) *RewardDB {
	return NewRewardDBWithTTL(providerAddr, specId, db, DefaultRewardTTL)
}

func NewRewardDBWithTTL(providerAddr string, specId string, db DB, ttl time.Duration) *RewardDB {
	return &RewardDB{
		providerAddr: providerAddr,
		specId:       specId,
		db:           db,
		ttl:          ttl,
	}
}

func (rs *RewardDB) assembleKey(epoch uint64, consumerAddr string, sessionId uint64, consumerKey string) string {
	keyParts := []string{
		rs.providerAddr + rs.specId,
		strconv.FormatUint(epoch, 10),
		consumerAddr,
		strconv.FormatUint(sessionId, 10),
	}

	if consumerKey != "" {
		keyParts = append(keyParts, consumerKey)
	}

	return strings.Join(keyParts, keySeparator)
}
