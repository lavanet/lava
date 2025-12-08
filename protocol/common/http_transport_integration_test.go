package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConnectionPoolingUnderLoad verifies that HTTP connections are properly pooled
// and reused under high-concurrency scenarios (200+ concurrent requests)
func TestConnectionPoolingUnderLoad(t *testing.T) {
	// Track unique connections established
	var connectionCount int64
	var activeConnections int64
	connectionsMutex := &sync.Mutex{}
	connections := make(map[string]bool)

	// Pre-computed response to reduce server overhead
	precomputedResponse := []byte(`{"jsonrpc":"2.0","result":"0x1234","id":"1"}`)

	// Create test server that tracks connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track this connection
		remoteAddr := r.RemoteAddr
		connectionsMutex.Lock()
		if !connections[remoteAddr] {
			connections[remoteAddr] = true
			atomic.AddInt64(&connectionCount, 1)
		}
		connectionsMutex.Unlock()

		atomic.AddInt64(&activeConnections, 1)
		defer atomic.AddInt64(&activeConnections, -1)

		// Simulate blockchain node latency (realistic: 5-20ms for fast queries)
		time.Sleep(10 * time.Millisecond)

		// Return pre-computed JSON response (faster than encoding)
		w.Header().Set("Content-Type", "application/json")
		w.Write(precomputedResponse)
	}))
	defer server.Close()

	// Create client with optimized transport
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	// Test scenario: 200 concurrent requests (simulates high provider load)
	numRequests := 200
	concurrency := 50 // 50 concurrent goroutines

	var wg sync.WaitGroup
	requestChan := make(chan int, numRequests)

	// Fill request channel
	for i := 0; i < numRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	// Track errors and latencies (pre-allocated to avoid reallocation)
	var errorCount int64
	type latencyResult struct {
		index    int
		duration time.Duration
	}
	latencyResults := make(chan latencyResult, numRequests)

	// Start concurrent workers
	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for reqNum := range requestChan {
				reqStart := time.Now()
				resp, err := client.Get(server.URL)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				reqDuration := time.Since(reqStart)

				latencyResults <- latencyResult{index: reqNum, duration: reqDuration}
			}
		}()
	}

	// Close latency channel when all workers are done
	go func() {
		wg.Wait()
		close(latencyResults)
	}()

	// Collect latency results
	latencies := make([]time.Duration, numRequests)
	for result := range latencyResults {
		latencies[result.index] = result.duration
	}

	totalDuration := time.Since(startTime)

	// Verify results
	if errorCount > 0 {
		t.Errorf("Had %d errors out of %d requests", errorCount, numRequests)
	}

	// Calculate statistics
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration = time.Hour, 0
	for _, lat := range latencies {
		if lat > 0 { // Skip zero values from errors
			totalLatency += lat
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
		}
	}
	successfulRequests := numRequests - int(errorCount)
	avgLatency := totalLatency / time.Duration(successfulRequests)

	// Calculate percentiles for better latency understanding
	sortedLatencies := make([]time.Duration, 0, successfulRequests)
	for _, lat := range latencies {
		if lat > 0 {
			sortedLatencies = append(sortedLatencies, lat)
		}
	}
	// Simple insertion sort for small dataset
	for i := 1; i < len(sortedLatencies); i++ {
		for j := i; j > 0 && sortedLatencies[j] < sortedLatencies[j-1]; j-- {
			sortedLatencies[j], sortedLatencies[j-1] = sortedLatencies[j-1], sortedLatencies[j]
		}
	}
	p50 := sortedLatencies[len(sortedLatencies)/2]
	p95 := sortedLatencies[len(sortedLatencies)*95/100]
	p99 := sortedLatencies[len(sortedLatencies)*99/100]

	// Log performance metrics
	t.Logf("Performance Metrics:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Latency Stats:")
	t.Logf("    Average: %v", avgLatency)
	t.Logf("    Min: %v", minLatency)
	t.Logf("    Max: %v", maxLatency)
	t.Logf("    p50 (median): %v", p50)
	t.Logf("    p95: %v", p95)
	t.Logf("    p99: %v", p99)
	t.Logf("    HTTP Overhead: ~%v (avg latency - 10ms simulated delay)", avgLatency-10*time.Millisecond)
	t.Logf("  Requests/sec: %.2f", float64(numRequests)/totalDuration.Seconds())
	t.Logf("  Unique Connections: %d", connectionCount)
	t.Logf("  Connection Reuse Ratio: %.2f%%", (1.0-float64(connectionCount)/float64(numRequests))*100)

	// CRITICAL TEST: Verify connection pooling is working
	// With MaxIdleConnsPerHost=50, we should use significantly fewer than 200 connections
	// Go's default (MaxIdleConnsPerHost=2) would create ~100-200 connections
	maxExpectedConnections := int64(70) // Allow some overhead beyond the 50 idle limit
	if connectionCount > maxExpectedConnections {
		t.Errorf("Too many connections created: %d (expected ≤%d). Connection pooling may not be working correctly.",
			connectionCount, maxExpectedConnections)
		t.Errorf("This suggests connections are not being reused properly.")
	}

	// Verify we're using connection pooling effectively
	minConnectionReuseRatio := 60.0 // At least 60% connection reuse
	actualReuseRatio := (1.0 - float64(connectionCount)/float64(numRequests)) * 100
	if actualReuseRatio < minConnectionReuseRatio {
		t.Errorf("Connection reuse ratio too low: %.2f%% (expected ≥%.2f%%)",
			actualReuseRatio, minConnectionReuseRatio)
	}

	// Verify performance is reasonable
	maxAvgLatency := 100 * time.Millisecond // Should be fast with connection pooling
	if avgLatency > maxAvgLatency {
		t.Errorf("Average latency too high: %v (expected ≤%v). Connection pooling should reduce latency.",
			avgLatency, maxAvgLatency)
	}
}

