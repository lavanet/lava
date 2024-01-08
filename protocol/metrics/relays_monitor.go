package metrics

import (
	"context"
	"sync"
	"time"
)

type RelaysMonitor struct {
	chainID      string
	apiInterface string

	relaySender func() error
	ticker      *time.Ticker
	interval    time.Duration
	lock        sync.RWMutex

	isHealthy bool
}

func NewRelaysMonitor(interval time.Duration, chainID, apiInterface string, relaySender func() error) *RelaysMonitor {
	return &RelaysMonitor{
		chainID:      chainID,
		apiInterface: apiInterface,
		relaySender:  relaySender,
		ticker:       time.NewTicker(time.Second * 15),
		interval:     interval,
		lock:         sync.RWMutex{},
		isHealthy:    false,
	}
}

func (sem *RelaysMonitor) Start(ctx context.Context) {
	go sem.startInner(ctx)
}

func (sem *RelaysMonitor) startInner(ctx context.Context) {
	for {
		select {
		case <-sem.ticker.C:
			err := sem.relaySender()
			sem.isHealthy = err != nil
		case <-ctx.Done():
			sem.ticker.Stop()
			return
		}
	}
}

func (sem *RelaysMonitor) LogRelay(ctx context.Context) {
	sem.lock.Lock()
	sem.isHealthy = true
	sem.ticker.Reset(time.Second * 15)
	sem.lock.Unlock()
}

func (sem *RelaysMonitor) IsHealthy() bool {
	sem.lock.RLock()
	defer sem.lock.RUnlock()
	return sem.isHealthy
}
