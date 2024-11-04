package updaters

import (
	"context"
	"sync"
	"time"

	"golang.org/x/exp/slices"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/v4/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/v4/utils"
	conflicttypes "github.com/lavanet/lava/v4/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

const (
	BlockResultRetry = 20
)

var TimeOutForFetchingLavaBlocks = time.Second * 5

type EventTracker struct {
	lock sync.RWMutex
	*StateQuery
	blockResults       *ctypes.ResultBlockResults
	latestUpdatedBlock int64
}

func (et *EventTracker) UpdateBlockResults(latestBlock int64) (err error) {
	ctx := context.Background()

	if latestBlock == 0 {
		var res *ctypes.ResultStatus
		for i := 0; i < 3; i++ {
			timeoutCtx, cancel := context.WithTimeout(ctx, TimeOutForFetchingLavaBlocks)
			res, err = et.StateQuery.Status(timeoutCtx)
			cancel()
			if err == nil {
				break
			}
		}
		if err != nil {
			return utils.LavaFormatWarning("could not get latest block height and requested latestBlock = 0", err)
		}
		latestBlock = res.SyncInfo.LatestBlockHeight
	}

	var blockResults *ctypes.ResultBlockResults
	for i := 0; i < BlockResultRetry; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, TimeOutForFetchingLavaBlocks)
		blockResults, err = et.StateQuery.BlockResults(timeoutCtx, &latestBlock)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond * time.Duration(i+1)) // need this so it doesn't just spam the attempts, and tendermint fails getting block results pretty often
	}
	if err != nil {
		return utils.LavaFormatError("could not get block result", err, utils.LogAttr("block_requested", latestBlock))
	}
	// lock for update after successful block result query
	et.lock.Lock()
	defer et.lock.Unlock()
	if latestBlock > et.latestUpdatedBlock {
		et.latestUpdatedBlock = latestBlock
		et.blockResults = blockResults
	} else {
		utils.LavaFormatDebug("event tracker got an outdated block", utils.Attribute{Key: "block", Value: latestBlock}, utils.Attribute{Key: "latestUpdatedBlock", Value: et.latestUpdatedBlock})
	}
	return nil
}

func (et *EventTracker) getLatestPaymentEvents() (payments []*rewardserver.PaymentRequest, err error) {
	et.lock.RLock()
	defer et.lock.RUnlock()
	transactionResults := et.blockResults.TxsResults
	for _, tx := range transactionResults {
		events := tx.Events
		for _, event := range events {
			if event.Type == utils.EventPrefix+pairingtypes.RelayPaymentEventName {
				paymentList, err := rewardserver.BuildPaymentFromRelayPaymentEvent(event, et.latestUpdatedBlock)
				if err != nil {
					return nil, utils.LavaFormatError("failed relay_payment_event parsing", err, utils.Attribute{Key: "event", Value: event})
				}

				utils.LavaFormatTrace("relay_payment_event", utils.LogAttr("payment", paymentList))

				payments = append(payments, paymentList...)
			}
		}
	}
	return payments, nil
}

