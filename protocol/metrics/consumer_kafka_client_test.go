package metrics

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

func TestNewConsumerKafkaClient(t *testing.T) {
	// Test with disabled flag
	client := NewConsumerKafkaClient(DisabledFlagOption, "test-topic", "", "", "", false, false)
	if client != nil {
		t.Errorf("Expected nil client when disabled, got %v", client)
	}

	// Test with valid address - this will try to connect but we'll close it immediately
	client = NewConsumerKafkaClient("localhost:9092", "test-topic", "", "", "", false, false)
	if client == nil {
		t.Errorf("Expected non-nil client with valid address, got nil")
		return
	}
	if client.kafkaAddress != "localhost:9092" {
		t.Errorf("Expected kafka address 'localhost:9092', got '%s'", client.kafkaAddress)
	}
	if client.topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", client.topic)
	}

	// Clean up immediately to avoid connection attempts
	if client != nil {
		client.Close()
	}
}

func TestNewConsumerKafkaClientWithConfig(t *testing.T) {
	// Test with custom configuration
	client := NewConsumerKafkaClientWithConfig(
		"localhost:9092",
		"test-topic",
		"", "", "",
		false, false,
		10*time.Second, // tickerInterval
		5,              // maxRetries
		2*time.Second,  // retryDelay
		15*time.Second, // maxTimeout
		500,            // maxRetryQueueSize
	)

	if client == nil {
		t.Errorf("Expected non-nil client with valid config, got nil")
		return
	}

	// Verify custom configuration
	if client.tickerInterval != 10*time.Second {
		t.Errorf("Expected ticker interval 10s, got %v", client.tickerInterval)
	}
	if client.maxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", client.maxRetries)
	}
	if client.retryDelay != 2*time.Second {
		t.Errorf("Expected retry delay 2s, got %v", client.retryDelay)
	}
	if client.maxTimeout != 15*time.Second {
		t.Errorf("Expected max timeout 15s, got %v", client.maxTimeout)
	}
	if client.maxRetryQueueSize != 500 {
		t.Errorf("Expected max retry queue size 500, got %d", client.maxRetryQueueSize)
	}

	// Clean up
	if client != nil {
		client.Close()
	}
}

func TestConsumerKafkaClient_AuthenticationFailure(t *testing.T) {
	// Test with invalid authentication mechanism
	client := NewConsumerKafkaClient("localhost:9092", "test-topic", "user", "pass", "INVALID-MECHANISM", false, false)
	if client != nil {
		t.Errorf("Expected nil client with invalid authentication mechanism, got %v", client)
		client.Close()
	}
}

func TestConsumerKafkaClient_SetRelayMetrics(t *testing.T) {
	// Skip this test if we can't connect to Kafka (which is expected in test environment)
	client := NewConsumerKafkaClient("localhost:9092", "test-topic", "", "", "", false, false)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}
	defer client.Close()

	// Test data
	relayMetrics := &RelayMetrics{
		ProjectHash: "test-hash",
		ChainID:     "test-chain",
		APIType:     "test-api",
		Timestamp:   time.Now(),
		Latency:     100,
		Success:     true,
	}

	// Test SetRelayMetrics
	client.SetRelayMetrics(relayMetrics)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Check if data was added to queue
	client.lock.Lock()
	queueLength := len(client.addQueue)
	client.lock.Unlock()

	if queueLength == 0 {
		t.Error("Expected data to be added to queue")
	}
}

func TestConsumerKafkaClient_RetryQueue(t *testing.T) {
	// Skip this test if we can't connect to Kafka (which is expected in test environment)
	client := NewConsumerKafkaClient("localhost:9092", "test-topic", "", "", "", false, false)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}
	defer client.Close()

	// Test data
	relayMetrics := &RelayMetrics{
		ProjectHash:  "test-hash",
		ChainID:      "test-chain",
		APIType:      "test-api",
		Timestamp:    time.Now(),
		Latency:      100,
		Success:      true,
		ComputeUnits: 10,
	}

	// Add metrics to queue
	client.SetRelayMetrics(relayMetrics)
	client.SetRelayMetrics(relayMetrics)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check if data was added to queue
	client.lock.Lock()
	queueLength := len(client.addQueue)
	client.lock.Unlock()

	if queueLength == 0 {
		t.Error("Expected data to be added to queue")
	}
}

func TestConsumerKafkaClient_RetryQueueProcessing(t *testing.T) {
	// Skip this test if we can't connect to Kafka (which is expected in test environment)
	// Use a shorter ticker interval for testing
	client := NewConsumerKafkaClientWithConfig("localhost:9092", "test-topic", "", "", "", false, false, 100*time.Millisecond, 3, 5*time.Second, 30*time.Second, 1000)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}
	defer client.Close()

	// Add items to retry queue directly
	testRequest := UpdateMetricsRequest{
		RecordDate:      "20240101",
		Hash:            "test-hash",
		Chain:           "test-chain",
		ApiType:         "test-api",
		RelaysInc:       1,
		CuInc:           10,
		LatencyToAdd:    100,
		LatencyAvgCount: 1,
	}

	client.lock.Lock()
	client.retryQueue = append(client.retryQueue, testRequest)
	initialRetryQueueLength := len(client.retryQueue)
	client.lock.Unlock()

	if initialRetryQueueLength != 1 {
		t.Errorf("Expected retry queue length 1, got %d", initialRetryQueueLength)
	}

	// Wait for ticker to process retry queue (wait for 2 ticker intervals to be sure)
	time.Sleep(250 * time.Millisecond)

	// Check if retry queue was processed
	client.lock.Lock()
	finalRetryQueueLength := len(client.retryQueue)
	client.lock.Unlock()

	// The retry queue should be processed and cleared
	if finalRetryQueueLength != 0 {
		t.Errorf("Expected retry queue to be processed and cleared, got length %d", finalRetryQueueLength)
	}
}