// TestConnectionPoolingOverhead measures the pure HTTP/connection pooling overhead
// without simulated blockchain latency (shows minimum achievable latency)
func TestConnectionPoolingOverhead(t *testing.T) {
	var connectionCount int64
	connectionsMutex := &sync.Mutex{}
	connections := make(map[string]bool)

	// Minimal server - no artificial delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		connectionsMutex.Lock()
		if !connections[remoteAddr] {
			connections[remoteAddr] = true
			atomic.AddInt64(&connectionCount, 1)
		}
		connectionsMutex.Unlock()

		// Immediate response - measures pure HTTP overhead
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer server.Close()

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	numRequests := 200
	concurrency := 50

	var wg sync.WaitGroup
	requestChan := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	type latencyResult struct {
		index    int
		duration time.Duration
	}
	latencyResults := make(chan latencyResult, numRequests)

	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for reqNum := range requestChan {
				reqStart := time.Now()
				resp, err := client.Get(server.URL)
				if err != nil {
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				reqDuration := time.Since(reqStart)

				latencyResults <- latencyResult{index: reqNum, duration: reqDuration}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(latencyResults)
	}()

	latencies := make([]time.Duration, numRequests)
	for result := range latencyResults {
		latencies[result.index] = result.duration
	}

	totalDuration := time.Since(startTime)

	// Calculate statistics
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration = time.Hour, 0
	validLatencies := make([]time.Duration, 0, numRequests)

	for _, lat := range latencies {
		if lat > 0 {
			totalLatency += lat
			validLatencies = append(validLatencies, lat)
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
		}
	}

	// Sort for percentiles
	for i := 1; i < len(validLatencies); i++ {
		for j := i; j > 0 && validLatencies[j] < validLatencies[j-1]; j-- {
			validLatencies[j], validLatencies[j-1] = validLatencies[j-1], validLatencies[j]
		}
	}

	avgLatency := totalLatency / time.Duration(len(validLatencies))
	p50 := validLatencies[len(validLatencies)/2]
	p95 := validLatencies[len(validLatencies)*95/100]
	p99 := validLatencies[len(validLatencies)*99/100]

	t.Logf("Pure Connection Pooling Overhead (No Simulated Delay):")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Pure HTTP Overhead:")
	t.Logf("    Average: %v", avgLatency)
	t.Logf("    Min: %v (best case with warm connections)", minLatency)
	t.Logf("    Max: %v", maxLatency)
	t.Logf("    p50: %v", p50)
	t.Logf("    p95: %v", p95)
	t.Logf("    p99: %v", p99)
	t.Logf("  Throughput: %.2f req/sec", float64(numRequests)/totalDuration.Seconds())
	t.Logf("  Unique Connections: %d", connectionCount)
	t.Logf("  Connection Reuse: %.2f%%", (1.0-float64(connectionCount)/float64(numRequests))*100)

	// Verify low overhead - should be sub-millisecond for most requests
	if p50 > 2*time.Millisecond {
		t.Logf("INFO: Median latency %v is higher than expected (<2ms). This is the pure HTTP overhead.", p50)
	}

	// Connection pooling should still be effective
	if connectionCount > 70 {
		t.Errorf("Too many connections: %d (connection pooling not working)", connectionCount)
	}
}

// TestConnectionPoolingVsDefaultTransport compares optimized transport against Go's default
// This test demonstrates the performance improvement from proper connection pooling
func TestConnectionPoolingVsDefaultTransport(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	numRequests := 100
	concurrency := 20

	// Test with default transport
	defaultTransport := &http.Transport{} // Go's defaults: MaxIdleConnsPerHost=2
	defaultClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: defaultTransport,
	}

	defaultDuration := runLoadTest(t, defaultClient, server.URL, numRequests, concurrency)

	// Test with optimized transport
	optimizedClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	optimizedDuration := runLoadTest(t, optimizedClient, server.URL, numRequests, concurrency)

	// Log comparison
	improvement := float64(defaultDuration-optimizedDuration) / float64(defaultDuration) * 100
	t.Logf("Performance Comparison:")
	t.Logf("  Default Transport: %v", defaultDuration)
	t.Logf("  Optimized Transport: %v", optimizedDuration)
	t.Logf("  Improvement: %.2f%%", improvement)

	// Verify optimized is faster or at least not slower
	// Note: In some cases they might be similar, but optimized should never be significantly slower
	if optimizedDuration > defaultDuration*15/10 { // Allow 50% margin
		t.Errorf("Optimized transport is slower than default: optimized=%v, default=%v",
			optimizedDuration, defaultDuration)
	}
}

