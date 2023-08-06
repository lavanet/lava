package rewardserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/rand"
	terderminttypes "github.com/tendermint/tendermint/abci/types"
)

const (
	RewardServerStorageFlagName = "reward-server-storage"
	DefaultRewardServerStorage  = ".storage/rewardserver"
	RewardTTLFlagName           = "reward-ttl"
	DefaultRewardTTL            = 24 * 60 * time.Minute
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

func (pr *PaymentRequest) String() string {
	return fmt.Sprintf("cu: %d, BlockHeightDeadline: %d, Amount:%s, Client:%s, UniqueIdentifier:%d, Description:%s, chainID:%s",
		pr.CU, pr.BlockHeightDeadline, pr.Amount.String(), pr.Client.String(), pr.UniqueIdentifier, pr.Description, pr.ChainID)
}

type ConsumerRewards struct {
	epoch    uint64
	consumer string
	proofs   map[uint64]*pairingtypes.RelaySession // key is sessionID
}

func (csrw *ConsumerRewards) PrepareRewardsForClaim() (retProofs []*pairingtypes.RelaySession, errRet error) {
	for _, proof := range csrw.proofs {
		retProofs = append(retProofs, proof)
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
	serverID         uint64
	expectedPayments []PaymentRequest
	totalCUServiced  uint64
	totalCUPaid      uint64
	providerMetrics  *metrics.ProviderMetricsManager
	rewardDB         *RewardDB
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string) error
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
}

func (rws *RewardServer) SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr string, apiInterface string) (existingCU uint64, updatedWithProof bool) {
	consumerRewardsKey := getKeyForConsumerRewards(proof.SpecId, apiInterface, consumerAddr)

	savedProof, saved, err := rws.rewardDB.Save(consumerAddr, consumerRewardsKey, proof)
	if err != nil {
		utils.LavaFormatError("failed saving proof", err, utils.Attribute{Key: "proof", Value: proof})
		return 0, false
	}

	if saved {
		return 0, true
	}

	return savedProof.CuSum, false
}

func (rws *RewardServer) UpdateEpoch(epoch uint64) {
	ctx := context.Background()
	_ = rws.sendRewardsClaim(ctx, epoch)
	_, _ = rws.identifyMissingPayments(ctx)
}

func (rws *RewardServer) sendRewardsClaim(ctx context.Context, epoch uint64) error {
	rewardsToClaim, err := rws.gatherRewardsForClaim(ctx, epoch)
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
		err = rws.rewardsTxSender.TxRelayPayment(ctx, rewardsToClaim, strconv.FormatUint(rws.serverID, 10))
		if err != nil {
			return utils.LavaFormatError("failed sending rewards claim", err)
		}

		err = rws.rewardDB.DeleteClaimedRewards(rewardsToClaim)
		if err != nil {
			utils.LavaFormatWarning("failed deleting claimed rewards", err)
		}

		utils.LavaFormatDebug("sent rewards claim", utils.Attribute{Key: "rewardsToClaim", Value: len(rewardsToClaim)})
	} else {
		utils.LavaFormatDebug("no rewards to claim")
	}
	return nil
}

