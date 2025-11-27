package rpcprovider

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/semaphore"
)

// BucketType represents the resource bucket category for request classification
type BucketType string

const (
	// BucketHeavy represents high-resource requests (high CU, debug/trace methods)
	BucketHeavy BucketType = "heavy"
	// BucketNormal represents standard requests
	BucketNormal BucketType = "normal"
)

// String returns the string representation of the bucket type
func (b BucketType) String() string {
	return string(b)
}

// MethodConfig defines resource limits for specific method types
type MethodConfig struct {
	MaxConcurrent int64         // Max concurrent executions
	MemoryPerCall uint64        // Estimated memory per call (bytes)
	QueueSize     int           // Max queued requests
	Timeout       time.Duration // Max time in queue
}

// ResourceLimiter manages concurrent execution based on method type and memory
type ResourceLimiter struct {
	heavySemaphore  *semaphore.Weighted
	normalSemaphore *semaphore.Weighted

	heavyQueue chan *queuedRequest

	config map[BucketType]*MethodConfig

	// Memory monitoring
	memoryThreshold uint64 // Max memory usage (bytes)
	memoryLock      sync.RWMutex
	currentMemory   uint64

	// Metrics
	metrics *ResourceLimiterMetrics

	// CU threshold for determining heavy methods
	cuThreshold uint64

	enabled bool
}

type queuedRequest struct {
	ctx      context.Context
	execute  func() error
	result   chan error
	enqueued time.Time
}

type ResourceLimiterMetrics struct {
	TotalRejected      uint64
	TotalQueued        uint64
	TotalTimeout       uint64
	HeavyInFlight      uint64
	NormalInFlight     uint64
	EstimatedMemoryUse uint64
	lock               sync.RWMutex

	// Prometheus metrics
	rejectedRequestsMetric *prometheus.CounterVec
	queuedRequestsMetric   *prometheus.CounterVec
	timeoutRequestsMetric  *prometheus.CounterVec
	inFlightRequestsMetric *prometheus.GaugeVec
	estimatedMemoryMetric  prometheus.Gauge
	queueWaitTimeMetric    *prometheus.HistogramVec
}

// NewResourceLimiter creates a new resource limiter
// cuThreshold: CU value above which methods are classified as "heavy" (recommended: 100, minimum: 10)
// heavyMaxConcurrent: Max concurrent heavy (high-CU/debug/trace) method calls
// heavyQueueSize: Queue size for heavy methods
// normalMaxConcurrent: Max concurrent normal method calls
// Note: cuThreshold should be validated before calling this function
func NewResourceLimiter(enabled bool, memoryThresholdGB uint64, cuThreshold uint64, heavyMaxConcurrent int64, heavyQueueSize int, normalMaxConcurrent int64) *ResourceLimiter {
	if !enabled {
		return &ResourceLimiter{enabled: false}
	}

	// Default configurations
	config := map[BucketType]*MethodConfig{
		BucketHeavy: {
			MaxConcurrent: heavyMaxConcurrent, // 2 concurrent heavy (debug/trace) calls
			MemoryPerCall: 512 * 1024 * 1024,  // Estimate 512MB per heavy call
			QueueSize:     heavyQueueSize,     // Queue up to 5 more
			Timeout:       30 * time.Second,
		},
		BucketNormal: {
			MaxConcurrent: normalMaxConcurrent, // 100 concurrent normal calls
			MemoryPerCall: 1 * 1024 * 1024,     // 1MB per normal call
			QueueSize:     0,                   // No queue, reject if full
			Timeout:       0,
		},
	}

	memoryThreshold := memoryThresholdGB * 1024 * 1024 * 1024

	// Create Prometheus metrics (with nil checks for testing)
	metricsInstance := createResourceLimiterMetrics()

	rl := &ResourceLimiter{
		heavySemaphore:  semaphore.NewWeighted(config[BucketHeavy].MaxConcurrent),
		normalSemaphore: semaphore.NewWeighted(config[BucketNormal].MaxConcurrent),
		heavyQueue:      make(chan *queuedRequest, config[BucketHeavy].QueueSize),
		config:          config,
		memoryThreshold: memoryThreshold,
		cuThreshold:     cuThreshold,
		metrics:         metricsInstance,
		enabled:         true,
	}

	// Start queue worker for heavy bucket
	go rl.processQueue(BucketHeavy, rl.heavyQueue, rl.heavySemaphore)

	// Start memory monitor
	go rl.monitorMemory()

	return rl
}