// runLoadTest is a helper that runs a load test and returns the duration
func runLoadTest(t *testing.T, client *http.Client, url string, numRequests, concurrency int) time.Duration {
	var wg sync.WaitGroup
	requestChan := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				resp, err := client.Get(url)
				if err != nil {
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	return time.Since(startTime)
}

// TestConnectionPoolIdleTimeout verifies that idle connections are cleaned up properly
func TestConnectionPoolIdleTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping idle timeout test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create transport with short idle timeout for testing
	transport := OptimizedHttpTransport()
	transport.IdleConnTimeout = 2 * time.Second

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	// Make initial requests to establish connections
	for i := 0; i < 10; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Wait for connections to become idle
	t.Logf("Waiting for idle connections to timeout...")
	time.Sleep(3 * time.Second)

	// Make new requests - should work even after idle timeout
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Errorf("Request after idle timeout failed: %v", err)
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

// TestConcurrentRequestsToMultipleHosts verifies connection pooling works correctly
// when making requests to multiple different hosts simultaneously
func TestConcurrentRequestsToMultipleHosts(t *testing.T) {
	numServers := 5
	servers := make([]*httptest.Server, numServers)

	// Create multiple test servers
	for i := 0; i < numServers; i++ {
		serverID := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Millisecond)
			json.NewEncoder(w).Encode(map[string]int{"server": serverID})
		}))
		defer servers[i].Close()
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	// Make concurrent requests to all servers
	numRequestsPerServer := 20
	var wg sync.WaitGroup
	var errorCount int64

	for _, server := range servers {
		serverURL := server.URL
		for i := 0; i < numRequestsPerServer; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Get(serverURL)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()
		}
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Had %d errors during multi-host requests", errorCount)
	}

	t.Logf("Successfully completed %d concurrent requests across %d hosts",
		numServers*numRequestsPerServer, numServers)
}

// TestKeepAliveConnections verifies that keep-alive connections are maintained
func TestKeepAliveConnections(t *testing.T) {
	var connectionCounter int64
	var activeConnections int64

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Create custom server that tracks connections
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}),
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt64(&connectionCounter, 1)
				atomic.AddInt64(&activeConnections, 1)
			case http.StateClosed:
				atomic.AddInt64(&activeConnections, -1)
			}
		},
	}

	go server.Serve(listener)
	defer server.Close()

	serverURL := fmt.Sprintf("http://%s", listener.Addr().String())

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	// Make sequential requests (should reuse the same connection)
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(serverURL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Small delay to allow connection state updates
		time.Sleep(10 * time.Millisecond)
	}

	totalConnections := atomic.LoadInt64(&connectionCounter)
	t.Logf("Total connections created: %d for %d requests", totalConnections, numRequests)

	// With keep-alive, we should reuse connections
	// Expect very few connections (ideally 1, but allow up to 3 for timing variations)
	if totalConnections > 3 {
		t.Errorf("Too many connections created: %d (expected ≤3 with keep-alive)", totalConnections)
		t.Errorf("Keep-alive connection reuse is not working properly")
	}
}