func (rws *RewardServer) identifyMissingPayments(ctx context.Context) (missingPayments bool, err error) {
	lastBlockInMemory, err := rws.rewardsTxSender.EarliestBlockInMemory(ctx)
	if err != nil {
		return false, err
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

func (rws *RewardServer) gatherRewardsForClaim(ctx context.Context, currentEpoch uint64) (rewardsForClaim []*pairingtypes.RelaySession, errRet error) {
	blockDistanceForEpochValidity, err := rws.rewardsTxSender.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("gatherRewardsForClaim failed to GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment", err)
	}

	earliestSavedEpoch, err := rws.rewardsTxSender.EarliestBlockInMemory(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("gatherRewardsForClaim failed to GetEpochsToSave", err)
	}

	if blockDistanceForEpochValidity > currentEpoch {
		return nil, utils.LavaFormatWarning("gatherRewardsForClaim current epoch is too low to claim rewards", nil, utils.Attribute{Key: "current epoch", Value: currentEpoch})
	}
	activeEpochThreshold := currentEpoch - blockDistanceForEpochValidity

	rewards, err := rws.rewardDB.FindAll()
	if err != nil {
		return nil, utils.LavaFormatError("gatherRewardsForClaim failed to FindAll", err)
	}

	for epoch, epochRewards := range rewards {
		if lavasession.IsEpochValidForUse(epoch, activeEpochThreshold) {
			// Epoch is still active so we don't claim the rewards yet.
			continue
		}

		if epoch < earliestSavedEpoch {
			err := rws.rewardDB.DeleteEpochRewards(epoch)
			if err != nil {
				utils.LavaFormatWarning("failed deleting epoch", err, utils.Attribute{Key: "epoch", Value: epoch})
			}

			// Epoch is too old, we can't claim the rewards anymore.
			continue
		}

		for _, consumerRewards := range epochRewards.consumerRewards {
			claimables, err := consumerRewards.PrepareRewardsForClaim()
			if err != nil {
				// can't claim this now
				continue
			}
			rewardsForClaim = append(rewardsForClaim, claimables...)
		}
	}
	return rewardsForClaim, errRet
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
		go rws.providerMetrics.AddPayment(payment.ChainID, payment.CU)
		removedPayment := rws.RemoveExpectedPayment(payment.CU, payment.Client, payment.BlockHeightDeadline, payment.UniqueIdentifier, payment.ChainID)
		if !removedPayment {
			utils.LavaFormatWarning("tried removing payment that wasn't expected", nil, utils.Attribute{Key: "payment", Value: payment})
		}
	}
}

func NewRewardServer(rewardsTxSender RewardsTxSender, providerMetrics *metrics.ProviderMetricsManager, rewardDB *RewardDB) *RewardServer {
	rws := &RewardServer{totalCUServiced: 0, totalCUPaid: 0}
	rws.serverID = uint64(rand.Int63())
	rws.rewardsTxSender = rewardsTxSender
	rws.expectedPayments = []PaymentRequest{}
	rws.providerMetrics = providerMetrics
	rws.rewardDB = rewardDB

	return rws
}

func BuildPaymentFromRelayPaymentEvent(event terderminttypes.Event, block int64) ([]*PaymentRequest, error) {
	type mapCont struct {
		attributes map[string]string
		index      int
	}
	attributesList := []*mapCont{}
	appendToAttributeList := func(idx int, key string, value string) {
		var mapContToChange *mapCont
		for _, mapCont := range attributesList {
			if mapCont.index != idx {
				continue
			}
			mapContToChange = mapCont
			break
		}
		if mapContToChange == nil {
			mapContToChange = &mapCont{attributes: map[string]string{}, index: idx}
			attributesList = append(attributesList, mapContToChange)
		}
		mapContToChange.attributes[key] = value
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
			if index < 0 {
				utils.LavaFormatError("failed building PaymentRequest from relay_payment event, index returned unreasonable value", nil, utils.Attribute{Key: "index", Value: index})
			}
		}
		appendToAttributeList(index, attrKey, string(attribute.Value))
	}
	payments := []*PaymentRequest{}
	for idx, mapCont := range attributesList {
		attributes := mapCont.attributes
		chainID, ok := attributes["chainID"]
		if !ok {
			errStringAllAttrs := ""
			for _, mapCont := range attributesList {
				errStringAllAttrs += fmt.Sprintf("%#v,", *mapCont)
			}
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event  missing field chainID", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx}, utils.Attribute{Key: "attributesList", Value: errStringAllAttrs})
		}
		mint, ok := attributes["Mint"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field Mint", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
		}
		mintedCoins, err := sdk.ParseCoinNormalized(mint)
		if err != nil {
			return nil, err
		}
		cu_str, ok := attributes["CU"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field CU", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
		}
		cu, err := strconv.ParseUint(cu_str, 10, 64)
		if err != nil {
			return nil, err
		}
		consumer, ok := attributes["client"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field client", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
		}
		consumerAddr, err := sdk.AccAddressFromBech32(consumer)
		if err != nil {
			return nil, err
		}

		uniqueIdentifier, ok := attributes["uniqueIdentifier"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field uniqueIdentifier", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
		}
		uniqueID, err := strconv.ParseUint(uniqueIdentifier, 10, 64)
		if err != nil {
			return nil, err
		}
		description, ok := attributes["descriptionString"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field descriptionString", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
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
