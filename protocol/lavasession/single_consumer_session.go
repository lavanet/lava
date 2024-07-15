package lavasession

import (
	"math"
	"sort"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type SingleConsumerSession struct {
	CuSum         uint64
	LatestRelayCu uint64 // set by GetSessions cuNeededForSession
	QoSInfo       QoSReport
	SessionId     int64
	Parent        *ConsumerSessionsWithProvider
	lock          utils.LavaMutex
	RelayNum      uint64
	LatestBlock   int64
	// Each session will holds a pointer to a connection, if the connection is lost, this session will be banned (wont be picked)
	EndpointConnection *EndpointConnection
	BlockListed        bool // if session lost sync we blacklist it.
	ConsecutiveErrors  []error
	errorsCount        uint64
	relayProcessor     UsedProvidersInf
	providerUniqueId   string
}

// returns the expected latency to a threshold.
func (cs *SingleConsumerSession) CalculateExpectedLatency(timeoutGivenToRelay time.Duration) time.Duration {
	expectedLatency := (timeoutGivenToRelay / 2)
	return expectedLatency
}

// cs should be locked here to use this method, returns the computed qos or zero if last qos is nil or failed to compute.
func (cs *SingleConsumerSession) getQosComputedResultOrZero() sdk.Dec {
	if cs.QoSInfo.LastExcellenceQoSReport != nil {
		qosComputed, errComputing := cs.QoSInfo.LastExcellenceQoSReport.ComputeQoSExcellence()
		if errComputing == nil { // if we failed to compute the qos will be 0 so this provider wont be picked to return the error in case we get it
			return qosComputed
		}
		utils.LavaFormatError("Failed computing QoS used for error parsing", errComputing, utils.LogAttr("Report", cs.QoSInfo.LastExcellenceQoSReport))
	}
	return sdk.ZeroDec()
}

func (cs *SingleConsumerSession) CalculateQoS(latency, expectedLatency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {
	// Add current Session QoS
	cs.QoSInfo.TotalRelays++    // increase total relays
	cs.QoSInfo.AnsweredRelays++ // increase answered relays

	if cs.QoSInfo.LastQoSReport == nil {
		cs.QoSInfo.LastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage, scaledAvailabilityScore := CalculateAvailabilityScore(&cs.QoSInfo)
	cs.QoSInfo.LastQoSReport.Availability = scaledAvailabilityScore
	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Availability) {
		utils.LavaFormatDebug("QoS Availability report", utils.Attribute{Key: "Availability", Value: cs.QoSInfo.LastQoSReport.Availability}, utils.Attribute{Key: "down percent", Value: downtimePercentage})
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
	cs.QoSInfo.LatencyScoreList = insertSorted(cs.QoSInfo.LatencyScoreList, latencyScore)
	cs.QoSInfo.LastQoSReport.Latency = cs.QoSInfo.LatencyScoreList[int(float64(len(cs.QoSInfo.LatencyScoreList))*PercentileToCalculateLatency)]

	// checking if we have enough information to calculate the sync score for the providers, if we haven't talked
	// with enough providers we don't have enough information and we will wait to have more information before setting the sync score
	shouldCalculateSyncScore := int64(numOfProviders) > int64(math.Ceil(float64(servicersToCount)*MinProvidersForSync))
	if shouldCalculateSyncScore { //
		if blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			cs.QoSInfo.SyncScoreSum++
		}
		cs.QoSInfo.TotalSyncScore++
		cs.QoSInfo.LastQoSReport.Sync = sdk.NewDec(cs.QoSInfo.SyncScoreSum).QuoInt64(cs.QoSInfo.TotalSyncScore)
		if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Sync) {
			utils.LavaFormatDebug("QoS Sync report",
				utils.Attribute{Key: "Sync", Value: cs.QoSInfo.LastQoSReport.Sync},
				utils.Attribute{Key: "block diff", Value: blockHeightDiff},
				utils.Attribute{Key: "sync score", Value: strconv.FormatInt(cs.QoSInfo.SyncScoreSum, 10) + "/" + strconv.FormatInt(cs.QoSInfo.TotalSyncScore, 10)},
				utils.Attribute{Key: "session_id", Value: cs.SessionId},
				utils.Attribute{Key: "provider", Value: cs.Parent.PublicLavaAddress},
			)
		}
	} else {
		// we prefer to give them a score of 1 when there is no other data, since otherwise we damage their payments
		cs.QoSInfo.LastQoSReport.Sync = sdk.NewDec(1)
	}
}

func (scs *SingleConsumerSession) SetUsageForSession(cuNeededForSession uint64, qoSExcellenceReport *pairingtypes.QualityOfServiceReport, usedProviders UsedProvidersInf) error {
	scs.LatestRelayCu = cuNeededForSession // set latestRelayCu
	scs.RelayNum += RelayNumberIncrement   // increase relayNum
	if scs.RelayNum > 1 {
		// we only set excellence for sessions with more than one successful relays, this guarantees data within the epoch exists
		scs.QoSInfo.LastExcellenceQoSReport = qoSExcellenceReport
	}
	scs.relayProcessor = usedProviders
	return nil
}

func (scs *SingleConsumerSession) Free(err error) {
	if scs.relayProcessor != nil {
		scs.relayProcessor.RemoveUsed(scs.Parent.PublicLavaAddress, err)
		scs.relayProcessor = nil
	}
	scs.EndpointConnection.decreaseSessionUsingConnection()
	scs.lock.Unlock()
}

func (session *SingleConsumerSession) TryUseSession() (blocked bool, ok bool) {
	if session.lock.TryLock() {
		if session.BlockListed { // this session cannot be used.
			session.lock.Unlock()
			return true, false
		}
		if session.relayProcessor != nil {
			utils.LavaFormatError("session misuse detected, usedProviders isn't nil, missing Free call, blocking", nil, utils.LogAttr("session", session.SessionId))
			session.BlockListed = true
			session.lock.Unlock()
			return true, false
		}
		session.EndpointConnection.addSessionUsingConnection()
		return false, true
	}
	return false, false
}

// Verify the consumerSession is locked when getting to this function, if its not locked throw an error
func (consumerSession *SingleConsumerSession) VerifyLock() error {
	if consumerSession.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.Free(nil)
		// if failed to lock we should block session as it seems like a very rare case.
		consumerSession.BlockListed = true // block this session from future usages
		utils.LavaFormatError("Verify Lock failed on session Failure, blocking session", nil, utils.LogAttr("consumerSession", consumerSession))
		return LockMisUseDetectedError
	}
	return nil
}

func (scs *SingleConsumerSession) VerifyProviderUniqueId(providerUniqueId string) bool {
	if scs.providerUniqueId == "" {
		utils.LavaFormatTrace("First time getting providerUniqueId for SingleConsumerSession",
			utils.LogAttr("sessionId", scs.SessionId),
			utils.LogAttr("providerUniqueId", providerUniqueId),
		)

		scs.providerUniqueId = providerUniqueId
		return true
	}

	return providerUniqueId == scs.providerUniqueId
}

func (scs *SingleConsumerSession) GetProviderUniqueId() string {
	return scs.providerUniqueId
}