func TestConsumerKafkaClient_AddToRetryQueue(t *testing.T) {
	// Skip this test if we can't connect to Kafka (which is expected in test environment)
	client := NewConsumerKafkaClient("localhost:9092", "test-topic", "", "", "", false, false)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}
	defer client.Close()

	// Test data
	testRequests := []UpdateMetricsRequest{
		{
			RecordDate:      "20240101",
			Hash:            "test-hash-1",
			Chain:           "test-chain",
			ApiType:         "test-api",
			RelaysInc:       1,
			CuInc:           10,
			LatencyToAdd:    100,
			LatencyAvgCount: 1,
		},
		{
			RecordDate:      "20240101",
			Hash:            "test-hash-2",
			Chain:           "test-chain",
			ApiType:         "test-api",
			RelaysInc:       2,
			CuInc:           20,
			LatencyToAdd:    200,
			LatencyAvgCount: 2,
		},
	}

	// Add to retry queue
	client.addToRetryQueue(testRequests)

	// Check if items were added to retry queue
	client.lock.Lock()
	retryQueueLength := len(client.retryQueue)
	client.lock.Unlock()

	if retryQueueLength != 2 {
		t.Errorf("Expected 2 items in retry queue, got %d", retryQueueLength)
	}
}

func TestConsumerKafkaClient_RetryQueueCapacity(t *testing.T) {
	// Skip this test if we can't connect to Kafka (which is expected in test environment)
	// Create client with small retry queue capacity
	client := NewConsumerKafkaClientWithConfig("localhost:9092", "test-topic", "", "", "", false, false, 30*time.Second, 3, 5*time.Second, 30*time.Second, 3)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}
	defer client.Close()

	// Test data - more than the capacity
	testRequests := []UpdateMetricsRequest{
		{RecordDate: "20240101", Hash: "test-hash-1", Chain: "test-chain", ApiType: "test-api", RelaysInc: 1, CuInc: 10, LatencyToAdd: 100, LatencyAvgCount: 1},
		{RecordDate: "20240101", Hash: "test-hash-2", Chain: "test-chain", ApiType: "test-api", RelaysInc: 2, CuInc: 20, LatencyToAdd: 200, LatencyAvgCount: 2},
		{RecordDate: "20240101", Hash: "test-hash-3", Chain: "test-chain", ApiType: "test-api", RelaysInc: 3, CuInc: 30, LatencyToAdd: 300, LatencyAvgCount: 3},
		{RecordDate: "20240101", Hash: "test-hash-4", Chain: "test-chain", ApiType: "test-api", RelaysInc: 4, CuInc: 40, LatencyToAdd: 400, LatencyAvgCount: 4},
		{RecordDate: "20240101", Hash: "test-hash-5", Chain: "test-chain", ApiType: "test-api", RelaysInc: 5, CuInc: 50, LatencyToAdd: 500, LatencyAvgCount: 5},
	}

	// Check initial state
	client.lock.Lock()
	initialLength := len(client.retryQueue)
	client.lock.Unlock()
	t.Logf("Initial retry queue length: %d", initialLength)

	// Add to retry queue
	client.addToRetryQueue(testRequests)

	// Check if retry queue respects capacity limit
	client.lock.Lock()
	retryQueueLength := len(client.retryQueue)
	client.lock.Unlock()

	t.Logf("Final retry queue length: %d, expected: 3", retryQueueLength)
	// Should only have 3 items (capacity limit), dropping the oldest 2
	if retryQueueLength != 3 {
		t.Errorf("Expected 3 items in retry queue (capacity limit), got %d", retryQueueLength)
	}
}

func TestConsumerKafkaClient_NilSafety(t *testing.T) {
	// Test nil client safety
	var client *ConsumerKafkaClient = nil

	// These should not panic
	client.SetRelayMetrics(&RelayMetrics{})
	client.Close()

	utils.LavaFormatInfo("Nil safety tests passed")
}

func TestConsumerKafkaClient_Close(t *testing.T) {
	// Test client close functionality
	client := NewConsumerKafkaClient("localhost:9092", "test-topic", "", "", "", false, false)
	if client == nil {
		t.Skip("Kafka client is nil, skipping test")
	}

	// Close the client
	client.Close()

	// Verify that the context is cancelled
	select {
	case <-client.ctx.Done():
		// Expected - context should be cancelled
	default:
		t.Error("Expected context to be cancelled after Close()")
	}
}
