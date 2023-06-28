package rewardserver

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
)

const keySeparator = "."

type RewardStore struct {
	db DB
}

type RewardEntity struct {
	Epoch        uint64
	ConsumerAddr string
	ConsumerKey  string
	SessionId    uint64
	Proof        *pairingtypes.RelaySession
}

func (rs *RewardStore) Save(ctx context.Context, consumerAddr string, consumerKey string, proof *pairingtypes.RelaySession) (bool, error) {
	key := assembleKey(uint64(proof.Epoch), consumerAddr, proof.SessionId, consumerKey)

	re := &RewardEntity{
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

	rs.db.Save(ctx, key, buf)

	return true, nil
}

func (rs *RewardStore) FindOne(
	ctx context.Context,
	epoch uint64,
	consumerAddr string,
	consumerKey string,
	sessionId uint64,
) (*pairingtypes.RelaySession, error) {
	key := assembleKey(epoch, consumerAddr, sessionId, consumerKey)

	rawReward, err := rs.db.FindOne(ctx, key)
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

func (rs *RewardStore) FindAll(ctx context.Context) (map[uint64]*EpochRewards, error) {
	rawRewards, err := rs.db.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]*EpochRewards)
	for key, rewards := range rawRewards {
		epoch, consumerAddr, sessionId, consumerKey := keyParts(key)

		re := RewardEntity{}
		err := json.Unmarshal(rewards, &re)
		if err != nil {
			utils.LavaFormatError("failed to decode proof: %s", err)
			continue
		}

		epochRewards, ok := result[epoch]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{sessionId: re.Proof}
			consumerRewards := map[string]*ConsumerRewards{consumerKey: &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs}}
			result[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewards}
			continue
		}

		consumerRewards, ok := epochRewards.consumerRewards[consumerKey]
		if !ok {
			proofs := map[uint64]*pairingtypes.RelaySession{sessionId: re.Proof}
			epochRewards.consumerRewards[consumerKey] = &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs}
			continue
		}

		_, ok = consumerRewards.proofs[sessionId]
		if !ok {
			consumerRewards.proofs[sessionId] = re.Proof
			continue
		}
	}
	return result, nil
}

func (rs *RewardStore) DeleteClaimedRewards(ctx context.Context, claimedRewards []*pairingtypes.RelaySession) error {
	var deletedPrefixes []string
	for _, claimedReward := range claimedRewards {
		consumer, err := sigs.ExtractSignerAddress(claimedReward)
		if err != nil {
			utils.LavaFormatError("failed to extract consumer address: %s", err)
			continue
		}

		prefix := assembleKey(uint64(claimedReward.Epoch), consumer.String(), claimedReward.SessionId, "")
		if slices.Contains(deletedPrefixes, prefix) {
			continue
		}

		err = rs.db.DeletePrefix(ctx, prefix)
		if err != nil {
			utils.LavaFormatError("failed to delete rewards: %s", err)
			continue
		}

		deletedPrefixes = append(deletedPrefixes, prefix)
	}

	return nil
}

func NewRewardStore(db DB) *RewardStore {
	return &RewardStore{
		db: db,
	}
}

func assembleKey(epoch uint64, consumerAddr string, sessionId uint64, consumerKey string) string {
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

func keyParts(key string) (epoch uint64, consumerAddr string, sessionId uint64, consumerKey string) {
	parts := strings.Split(key, keySeparator)

	epoch, _ = strconv.ParseUint(parts[0], 10, 64)
	consumerAddr = parts[1]
	sessionId, _ = strconv.ParseUint(parts[2], 10, 64)
	consumerKey = parts[3]

	return
}
