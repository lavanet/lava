package rewardserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	terderminttypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/rand"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

const (
	RewardServerStorageFlagName         = "reward-server-storage"
	DefaultRewardServerStorage          = ".storage/rewardserver"
	RewardTTLFlagName                   = "reward-ttl"
	DefaultRewardTTL                    = 24 * 60 * time.Minute
	MaxDBSaveRetries                    = 10
	RewardsSnapshotThresholdFlagName    = "proofs-snapshot-threshold"
	DefaultRewardsSnapshotThreshold     = 1000
	RewardsSnapshotTimeoutSecFlagName   = "proofs-snapshot-timeout-sec"
	DefaultRewardsSnapshotTimeoutSec    = 30
	MaxPaymentRequestsRetiresForSession = 3
	RewardServerMaxRelayRetires         = 3
	splitRewardsIntoChunksSize          = 500 // if the reward array is larger than this it will split it into chunks and send multiple requests instead of a huge one
)

type PaymentRequest struct {
	CU                  uint64
	BlockHeightDeadline int64
	PaymentEpoch        uint64
	Amount              sdk.Coin
	Client              sdk.AccAddress
	UniqueIdentifier    uint64
	Description         string
	ChainID             string
	ConsumerRewardsKey  string
}

func (pr *PaymentRequest) String() string {
	return fmt.Sprintf("cu: %d, BlockHeightDeadline: %d, PaymentEpoch: %d, Amount:%s, Client:%s, UniqueIdentifier:%d, Description:%s, chainID:%s, ConsumerRewardsKey:%s",
		pr.CU, pr.BlockHeightDeadline, pr.PaymentEpoch, pr.Amount.String(), pr.Client.String(), pr.UniqueIdentifier, pr.Description, pr.ChainID, pr.ConsumerRewardsKey)
}

type ConsumerRewards struct {
	epoch    uint64
	consumer string
	proofs   map[uint64]*pairingtypes.RelaySession // key is sessionID
}

func (csrw *ConsumerRewards) PrepareRewardsForClaim() (retProofs []*pairingtypes.RelaySession, errRet error) {
	utils.LavaFormatDebug("Adding reward ids for claim", utils.LogAttr("number_of_proofs", len(csrw.proofs)))
	for _, proof := range csrw.proofs {
		retProofs = append(retProofs, proof)
	}
	return
}

type EpochRewards struct {
	epoch           uint64
	consumerRewards map[string]*ConsumerRewards // key is consumerRewardsKey
}

type RelaySessionsToRetryAttempts struct {
	relaySession                *pairingtypes.RelaySession
	paymentRequestRetryAttempts uint64
}

type PaymentConfiguration struct {
	relaySessionChunks       [][]*pairingtypes.RelaySession // small chunks of relay session to request payments for
	shouldAddExpectedPayment bool
}

type RewardServer struct {
	rewardsTxSender                RewardsTxSender
	lock                           sync.RWMutex
	serverID                       uint64
	expectedPayments               []PaymentRequest
	totalCUServiced                uint64
	totalCUPaid                    uint64
	providerMetrics                *metrics.ProviderMetricsManager
	rewards                        map[uint64]*EpochRewards
	rewardDB                       *RewardDB
	rewardStoragePath              string
	rewardsSnapshotThreshold       uint64
	rewardsSnapshotTimeoutDuration time.Duration
	rewardsSnapshotTimer           *time.Timer
	rewardsSnapshotThresholdCh     chan struct{}
	failedRewardsPaymentRequests   map[uint64]*RelaySessionsToRetryAttempts // key is SessionId
	chainTrackerSpecsInf           ChainTrackerSpecsInf
}

type RewardsTxSender interface {
	TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error
	GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error)
	EarliestBlockInMemory(ctx context.Context) (uint64, error)
	GetEpochSize(ctx context.Context) (uint64, error)
	LatestBlock() int64
	GetAverageBlockTime() time.Duration
}

type ChainTrackerSpecsInf interface {
	GetLatestBlockNumForSpec(specID string) int64
}

