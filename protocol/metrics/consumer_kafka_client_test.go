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
	client.lock.RLock()
	queueLength := len(client.addQueue)
	client.lock.RUnlock()

	if queueLength == 0 {
		t.Error("Expected data to be added to queue")
	}
}

func TestConsumerKafkaClient_AggregateRelayData(t *testing.T) {
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

	// Add multiple metrics
	client.SetRelayMetrics(relayMetrics)
	client.SetRelayMetrics(relayMetrics)

	// Wait for aggregation
	time.Sleep(100 * time.Millisecond)

	// Check if data was aggregated
	client.lock.RLock()
	queueLength := len(client.addQueue)
	client.lock.RUnlock()

	if queueLength == 0 {
		t.Error("Expected aggregated data in queue")
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
