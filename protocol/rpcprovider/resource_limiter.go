package rpcprovider

import (
	"context"
	"fmt"
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
	QueueSize     int           // Max queued requests
	Timeout       time.Duration // Max time in queue
}

// ResourceLimiter manages concurrent execution based on method type
type ResourceLimiter struct {
	heavySemaphore  *semaphore.Weighted
	normalSemaphore *semaphore.Weighted

	heavyQueue chan *queuedRequest

	config map[BucketType]*MethodConfig

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
	TotalRejected  uint64
	TotalQueued    uint64
	TotalTimeout   uint64
	HeavyInFlight  uint64
	NormalInFlight uint64
	lock           sync.RWMutex

	// Prometheus metrics
	rejectedRequestsMetric *prometheus.CounterVec
	queuedRequestsMetric   *prometheus.CounterVec
	timeoutRequestsMetric  *prometheus.CounterVec
	inFlightRequestsMetric *prometheus.GaugeVec
	queueWaitTimeMetric    *prometheus.HistogramVec
}

// NewResourceLimiter creates a new resource limiter
// endpointName: Unique name for this provider endpoint (used for metric differentiation)
// cuThreshold: CU value above which methods are classified as "heavy" (recommended: 100, minimum: 10)
// heavyMaxConcurrent: Max concurrent heavy (high-CU/debug/trace) method calls
// heavyQueueSize: Queue size for heavy methods
// normalMaxConcurrent: Max concurrent normal method calls
// Note: cuThreshold should be validated before calling this function
func NewResourceLimiter(enabled bool, endpointName string, cuThreshold uint64, heavyMaxConcurrent int64, heavyQueueSize int, normalMaxConcurrent int64) *ResourceLimiter {
	if !enabled {
		return &ResourceLimiter{enabled: false}
	}

	// Default configurations
	config := map[BucketType]*MethodConfig{
		BucketHeavy: {
			MaxConcurrent: heavyMaxConcurrent, // 2 concurrent heavy (debug/trace) calls
			QueueSize:     heavyQueueSize,     // Queue up to 5 more
			Timeout:       30 * time.Second,
		},
		BucketNormal: {
			MaxConcurrent: normalMaxConcurrent, // 100 concurrent normal calls
			QueueSize:     0,                   // No queue, reject if full
			Timeout:       0,
		},
	}

	// Create Prometheus metrics with endpoint name as constant label
	metricsInstance := createResourceLimiterMetrics(endpointName)

	rl := &ResourceLimiter{
		heavySemaphore:  semaphore.NewWeighted(config[BucketHeavy].MaxConcurrent),
		normalSemaphore: semaphore.NewWeighted(config[BucketNormal].MaxConcurrent),
		heavyQueue:      make(chan *queuedRequest, config[BucketHeavy].QueueSize),
		config:          config,
		cuThreshold:     cuThreshold,
		metrics:         metricsInstance,
		enabled:         true,
	}

	// Start queue worker for heavy bucket
	go rl.processQueue(BucketHeavy, rl.heavyQueue, rl.heavySemaphore)

	return rl
}

// createResourceLimiterMetrics creates metrics instance with endpoint-specific labels
// Uses ConstLabels to automatically differentiate metrics per endpoint without re-registration conflicts
func createResourceLimiterMetrics(endpointName string) *ResourceLimiterMetrics {
	// ConstLabels are applied to all metrics from this endpoint, preventing registration conflicts
	constLabels := prometheus.Labels{"endpoint": endpointName}

	return &ResourceLimiterMetrics{
		rejectedRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "lava_provider_resource_limiter_rejections_total",
			Help:        "Total number of requests rejected by resource limiter",
			ConstLabels: constLabels,
		}, []string{"bucket", "reason"}),
		queuedRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "lava_provider_resource_limiter_queued_total",
			Help:        "Total number of requests queued by resource limiter",
			ConstLabels: constLabels,
		}, []string{"bucket"}),
		timeoutRequestsMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "lava_provider_resource_limiter_timeouts_total",
			Help:        "Total number of requests that timed out in queue",
			ConstLabels: constLabels,
		}, []string{"bucket"}),
		inFlightRequestsMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name:        "lava_provider_resource_limiter_in_flight",
			Help:        "Number of requests currently executing",
			ConstLabels: constLabels,
		}, []string{"bucket"}),
		queueWaitTimeMetric: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "lava_provider_resource_limiter_queue_wait_seconds",
			Help:        "Time requests spent waiting in queue",
			Buckets:     prometheus.ExponentialBuckets(0.1, 2, 10),
			ConstLabels: constLabels,
		}, []string{"bucket"}),
	}
}

// selectBucket determines the resource category for a method
// First checks for batch methods (with &), then CU, then method name prefixes
func (rl *ResourceLimiter) selectBucket(computeUnits uint64, methodName string) BucketType {
	// Priority 0: Check if method contains ampersands (batch methods)
	// Batch methods should always use the normal bucket, bypassing resource limiter
	if strings.Contains(methodName, "&") {
		return BucketNormal
	}

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

	// Try immediate execution
	sem := rl.getSemaphore(bucket)
	if sem.TryAcquire(1) {
		return rl.executeWithSemaphore(bucket, sem, execute)
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
	execute func() error,
) error {
	defer sem.Release(1)

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
	// Create a timeout context so the request gets canceled if it waits too long
	// This ensures processQueue will skip execution for timed-out requests
	queueCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel() // Always cancel to free resources

	qr := &queuedRequest{
		ctx:      queueCtx,
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

		// Wait for result or context cancellation (parent or timeout)
		select {
		case err := <-qr.result:
			return err
		case <-queueCtx.Done():
			// Context canceled (either parent or timeout)
			// Check if it was a timeout specifically
			if ctx.Err() == nil && queueCtx.Err() == context.DeadlineExceeded {
				rl.metrics.incrementTimeoutWithLabels(bucket)
				return fmt.Errorf("request timeout in queue after %v", cfg.Timeout)
			}
			return queueCtx.Err()
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
	for qr := range queue {
		// Check if context was canceled/timed out before we process
		// This prevents executing requests that already timed out in the queue
		if qr.ctx.Err() != nil {
			utils.LavaFormatDebug("Skipping queued request - context already canceled",
				utils.LogAttr("bucket", bucket.String()),
				utils.LogAttr("error", qr.ctx.Err().Error()),
				utils.LogAttr("wait_time", time.Since(qr.enqueued)),
			)
			qr.result <- qr.ctx.Err()
			continue
		}

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
		err := rl.executeWithSemaphore(bucket, sem, qr.execute)
		qr.result <- err
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

func (m *ResourceLimiterMetrics) recordQueueWaitTime(bucket BucketType, duration time.Duration) {
	if m.queueWaitTimeMetric != nil {
		m.queueWaitTimeMetric.WithLabelValues(bucket.String()).Observe(duration.Seconds())
	}
}