func (rws *RewardServer) SendNewProof(ctx context.Context, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr string, apiInterface string) (existingCU uint64, updatedWithProof bool) {
	consumerRewardsKey := getKeyForConsumerRewards(proof.SpecId, consumerAddr)

	existingCU, updatedWithProof = rws.saveProofInMemory(ctx, consumerRewardsKey, proof, epoch, consumerAddr)

	if proof.RelayNum%rws.rewardsSnapshotThreshold == 0 {
		rws.rewardsSnapshotThresholdCh <- struct{}{}
	}

	return existingCU, updatedWithProof
}

func (rws *RewardServer) saveProofInMemory(ctx context.Context, consumerRewardsKey string, proof *pairingtypes.RelaySession, epoch uint64, consumerAddr string) (existingCU uint64, updatedWithProof bool) {
	rws.lock.Lock() // assuming 99% of the time we will need to write the new entry so there's no use in doing the read lock first to check stuff
	defer rws.lock.Unlock()

	epochRewards, ok := rws.rewards[epoch]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelaySession{proof.SessionId: proof}
		consumerRewardsMap := map[string]*ConsumerRewards{consumerRewardsKey: {epoch: epoch, consumer: consumerAddr, proofs: proofs}}
		rws.rewards[epoch] = &EpochRewards{epoch: epoch, consumerRewards: consumerRewardsMap}
		return 0, true
	}

	consumerRewards, ok := epochRewards.consumerRewards[consumerRewardsKey]
	if !ok {
		proofs := map[uint64]*pairingtypes.RelaySession{proof.SessionId: proof}
		consumerRewards := &ConsumerRewards{epoch: epoch, consumer: consumerAddr, proofs: proofs}
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

	if relayProof.Badge != nil && proof.Badge == nil {
		proof.Badge = relayProof.Badge
	}

	consumerRewards.proofs[proof.SessionId] = proof
	return 0, true
}

func (rws *RewardServer) UpdateEpoch(epoch uint64) {
	go rws.runRewardServerEpochUpdate(epoch)
}

func (rws *RewardServer) runRewardServerEpochUpdate(epoch uint64) {
	ctx := context.Background()
	rws.AddRewardDelayForUnifiedRewardDistribution(ctx, epoch)
	rws.sendRewardsClaim(ctx, epoch)
	rws.identifyMissingPayments(ctx)
}

func (rws *RewardServer) getEpochSizeWithRetry(ctx context.Context) (epochSize uint64, err error) {
	for i := 0; i < RewardServerMaxRelayRetires; i++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, common.CommunicateWithLocalLavaNodeTimeout)
		epochSize, err = rws.rewardsTxSender.GetEpochSize(ctxWithTimeout)
		cancel()
		if err == nil {
			break
		}
		utils.LavaFormatDebug("failed getting epoch size, retrying...", utils.LogAttr("retry_#", i+1))
		time.Sleep(50 * time.Duration(i+1) * time.Millisecond)
	}

	return epochSize, err
}

func (rws *RewardServer) getEarliestBlockInMemoryWithRetry(ctx context.Context) (earliestBlock uint64, err error) {
	for i := 0; i < RewardServerMaxRelayRetires; i++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, common.CommunicateWithLocalLavaNodeTimeout)
		earliestBlock, err = rws.rewardsTxSender.EarliestBlockInMemory(ctxWithTimeout)
		cancel()
		if err == nil {
			break
		}
		utils.LavaFormatDebug("failed getting earliest block in memory, retrying...", utils.LogAttr("retry_#", i+1))
		time.Sleep(50 * time.Duration(i+1) * time.Millisecond)
	}

	return earliestBlock, err
}

