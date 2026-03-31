package qos

import (
	"sync"

	pairingtypes "github.com/lavanet/lava/v5/types/relay"
)

type QoSReport struct {
	lastQoSReport           *pairingtypes.QualityOfServiceReport
	lastReputationQoSReport *pairingtypes.QualityOfServiceReport
	latencyScoreList        []float64
	syncScoreSum            int64
	totalSyncScore          int64
	totalRelays             uint64
	answeredRelays          uint64
	lock                    sync.RWMutex
}

func (qr *QoSReport) mutate(mutator Mutator) {
	qr.lock.Lock()
	defer qr.lock.Unlock()
	mutator.Mutate(qr)
}

func (qr *QoSReport) getLastQoSReport() *pairingtypes.QualityOfServiceReport {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.lastQoSReport
}

func (qr *QoSReport) getLastReputationQoSReport() *pairingtypes.QualityOfServiceReport {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.lastReputationQoSReport
}

func (qr *QoSReport) getAnsweredRelays() uint64 {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.answeredRelays
}

func (qr *QoSReport) getTotalRelays() uint64 {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.totalRelays
}

func (qr *QoSReport) getSyncScoreSum() int64 {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.syncScoreSum
}

func (qr *QoSReport) getTotalSyncScore() int64 {
	qr.lock.RLock()
	defer qr.lock.RUnlock()
	return qr.totalSyncScore
}
