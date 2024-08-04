package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v2/utils"
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

func NewRelaysMonitor(interval time.Duration, chainID, apiInterface string) *RelaysMonitor {
	return &RelaysMonitor{
		chainID:      chainID,
		apiInterface: apiInterface,
		ticker:       time.NewTicker(interval),
		interval:     interval,
		isHealthy:    1, // setting process to healthy by default, after init relays we know if its truly healthy or not.
	}
}

func (sem *RelaysMonitor) SetRelaySender(relaySender func() (bool, error)) {
	if sem == nil {
		return
	}

	sem.lock.Lock()
	defer sem.lock.Unlock()
	sem.relaySender = relaySender
}

func (sem *RelaysMonitor) Start(ctx context.Context) {
	if sem == nil {
		return
	}

	// We run the relaySender right away, because we call this function from the RPCConsumerServer on it's initialization.
	// This means that the relaySender will be called right away, and we don't have to wait for the ticker to fire.
	// There is a difference between the first call to relaySender and the subsequent calls.
	// To see the difference, please refer to the call to NewRelaysMonitor in RPCConsumerServer.

	go func() {
		success, _ := sem.relaySender()
		sem.storeHealthStatus(success)
	}()
	go sem.startInner(ctx)
}

func (sem *RelaysMonitor) startInner(ctx context.Context) {
	for {
		select {
		case <-sem.ticker.C:
			success, _ := sem.relaySender()
			utils.LavaFormatInfo("Health Check Interval Check",
				utils.LogAttr("chain", sem.chainID),
				utils.LogAttr("apiInterface", sem.apiInterface),
				utils.LogAttr("health result", success),
			)
			sem.storeHealthStatus(success)
		case <-ctx.Done():
			sem.ticker.Stop()
			return
		}
	}
}

func (sem *RelaysMonitor) LogRelay() {
	if sem == nil {
		return
	}

	sem.lock.Lock()
	defer sem.lock.Unlock()

	sem.storeHealthStatus(true)
	sem.ticker.Reset(sem.interval)
}

func (sem *RelaysMonitor) IsHealthy() bool {
	if sem == nil {
		return false
	}

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
