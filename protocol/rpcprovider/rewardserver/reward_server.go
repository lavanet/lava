package rewardserver

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
)

type PaymentRequest struct {
	CU                  uint64
	BlockHeightDeadline int64
	Amount              sdk.Coin
	Client              sdk.AccAddress
	UniqueIdentifier    uint64
	Description         string
	ChainID             string
}

type ConsumerRewards struct {
	epoch                 uint64
	consumer              string
	proofs                map[uint64]*pairingtypes.RelaySession // key is sessionID
	dataReliabilityProofs []*pairingtypes.VRFData
}

func (csrw *ConsumerRewards) PrepareRewardsForClaim() (retProofs []*pairingtypes.RelaySession, retVRFs []*pairingtypes.VRFData, errRet error) {
	for _, proof := range csrw.proofs {
		retProofs = append(retProofs, proof)
	}
	// add data reliability proofs
	dataReliabilityProofs := len(csrw.dataReliabilityProofs)
	if len(retProofs) > 0 && dataReliabilityProofs > 0 {
		for idx := range retProofs {
			if idx > dataReliabilityProofs-1 {
				break
			}
			retVRFs = append(retVRFs, csrw.dataReliabilityProofs[idx])
		}
	}
	return
}

type EpochRewards struct {
	epoch           uint64
	consumerRewards map[string]*ConsumerRewards // key is consumer
}

type RewardServer struct {
	rewardsTxSender  RewardsTxSender
	lock             sync.RWMutex
	rewards          map[uint64]*EpochRewards
	serverID         uint64
	expectedPayments []PaymentRequest
	totalCUServiced  uint64
	totalCUPaid      uint64
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, dataReliabilityProofs []*pairingtypes.VRFData, description string) error
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
}

func (rws *RewardServer) SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr string, apiInterface string) (existingCU uint64, updatedWithProof bool) {
	rws.lock.Lock() // assuming 99% of the time we will need to write the new entry so there's no use in doing the read lock first to check stuff
	defer rws.lock.Unlock()
	consumerRewardsKey := getKeyForConsumerRewards(proof.SpecId, apiInterface, consumerAddr)
	epochRewards, ok := rws.rewards[epoch]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelaySession{proof.SessionId: proof}
		consumerRewardsMap := map[string]*ConsumerRewards{consumerRewardsKey: {epoch: epoch, consumer: consumerAddr, proofs: proofs, dataReliabilityProofs: []*pairingtypes.VRFData{}}}
		rws.rewards[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewardsMap}
		return 0, true
	}
	consumerRewards, ok := epochRewards.consumerRewards[consumerRewardsKey]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelaySession{proof.SessionId: proof}
		consumerRewards := &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs, dataReliabilityProofs: []*pairingtypes.VRFData{}}
		epochRewards.consumerRewards[consumerRewardsKey] = consumerRewards
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

func (rws *RewardServer) SendNewDataReliabilityProof(ctx context.Context, dataReliability *pairingtypes.VRFData, epoch uint64, consumerAddr string, specId string, apiInterface string) (updatedWithProof bool) {
	rws.lock.Lock() // assuming 99% of the time we will need to write the new entry so there's no use in doing the read lock first to check stuff
	defer rws.lock.Unlock()
	consumerRewardsKey := getKeyForConsumerRewards(specId, apiInterface, consumerAddr)
	epochRewards, ok := rws.rewards[epoch]
	if !ok {
		consumerRewardsMap := map[string]*ConsumerRewards{(consumerRewardsKey): {epoch: epoch, consumer: consumerAddr, proofs: map[uint64]*pairingtypes.RelaySession{}, dataReliabilityProofs: []*pairingtypes.VRFData{dataReliability}}}
		rws.rewards[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewardsMap}
		return true
	}
	consumerRewards, ok := epochRewards.consumerRewards[consumerRewardsKey]
	if !ok {
		consumerRewards := &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: map[uint64]*pairingtypes.RelaySession{}, dataReliabilityProofs: []*pairingtypes.VRFData{dataReliability}}
		epochRewards.consumerRewards[consumerRewardsKey] = consumerRewards
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
	_ = rws.sendRewardsClaim(ctx, epoch)
	_, _ = rws.identifyMissingPayments(ctx)
}

func (rws *RewardServer) sendRewardsClaim(ctx context.Context, epoch uint64) error {
	rewardsToClaim, dataReliabilityProofs, err := rws.gatherRewardsForClaim(ctx, epoch)
	if err != nil {
		return err
	}
	for _, relay := range rewardsToClaim {
		consumerAddr, err := sigs.ExtractSignerAddress(relay)
		if err != nil {
			utils.LavaFormatError("invalid consumer address extraction from relay", err, utils.Attribute{Key: "relay", Value: relay})
			continue
		}
		expectedPay := PaymentRequest{ChainID: relay.SpecId, CU: relay.CuSum, BlockHeightDeadline: relay.Epoch, Amount: sdk.Coin{}, Client: consumerAddr, UniqueIdentifier: relay.SessionId, Description: strconv.FormatUint(rws.serverID, 10)}
		rws.addExpectedPayment(expectedPay)
		rws.updateCUServiced(relay.CuSum)
	}
	if len(rewardsToClaim) > 0 {
		err = rws.rewardsTxSender.TxRelayPayment(ctx, rewardsToClaim, dataReliabilityProofs, strconv.FormatUint(rws.serverID, 10))
		if err != nil {
			return utils.LavaFormatError("failed sending rewards claim", err)
		}
	} else {
		utils.LavaFormatDebug("no rewards to claim")
	}
	return nil
}

