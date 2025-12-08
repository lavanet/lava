package memoryutils

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

const (
	// GC monitoring defaults
	gcMonitoringCheckInterval = 5 * time.Second
)

// memoryLogsEnabled is an atomic flag to control whether memory logs are enabled
var memoryLogsEnabled int32

// MemoryStats holds current memory statistics
type MemoryStats struct {
	AllocMB        float64
	TotalAllocMB   float64
	SysMB          float64
	NumGC          uint32
	HeapAllocMB    float64
	HeapSysMB      float64
	HeapIdleMB     float64
	HeapInUseMB    float64
	StackInUseMB   float64
	GoroutineCount int
}

// GetMemoryStats returns current memory statistics
func GetMemoryStats() *MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryStats{
		AllocMB:        float64(m.Alloc) / (1024 * 1024),
		TotalAllocMB:   float64(m.TotalAlloc) / (1024 * 1024),
		SysMB:          float64(m.Sys) / (1024 * 1024),
		NumGC:          m.NumGC,
		HeapAllocMB:    float64(m.HeapAlloc) / (1024 * 1024),
		HeapSysMB:      float64(m.HeapSys) / (1024 * 1024),
		HeapIdleMB:     float64(m.HeapIdle) / (1024 * 1024),
		HeapInUseMB:    float64(m.HeapInuse) / (1024 * 1024),
		StackInUseMB:   float64(m.StackInuse) / (1024 * 1024),
		GoroutineCount: runtime.NumGoroutine(),
	}
}

// EnableMemoryLogs enables memory tracking logs
func EnableMemoryLogs(enabled bool) {
	if enabled {
		atomic.StoreInt32(&memoryLogsEnabled, 1)
	} else {
		atomic.StoreInt32(&memoryLogsEnabled, 0)
	}
}

// IsMemoryLogsEnabled returns whether memory logs are enabled
func IsMemoryLogsEnabled() bool {
	return atomic.LoadInt32(&memoryLogsEnabled) == 1
}

// LogMemoryAndMessageSize logs memory stats and message size with context
func LogMemoryAndMessageSize(ctx context.Context, stage string, messageSize int, additionalAttrs ...utils.Attribute) {
	if !IsMemoryLogsEnabled() {
		return
	}
	memStats := GetMemoryStats()

	attrs := []utils.Attribute{
		{Key: "stage", Value: stage},
		{Key: "message_size_bytes", Value: messageSize},
		{Key: "message_size_kb", Value: float64(messageSize) / 1024},
		{Key: "alloc_mb", Value: memStats.AllocMB},
		{Key: "total_alloc_mb", Value: memStats.TotalAllocMB},
		{Key: "sys_mb", Value: memStats.SysMB},
		{Key: "heap_alloc_mb", Value: memStats.HeapAllocMB},
		{Key: "heap_inuse_mb", Value: memStats.HeapInUseMB},
		{Key: "stack_inuse_mb", Value: memStats.StackInUseMB},
		{Key: "num_goroutines", Value: memStats.GoroutineCount},
		{Key: "num_gc", Value: memStats.NumGC},
		{Key: "GUID", Value: ctx},
	}

	// Add request ID if available
	if reqID, ok := utils.GetRequestId(ctx); ok {
		attrs = append(attrs, utils.Attribute{Key: utils.KEY_REQUEST_ID, Value: reqID})
	}

	// Add task ID if available
	if taskID, ok := utils.GetTaskId(ctx); ok {
		attrs = append(attrs, utils.Attribute{Key: utils.KEY_TASK_ID, Value: taskID})
	}

	// Add tx ID if available
	if txID, ok := utils.GetTxId(ctx); ok {
		attrs = append(attrs, utils.Attribute{Key: utils.KEY_TRANSACTION_ID, Value: txID})
	}

	attrs = append(attrs, additionalAttrs...)

	_ = utils.LavaFormatInfo("[MEMORY_TRACKING] "+stage, attrs...)
}

