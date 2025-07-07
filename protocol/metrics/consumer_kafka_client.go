package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"

	"github.com/lavanet/lava/v5/utils"
)

type ConsumerKafkaClient struct {
	kafkaAddress       string
	topic              string
	addQueue           []UpdateMetricsRequest
	ticker             *time.Ticker
	lock               sync.Mutex
	sendID             int
	isTickerRunning    bool
	isSendQueueRunning bool
	writer             *kafka.Writer
	ctx                context.Context
	cancel             context.CancelFunc
	// Retry mechanism fields
	retryQueue []UpdateMetricsRequest
	maxRetries int
	retryDelay time.Duration
	// Authentication fields
	username    string
	password    string
	mechanism   string
	tlsEnabled  bool
	tlsInsecure bool
	// Configuration fields
	tickerInterval time.Duration
	maxTimeout     time.Duration
	// Circuit breaker fields
	maxRetryQueueSize int
}

func NewConsumerKafkaClient(kafkaAddress string, topic string, username string, password string, mechanism string, tlsEnabled bool, tlsInsecure bool) *ConsumerKafkaClient {
	return NewConsumerKafkaClientWithConfig(kafkaAddress, topic, username, password, mechanism, tlsEnabled, tlsInsecure, 30*time.Second, 3, 5*time.Second, 30*time.Second, 1000)
}