func (et *EventTracker) getLatestVersionEvents(latestBlock int64) (updated bool, err error) {
	et.lock.RLock()
	defer et.lock.RUnlock()
	if et.latestUpdatedBlock != latestBlock {
		return false, utils.LavaFormatWarning("event results are different than expected", nil, utils.Attribute{Key: "requested latestBlock", Value: latestBlock}, utils.Attribute{Key: "current latestBlock", Value: et.latestUpdatedBlock})
	}
	for _, event := range et.blockResults.EndBlockEvents {
		if event.Type == utils.EventPrefix+"param_change" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "param" && attribute.Value == "Version" {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (et *EventTracker) getLatestDowntimeParamsUpdateEvents(latestBlock int64) (updated bool, err error) {
	// check DowntimeParams change proposal results
	et.lock.RLock()
	defer et.lock.RUnlock()
	if et.latestUpdatedBlock != latestBlock {
		return false, utils.LavaFormatWarning("event results are different than expected", nil, utils.Attribute{Key: "requested latestBlock", Value: latestBlock}, utils.Attribute{Key: "current latestBlock", Value: et.latestUpdatedBlock})
	}
	for _, event := range et.blockResults.EndBlockEvents {
		if event.Type == utils.EventPrefix+"param_change" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "param" && (attribute.Value == "DowntimeDuration" || attribute.Value == "EpochDuration") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (et *EventTracker) getLatestSpecModifyEvents(latestBlock int64) (updated bool, err error) {
	// SpecModifyEventName
	et.lock.RLock()
	defer et.lock.RUnlock()
	if et.latestUpdatedBlock != latestBlock {
		return false, utils.LavaFormatWarning("event results are different than expected", nil, utils.Attribute{Key: "requested latestBlock", Value: latestBlock}, utils.Attribute{Key: "current latestBlock", Value: et.latestUpdatedBlock})
	}
	eventsListToListenTo := []string{
		utils.EventPrefix + spectypes.SpecModifyEventName,
		utils.EventPrefix + spectypes.SpecRefreshEventName,
	}
	for _, event := range et.blockResults.EndBlockEvents {
		if slices.Contains(eventsListToListenTo, event.Type) {
			utils.LavaFormatInfo("Spec update event identified", utils.LogAttr("Event", event.Type))
			return true, nil
		}
	}
	return false, nil
}

func (et *EventTracker) getLatestVoteEvents(latestBlock int64) (votes []*reliabilitymanager.VoteParams, err error) {
	et.lock.RLock()
	defer et.lock.RUnlock()
	if et.latestUpdatedBlock != latestBlock {
		return nil, utils.LavaFormatWarning("event results are different than expected", nil, utils.Attribute{Key: "requested latestBlock", Value: latestBlock}, utils.Attribute{Key: "current latestBlock", Value: et.latestUpdatedBlock})
	}
	transactionResults := et.blockResults.TxsResults
	for _, tx := range transactionResults {
		events := tx.Events
		for _, event := range events {
			if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteDetectionEventName {
				vote, err := reliabilitymanager.BuildVoteParamsFromDetectionEvent(event)
				if err != nil {
					return nil, utils.LavaFormatError("failed conflict_vote_detection_event parsing", err, utils.Attribute{Key: "event", Value: event})
				}
				utils.LavaFormatDebug("conflict_vote_detection_event", utils.Attribute{Key: "voteID", Value: vote.VoteID})
				votes = append(votes, vote)
			}
		}
	}

	beginBlockEvents := et.blockResults.BeginBlockEvents
	for _, event := range beginBlockEvents {
		if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteRevealEventName {
			voteID, voteDeadline, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
			if err != nil {
				return nil, utils.LavaFormatError("failed conflict_vote_reveal_event parsing", err, utils.Attribute{Key: "event", Value: event})
			}
			vote_reveal := &reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: voteDeadline, ParamsType: reliabilitymanager.RevealVoteType}
			utils.LavaFormatDebug("conflict_vote_reveal_event", utils.Attribute{Key: "voteID", Value: voteID})
			votes = append(votes, vote_reveal)
		}
		if event.Type == utils.EventPrefix+conflicttypes.ConflictVoteResolvedEventName {
			voteID, _, err := reliabilitymanager.BuildBaseVoteDataFromEvent(event)
			if err != nil {
				if !reliabilitymanager.NoVoteDeadline.Is(err) {
					return nil, utils.LavaFormatError("failed conflict_vote_resolved_event parsing", err, utils.Attribute{Key: "event", Value: event})
				}
			}
			vote_resolved := &reliabilitymanager.VoteParams{VoteID: voteID, VoteDeadline: 0, ParamsType: reliabilitymanager.CloseVoteType, CloseVote: true}
			votes = append(votes, vote_resolved)
			utils.LavaFormatDebug("conflict_vote_resolved_event", utils.Attribute{Key: "voteID", Value: voteID})
		}
	}

	return votes, err
}

type tendermintRPC interface {
	BlockResults(
		ctx context.Context,
		height *int64,
	) (*ctypes.ResultBlockResults, error)
	ConsensusParams(
		ctx context.Context,
		height *int64,
	) (*ctypes.ResultConsensusParams, error)
}
