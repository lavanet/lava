package lavasession

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
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
	
	// Connection type - uses composition pattern for type safety
	// Either ProviderRelayConnection (rpcconsumer) or DirectRPCSessionConnection (rpcsmartrouter)
	Connection SessionConnection
	
	// Legacy field - maintained for backward compatibility
	// For provider-relay mode, this points to the same connection as Connection.(*ProviderRelayConnection).EndpointConnection
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
	// Use GetSessionQoSManager() to support both connection types
	qosManager := cs.GetSessionQoSManager()
	if qosManager == nil {
		return sdk.ZeroDec()
	}
	
	lastReputationReport := qosManager.GetLastReputationQoSReport(cs.epoch, cs.SessionId)
	if lastReputationReport != nil {
		computedReputation, errComputing := lastReputationReport.ComputeReputation()
		if errComputing == nil { // if we failed to compute the qos will be 0 so this provider wont be picked to return the error in case we get it
			return computedReputation
		}
		utils.LavaFormatDebug("Failed computing QoS used for error parsing, could happen if we have no sync data or one of the fields is zero",
			utils.LogAttr("Report", lastReputationReport),
			utils.LogAttr("error", errComputing),
		)
	}
	return sdk.ZeroDec()
}

func (scs *SingleConsumerSession) SetUsageForSession(cuNeededForSession uint64, reputationReport *pairingtypes.QualityOfServiceReport, usedProviders UsedProvidersInf, routerKey RouterKey) error {
	scs.LatestRelayCu = cuNeededForSession // set latestRelayCu
	scs.RelayNum += RelayNumberIncrement   // increase relayNum
	if scs.RelayNum > 1 {
		// we only set reputation for sessions with more than one successful relays, this guarantees data within the epoch exists
		scs.QoSManager.SetLastReputationQoSReport(scs.epoch, scs.SessionId, reputationReport)
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
	
	// Only decrease connection usage for provider-relay sessions
	// Direct RPC sessions don't have EndpointConnection
	if scs.EndpointConnection != nil {
	scs.EndpointConnection.decreaseSessionUsingConnection()
	}
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
		
		// Only increase connection usage for provider-relay sessions
		// Direct RPC sessions don't have EndpointConnection
		if session.EndpointConnection != nil {
		session.EndpointConnection.addSessionUsingConnection()
		}
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

// GetDirectConnection returns the DirectRPCConnection if this is a smart router session
// Returns (connection, true) if this is a direct RPC session, (nil, false) otherwise
func (scs *SingleConsumerSession) GetDirectConnection() (DirectRPCConnection, bool) {
	if drsc, ok := scs.Connection.(*DirectRPCSessionConnection); ok {
		return drsc.DirectConnection, true
	}
	return nil, false
}

// GetProviderConnection returns the EndpointConnection if this is a consumer session
// Returns (connection, true) if this is a provider-relay session, (nil, false) otherwise
func (scs *SingleConsumerSession) GetProviderConnection() (*EndpointConnection, bool) {
	if prc, ok := scs.Connection.(*ProviderRelayConnection); ok {
		return prc.EndpointConnection, true
	}
	return nil, false
}

// IsDirectRPC returns true if this session uses direct RPC (smart router mode)
func (scs *SingleConsumerSession) IsDirectRPC() bool {
	_, ok := scs.Connection.(*DirectRPCSessionConnection)
	return ok
}

// GetSessionQoSManager returns the QoS manager from the connection
// Falls back to the legacy QoSManager field if Connection is nil (backward compatibility)
func (scs *SingleConsumerSession) GetSessionQoSManager() *qos.QoSManager {
	if scs.Connection != nil {
		return scs.Connection.GetQoSManager()
	}
	// Fallback for backward compatibility
	return scs.QoSManager
}