func (rws *RewardServer) AddRewardDelayForUnifiedRewardDistribution(ctx context.Context, epochStart uint64) {
	epochSize, err := rws.getEpochSizeWithRetry(ctx)
	if err != nil {
		utils.LavaFormatError("Failed fetching epoch size in reward server delay, skipping delay", err)
		return
	}
	halfEpochSize := (epochSize / 2) + 1 // +1 for random to choose 0 to number
	chosenBlocksForRewardDelay := uint64(rand.Intn(int(halfEpochSize)))
	utils.LavaFormatDebug("Waiting for number of blocks to send the reward", utils.LogAttr("blocks_delay", chosenBlocksForRewardDelay))
	for {
		latestBlock := rws.rewardsTxSender.LatestBlock()
		if uint64(latestBlock)-epochStart >= chosenBlocksForRewardDelay {
			return
		}
		time.Sleep(rws.rewardsTxSender.GetAverageBlockTime()) // sleep the average block time, to wait for the next block.
	}
}

func (rws *RewardServer) sendRewardsClaim(ctx context.Context, epoch uint64) error {
	earliestSavedEpoch, err := rws.getEarliestBlockInMemoryWithRetry(ctx)
	if err != nil {
		return utils.LavaFormatError("sendRewardsClaim failed to get earliest block in memory", err)
	}

	// Handle Failed rewards claim with retry
	failedRewardRequestsToRetry := rws.gatherFailedRequestPaymentsToRetry(earliestSavedEpoch)
	if len(failedRewardRequestsToRetry) > 0 {
		utils.LavaFormatDebug("Found failed reward claims, retrying", utils.LogAttr("number_of_rewards", len((failedRewardRequestsToRetry))))
	}
	failedRewardsToClaimChunks := lavaslices.SplitGenericSliceIntoChunks(failedRewardRequestsToRetry, splitRewardsIntoChunksSize)
	failedRewardsLength := len(failedRewardsToClaimChunks)
	if failedRewardsLength > 1 {
		utils.LavaFormatDebug("Splitting Failed Reward claims into chunks", utils.LogAttr("chunk_size", splitRewardsIntoChunksSize), utils.LogAttr("number_of_chunks", failedRewardsLength))
	}

	// Handle new claims
	gatheredRewardsToClaim, err := rws.gatherRewardsForClaim(ctx, epoch, earliestSavedEpoch)
	if err != nil {
		return err
	}
	rewardsToClaimChunks := lavaslices.SplitGenericSliceIntoChunks(gatheredRewardsToClaim, splitRewardsIntoChunksSize)
	newRewardsLength := len(rewardsToClaimChunks)
	if newRewardsLength > 1 {
		utils.LavaFormatDebug("Splitting Reward claims into chunks", utils.LogAttr("chunk_size", splitRewardsIntoChunksSize), utils.LogAttr("number_of_chunks", newRewardsLength))
	}

	// payment chunk configurations
	paymentConfiguration := []*PaymentConfiguration{
		{ // adding the new rewards.
			relaySessionChunks:       rewardsToClaimChunks,
			shouldAddExpectedPayment: true,
		},
		{ // adding the failed rewards.
			relaySessionChunks:       failedRewardsToClaimChunks,
			shouldAddExpectedPayment: false,
		},
	}

	paymentWaitGroup := sync.WaitGroup{}
	paymentWaitGroup.Add(newRewardsLength + failedRewardsLength)

	// add expected pay and ask for rewards
	for _, paymentConfig := range paymentConfiguration {
		for _, rewardsToClaim := range paymentConfig.relaySessionChunks {
			if len(rewardsToClaim) == 0 {
				if paymentConfig.shouldAddExpectedPayment {
					utils.LavaFormatDebug("no new rewards to claim")
				} else {
					utils.LavaFormatDebug("no failed rewards to claim")
				}
				continue
			}
			go func(rewards []*pairingtypes.RelaySession, payment *PaymentConfiguration) { // send rewards asynchronously
				defer paymentWaitGroup.Done()
				specs := map[string]struct{}{}
				if payment.shouldAddExpectedPayment {
					for _, relay := range rewards {
						consumerAddr, err := sigs.ExtractSignerAddress(relay)
						if err != nil {
							utils.LavaFormatError("invalid consumer address extraction from relay", err, utils.Attribute{Key: "relay", Value: relay})
							continue
						}
						expectedPay := PaymentRequest{
							ChainID:             relay.SpecId,
							CU:                  relay.CuSum,
							BlockHeightDeadline: relay.Epoch,
							Amount:              sdk.Coin{},
							Client:              consumerAddr,
							UniqueIdentifier:    relay.SessionId,
							Description:         strconv.FormatUint(rws.serverID, 10),
							ConsumerRewardsKey:  getKeyForConsumerRewards(relay.SpecId, consumerAddr.String()),
						}
						rws.addExpectedPayment(expectedPay)
						rws.updateCUServiced(relay.CuSum)
						specs[relay.SpecId] = struct{}{}

						utils.LavaFormatTrace("Adding Payment for Spec",
							utils.LogAttr("spec", relay.SpecId),
							utils.LogAttr("Cu Sum", relay.CuSum),
							utils.LogAttr("epoch", relay.Epoch),
							utils.LogAttr("consumerAddr", consumerAddr),
							utils.LogAttr("number_of_relays_served", relay.RelayNum),
							utils.LogAttr("sessionId", relay.SessionId),
						)
					}
				} else { // just add the specs
					for _, relay := range failedRewardRequestsToRetry {
						utils.LavaFormatDebug("[sendRewardsClaim] retrying failed id", utils.LogAttr("id", relay.SessionId))
						specs[relay.SpecId] = struct{}{}
					}
				}
				err = rws.rewardsTxSender.TxRelayPayment(ctx, rewards, strconv.FormatUint(rws.serverID, 10), rws.latestBlockReports(specs))
				if err != nil {
					rws.updatePaymentRequestAttempt(rewards, false)
					utils.LavaFormatError("failed sending rewards claim", err)
					return
				}
				rws.updatePaymentRequestAttempt(rewards, true)
				utils.LavaFormatDebug("Sent rewards claim", utils.Attribute{Key: "number_of_relay_sessions_sent", Value: len(rewards)})
			}(rewardsToClaim, paymentConfig)
		}
	}
	utils.LavaFormatDebug("Waiting for all Payment groups to finish", utils.LogAttr("wait_group_size", newRewardsLength+failedRewardsLength))
	paymentWaitGroup.Wait()
	return nil
}

