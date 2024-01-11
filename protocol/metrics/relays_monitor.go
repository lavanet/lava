package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type RelaysMonitor struct {
	chainID      string
	apiInterface string

	relaySender func() (bool, error)
	ticker      *time.Ticker
	interval    time.Duration
	lock        sync.RWMutex

	isHealthy uint32
}

func NewRelaysMonitor(interval time.Duration, chainID, apiInterface string, relaySender func() (bool, error)) *RelaysMonitor {
	return &RelaysMonitor{
		chainID:      chainID,
		apiInterface: apiInterface,
		relaySender:  relaySender,
		ticker:       time.NewTicker(interval),
		interval:     interval,
		lock:         sync.RWMutex{},
		isHealthy:    1,
	}
}

func (sem *RelaysMonitor) Start(ctx context.Context) {
	go sem.startInner(ctx)
}

func (sem *RelaysMonitor) startInner(ctx context.Context) {
	for {
		select {
		case <-sem.ticker.C:
			success, _ := sem.relaySender()
			sem.storeHealthStatus(success)
		case <-ctx.Done():
			sem.ticker.Stop()
			return
		}
	}
}

func (sem *RelaysMonitor) LogRelay() {
	sem.lock.Lock()
	defer sem.lock.Unlock()

	sem.storeHealthStatus(true)
	sem.ticker.Reset(sem.interval)
}

func (sem *RelaysMonitor) IsHealthy() bool {
	return sem.loadHealthStatus()
}

func (sem *RelaysMonitor) storeHealthStatus(healthy bool) {
	value := uint32(0)
	if healthy {
		value = 1
	}
	atomic.StoreUint32(&sem.isHealthy, value)
}

func (sem *RelaysMonitor) loadHealthStatus() bool {
	return atomic.LoadUint32(&sem.isHealthy) == 1
}