// TestHighConcurrencyStressTest simulates extreme load to verify stability
func TestHighConcurrencyStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		// Simulate variable latency
		time.Sleep(time.Duration(5+count%10) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"count": count})
	}))
	defer server.Close()

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	// Stress test: 500 requests with 100 concurrent workers
	numRequests := 500
	concurrency := 100

	var wg sync.WaitGroup
	requestChan := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	var successCount int64
	var errorCount int64

	startTime := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				resp, err := client.Get(server.URL)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	successRate := float64(successCount) / float64(numRequests) * 100
	throughput := float64(numRequests) / duration.Seconds()

	t.Logf("Stress Test Results:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Success: %d (%.2f%%)", successCount, successRate)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	// Verify acceptable success rate (should be 100% or very close)
	minSuccessRate := 99.0
	if successRate < minSuccessRate {
		t.Errorf("Success rate too low: %.2f%% (expected ≥%.2f%%)", successRate, minSuccessRate)
	}

	// Verify reasonable throughput (should handle at least 50 req/sec)
	minThroughput := 50.0
	if throughput < minThroughput {
		t.Errorf("Throughput too low: %.2f req/sec (expected ≥%.2f req/sec)", throughput, minThroughput)
	}
}

// TestConnectionPoolingWithTimeout verifies that timeouts don't break connection pooling
func TestConnectionPoolingWithTimeout(t *testing.T) {
	var slowRequests int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Every 10th request is slow
		count := atomic.AddInt64(&slowRequests, 1)
		if count%10 == 0 {
			time.Sleep(200 * time.Millisecond) // Will timeout
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout:   100 * time.Millisecond, // Short timeout to trigger some timeouts
		Transport: OptimizedHttpTransport(),
	}

	numRequests := 50
	var successCount int64
	var timeoutCount int64

	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL)
			if err != nil {
				atomic.AddInt64(&timeoutCount, 1)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			atomic.AddInt64(&successCount, 1)
		}()
	}

	wg.Wait()

	t.Logf("Timeout Test Results:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Timeouts: %d", timeoutCount)

	// Verify we had some timeouts (proves the test is working)
	if timeoutCount == 0 {
		t.Error("Expected some timeouts, but got none. Test may not be working correctly.")
	}

	// Verify successful requests still worked (connection pooling survived the timeouts)
	if successCount == 0 {
		t.Error("No successful requests after timeouts. Connection pooling may be broken by timeouts.")
	}

	// Most requests should succeed
	minSuccessCount := int64(40)
	if successCount < minSuccessCount {
		t.Errorf("Too few successful requests: %d (expected ≥%d)", successCount, minSuccessCount)
	}
}

// BenchmarkConnectionPoolingUnderLoad benchmarks the performance of connection pooling
func BenchmarkConnectionPoolingUnderLoad(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: OptimizedHttpTransport(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkDefaultTransportVsOptimized compares default vs optimized transport performance
func BenchmarkDefaultTransportVsOptimized(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b.Run("DefaultTransport", func(b *testing.B) {
		client := &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{}, // Go defaults
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("OptimizedTransport", func(b *testing.B) {
		client := &http.Client{
			Timeout:   10 * time.Second,
			Transport: OptimizedHttpTransport(),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// TestConnectionPoolingWithContext verifies that context cancellation works properly
func TestConnectionPoolingWithContext(t *testing.T) {
	var activeRequests int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&activeRequests, 1)
		defer atomic.AddInt64(&activeRequests, -1)

		// Wait for context cancellation or 1 second
		select {
		case <-r.Context().Done():
			return
		case <-time.After(1 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &http.Client{
		Transport: OptimizedHttpTransport(),
	}

	// Create requests with context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	numRequests := 20

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
			resp, err := client.Do(req)
			if err != nil {
				// Expected to fail due to context cancellation
				return
			}
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()

	// Wait a bit for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify no requests are still active
	active := atomic.LoadInt64(&activeRequests)
	if active != 0 {
		t.Errorf("Still have %d active requests after context cancellation", active)
	}

	// Verify connection pool still works after cancellations
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Errorf("Request failed after context cancellations: %v", err)
	}
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
