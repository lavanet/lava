package lavasession

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/protocol/qos"
	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

type SingleConsumerSession struct {
	CuSum         uint64
	LatestRelayCu uint64 // set by GetSessions cuNeededForSession
	QoSManager    *qos.QoSManager
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
	usedProviders      UsedProvidersInf
	providerUniqueId   string
	StaticProvider     bool
	routerKey          RouterKey
	epoch              uint64
}

// returns the expected latency to a threshold.
func (cs *SingleConsumerSession) CalculateExpectedLatency(timeoutGivenToRelay time.Duration) time.Duration {
	expectedLatency := (timeoutGivenToRelay / 2)
	return expectedLatency
}

// cs should be locked here to use this method, returns the computed qos or zero if last qos is nil or failed to compute.
func (cs *SingleConsumerSession) getQosComputedResultOrZero() sdk.Dec {
	lastReputationReport := cs.QoSManager.GetLastReputationQoSReportRaw(cs.epoch, cs.SessionId)
	if lastReputationReport != nil {
		computedReputation, errComputing := lastReputationReport.ComputeQoSExcellence()
		if errComputing == nil { // if we failed to compute the qos will be 0 so this provider wont be picked to return the error in case we get it
			return computedReputation
		}
		utils.LavaFormatDebug("Failed computing QoS used for error parsing, could happen if we have no sync data or one of the fields is zero",
			utils.LogAttr("Report", cs.QoSManager.GetLastReputationQoSReportRaw(cs.epoch, cs.SessionId)),
			utils.LogAttr("error", errComputing),
		)
	}
	return sdk.ZeroDec()
}

func (scs *SingleConsumerSession) SetUsageForSession(cuNeededForSession uint64, reputationReport *pairingtypes.QualityOfServiceReport, rawReputationReport *pairingtypes.QualityOfServiceReport, usedProviders UsedProvidersInf, routerKey RouterKey) error {
	scs.LatestRelayCu = cuNeededForSession // set latestRelayCu
	scs.RelayNum += RelayNumberIncrement   // increase relayNum
	if scs.RelayNum > 1 {
		// we only set reputation for sessions with more than one successful relays, this guarantees data within the epoch exists
		scs.QoSManager.SetLastReputationQoSReportRaw(scs.epoch, scs.SessionId, reputationReport)
		scs.QoSManager.SetLastReputationQoSReportRaw(scs.epoch, scs.SessionId, rawReputationReport)
	}
	scs.usedProviders = usedProviders
	scs.routerKey = routerKey
	return nil
}

func (scs *SingleConsumerSession) Free(err error) {
	if scs.usedProviders != nil {
		scs.usedProviders.RemoveUsed(scs.Parent.PublicLavaAddress, scs.routerKey, err)
		scs.usedProviders = nil
	}
	scs.routerKey = NewRouterKey(nil)
	scs.EndpointConnection.decreaseSessionUsingConnection()
	scs.lock.Unlock()
}

func (session *SingleConsumerSession) TryUseSession() (blocked bool, ok bool) {
	if session.lock.TryLock() {
		if session.BlockListed { // this session cannot be used.
			session.lock.Unlock()
			return true, false
		}
		if session.usedProviders != nil {
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

func (scs *SingleConsumerSession) VerifyProviderUniqueIdAndStoreIfFirstTime(providerUniqueId string) bool {
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