// createResourceLimiterMetrics creates metrics instance
// Separated for testability
func createResourceLimiterMetrics() *ResourceLimiterMetrics {
	return &ResourceLimiterMetrics{
		rejectedRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lava_provider_resource_limiter_rejections_total",
			Help: "Total number of requests rejected by resource limiter",
		}, []string{"bucket", "reason"}),
		queuedRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lava_provider_resource_limiter_queued_total",
			Help: "Total number of requests queued by resource limiter",
		}, []string{"bucket"}),
		timeoutRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lava_provider_resource_limiter_timeouts_total",
			Help: "Total number of requests that timed out in queue",
		}, []string{"bucket"}),
		inFlightRequestsMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lava_provider_resource_limiter_in_flight",
			Help: "Number of requests currently executing",
		}, []string{"bucket"}),
		estimatedMemoryMetric: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "lava_provider_resource_limiter_memory_bytes",
			Help: "Estimated memory usage by resource limiter",
		}),
		queueWaitTimeMetric: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lava_provider_resource_limiter_queue_wait_seconds",
			Help:    "Time requests spent waiting in queue",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		}, []string{"bucket"}),
	}
}

// newResourceLimiterForTesting creates a limiter without Prometheus metrics for testing
func newResourceLimiterForTesting(enabled bool, memoryThresholdGB uint64, cuThreshold uint64) *ResourceLimiter {
	if !enabled {
		return &ResourceLimiter{enabled: false}
	}

	config := map[BucketType]*MethodConfig{
		BucketHeavy: {
			MaxConcurrent: 2,
			MemoryPerCall: 512 * 1024 * 1024,
			QueueSize:     5,
			Timeout:       30 * time.Second,
		},
		BucketNormal: {
			MaxConcurrent: 100,
			MemoryPerCall: 1 * 1024 * 1024,
			QueueSize:     0,
			Timeout:       0,
		},
	}

	memoryThreshold := memoryThresholdGB * 1024 * 1024 * 1024

	// Create minimal metrics without Prometheus registration
	metricsInstance := &ResourceLimiterMetrics{}

	rl := &ResourceLimiter{
		heavySemaphore:  semaphore.NewWeighted(config[BucketHeavy].MaxConcurrent),
		normalSemaphore: semaphore.NewWeighted(config[BucketNormal].MaxConcurrent),
		heavyQueue:      make(chan *queuedRequest, config[BucketHeavy].QueueSize),
		config:          config,
		memoryThreshold: memoryThreshold,
		cuThreshold:     cuThreshold,
		metrics:         metricsInstance,
		enabled:         true,
	}

	// Start queue worker for heavy bucket
	go rl.processQueue(BucketHeavy, rl.heavyQueue, rl.heavySemaphore)

	// Start memory monitor
	go rl.monitorMemory()

	return rl
}

// selectBucket determines the resource category for a method
// First checks CU, then checks method name prefixes
func (rl *ResourceLimiter) selectBucket(computeUnits uint64, methodName string) BucketType {
	// Priority 1: Check if CU is high (primary indicator)
	if computeUnits >= rl.cuThreshold {
		return BucketHeavy
	}

	// Priority 2: Check method name prefixes (secondary check)
	lower := strings.ToLower(methodName)
	if strings.HasPrefix(lower, "debug_") || strings.HasPrefix(lower, "trace_") {
		return BucketHeavy
	}

	return BucketNormal
}

// Acquire attempts to acquire resources for executing a request
func (rl *ResourceLimiter) Acquire(ctx context.Context, computeUnits uint64, methodName string, execute func() error) error {
	if rl == nil || !rl.enabled {
		return execute() // Pass through if nil or disabled
	}

	bucket := rl.selectBucket(computeUnits, methodName)
	cfg := rl.config[bucket]

	// Check memory before admitting
	if !rl.canAdmitRequest(cfg.MemoryPerCall) {
		rl.metrics.incrementRejectedWithLabels(bucket, "memory_limit")
		return fmt.Errorf("provider memory limit reached, rejecting %s request", bucket)
	}

	// Try immediate execution
	sem := rl.getSemaphore(bucket)
	if sem.TryAcquire(1) {
		return rl.executeWithSemaphore(bucket, sem, cfg, execute)
	}

	// If can't acquire immediately, try queue (if available)
	if cfg.QueueSize > 0 {
		return rl.enqueueRequest(ctx, bucket, cfg, execute)
	}

	// No queue, reject
	rl.metrics.incrementRejectedWithLabels(bucket, "max_concurrent")
	return fmt.Errorf("provider busy: max concurrent %s requests (%d) reached",
		bucket, cfg.MaxConcurrent)
}

