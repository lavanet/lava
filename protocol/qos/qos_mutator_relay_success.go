package qos

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// Mutator for relay success
type QoSMutatorRelaySuccess struct {
	*QoSMutatorBase
	latency          time.Duration
	expectedLatency  time.Duration
	blockHeightDiff  int64
	numOfProviders   int
	servicersToCount int64
	providerAddress  string
}

func (qoSMutatorRelaySuccess *QoSMutatorRelaySuccess) calculateAvailabilityScore(qosReport *QoSReport) (downtimePercentageRet, scaledAvailabilityScoreRet float64) {
	downtimePercentage := float64(qosReport.totalRelays-qosReport.answeredRelays) / float64(qosReport.totalRelays)
	scaledAvailabilityScore := math.Max(0.0, (AvailabilityPercentage-downtimePercentage)/AvailabilityPercentage)
	return downtimePercentage, scaledAvailabilityScore
}

func (qoSMutatorRelaySuccess *QoSMutatorRelaySuccess) Mutate(report *QoSReport) {
	report.totalRelays++
	report.answeredRelays++

	if report.lastQoSReport == nil {
		report.lastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage, scaledAvailabilityScore := qoSMutatorRelaySuccess.calculateAvailabilityScore(report)
	report.lastQoSReport.Availability = scaledAvailabilityScore
	if 1.0 > report.lastQoSReport.Availability {
		utils.LavaFormatDebug("QoS Availability report",
			utils.LogAttr("availability", report.lastQoSReport.Availability),
			utils.LogAttr("down_percent", downtimePercentage),
			utils.LogAttr("session_id", qoSMutatorRelaySuccess.sessionId),
			utils.LogAttr("provider", qoSMutatorRelaySuccess.providerAddress),
		)
	}

	if qoSMutatorRelaySuccess.latency == 0 {
		qoSMutatorRelaySuccess.latency = 1 * time.Microsecond
	}

	latencyScore := math.Min(1.0, float64(qoSMutatorRelaySuccess.expectedLatency)/float64(qoSMutatorRelaySuccess.latency))

	insertSorted := func(list []float64, value float64) []float64 {
		index := sort.Search(len(list), func(i int) bool {
			return list[i] >= value
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
	shouldCalculateSyncScore := int64(qoSMutatorRelaySuccess.numOfProviders) > int64(math.Ceil(float64(qoSMutatorRelaySuccess.servicersToCount)*MinProvidersForSync))
	if shouldCalculateSyncScore { //
		if qoSMutatorRelaySuccess.blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			report.syncScoreSum++
		}
		report.totalSyncScore++
		report.lastQoSReport.Sync = float64(report.syncScoreSum) / float64(report.totalSyncScore)
		if 1.0 > report.lastQoSReport.Sync {
			utils.LavaFormatDebug("QoS Sync report",
				utils.LogAttr("sync", report.lastQoSReport.Sync),
				utils.LogAttr("block_diff", qoSMutatorRelaySuccess.blockHeightDiff),
				utils.LogAttr("sync_score", strconv.FormatInt(report.syncScoreSum, 10)+"/"+strconv.FormatInt(report.totalSyncScore, 10)),
				utils.LogAttr("session_id", qoSMutatorRelaySuccess.sessionId),
				utils.LogAttr("provider", qoSMutatorRelaySuccess.providerAddress),
			)
		}
	} else {
		// we prefer to give them a score of 1 when there is no other data, since otherwise we damage their payments
		report.lastQoSReport.Sync = 1.0
	}
}
