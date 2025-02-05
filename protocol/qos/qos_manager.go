package qos

import (
	"sync"
	"time"

	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

type QoSManager struct {
	qosReports map[uint64]map[int64]*QoSReport // first key is the epoch, second key is the session id
	lock       sync.RWMutex
}

func NewQoSManager() *QoSManager {
	qosManager := &QoSManager{}
	qosManager.qosReports = make(map[uint64]map[int64]*QoSReport)
	return qosManager
}

func (qosManager *QoSManager) fetchOrSetSessionFromMap(epoch uint64, sessionId int64) *QoSReport {
	qosManager.lock.Lock()
	defer qosManager.lock.Unlock()
	if qosManager.qosReports[epoch] == nil {
		qosManager.qosReports[epoch] = make(map[int64]*QoSReport)
	}
	if qosManager.qosReports[epoch][sessionId] == nil {
		qosManager.qosReports[epoch][sessionId] = &QoSReport{}
	}
	return qosManager.qosReports[epoch][sessionId]
}

func (qosManager *QoSManager) createQoSMutatorBase(epoch uint64, sessionId int64) *QoSMutatorBase {
	qosMutatorBase := &QoSMutatorBase{
		epoch:     epoch,
		sessionId: sessionId,
	}
	return qosMutatorBase
}

func (qm *QoSManager) mutate(mutator Mutator) {
	qosReport := qm.fetchOrSetSessionFromMap(mutator.GetEpochAndSessionId())
	qosReport.mutate(mutator)
}

func (qosManager *QoSManager) CalculateQoS(epoch uint64, sessionId int64, providerAddress string, latency, expectedLatency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {
	qosManager.mutate(&QoSMutatorRelaySuccess{
		QoSMutatorBase:   qosManager.createQoSMutatorBase(epoch, sessionId),
		providerAddress:  providerAddress,
		latency:          latency,
		expectedLatency:  expectedLatency,
		blockHeightDiff:  blockHeightDiff,
		numOfProviders:   numOfProviders,
		servicersToCount: servicersToCount,
	})
}

func (qosManager *QoSManager) AddFailedRelay(epoch uint64, sessionId int64) {
	qosManager.mutate(&QoSMutatorRelayFailure{
		QoSMutatorBase: qosManager.createQoSMutatorBase(epoch, sessionId),
	})
}

func (qosManager *QoSManager) SetLastReputationQoSReport(epoch uint64, sessionId int64, report *pairingtypes.QualityOfServiceReport) {
	qosManager.mutate(&QoSMutatorSetReputation{
		QoSMutatorBase: qosManager.createQoSMutatorBase(epoch, sessionId),
		report:         report,
	})
}

func (qosManager *QoSManager) getQoSReport(epoch uint64, sessionId int64) *QoSReport {
	qosManager.lock.RLock()
	defer qosManager.lock.RUnlock()
	if qosManager.qosReports[epoch] == nil {
		return nil
	}
	return qosManager.qosReports[epoch][sessionId]
}

func (qosManager *QoSManager) GetLastQoSReport(epoch uint64, sessionId int64) *pairingtypes.QualityOfServiceReport {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return nil
	}
	return qosReport.getLastQoSReport()
}

func (qosManager *QoSManager) GetLastReputationQoSReport(epoch uint64, sessionId int64) *pairingtypes.QualityOfServiceReport {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return nil
	}
	return qosReport.getLastReputationQoSReport()
}

func (qosManager *QoSManager) GetAnsweredRelays(epoch uint64, sessionId int64) uint64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}
	return qosReport.getAnsweredRelays()
}

func (qosManager *QoSManager) GetTotalRelays(epoch uint64, sessionId int64) uint64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}
	return qosReport.getTotalRelays()
}

func (qosManager *QoSManager) GetSyncScoreSum(epoch uint64, sessionId int64) int64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}
	return qosReport.getSyncScoreSum()
}

func (qosManager *QoSManager) GetTotalSyncScore(epoch uint64, sessionId int64) int64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}
	return qosReport.getTotalSyncScore()
}