func (rl *ResourceLimiter) getSemaphore(bucket BucketType) *semaphore.Weighted {
	if bucket == BucketHeavy {
		return rl.heavySemaphore
	}
	return rl.normalSemaphore
}

func (rl *ResourceLimiter) executeWithSemaphore(
	bucket BucketType,
	sem *semaphore.Weighted,
	cfg *MethodConfig,
	execute func() error,
) error {
	defer sem.Release(1)

	// Reserve memory
	rl.reserveMemory(cfg.MemoryPerCall)
	defer rl.releaseMemory(cfg.MemoryPerCall)

	// Update metrics
	rl.metrics.incrementInFlight(bucket)
	defer rl.metrics.decrementInFlight(bucket)

	// Execute
	startTime := time.Now()
	err := execute()
	duration := time.Since(startTime)

	utils.LavaFormatDebug("Resource limiter executed request",
		utils.LogAttr("bucket", bucket.String()),
		utils.LogAttr("duration", duration),
		utils.LogAttr("error", err != nil),
	)

	return err
}

func (rl *ResourceLimiter) enqueueRequest(
	ctx context.Context,
	bucket BucketType,
	cfg *MethodConfig,
	execute func() error,
) error {
	qr := &queuedRequest{
		ctx:      ctx,
		execute:  execute,
		result:   make(chan error, 1),
		enqueued: time.Now(),
	}

	queue := rl.getQueue(bucket)

	// Try to enqueue with timeout
	select {
	case queue <- qr:
		rl.metrics.incrementQueuedWithLabels(bucket)

		// Log current queue depth after enqueuing
		queueDepth := len(queue)
		utils.LavaFormatDebug("Request enqueued",
			utils.LogAttr("bucket", bucket.String()),
			utils.LogAttr("queue_depth", queueDepth),
			utils.LogAttr("max_queue_size", cfg.QueueSize),
		)

		// Wait for result or timeout
		select {
		case err := <-qr.result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cfg.Timeout):
			rl.metrics.incrementTimeoutWithLabels(bucket)
			return fmt.Errorf("request timeout in queue after %v", cfg.Timeout)
		}

	default:
		// Queue is full
		rl.metrics.incrementRejectedWithLabels(bucket, "queue_full")
		queueDepth := len(queue)
		utils.LavaFormatWarning("Request rejected - queue full", nil,
			utils.LogAttr("bucket", bucket.String()),
			utils.LogAttr("queue_depth", queueDepth),
			utils.LogAttr("max_queue_size", cfg.QueueSize),
		)
		return fmt.Errorf("provider queue full: %s queue size %d", bucket, cfg.QueueSize)
	}
}

func (rl *ResourceLimiter) getQueue(bucket BucketType) chan *queuedRequest {
	if bucket == BucketHeavy {
		return rl.heavyQueue
	}
	return nil
}

func (rl *ResourceLimiter) processQueue(
	bucket BucketType,
	queue chan *queuedRequest,
	sem *semaphore.Weighted,
) {
	cfg := rl.config[bucket]

	for qr := range queue {
		// Wait for semaphore
		if err := sem.Acquire(qr.ctx, 1); err != nil {
			qr.result <- err
			continue
		}

		waitTime := time.Since(qr.enqueued)
		queueDepth := len(queue) // Remaining requests in queue after dequeuing this one
		rl.metrics.recordQueueWaitTime(bucket, waitTime)
		utils.LavaFormatDebug("Processing queued request",
			utils.LogAttr("bucket", bucket.String()),
			utils.LogAttr("waitTime", waitTime),
			utils.LogAttr("remaining_queue_depth", queueDepth),
		)

		// Execute
		err := rl.executeWithSemaphore(bucket, sem, cfg, qr.execute)
		qr.result <- err
	}
}

// Memory management
func (rl *ResourceLimiter) canAdmitRequest(estimatedMemory uint64) bool {
	rl.memoryLock.RLock()
	defer rl.memoryLock.RUnlock()

	// Check if adding this request would exceed threshold
	if rl.currentMemory+estimatedMemory > rl.memoryThreshold {
		return false
	}

	// Also check actual system memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// If heap usage > 80% of threshold, reject
	return m.HeapAlloc <= (rl.memoryThreshold * 80 / 100)
}

