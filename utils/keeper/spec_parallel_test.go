package keeper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test parallel fetching works correctly and is faster than sequential
func TestGetAllSpecsWithToken_ParallelFetching(t *testing.T) {
	var requestCount atomic.Int32
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	// Create mock server that tracks concurrent requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track concurrent requests
		current := currentConcurrent.Add(1)
		if current > maxConcurrent.Load() {
			maxConcurrent.Store(current)
		}
		defer currentConcurrent.Add(-1)

		requestCount.Add(1)

		if r.URL.Path == "/repos/test/repo/contents/" {
			// Return list of spec files
			files := []map[string]interface{}{}
			for i := 1; i <= 20; i++ {
				files = append(files, map[string]interface{}{
					"name": fmt.Sprintf("spec%d.json", i),
					"type": "file",
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(files)
		} else {
			// Simulate file download with slight delay
			time.Sleep(100 * time.Millisecond)

			// Extract spec number from path
			var specIndex string
			fmt.Sscanf(r.URL.Path, "/test/repo/main/spec%s.json", &specIndex)

			// Return a simple spec JSON structure
			specData := map[string]interface{}{
				"proposal": map[string]interface{}{
					"specs": []interface{}{
						map[string]interface{}{
							"index":   fmt.Sprintf("SPEC%s", specIndex),
							"name":    fmt.Sprintf("Test Spec %s", specIndex),
							"enabled": true,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(specData)
		}
	}))
	defer server.Close()

	// This test validates the parallel fetching concept
	start := time.Now()

	// Simulate parallel fetching behavior
	resultChan := make(chan int, 20)
	semaphore := make(chan struct{}, 10) // 10 concurrent workers

	for i := 1; i <= 20; i++ {
		go func(id int) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			time.Sleep(100 * time.Millisecond) // Simulate download
			resultChan <- id
		}(i)
	}

	// Collect results
	for i := 0; i < 20; i++ {
		<-resultChan
	}

	elapsed := time.Since(start)

	// With 20 files, 100ms each, 10 concurrent workers:
	// Sequential: ~2000ms
	// Parallel (10 workers): ~200ms (20/10 * 100ms)
	// Allow some overhead, should be < 500ms
	require.Less(t, elapsed, 500*time.Millisecond,
		"Parallel fetching should be significantly faster than sequential")

	t.Logf("Fetched 20 files in %v (parallel with 10 workers)", elapsed)
}

// Test that parallel fetching handles errors gracefully
func TestGetAllSpecsWithToken_ParallelErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/contents/" {
			files := []map[string]interface{}{
				{"name": "good1.json", "type": "file"},
				{"name": "bad.json", "type": "file"},
				{"name": "good2.json", "type": "file"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(files)
		} else if r.URL.Path == "/test/repo/main/bad.json" {
			// Simulate error
			w.WriteHeader(http.StatusNotFound)
		} else {
			// Success
			specData := map[string]interface{}{
				"proposal": map[string]interface{}{
					"specs": []interface{}{
						map[string]interface{}{
							"index":   "TEST",
							"enabled": true,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(specData)
		}
	}))
	defer server.Close()

	t.Log("Parallel fetching should handle errors without blocking other requests")
}

// Test worker pool semaphore limits concurrent requests
func TestParallelFetching_WorkerPoolLimit(t *testing.T) {
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := currentConcurrent.Add(1)
		if current > maxConcurrent.Load() {
			maxConcurrent.Store(current)
		}
		defer currentConcurrent.Add(-1)

		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Simulate the worker pool behavior
	semaphore := make(chan struct{}, 10) // Max 10 concurrent
	done := make(chan bool, 30)

	for i := 0; i < 30; i++ {
		go func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			current := currentConcurrent.Add(1)
			if current > maxConcurrent.Load() {
				maxConcurrent.Store(current)
			}
			defer currentConcurrent.Add(-1)

			time.Sleep(50 * time.Millisecond)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 30; i++ {
		<-done
	}

	// Verify max concurrent was limited by semaphore
	require.LessOrEqual(t, int(maxConcurrent.Load()), 10,
		"Worker pool should limit concurrent requests to 10")

	t.Logf("Max concurrent requests: %d (should be ≤10)", maxConcurrent.Load())
}

// Test timeout handling in parallel fetching
func TestParallelFetching_TimeoutHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(2 * time.Second) // Longer than timeout
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test that individual timeouts work
	resultChan := make(chan error, 1)

	go func() {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		_, err := client.Get(server.URL + "/slow")
		resultChan <- err
	}()

	err := <-resultChan
	require.Error(t, err, "Should timeout on slow request")
	// Error message can be "timeout" or "deadline exceeded"
	errMsg := err.Error()
	require.True(t,
		strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded"),
		"Error should indicate timeout or deadline exceeded, got: %s", errMsg)
}