func NewConsumerKafkaClientWithConfig(kafkaAddress string, topic string, username string, password string, mechanism string, tlsEnabled bool, tlsInsecure bool, tickerInterval time.Duration, maxRetries int, retryDelay time.Duration, maxTimeout time.Duration, maxRetryQueueSize int) *ConsumerKafkaClient {
	if kafkaAddress == DisabledFlagOption {
		utils.LavaFormatInfo("Running with Consumer Kafka Client Disabled")
		return nil
	}

	// Parse kafka address
	addresses := strings.Split(kafkaAddress, ",")
	for i, addr := range addresses {
		addresses[i] = strings.TrimSpace(addr)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Configure Kafka writer with authentication
	writer := &kafka.Writer{
		Addr:     kafka.TCP(addresses...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	// Configure authentication if provided
	if username != "" && password != "" {
		var authErr error
		switch mechanism {
		case "PLAIN":
			writer.Transport = &kafka.Transport{
				SASL: plain.Mechanism{
					Username: username,
					Password: password,
				},
			}
		case "SCRAM-SHA-256":
			saslMechanism, err := scram.Mechanism(scram.SHA256, username, password)
			if err != nil {
				authErr = err
				utils.LavaFormatWarning("Failed to create SCRAM-SHA-256 mechanism", err)
			} else {
				writer.Transport = &kafka.Transport{
					SASL: saslMechanism,
				}
			}
		case "SCRAM-SHA-512":
			saslMechanism, err := scram.Mechanism(scram.SHA512, username, password)
			if err != nil {
				authErr = err
				utils.LavaFormatWarning("Failed to create SCRAM-SHA-512 mechanism", err)
			} else {
				writer.Transport = &kafka.Transport{
					SASL: saslMechanism,
				}
			}
		default:
			authErr = fmt.Errorf("unsupported SASL mechanism: %s", mechanism)
			utils.LavaFormatWarning("Unsupported SASL mechanism", authErr, utils.LogAttr("mechanism", mechanism), utils.LogAttr("supported", "PLAIN, SCRAM-SHA-256, SCRAM-SHA-512"))
		}

		// If authentication failed, return nil to prevent connection issues
		if authErr != nil {
			utils.LavaFormatError("Failed to configure Kafka authentication", authErr, utils.LogAttr("mechanism", mechanism))
			cancel()
			return nil
		}
	}

	// Configure TLS if enabled
	if tlsEnabled {
		var transport *kafka.Transport
		if writer.Transport == nil {
			transport = &kafka.Transport{}
			writer.Transport = transport
		} else {
			var ok bool
			transport, ok = writer.Transport.(*kafka.Transport)
			if !ok {
				transport = &kafka.Transport{}
				writer.Transport = transport
			}
		}
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if tlsInsecure {
			tlsConfig.InsecureSkipVerify = true
		}
		transport.TLS = tlsConfig
	}

	consumerKafkaClient := &ConsumerKafkaClient{
		kafkaAddress:       kafkaAddress,
		topic:              topic,
		addQueue:           []UpdateMetricsRequest{},
		ticker:             time.NewTicker(tickerInterval),
		lock:               sync.Mutex{},
		sendID:             0,
		isTickerRunning:    false,
		isSendQueueRunning: false,
		writer:             writer,
		ctx:                ctx,
		cancel:             cancel,
		retryQueue:         []UpdateMetricsRequest{},
		maxRetries:         maxRetries,
		retryDelay:         retryDelay,
		username:           username,
		password:           password,
		mechanism:          mechanism,
		tlsEnabled:         tlsEnabled,
		tlsInsecure:        tlsInsecure,
		tickerInterval:     tickerInterval,
		maxTimeout:         maxTimeout,
		maxRetryQueueSize:  maxRetryQueueSize,
	}

	utils.LavaFormatInfo("Starting Consumer Kafka Client", utils.LogAttr("kafka_address", kafkaAddress), utils.LogAttr("topic", topic), utils.LogAttr("auth_mechanism", mechanism), utils.LogAttr("tls_enabled", tlsEnabled), utils.LogAttr("tls_insecure", tlsInsecure), utils.LogAttr("ticker_interval", tickerInterval), utils.LogAttr("max_retries", maxRetries), utils.LogAttr("max_retry_queue_size", maxRetryQueueSize))
	consumerKafkaClient.relayDataSendQueueStart()
	return consumerKafkaClient
}

func (cuc *ConsumerKafkaClient) relayDataSendQueueStart() {
	if cuc == nil {
		return
	}
	cuc.lock.Lock()
	if cuc.isTickerRunning {
		cuc.lock.Unlock()
		return
	}
	cuc.isTickerRunning = true
	cuc.lock.Unlock()

	utils.LavaFormatDebug("[CKC] Starting relayDataSendQueueStart loop")
	go func() {
		defer func() {
			cuc.lock.Lock()
			cuc.isTickerRunning = false
			cuc.lock.Unlock()
			utils.LavaFormatDebug("[CKC] Stopping relayDataSendQueueStart loop")
		}()

		for {
			select {
			case <-cuc.ctx.Done():
				return
			case <-cuc.ticker.C:
				cuc.relayDataSendQueueTick()
			}
		}
	}()
}

func (cuc *ConsumerKafkaClient) relayDataSendQueueTick() {
	if cuc == nil {
		return
	}

	cuc.lock.Lock()
	defer cuc.lock.Unlock()

	// Process both addQueue and retryQueue
	var sendQueue []UpdateMetricsRequest

	// Add items from addQueue
	if len(cuc.addQueue) > 0 {
		sendQueue = append(sendQueue, cuc.addQueue...)
		cuc.addQueue = cuc.addQueue[:0] // Reset slice without reallocating
	}

	// Add items from retryQueue
	retryQueueLength := len(cuc.retryQueue)
	if retryQueueLength > 0 {
		sendQueue = append(sendQueue, cuc.retryQueue...)
		cuc.retryQueue = cuc.retryQueue[:0] // Reset slice without reallocating
		utils.LavaFormatDebug("[CKC] Processing retry queue", utils.LogAttr("retry_items", retryQueueLength))
	}

	if !cuc.isSendQueueRunning && len(sendQueue) > 0 {
		cuc.isSendQueueRunning = true
		cuc.sendID++
		utils.LavaFormatDebug("[CKC] Swapped queues", utils.LogAttr("sendQueue_length", len(sendQueue)), utils.LogAttr("send_id", cuc.sendID))

		sendID := cuc.sendID

		// Use a separate goroutine to avoid holding the lock during send
		go func() {
			defer func() {
				// Ensure lock is always released, even if sendKafkaData panics
				if r := recover(); r != nil {
					utils.LavaFormatError("Panic in sendKafkaData", fmt.Errorf("panic: %v", r))
				}

				cuc.lock.Lock()
				cuc.isSendQueueRunning = false
				cuc.lock.Unlock()
			}()

			cuc.sendKafkaData(sendQueue, sendID)
		}()
	}
}

func (cuc *ConsumerKafkaClient) appendQueue(request UpdateMetricsRequest) {
	if cuc == nil {
		return
	}
	cuc.lock.Lock()
	defer cuc.lock.Unlock()
	cuc.addQueue = append(cuc.addQueue, request)
}

func (cuc *ConsumerKafkaClient) SetRelayMetrics(relayMetric *RelayMetrics) {
	if cuc == nil {
		return
	}
	request := UpdateMetricsRequest{
		RecordDate:      relayMetric.Timestamp.Format("20060102"),
		Hash:            relayMetric.ProjectHash,
		Chain:           relayMetric.ChainID,
		ApiType:         relayMetric.APIType,
		RelaysInc:       1,
		CuInc:           int(relayMetric.ComputeUnits),
		LatencyToAdd:    uint64(relayMetric.Latency),
		LatencyAvgCount: 1,
	}
	cuc.appendQueue(request)
}

func (cuc *ConsumerKafkaClient) sendKafkaData(requests []UpdateMetricsRequest, sendID int) {
	if cuc == nil || len(requests) == 0 {
		return
	}

	// Convert requests to JSON
	jsonData, err := json.Marshal(requests)
	if err != nil {
		utils.LavaFormatError("Failed to marshal Kafka data", err)
		return
	}

	// Send to Kafka with retry mechanism
	success := cuc.sendWithRetry(jsonData, sendID, len(requests))

	if success {
		utils.LavaFormatDebug("Successfully sent data to Kafka", utils.LogAttr("send_id", sendID), utils.LogAttr("records", len(requests)))
	} else {
		// If all retries failed, add to retry queue for next ticker cycle
		cuc.addToRetryQueue(requests)
		utils.LavaFormatWarning("All retries failed, added to retry queue", nil, utils.LogAttr("send_id", sendID), utils.LogAttr("records", len(requests)))
	}
}

func (cuc *ConsumerKafkaClient) sendWithRetry(jsonData []byte, sendID int, recordCount int) bool {
	for attempt := 0; attempt <= cuc.maxRetries; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-cuc.ctx.Done():
			utils.LavaFormatWarning("Context cancelled during Kafka send", nil, utils.LogAttr("send_id", sendID))
			return false
		default:
		}

		// Calculate timeout with exponential backoff
		timeout := time.Duration(attempt+1) * cuc.retryDelay
		if timeout > cuc.maxTimeout {
			timeout = cuc.maxTimeout
		}

		// Use the client's context with timeout
		ctx, cancel := context.WithTimeout(cuc.ctx, timeout)

		err := cuc.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(fmt.Sprintf("lava-relay-%d", sendID)),
			Value: jsonData,
		})

		cancel()

		if err == nil {
			// Success!
			if attempt > 0 {
				utils.LavaFormatInfo("Kafka send succeeded after retry", utils.LogAttr("send_id", sendID), utils.LogAttr("attempt", attempt+1), utils.LogAttr("records", recordCount))
			}
			return true
		}

		// Categorize error for better handling
		errorType := "unknown"
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no route to host") {
			errorType = "network"
		} else if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "authorization") {
			errorType = "auth"
		} else if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		}

		// Log the error with attempt information
		if attempt < cuc.maxRetries {
			utils.LavaFormatWarning("Kafka send failed, will retry", err, utils.LogAttr("send_id", sendID), utils.LogAttr("attempt", attempt+1), utils.LogAttr("max_retries", cuc.maxRetries), utils.LogAttr("error_type", errorType))

			// Wait before retry (exponential backoff) - use select to allow context cancellation
			backoffDelay := time.Duration(attempt+1) * cuc.retryDelay
			select {
			case <-time.After(backoffDelay):
				// Continue to next attempt
			case <-cuc.ctx.Done():
				utils.LavaFormatWarning("Context cancelled during retry backoff", nil, utils.LogAttr("send_id", sendID))
				return false
			}
		} else {
			// Final attempt failed
			utils.LavaFormatError("Failed to send data to Kafka after all retries", err, utils.LogAttr("send_id", sendID), utils.LogAttr("attempts", cuc.maxRetries+1), utils.LogAttr("records", recordCount), utils.LogAttr("error_type", errorType))
		}
	}

	return false
}