// MemoryProfiler helps track memory changes over time
type MemoryProfiler struct {
	startStats *MemoryStats
	startTime  time.Time
	mu         sync.Mutex
}

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{
		startStats: GetMemoryStats(),
		startTime:  time.Now(),
	}
}

// GetDelta returns the change in memory since profiler creation
func (mp *MemoryProfiler) GetDelta() map[string]interface{} {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	currentStats := GetMemoryStats()

	return map[string]interface{}{
		"alloc_mb_delta":      currentStats.AllocMB - mp.startStats.AllocMB,
		"heap_alloc_mb_delta": currentStats.HeapAllocMB - mp.startStats.HeapAllocMB,
		"sys_mb_delta":        currentStats.SysMB - mp.startStats.SysMB,
		"goroutine_delta":     currentStats.GoroutineCount - mp.startStats.GoroutineCount,
		"gc_count_delta":      currentStats.NumGC - mp.startStats.NumGC,
		"duration_ms":         time.Since(mp.startTime).Milliseconds(),
	}
}

// LogDelta logs the memory delta
func (mp *MemoryProfiler) LogDelta(ctx context.Context, stage string, additionalAttrs ...utils.Attribute) {
	if !IsMemoryLogsEnabled() {
		return
	}
	delta := mp.GetDelta()

	attrs := []utils.Attribute{
		{Key: "stage", Value: stage},
		{Key: "alloc_mb_delta", Value: delta["alloc_mb_delta"]},
		{Key: "heap_alloc_mb_delta", Value: delta["heap_alloc_mb_delta"]},
		{Key: "sys_mb_delta", Value: delta["sys_mb_delta"]},
		{Key: "goroutine_delta", Value: delta["goroutine_delta"]},
		{Key: "gc_count_delta", Value: delta["gc_count_delta"]},
		{Key: "duration_ms", Value: delta["duration_ms"]},
		{Key: "GUID", Value: ctx},
	}

	attrs = append(attrs, additionalAttrs...)

	_ = utils.LavaFormatInfo("[MEMORY_TRACKING] Delta: "+stage, attrs...)
}

// StartMemoryGC starts a goroutine that monitors heap memory usage and calls GC
// when heap in use exceeds the threshold. If thresholdGB is 0, the routine is disabled.
// The goroutine runs until ctx is cancelled.
func StartMemoryGC(ctx context.Context, thresholdGB float64) {
	// If threshold is 0, the routine is disabled
	if thresholdGB <= 0 {
		_ = utils.LavaFormatInfo("Memory GC monitoring disabled (threshold is 0)")
		return
	}

	thresholdBytes := uint64(thresholdGB * 1024 * 1024 * 1024)

	_ = utils.LavaFormatInfo("Starting memory GC monitoring goroutine",
		utils.LogAttr("threshold_gb", thresholdGB),
		utils.LogAttr("threshold_bytes", thresholdBytes),
	)

	go func() {
		ticker := time.NewTicker(gcMonitoringCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				_ = utils.LavaFormatInfo("Memory GC monitoring goroutine stopped")
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				// Check if heap in use exceeds threshold
				if m.HeapInuse > thresholdBytes {
					heapInUseMB := float64(m.HeapInuse) / (1024 * 1024)
					heapInUseGB := float64(m.HeapInuse) / (1024 * 1024 * 1024)
					_ = utils.LavaFormatInfo("Heap memory exceeds threshold, triggering GC",
						utils.LogAttr("heap_inuse_gb", heapInUseGB),
						utils.LogAttr("threshold_gb", thresholdGB),
						utils.LogAttr("num_gc_before", m.NumGC),
					)

					// Trigger garbage collection
					runtime.GC()

					// Read stats again after GC to see the effect
					runtime.ReadMemStats(&m)
					heapInUseAfterMB := float64(m.HeapInuse) / (1024 * 1024)

					_ = utils.LavaFormatInfo("GC completed",
						utils.LogAttr("heap_inuse_mb_before", heapInUseMB),
						utils.LogAttr("heap_inuse_mb_after", heapInUseAfterMB),
						utils.LogAttr("num_gc_after", m.NumGC),
					)
				}
			}
		}
	}()
}
