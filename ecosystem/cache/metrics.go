package cache

import (
	"net/http"
	"sync"

	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DisabledFlagOption = "disabled"
	totalHitsKey       = "total_hits"
	totalMissesKey     = "total_misses"
)

type CacheMetrics struct {
	lock         sync.RWMutex
	totalHits    *prometheus.CounterVec
	totalMisses  *prometheus.CounterVec
	apiSpecifics *prometheus.GaugeVec
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

	apiSpecifics := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_api_specifics",
		Help: "api specific information",
	}, []string{"requested_block", "chain_id", "method", "api_interface", "result"})

	prometheus.MustRegister(totalHits)
	prometheus.MustRegister(totalMisses)
	prometheus.MustRegister(apiSpecifics)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		utils.LavaFormatInfo("prometheus endpoint listening", utils.Attribute{Key: "Listen Address", Value: listenAddress})
		http.ListenAndServe(listenAddress, nil)
	}()
	return &CacheMetrics{
		totalHits:    totalHits,
		totalMisses:  totalMisses,
		apiSpecifics: apiSpecifics,
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

func (c *CacheMetrics) AddApiSpecific(block int64, chainId string, method string, apiInterface string, hit bool) {
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
		c.apiSpecifics.WithLabelValues(requestedBlock, chainId, method, apiInterface, "hit").Add(1) // Removed "specifics" label
		c.addHit()
	} else {
		c.apiSpecifics.WithLabelValues(requestedBlock, chainId, method, apiInterface, "miss").Add(1) // Removed "specifics" label
		c.addMiss()
	}
}