func (rws *RewardServer) identifyMissingPayments(ctx context.Context) (missingPayments bool, err error) {
	lastBlockInMemory, err := rws.getEarliestBlockInMemoryWithRetry(ctx)
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
	rws.expectedPayments = append(rws.expectedPayments, expectedPay)
	rws.lock.Unlock()
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

func (rws *RewardServer) gatherRewardsForClaim(ctx context.Context, currentEpoch uint64, earliestSavedEpoch uint64) (rewardsForClaim []*pairingtypes.RelaySession, errRet error) {
	blockDistanceForEpochValidity, err := rws.rewardsTxSender.GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("gatherRewardsForClaim failed to GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment", err)
	}

	if blockDistanceForEpochValidity > currentEpoch {
		return nil, utils.LavaFormatWarning("gatherRewardsForClaim current epoch is too low to claim rewards", nil, utils.Attribute{Key: "current epoch", Value: currentEpoch})
	}

	activeEpochThreshold := currentEpoch - blockDistanceForEpochValidity
	rws.lock.Lock()
	defer rws.lock.Unlock()
	for epoch, epochRewards := range rws.rewards {
		if epoch < earliestSavedEpoch {
			delete(rws.rewards, epoch)
			err := rws.rewardDB.DeleteEpochRewards(epoch)
			if err != nil {
				utils.LavaFormatWarning("gatherRewardsForClaim failed deleting epoch from rewardDB", err, utils.Attribute{Key: "epoch", Value: epoch})
			}
			// Epoch is too old, we can't claim the rewards anymore.
			continue
		}

		if lavasession.IsEpochValidForUse(epoch, activeEpochThreshold) {
			// Epoch is still active so we don't claim the rewards yet.
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
	return rewardsForClaim, errRet
}

func (rws *RewardServer) SubscribeStarted(consumer string, epoch uint64, subscribeID string) {
	// TODO: hold off reward claims for subscription while this is still active
}

func (rws *RewardServer) SubscribeEnded(consumer string, epoch uint64, subscribeID string) {
	// TODO: can collect now
}

func (rws *RewardServer) updateCUServiced(cu uint64) {
	atomic.AddUint64(&rws.totalCUServiced, cu)
}

func (rws *RewardServer) updateCUPaid(cu uint64) {
	atomic.AddUint64(&rws.totalCUPaid, cu)
}

func (rws *RewardServer) AddDataBase(specId string, providerPublicAddress string, shardID uint) {
	// the db itself doesn't need locks. as it self manages locks inside.
	// but opening a db can race. (NewLocalDB) so we lock this method.
	// Also, we construct the in-memory rewards from the DB, so that needs a lock as well
	rws.lock.Lock()
	defer rws.lock.Unlock()
	found := rws.rewardDB.DBExists(specId)
	if !found {
		rws.rewardDB.AddDB(NewLocalDB(rws.rewardStoragePath, providerPublicAddress, specId, shardID))
		rws.restoreRewardsFromDB(specId)
	}
}

func (rws *RewardServer) CloseAllDataBases() error {
	return rws.rewardDB.Close()
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
		err = rws.rewardDB.DeleteClaimedRewards(payment.PaymentEpoch, payment.Client.String(), payment.UniqueIdentifier, payment.ConsumerRewardsKey)
		if err != nil {
			utils.LavaFormatWarning("failed deleting claimed rewards", err)
		}
	}
}

func NewRewardServer(rewardsTxSender RewardsTxSender, providerMetrics *metrics.ProviderMetricsManager, rewardDB *RewardDB, rewardStoragePath string, rewardsSnapshotThreshold uint, rewardsSnapshotTimeoutSec uint, chainTrackerSpecsInf ChainTrackerSpecsInf) *RewardServer {
	rws := &RewardServer{totalCUServiced: 0, totalCUPaid: 0}
	rws.serverID = uint64(rand.Int63())
	rws.rewardsTxSender = rewardsTxSender
	rws.expectedPayments = []PaymentRequest{}
	rws.providerMetrics = providerMetrics
	rws.rewards = map[uint64]*EpochRewards{}
	rws.rewardDB = rewardDB
	rws.rewardStoragePath = rewardStoragePath
	rws.rewardsSnapshotThreshold = uint64(rewardsSnapshotThreshold)
	rws.rewardsSnapshotTimeoutDuration = time.Duration(rewardsSnapshotTimeoutSec) * time.Second
	rws.rewardsSnapshotTimer = time.NewTimer(rws.rewardsSnapshotTimeoutDuration)
	rws.rewardsSnapshotThresholdCh = make(chan struct{})
	rws.failedRewardsPaymentRequests = make(map[uint64]*RelaySessionsToRetryAttempts)
	rws.chainTrackerSpecsInf = chainTrackerSpecsInf

	go rws.saveRewardsSnapshotToDBJob()
	return rws
}

func (rws *RewardServer) saveRewardsSnapshotToDBJob() {
	for {
		select {
		case <-rws.rewardsSnapshotTimer.C:
			rws.resetSnapshotTimerAndSaveRewardsSnapshotToDB()
		case <-rws.rewardsSnapshotThresholdCh:
			rws.resetSnapshotTimerAndSaveRewardsSnapshotToDB()
		}
	}
}

func (rws *RewardServer) resetSnapshotTimerAndSaveRewardsSnapshotToDB() {
	// We lock without defer because the DB is already locking itself
	rws.lock.RLock()
	defer rws.lock.RUnlock()
	rws.rewardsSnapshotTimer.Reset(rws.rewardsSnapshotTimeoutDuration)

	rewardEntities := []*RewardEntity{}
	for epoch, epochRewards := range rws.rewards {
		for consumerRewardKey, consumerRewards := range epochRewards.consumerRewards {
			for sessionId, proof := range consumerRewards.proofs {
				rewardEntity := &RewardEntity{
					Epoch:        epoch,
					ConsumerAddr: consumerRewards.consumer,
					ConsumerKey:  consumerRewardKey,
					SessionId:    sessionId,
					Proof:        proof,
				}
				rewardEntities = append(rewardEntities, rewardEntity)
			}
		}
	}

	if len(rewardEntities) == 0 {
		return
	}
	utils.LavaFormatDebug("saving rewards snapshot to the DB", utils.Attribute{Key: "proofs", Value: len(rewardEntities)})

	var err error
	for i := 0; i < MaxDBSaveRetries; i++ {
		err = rws.rewardDB.BatchSave(rewardEntities)
		if err == nil {
			utils.LavaFormatInfo("Saved rewards snapshot to the DB successfully", utils.Attribute{Key: "proofs", Value: len(rewardEntities)})
			return
		}
		utils.LavaFormatDebug("failed saving proofs snapshot to rewardDB. Retrying...",
			utils.Attribute{Key: "errorReceived", Value: err},
			utils.Attribute{Key: "attempt", Value: i + 1},
			utils.Attribute{Key: "maxAttempts", Value: MaxDBSaveRetries},
		)
	}
	utils.LavaFormatError("failed saving proofs snapshot to rewardDB. Reached maximum attempts", err,
		utils.Attribute{Key: "maxAttempts", Value: MaxDBSaveRetries})
}

func (rws *RewardServer) restoreRewardsFromDB(specId string) (err error) {
	// Pay Attention! This function should be called inside the RewardServer lock

	earliestSavedEpoch, err := rws.getEarliestBlockInMemoryWithRetry(context.Background())
	if err != nil {
		return utils.LavaFormatError("restoreRewardsFromDB failed to get earliest block in memory", err)
	}

	rewards, err := rws.rewardDB.FindAllInDB(specId)
	if err != nil {
		return utils.LavaFormatError("restoreRewardsFromDB failed to FindAllInDB", err, utils.Attribute{Key: "specId", Value: specId})
	}

	for epoch, epochRewardsFromDb := range rewards {
		if epoch < earliestSavedEpoch {
			err := rws.rewardDB.DeleteEpochRewards(epoch)
			if err != nil {
				utils.LavaFormatWarning("restoreRewardsFromDB failed deleting epoch from rewardDB", err, utils.Attribute{Key: "epoch", Value: epoch})
			}

			// Epoch is too old, we can't claim the rewards anymore.
			continue
		}

		// This is happening for every DB because there might be different rewards from the same epoch but from a different specId
		epochRewards, ok := rws.rewards[epoch]
		if !ok {
			rws.rewards[epoch] = epochRewardsFromDb
			continue
		}

		// ConsumerRewardsKey is made from a combination of specId + apiInterface + consumerAddress.
		// So if it's a different DB, it's a different specId, which also means different consumerRewards
		for consumerRewardsKey, consumerRewardsFromDb := range epochRewardsFromDb.consumerRewards {
			epochRewards.consumerRewards[consumerRewardsKey] = consumerRewardsFromDb
		}
	}

	utils.LavaFormatInfo("restored rewards from DB", utils.Attribute{Key: "proofs", Value: len(rewards)})

	return nil
}

func (rws *RewardServer) latestBlockReports(specs map[string]struct{}) (latestBlockReports []*pairingtypes.LatestBlockReport) {
	latestBlockReports = []*pairingtypes.LatestBlockReport{}
	if rws.chainTrackerSpecsInf == nil {
		return
	}
	for spec := range specs {
		latestBlock := rws.chainTrackerSpecsInf.GetLatestBlockNumForSpec(spec)
		if latestBlock < 0 {
			continue
		}
		blockReport := &pairingtypes.LatestBlockReport{
			SpecId:      spec,
			LatestBlock: uint64(latestBlock),
		}
		latestBlockReports = append(latestBlockReports, blockReport)
	}
	return
}

func (rws *RewardServer) gatherFailedRequestPaymentsToRetry(earliestSavedEpoch uint64) (rewardsForClaim []*pairingtypes.RelaySession) {
	rws.lock.Lock()
	defer rws.lock.Unlock()

	if len(rws.failedRewardsPaymentRequests) == 0 {
		return
	}

	var sessionsToDelete []uint64
	for key, val := range rws.failedRewardsPaymentRequests {
		if val.relaySession.Epoch < int64(earliestSavedEpoch) {
			sessionsToDelete = append(sessionsToDelete, key)
			continue
		}
		rewardsForClaim = append(rewardsForClaim, val.relaySession)
	}

	for _, sessionId := range sessionsToDelete {
		rws.deleteRelaySessionFromRewardDB(rws.failedRewardsPaymentRequests[sessionId].relaySession)
		delete(rws.failedRewardsPaymentRequests, sessionId)
	}

	return
}

func (rws *RewardServer) updatePaymentRequestAttempt(paymentRequests []*pairingtypes.RelaySession, success bool) {
	rws.lock.Lock()
	defer rws.lock.Unlock()
	for _, relaySession := range paymentRequests {
		sessionId := relaySession.SessionId
		sessionWithAttempts, found := rws.failedRewardsPaymentRequests[sessionId]
		if !found {
			if !success {
				rws.failedRewardsPaymentRequests[sessionId] = &RelaySessionsToRetryAttempts{
					relaySession:                relaySession,
					paymentRequestRetryAttempts: 1,
				}
			}
			continue
		}

		if success {
			delete(rws.failedRewardsPaymentRequests, sessionId)
			continue
		}

		sessionWithAttempts.paymentRequestRetryAttempts++
		if sessionWithAttempts.paymentRequestRetryAttempts >= MaxPaymentRequestsRetiresForSession {
			utils.LavaFormatInfo("Rewards for session are being removed due to surpassing the maximum allowed retries for payment requests.",
				utils.Attribute{Key: "sessionIds", Value: sessionId},
				utils.Attribute{Key: "maxRetriesAllowed", Value: MaxPaymentRequestsRetiresForSession},
			)
			delete(rws.failedRewardsPaymentRequests, sessionId)
			rws.deleteRelaySessionFromRewardDB(relaySession)
			continue
		}

		rws.failedRewardsPaymentRequests[sessionId] = sessionWithAttempts
	}
}

func (rws *RewardServer) deleteRelaySessionFromRewardDB(relaySession *pairingtypes.RelaySession) error {
	// Must be called inside a lock!
	consumerAddr, err := sigs.ExtractSignerAddress(relaySession)
	if err != nil {
		return utils.LavaFormatError("invalid consumer address extraction from relay", err, utils.Attribute{Key: "relay", Value: relaySession})
	}

	rws.rewardDB.DeleteClaimedRewards(uint64(relaySession.Epoch), consumerAddr.String(), relaySession.SessionId,
		getKeyForConsumerRewards(relaySession.SpecId, consumerAddr.String()))
	return nil
}

func BuildPaymentFromRelayPaymentEvent(event terderminttypes.Event, block int64) ([]*PaymentRequest, error) {
	type mapCont struct {
		attributes map[string]string
		index      int
	}
	attributesList := []*mapCont{}
	appendToAttributeList := func(idx int, key, value string) {
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
		splittedAttrs := strings.SplitN(attribute.Key, ".", 2)
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
		appendToAttributeList(index, attrKey, attribute.Value)
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
		epochString, ok := attributes["epoch"]
		if !ok {
			return nil, utils.LavaFormatError("failed building PaymentRequest from relay_payment event missing field epoch", nil, utils.Attribute{Key: "attributes", Value: attributes}, utils.Attribute{Key: "idx", Value: idx})
		}
		epoch, err := strconv.ParseUint(epochString, 10, 64)
		if err != nil {
			return nil, err
		}
		payment := &PaymentRequest{
			CU:                  cu,
			BlockHeightDeadline: block,
			PaymentEpoch:        epoch,
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

func getKeyForConsumerRewards(specId string, consumerAddress string) string {
	return specId + consumerAddress
}