func (cuc *ConsumerKafkaClient) addToRetryQueue(requests []UpdateMetricsRequest) {
	if cuc == nil {
		return
	}
	cuc.lock.Lock()
	defer cuc.lock.Unlock()

	// Check if retry queue is at capacity
	if len(cuc.retryQueue)+len(requests) > cuc.maxRetryQueueSize {
		// Calculate how many items we need to drop
		overflow := len(cuc.retryQueue) + len(requests) - cuc.maxRetryQueueSize
		if overflow > 0 {
			utils.LavaFormatWarning("Retry queue at capacity, dropping oldest items", nil, utils.LogAttr("dropped_items", overflow), utils.LogAttr("max_retry_queue_size", cuc.maxRetryQueueSize))
			// Remove oldest items from the beginning
			if overflow >= len(cuc.retryQueue) {
				cuc.retryQueue = cuc.retryQueue[:0]
			} else {
				cuc.retryQueue = cuc.retryQueue[overflow:]
			}
		}
	}

	cuc.retryQueue = append(cuc.retryQueue, requests...)

	// If we still exceed capacity after adding, drop the oldest items
	if len(cuc.retryQueue) > cuc.maxRetryQueueSize {
		excess := len(cuc.retryQueue) - cuc.maxRetryQueueSize
		cuc.retryQueue = cuc.retryQueue[excess:]
		utils.LavaFormatWarning("Dropped excess items after adding", nil, utils.LogAttr("dropped_items", excess), utils.LogAttr("max_retry_queue_size", cuc.maxRetryQueueSize))
	}

	utils.LavaFormatInfo("Added failed requests to retry queue", utils.LogAttr("retry_queue_length", len(cuc.retryQueue)), utils.LogAttr("max_retry_queue_size", cuc.maxRetryQueueSize))
}

func (cuc *ConsumerKafkaClient) Close() {
	if cuc == nil {
		return
	}

	// Cancel context to stop all operations
	cuc.cancel()

	// Wait for any in-flight operations to complete
	time.Sleep(100 * time.Millisecond)

	if cuc.ticker != nil {
		cuc.ticker.Stop()
	}

	if cuc.writer != nil {
		cuc.writer.Close()
	}

	utils.LavaFormatDebug("[CKC] Kafka client closed")
}
