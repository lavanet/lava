package qos

import (
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
)

type QoSReport struct {
	lastQoSReport           *pairingtypes.QualityOfServiceReport
	lastReputationQoSReport *pairingtypes.QualityOfServiceReport
	latencyScoreList        []sdk.Dec
	syncScoreSum            int64
	totalSyncScore          int64
	totalRelays             uint64
	answeredRelays          uint64
	lock                    sync.RWMutex
}

type DoneChan <-chan struct{}

type QoSManager struct {
	qosReports    map[uint64]map[int64]*QoSReport // first key is the epoch, second key is the session id
	mutatorsQueue chan Mutator
	lock          sync.RWMutex
}

func NewQoSManager() *QoSManager {
	qosManager := &QoSManager{}
	qosManager.qosReports = make(map[uint64]map[int64]*QoSReport)
	qosManager.mutatorsQueue = make(chan Mutator, 10000000) // Buffer of 10 Million mutators
	go qosManager.processMutations()
	return qosManager
}

func (qosManager *QoSManager) processMutations() {
	for mutator := range qosManager.mutatorsQueue {
		epoch, sessionId := mutator.GetEpochAndSessionId()
		qosReport := qosManager.fetchOrSetSessionFromMap(epoch, sessionId)
		func() {
			qosReport.lock.Lock()
			defer qosReport.lock.Unlock()
			mutator.Mutate(qosReport)
		}()
	}
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

func (qosManager *QoSManager) createQoSMutatorBase(epoch uint64, sessionId int64) (*QoSMutatorBase, chan struct{}) {
	doneChan := make(chan struct{}, 1) // Must be buffered to avoid freezing the queue
	qosMutatorBase := &QoSMutatorBase{
		epoch:     epoch,
		sessionId: sessionId,
		doneChan:  doneChan,
	}
	return qosMutatorBase, doneChan
}

func (qosManager *QoSManager) CalculateQoS(epoch uint64, sessionId int64, providerAddress string, latency, expectedLatency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) DoneChan {
	qosMutatorBase, doneChan := qosManager.createQoSMutatorBase(epoch, sessionId)
	qosManager.mutatorsQueue <- &QoSMutatorRelaySuccess{
		QoSMutatorBase:   *qosMutatorBase,
		providerAddress:  providerAddress,
		latency:          latency,
		expectedLatency:  expectedLatency,
		blockHeightDiff:  blockHeightDiff,
		numOfProviders:   numOfProviders,
		servicersToCount: servicersToCount,
	}
	return doneChan
}

func (qosManager *QoSManager) AddFailedRelay(epoch uint64, sessionId int64) DoneChan {
	qosMutatorBase, doneChan := qosManager.createQoSMutatorBase(epoch, sessionId)
	qosManager.mutatorsQueue <- &QoSMutatorRelayFailure{
		QoSMutatorBase: *qosMutatorBase,
	}
	return doneChan
}

func (qosManager *QoSManager) SetLastReputationQoSReport(epoch uint64, sessionId int64, report *pairingtypes.QualityOfServiceReport) DoneChan {
	qosMutatorBase, doneChan := qosManager.createQoSMutatorBase(epoch, sessionId)
	qosManager.mutatorsQueue <- &QoSMutatorSetReputation{
		QoSMutatorBase: *qosMutatorBase,
		report:         report,
	}
	return doneChan
}

func (qosManager *QoSManager) getQoSReport(epoch uint64, sessionId int64) *QoSReport {
	qosManager.lock.RLock()
	defer qosManager.lock.RUnlock()
	if qosManager.qosReports[epoch] == nil || qosManager.qosReports[epoch][sessionId] == nil {
		return nil
	}
	return qosManager.qosReports[epoch][sessionId]
}

func (qosManager *QoSManager) GetLastQoSReport(epoch uint64, sessionId int64) *pairingtypes.QualityOfServiceReport {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return nil
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.lastQoSReport
}

func (qosManager *QoSManager) GetLastReputationQoSReport(epoch uint64, sessionId int64) *pairingtypes.QualityOfServiceReport {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return nil
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.lastReputationQoSReport
}

func (qosManager *QoSManager) GetAnsweredRelays(epoch uint64, sessionId int64) uint64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.answeredRelays
}

func (qosManager *QoSManager) GetTotalRelays(epoch uint64, sessionId int64) uint64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.totalRelays
}

func (qosManager *QoSManager) GetSyncScoreSum(epoch uint64, sessionId int64) int64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.syncScoreSum
}

func (qosManager *QoSManager) GetTotalSyncScore(epoch uint64, sessionId int64) int64 {
	qosReport := qosManager.getQoSReport(epoch, sessionId)
	if qosReport == nil {
		return 0
	}

	qosReport.lock.RLock()
	defer qosReport.lock.RUnlock()
	return qosReport.totalSyncScore
}