func (rws *RewardServer) identifyMissingPayments(ctx context.Context) (missingPayments bool, err error) {
	lastBlockInMemory, err := rws.rewardsTxSender.EarliestBlockInMemory(ctx)
	if err != nil {
		return
	}
	rws.lock.Lock()
	defer rws.lock.Unlock()

	var updatedExpectedPayments []PaymentRequest

	for idx, expectedPay := range rws.expectedPayments {
		// Exclude and log missing payments
		if uint64(expectedPay.BlockHeightDeadline) < lastBlockInMemory {
			utils.LavaFormatError("Identified Missing Payment", nil,
				utils.Attribute{Key: "expectedPay.CU", Value: expectedPay.CU},
				utils.Attribute{Key: "expectedPay.BlockHeightDeadline", Value: expectedPay.BlockHeightDeadline},
				utils.Attribute{Key: "lastBlockInMemory", Value: lastBlockInMemory},
			)
			missingPayments = true
			continue
		}

		// Include others
		updatedExpectedPayments = append(updatedExpectedPayments, rws.expectedPayments[idx])
	}

	// Update expectedPayment
	rws.expectedPayments = updatedExpectedPayments

	// can be modified in this race window, so we double-check

	utils.LavaFormatInfo("Service report",
		utils.Attribute{Key: "total CU serviced", Value: rws.cUServiced()},
		utils.Attribute{Key: "total CU that got paid", Value: rws.paidCU()},
	)
	return missingPayments, err
}

func (rws *RewardServer) cUServiced() uint64 {
	return atomic.LoadUint64(&rws.totalCUServiced)
}

func (rws *RewardServer) paidCU() uint64 {
	return atomic.LoadUint64(&rws.totalCUPaid)
}

func (rws *RewardServer) addExpectedPayment(expectedPay PaymentRequest) {
	rws.lock.Lock() // this can be a separate lock, if we have performance issues
	defer rws.lock.Unlock()
	rws.expectedPayments = append(rws.expectedPayments, expectedPay)
}

func (rws *RewardServer) RemoveExpectedPayment(paidCUToFInd uint64, expectedClient sdk.AccAddress, blockHeight int64, uniqueID uint64, chainID string) bool {
	rws.lock.Lock() // this can be a separate lock, if we have performance issues
	defer rws.lock.Unlock()
	for idx, expectedPayment := range rws.expectedPayments {
		// TODO: make sure the payment is not too far from expected block, expectedPayment.BlockHeightDeadline == blockHeight
		if expectedPayment.CU == paidCUToFInd && expectedPayment.Client.Equals(expectedClient) && uniqueID == expectedPayment.UniqueIdentifier && chainID == expectedPayment.ChainID {
			// found payment for expected payment
			rws.expectedPayments[idx] = rws.expectedPayments[len(rws.expectedPayments)-1] // replace the element at delete index with the last one
			rws.expectedPayments = rws.expectedPayments[:len(rws.expectedPayments)-1]     // remove last element
			return true
		}
	}
	return false
}

func (rws *RewardServer) gatherRewardsForClaim(ctx context.Context, currentEpoch uint64) (rewardsForClaim []*pairingtypes.RelaySession, dataReliabilityProofs []*pairingtypes.VRFData, errRet error) {
	rws.lock.Lock()
	defer rws.lock.Unlock()
	blockDistanceForEpochValidity, err := rws.rewardsTxSender.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
	if err != nil {
		return nil, nil, utils.LavaFormatError("gatherRewardsForClaim failed to GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment", err)
	}

	if blockDistanceForEpochValidity > currentEpoch {
		return nil, nil, utils.LavaFormatWarning("gatherRewardsForClaim current epoch is too low to claim rewards", nil, utils.Attribute{Key: "current epoch", Value: currentEpoch})
	}
	activeEpochThreshold := currentEpoch - blockDistanceForEpochValidity
	for epoch, epochRewards := range rws.rewards {
		if lavasession.IsEpochValidForUse(epoch, activeEpochThreshold) {
			// Epoch is still active so we don't claim the rewards yet.
			continue
		}

		for consumerAddr, rewards := range epochRewards.consumerRewards {
			claimables, dataReliabilities, err := rewards.PrepareRewardsForClaim()
			if err != nil {
				// can't claim this now
				continue
			}
			rewardsForClaim = append(rewardsForClaim, claimables...)
			dataReliabilityProofs = append(dataReliabilityProofs, dataReliabilities...)
			delete(epochRewards.consumerRewards, consumerAddr)
		}
		if len(epochRewards.consumerRewards) == 0 {
			delete(rws.rewards, epoch)
		}
	}
	return rewardsForClaim, dataReliabilityProofs, errRet
}

