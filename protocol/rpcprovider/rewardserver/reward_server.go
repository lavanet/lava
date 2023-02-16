package rewardserver

import (
	"context"
	"strconv"
	"sync"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const (
	StaleEpochDistance = 2
)

type ConsumerRewards struct {
	epoch                 uint64
	consumer              string
	proofs                map[uint64]*pairingtypes.RelayRequest // key is sessionID
	dataReliabilityProofs []*pairingtypes.VRFData
}

func (csrw *ConsumerRewards) PrepareRewardsForClaim() (retProofs []*pairingtypes.RelayRequest, errRet error) {
	for _, proof := range csrw.proofs {
		retProofs = append(retProofs, proof)
	}
	dataReliabilityProofs := len(csrw.dataReliabilityProofs)
	if len(retProofs) > 0 && dataReliabilityProofs > 0 {
		for idx := range retProofs {
			if idx > dataReliabilityProofs-1 {
				break
			}
			retProofs[idx].DataReliability = csrw.dataReliabilityProofs[idx]
		}
	}
	return
}

type EpochRewards struct {
	epoch           uint64
	consumerRewards map[string]*ConsumerRewards // key is consumer
}

type RewardServer struct {
	rewardsTxSender RewardsTxSender
	lock            sync.RWMutex
	rewards         map[uint64]*EpochRewards
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelayRequest)
	GetEpochSize(ctx context.Context) (uint64, error)
}

func (rws *RewardServer) SendNewProof(ctx context.Context, proof *pairingtypes.RelayRequest, epoch uint64, consumerAddr string) (existingCU uint64, updatedWithProof bool) {
	rws.lock.Lock() // assuming 99% of the time we will need to write the new entry so there's no use in doing the read lock first to check stuff
	defer rws.lock.Unlock()
	epochRewards, ok := rws.rewards[epoch]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelayRequest{proof.SessionId: proof}
		consumerRewardsMap := map[string]*ConsumerRewards{consumerAddr: {epoch: epoch, consumer: consumerAddr, proofs: proofs, dataReliabilityProofs: []*pairingtypes.VRFData{}}}
		rws.rewards[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewardsMap}
		return 0, true
	}
	consumerRewards, ok := epochRewards.consumerRewards[consumerAddr]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelayRequest{proof.SessionId: proof}
		consumerRewards := &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs, dataReliabilityProofs: []*pairingtypes.VRFData{}}
		epochRewards.consumerRewards[consumerAddr] = consumerRewards
		return 0, true
	}
	relayProof, ok := consumerRewards.proofs[proof.SessionId]
	if !ok {
		consumerRewards.proofs[proof.SessionId] = proof
		return 0, true
	}
	cuSumStored := relayProof.CuSum
	if cuSumStored >= proof.CuSum {
		return cuSumStored, false
	}
	consumerRewards.proofs[proof.SessionId] = proof
	return 0, true
}

func (rws *RewardServer) SendNewDataReliabilityProof(ctx context.Context, dataReliability *pairingtypes.VRFData, epoch uint64, consumerAddr string) (updatedWithProof bool) {
	rws.lock.Lock() // assuming 99% of the time we will need to write the new entry so there's no use in doing the read lock first to check stuff
	defer rws.lock.Unlock()
	epochRewards, ok := rws.rewards[epoch]
	if !ok {
		consumerRewardsMap := map[string]*ConsumerRewards{consumerAddr: {epoch: epoch, consumer: consumerAddr, proofs: map[uint64]*pairingtypes.RelayRequest{}, dataReliabilityProofs: []*pairingtypes.VRFData{dataReliability}}}
		rws.rewards[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewardsMap}
		return true
	}
	consumerRewards, ok := epochRewards.consumerRewards[consumerAddr]
	if !ok {
		consumerRewards := &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: map[uint64]*pairingtypes.RelayRequest{}, dataReliabilityProofs: []*pairingtypes.VRFData{dataReliability}}
		epochRewards.consumerRewards[consumerAddr] = consumerRewards
		return true
	}
	if len(consumerRewards.dataReliabilityProofs) == 0 {
		consumerRewards.dataReliabilityProofs = []*pairingtypes.VRFData{dataReliability}
		return true
	}
	return false // currently support only one per epoch
}

func (rws *RewardServer) UpdateEpoch(epoch uint64) {
	ctx := context.Background()
	rws.gatherRewardsForClaim(ctx, epoch)
}

func (rws *RewardServer) gatherRewardsForClaim(ctx context.Context, current_epoch uint64) (rewardsForClaim []*pairingtypes.RelayRequest, errRet error) {
	rws.lock.Lock()
	defer rws.lock.Unlock()
	epochSize, err := rws.rewardsTxSender.GetEpochSize(ctx)
	if err != nil {
		return nil, err
	}
	if epochSize*StaleEpochDistance > current_epoch {
		return nil, utils.LavaFormatError("current epoch too low", nil, &map[string]string{"current epoch": strconv.FormatUint(current_epoch, 10)})
	}
	target_epoch_to_claim_rewards := current_epoch - epochSize*StaleEpochDistance
	for epoch, epochRewards := range rws.rewards {
		if epoch >= uint64(target_epoch_to_claim_rewards) {
			continue
		}

		for consumerAddr, rewards := range epochRewards.consumerRewards {
			claimables, err := rewards.PrepareRewardsForClaim()
			if err != nil {
				// can't claim this now
				continue
			}
			rewardsForClaim = append(rewardsForClaim, claimables...)
			delete(epochRewards.consumerRewards, consumerAddr)
		}
		if len(epochRewards.consumerRewards) == 0 {
			delete(rws.rewards, epoch)
		}
	}
	return
}

func (rws *RewardServer) SubscribeStarted(consumer string, epoch uint64, subscribeID string) {
	// hold off reward claims for subscription while this is still active
}

func (rws *RewardServer) SubscribeEnded(consumer string, epoch uint64, subscribeID string) {
	// can collect now
}

func NewRewardServer(rewardsTxSender RewardsTxSender) *RewardServer {
	//
	rws := &RewardServer{}
	rws.rewardsTxSender = rewardsTxSender
	// TODO: load this from persistency
	rws.rewards = map[uint64]*EpochRewards{}
	return rws
}
