package qos

import (
	"math"
	"sort"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

// Base interface for all mutators
type Mutator interface {
	Mutate(report *QoSReport)
	GetEpochAndSessionId() (epoch uint64, sessionId int64)
}

type QoSUpdateTaskBase struct {
	epoch     uint64
	sessionId int64
}

func (qoSUpdateTaskBase *QoSUpdateTaskBase) GetEpochAndSessionId() (epoch uint64, sessionId int64) {
	return qoSUpdateTaskBase.epoch, qoSUpdateTaskBase.sessionId
}

// Mutator for relay success
type QoSUpdateTaskRelaySuccess struct {
	QoSUpdateTaskBase
	latency          time.Duration
	expectedLatency  time.Duration
	blockHeightDiff  int64
	numOfProviders   int
	servicersToCount int64
	providerAddress  string
}

func (qoSUpdateTaskRelaySuccess *QoSUpdateTaskRelaySuccess) Mutate(report *QoSReport) {
	report.lock.Lock()
	defer report.lock.Unlock()
	report.totalRelays++
	report.answeredRelays++

	if report.lastQoSReport == nil {
		report.lastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage := sdk.NewDecWithPrec(int64(report.totalRelays-report.answeredRelays), 0).Quo(sdk.NewDecWithPrec(int64(report.totalRelays), 0))
	scaledAvailabilityScore := sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePercentage).Quo(AvailabilityPercentage))
	report.lastQoSReport.Availability = scaledAvailabilityScore
	if sdk.OneDec().GT(report.lastQoSReport.Availability) {
		utils.LavaFormatDebug("QoS Availability report", utils.Attribute{Key: "Availability", Value: report.lastQoSReport.Availability}, utils.Attribute{Key: "down percent", Value: downtimePercentage})
	}

	latencyScore := sdk.MinDec(sdk.OneDec(), sdk.NewDecFromInt(sdk.NewInt(int64(qoSUpdateTaskRelaySuccess.expectedLatency))).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(qoSUpdateTaskRelaySuccess.latency)))))

	insertSorted := func(list []sdk.Dec, value sdk.Dec) []sdk.Dec {
		index := sort.Search(len(list), func(i int) bool {
			return list[i].GTE(value)
		})
		if len(list) == index { // nil or empty slice or after last element
			return append(list, value)
		}
		list = append(list[:index+1], list[index:]...) // index < len(a)
		list[index] = value
		return list
	}
	report.latencyScoreList = insertSorted(report.latencyScoreList, latencyScore)
	report.lastQoSReport.Latency = report.latencyScoreList[int(float64(len(report.latencyScoreList))*PercentileToCalculateLatency)]

	// checking if we have enough information to calculate the sync score for the providers, if we haven't talked
	// with enough providers we don't have enough information and we will wait to have more information before setting the sync score
	shouldCalculateSyncScore := int64(qoSUpdateTaskRelaySuccess.numOfProviders) > int64(math.Ceil(float64(qoSUpdateTaskRelaySuccess.servicersToCount)*MinProvidersForSync))
	if shouldCalculateSyncScore { //
		if qoSUpdateTaskRelaySuccess.blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			report.syncScoreSum++
		}
		report.totalSyncScore++
		report.lastQoSReport.Sync = sdk.NewDec(report.syncScoreSum).QuoInt64(report.totalSyncScore)
		if sdk.OneDec().GT(report.lastQoSReport.Sync) {
			utils.LavaFormatDebug("QoS Sync report",
				utils.Attribute{Key: "Sync", Value: report.lastQoSReport.Sync},
				utils.Attribute{Key: "block diff", Value: qoSUpdateTaskRelaySuccess.blockHeightDiff},
				utils.Attribute{Key: "sync score", Value: strconv.FormatInt(report.syncScoreSum, 10) + "/" + strconv.FormatInt(report.totalSyncScore, 10)},
				utils.Attribute{Key: "session_id", Value: qoSUpdateTaskRelaySuccess.sessionId},
				utils.Attribute{Key: "provider", Value: qoSUpdateTaskRelaySuccess.providerAddress},
			)
		}
	} else {
		// we prefer to give them a score of 1 when there is no other data, since otherwise we damage their payments
		report.lastQoSReport.Sync = sdk.NewDec(1)
	}
}

// Mutator for relay failure
type QoSUpdateTaskRelayFailure struct {
	QoSUpdateTaskBase
}

func (qoSUpdateTaskRelayFailure *QoSUpdateTaskRelayFailure) Mutate(report *QoSReport) {
	report.lock.Lock()
	defer report.lock.Unlock()
	report.totalRelays++
}

// Mutator to set usage for a session
type QoSUpdateTaskSetReputationRaw struct {
	QoSUpdateTaskBase
	report *pairingtypes.QualityOfServiceReport
}

func (qoSUpdateTaskSetReputationRaw *QoSUpdateTaskSetReputationRaw) Mutate(report *QoSReport) {
	report.lock.Lock()
	defer report.lock.Unlock()
	report.lastReputationQoSReportRaw = qoSUpdateTaskSetReputationRaw.report
}