func (rws *RewardServer) SubscribeStarted(consumer string, epoch uint64, subscribeID string) {
	// TODO: hold off reward claims for subscription while this is still active
}

func (rws *RewardServer) SubscribeEnded(consumer string, epoch uint64, subscribeID string) {
	// TODO: can collect now
}

func (rws *RewardServer) updateCUServiced(cu uint64) {
	rws.lock.Lock()
	defer rws.lock.Unlock()
	currentCU := atomic.LoadUint64(&rws.totalCUServiced)
	atomic.StoreUint64(&rws.totalCUServiced, currentCU+cu)
}

func (rws *RewardServer) updateCUPaid(cu uint64) {
	rws.lock.Lock()
	defer rws.lock.Unlock()
	currentCU := atomic.LoadUint64(&rws.totalCUPaid)
	atomic.StoreUint64(&rws.totalCUPaid, currentCU+cu)
}

func (rws *RewardServer) Description() string {
	return strconv.FormatUint(rws.serverID, 10)
}

func (rws *RewardServer) PaymentHandler(payment *PaymentRequest) {
	serverID, err := strconv.ParseUint(payment.Description, 10, 64)
	if err != nil {
		utils.LavaFormatError("failed parsing description as server id", err, utils.Attribute{Key: "description", Value: payment.Description})
		return
	}
	if serverID == rws.serverID {
		rws.updateCUPaid(payment.CU)
		removedPayment := rws.RemoveExpectedPayment(payment.CU, payment.Client, payment.BlockHeightDeadline, payment.UniqueIdentifier, payment.ChainID)
		if !removedPayment {
			utils.LavaFormatWarning("tried removing payment that wasn;t expected", nil, utils.Attribute{Key: "payment", Value: payment})
		}
	}
}

func NewRewardServer(rewardsTxSender RewardsTxSender) *RewardServer {
	//
	rws := &RewardServer{totalCUServiced: 0, totalCUPaid: 0}
	rws.serverID = uint64(rand.Int63())
	rws.rewardsTxSender = rewardsTxSender
	rws.expectedPayments = []PaymentRequest{}
	// TODO: load this from persistency
	rws.rewards = map[uint64]*EpochRewards{}
	return rws
}

func BuildPaymentFromRelayPaymentEvent(event terderminttypes.Event, block int64) ([]*PaymentRequest, error) {
	attributesList := []map[string]string{}
	appendToAttributeList := func(attributesList []map[string]string, idx int, key string, value string) {
		for len(attributesList) <= idx {
			attributesList = append(attributesList, map[string]string{})
		}
		attributesList[idx] = map[string]string{key: value}
	}
	for _, attribute := range event.Attributes {
		splittedAttrs := strings.SplitN(string(attribute.Key), ".", 2)
		attrKey := splittedAttrs[0]
		index := 0
		if len(splittedAttrs) > 1 {
			var err error
			index, err = strconv.Atoi(splittedAttrs[1])
			if err != nil {
				utils.LavaFormatError("failed building PaymentRequest from relay_payment event, could not parse index after a .", nil, utils.Attribute{Key: "attribute", Value: attribute.Key})
			}
			if index < 0 || index > len(event.Attributes) {
				utils.LavaFormatError("failed building PaymentRequest from relay_payment event, index returned unreasonable value", nil, utils.Attribute{Key: "index", Value: index})
			}
		}
		appendToAttributeList(attributesList, index, attrKey, string(attribute.Value))
	}
	payments := []*PaymentRequest{}
	for _, attributes := range attributesList {
		chainID, ok := attributes["chainID"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		mint, ok := attributes["Mint"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		mintedCoins, err := sdk.ParseCoinNormalized(mint)
		if err != nil {
			return nil, err
		}
		cu_str, ok := attributes["CU"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		cu, err := strconv.ParseUint(cu_str, 10, 64)
		if err != nil {
			return nil, err
		}
		consumer, ok := attributes["client"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		consumerAddr, err := sdk.AccAddressFromBech32(consumer)
		if err != nil {
			return nil, err
		}

		uniqueIdentifier, ok := attributes["uniqueIdentifier"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		uniqueID, err := strconv.ParseUint(uniqueIdentifier, 10, 64)
		if err != nil {
			return nil, err
		}
		description, ok := attributes["descriptionString"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event", nil, utils.Attribute{Key: "attributes", Value: attributes})
		}
		payment := &PaymentRequest{
			CU:                  cu,
			BlockHeightDeadline: block,
			Amount:              mintedCoins,
			Client:              consumerAddr,
			Description:         description,
			UniqueIdentifier:    uniqueID,
			ChainID:             chainID,
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

func getKeyForConsumerRewards(specId string, apiInterface string, consumerAddress string) string {
	return specId + apiInterface + consumerAddress
}
