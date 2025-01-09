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

type QoSManager struct {
	lastQoSReport              *pairingtypes.QualityOfServiceReport
	lastReputationQoSReport    *pairingtypes.QualityOfServiceReport
	lastReputationQoSReportRaw *pairingtypes.QualityOfServiceReport
	latencyScoreList           []sdk.Dec
	syncScoreSum               int64
	totalSyncScore             int64
	totalRelays                uint64
	answeredRelays             uint64
}

func NewQoSManager() *QoSManager {
	return &QoSManager{
		lastQoSReport:              &pairingtypes.QualityOfServiceReport{},
		lastReputationQoSReport:    &pairingtypes.QualityOfServiceReport{},
		lastReputationQoSReportRaw: &pairingtypes.QualityOfServiceReport{},
		latencyScoreList:           []sdk.Dec{},
		syncScoreSum:               0,
		totalSyncScore:             0,
		totalRelays:                0,
		answeredRelays:             0,
	}
}

func (qosReport *QoSManager) CalculateAvailabilityScore() (downtimePercentageRet, scaledAvailabilityScoreRet sdk.Dec) {
	downtimePercentage := sdk.NewDecWithPrec(int64(qosReport.totalRelays-qosReport.answeredRelays), 0).Quo(sdk.NewDecWithPrec(int64(qosReport.totalRelays), 0))
	scaledAvailabilityScore := sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePercentage).Quo(AvailabilityPercentage))
	return downtimePercentage, scaledAvailabilityScore
}

func (qosReport *QoSManager) GetLastQoSReport() *pairingtypes.QualityOfServiceReport {
	return qosReport.lastQoSReport
}

func (qosReport *QoSManager) GetLastReputationQoSReportRaw() *pairingtypes.QualityOfServiceReport {
	return qosReport.lastReputationQoSReportRaw
}

func (qosReport *QoSManager) SetLastReputationQoSReportRaw(report *pairingtypes.QualityOfServiceReport) {
	qosReport.lastReputationQoSReportRaw = report
}

func (qosReport *QoSManager) IncTotalRelays() {
	qosReport.totalRelays++
}

func (cs *QoSManager) CalculateQoS(latency, expectedLatency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {
	// Add current Session QoS
	cs.totalRelays++    // increase total relays
	cs.answeredRelays++ // increase answered relays

	if cs.lastQoSReport == nil {
		cs.lastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage, scaledAvailabilityScore := cs.CalculateAvailabilityScore()
	cs.lastQoSReport.Availability = scaledAvailabilityScore
	if sdk.OneDec().GT(cs.lastQoSReport.Availability) {
		utils.LavaFormatDebug("QoS Availability report", utils.Attribute{Key: "Availability", Value: cs.lastQoSReport.Availability}, utils.Attribute{Key: "down percent", Value: downtimePercentage})
	}

	latencyScore := sdk.MinDec(sdk.OneDec(), sdk.NewDecFromInt(sdk.NewInt(int64(expectedLatency))).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(latency)))))

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
	cs.latencyScoreList = insertSorted(cs.latencyScoreList, latencyScore)
	cs.lastQoSReport.Latency = cs.latencyScoreList[int(float64(len(cs.latencyScoreList))*PercentileToCalculateLatency)]

	// checking if we have enough information to calculate the sync score for the providers, if we haven't talked
	// with enough providers we don't have enough information and we will wait to have more information before setting the sync score
	shouldCalculateSyncScore := int64(numOfProviders) > int64(math.Ceil(float64(servicersToCount)*MinProvidersForSync))
	if shouldCalculateSyncScore { //
		if blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			cs.syncScoreSum++
		}
		cs.totalSyncScore++
		cs.lastQoSReport.Sync = sdk.NewDec(cs.syncScoreSum).QuoInt64(cs.totalSyncScore)
		if sdk.OneDec().GT(cs.lastQoSReport.Sync) {
			utils.LavaFormatDebug("QoS Sync report",
				utils.Attribute{Key: "Sync", Value: cs.lastQoSReport.Sync},
				utils.Attribute{Key: "block diff", Value: blockHeightDiff},
				utils.Attribute{Key: "sync score", Value: strconv.FormatInt(cs.syncScoreSum, 10) + "/" + strconv.FormatInt(cs.totalSyncScore, 10)},
				// utils.Attribute{Key: "session_id", Value: cs.SessionId},
				// utils.Attribute{Key: "provider", Value: cs.Parent.PublicLavaAddress},
			)
		}
	} else {
		// we prefer to give them a score of 1 when there is no other data, since otherwise we damage their payments
		cs.lastQoSReport.Sync = sdk.NewDec(1)
	}
}

func (qosReport *QoSManager) GetAnsweredRelays() uint64 {
	return qosReport.answeredRelays
}

func (qosReport *QoSManager) SetAnsweredRelays(answeredRelays uint64) {
	qosReport.answeredRelays = answeredRelays
}

func (qosReport *QoSManager) GetTotalRelays() uint64 {
	return qosReport.totalRelays
}

func (qosReport *QoSManager) SetTotalRelays(totalRelays uint64) {
	qosReport.totalRelays = totalRelays
}

func (qosReport *QoSManager) GetSyncScoreSum() int64 {
	return qosReport.syncScoreSum
}

func (qosReport *QoSManager) GetTotalSyncScore() int64 {
	return qosReport.totalSyncScore
}
