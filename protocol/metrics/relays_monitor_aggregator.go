package metrics

import (
	"context"
	"sync"
	"time"
)

type RelaysMonitorAggregator struct {
	relaysMonitors  map[string]*RelaysMonitor // key is endpoint: chainID+apiInterface
	ticker          *time.Ticker
	rpcConsumerLogs *ConsumerMetricsManager
	lock            sync.RWMutex
}

func NewRelaysMonitorAggregator(interval time.Duration, rpcConsumerLogs *ConsumerMetricsManager) *RelaysMonitorAggregator {
	return &RelaysMonitorAggregator{
		relaysMonitors:  map[string]*RelaysMonitor{},
		ticker:          time.NewTicker(interval),
		rpcConsumerLogs: rpcConsumerLogs,
		lock:            sync.RWMutex{},
	}
}

func (rma *RelaysMonitorAggregator) RegisterRelaysMonitor(rpcEndpointKey string, relaysMonitor *RelaysMonitor) {
	rma.lock.Lock()
	rma.relaysMonitors[rpcEndpointKey] = relaysMonitor
	rma.lock.Unlock()
}

func (rma *RelaysMonitorAggregator) StartMonitoring(ctx context.Context) {
	go func() {
		for {
			select {
			case <-rma.ticker.C:
				go rma.runHealthCheck()
			case <-ctx.Done():
				rma.ticker.Stop()
				return
			}
		}
	}()
}

func (rma *RelaysMonitorAggregator) runHealthCheck() {
	rma.lock.RLock()
	defer rma.lock.RUnlock()

	// If at least one of the relays monitors is healthy, we set the status to TRUE, otherwise we set it to FALSE.
	for _, relaysMonitor := range rma.relaysMonitors {
		if relaysMonitor.IsHealthy() {
			rma.rpcConsumerLogs.SetEndpointsHealthChecksOkStatus(true)
			return
		}
	}

	rma.rpcConsumerLogs.SetEndpointsHealthChecksOkStatus(false)
}