func (rl *ResourceLimiter) reserveMemory(amount uint64) {
	rl.memoryLock.Lock()
	defer rl.memoryLock.Unlock()
	rl.currentMemory += amount
	rl.metrics.updateMemory(rl.currentMemory)
}

func (rl *ResourceLimiter) releaseMemory(amount uint64) {
	rl.memoryLock.Lock()
	defer rl.memoryLock.Unlock()
	if rl.currentMemory >= amount {
		rl.currentMemory -= amount
	} else {
		rl.currentMemory = 0
	}
	rl.metrics.updateMemory(rl.currentMemory)
}

func (rl *ResourceLimiter) monitorMemory() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		rl.memoryLock.RLock()
		reserved := rl.currentMemory
		rl.memoryLock.RUnlock()

		// Get current queue depths
		heavyQueueDepth := len(rl.heavyQueue)
		normalQueueDepth := 0 // Normal bucket has no queue (QueueSize: 0)

		utils.LavaFormatDebug("Provider memory status",
			utils.LogAttr("heap_alloc_mb", m.HeapAlloc/(1024*1024)),
			utils.LogAttr("reserved_mb", reserved/(1024*1024)),
			utils.LogAttr("threshold_mb", rl.memoryThreshold/(1024*1024)),
			utils.LogAttr("heavy_in_flight", rl.metrics.getInFlight(BucketHeavy)),
			utils.LogAttr("normal_in_flight", rl.metrics.getInFlight(BucketNormal)),
			utils.LogAttr("heavy_queue_depth", heavyQueueDepth),
			utils.LogAttr("normal_queue_depth", normalQueueDepth),
		)
	}
}

// Metrics methods
func (m *ResourceLimiterMetrics) incrementRejected() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.TotalRejected++
}

func (m *ResourceLimiterMetrics) incrementRejectedWithLabels(bucket BucketType, reason string) {
	m.incrementRejected()
	if m.rejectedRequestsMetric != nil {
		m.rejectedRequestsMetric.WithLabelValues(bucket.String(), reason).Inc()
	}
}

func (m *ResourceLimiterMetrics) incrementQueued() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.TotalQueued++
}

func (m *ResourceLimiterMetrics) incrementQueuedWithLabels(bucket BucketType) {
	m.incrementQueued()
	if m.queuedRequestsMetric != nil {
		m.queuedRequestsMetric.WithLabelValues(bucket.String()).Inc()
	}
}

func (m *ResourceLimiterMetrics) incrementTimeout() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.TotalTimeout++
}

func (m *ResourceLimiterMetrics) incrementTimeoutWithLabels(bucket BucketType) {
	m.incrementTimeout()
	if m.timeoutRequestsMetric != nil {
		m.timeoutRequestsMetric.WithLabelValues(bucket.String()).Inc()
	}
}

func (m *ResourceLimiterMetrics) incrementInFlight(bucket BucketType) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if bucket == BucketHeavy {
		m.HeavyInFlight++
	} else {
		m.NormalInFlight++
	}
	if m.inFlightRequestsMetric != nil {
		m.inFlightRequestsMetric.WithLabelValues(bucket.String()).Inc()
	}
}

func (m *ResourceLimiterMetrics) decrementInFlight(bucket BucketType) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if bucket == BucketHeavy {
		if m.HeavyInFlight > 0 {
			m.HeavyInFlight--
		}
	} else if m.NormalInFlight > 0 {
		m.NormalInFlight--
	}
	if m.inFlightRequestsMetric != nil {
		m.inFlightRequestsMetric.WithLabelValues(bucket.String()).Dec()
	}
}

func (m *ResourceLimiterMetrics) getInFlight(bucket BucketType) uint64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if bucket == BucketHeavy {
		return m.HeavyInFlight
	}
	return m.NormalInFlight
}

func (m *ResourceLimiterMetrics) updateMemory(current uint64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.EstimatedMemoryUse = current
	if m.estimatedMemoryMetric != nil {
		m.estimatedMemoryMetric.Set(float64(current))
	}
}

func (m *ResourceLimiterMetrics) recordQueueWaitTime(bucket BucketType, duration time.Duration) {
	if m.queueWaitTimeMetric != nil {
		m.queueWaitTimeMetric.WithLabelValues(bucket.String()).Observe(duration.Seconds())
	}
}
