package cache

import (
	"net/http"
	"sync"

	"github.com/lavanet/lava/v5/utils"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DisabledFlagOption = "disabled"
	totalHitsKey       = "total_hits"
	totalMissesKey     = "total_misses"
	totalExpiredKey    = "total_expired"
)

type CacheMetrics struct {
	lock             sync.RWMutex
	totalHits        *prometheus.CounterVec
	totalMisses      *prometheus.CounterVec
	totalExpired     *prometheus.CounterVec
	currentCacheSize *prometheus.Gauge
	apiSpecifics     *prometheus.GaugeVec
}

func NewCacheMetricsServer(listenAddress string) *CacheMetrics {
	if listenAddress == DisabledFlagOption {
		utils.LavaFormatWarning("prometheus endpoint inactive, option is disabled", nil)
		return nil
	}
	totalHits := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_total_hits",
		Help: "The total number of hits the cache server managed to reply.",
	}, []string{totalHitsKey})

	totalMisses := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_total_misses",
		Help: "The total number of misses the cache server could not reply.",
	}, []string{totalMissesKey})

	totalExpired := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_total_expired",
		Help: "The total number of expired/evicted keys from the cache.",
	}, []string{totalExpiredKey})

	currentCacheSize := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cache_current_size",
		Help: "The current number of items in the cache.",
	})

	apiSpecificsLabelNames := []string{"requested_block", "chain_id", "result"}
	apiSpecifics := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_api_specifics",
		Help: "api specific information",
	}, apiSpecificsLabelNames)

	prometheus.MustRegister(totalHits)
	prometheus.MustRegister(totalMisses)
	prometheus.MustRegister(totalExpired)
	prometheus.MustRegister(currentCacheSize)
	prometheus.MustRegister(apiSpecifics)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: listenAddress})
		http.ListenAndServe(listenAddress, nil)
	}()

	// Initialize metrics to 0 so they appear immediately in /metrics endpoint
	totalExpired.WithLabelValues(totalExpiredKey).Add(0)
	currentCacheSize.Set(0)

	return &CacheMetrics{
		totalHits:        totalHits,
		totalMisses:      totalMisses,
		totalExpired:     totalExpired,
		currentCacheSize: &currentCacheSize,
		apiSpecifics:     apiSpecifics,
	}
}

func (c *CacheMetrics) addHit() {
	if c == nil {
		return
	}
	c.totalHits.WithLabelValues(totalHitsKey).Add(1)
}

func (c *CacheMetrics) addMiss() {
	if c == nil {
		return
	}
	c.totalMisses.WithLabelValues(totalMissesKey).Add(1)
}

func (c *CacheMetrics) AddApiSpecific(block int64, chainId string, hit bool) {
	if c == nil {
		return
	}
	requestedBlock := "specific"
	if spectypes.LATEST_BLOCK == block {
		requestedBlock = "latest"
	} else if spectypes.NOT_APPLICABLE == block {
		requestedBlock = "not applicable"
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if hit {
		c.apiSpecificWithMethodIfNeeded(requestedBlock, chainId, "hit")
		c.addHit()
	} else {
		c.apiSpecificWithMethodIfNeeded(requestedBlock, chainId, "miss")
		c.addMiss()
	}
}

func (c *CacheMetrics) apiSpecificWithMethodIfNeeded(requestedBlock, chainId, hitOrMiss string) {
	c.apiSpecifics.WithLabelValues(requestedBlock, chainId, hitOrMiss).Add(1) // Removed "specifics" label
}

func (c *CacheMetrics) AddExpired() {
	if c == nil {
		return
	}
	c.totalExpired.WithLabelValues(totalExpiredKey).Add(1)
}

func (c *CacheMetrics) SetCacheSize(size int64) {
	if c == nil {
		return
	}
	(*c.currentCacheSize).Set(float64(size))
}
