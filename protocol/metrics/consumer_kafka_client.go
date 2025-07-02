package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/segmentio/kafka-go"

	"github.com/lavanet/lava/v5/utils"
)

type ConsumerKafkaClient struct {
	kafkaAddress       string
	topic              string
	addQueue           []UpdateMetricsRequest
	ticker             *time.Ticker
	lock               sync.RWMutex
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
}

func NewConsumerKafkaClient(kafkaAddress string, topic string) *ConsumerKafkaClient {
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

	writer := &kafka.Writer{
		Addr:     kafka.TCP(addresses...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	consumerKafkaClient := &ConsumerKafkaClient{
		kafkaAddress:       kafkaAddress,
		topic:              topic,
		addQueue:           []UpdateMetricsRequest{},
		ticker:             time.NewTicker(30 * time.Second),
		lock:               sync.RWMutex{},
		sendID:             0,
		isTickerRunning:    false,
		isSendQueueRunning: false,
		writer:             writer,
		ctx:                ctx,
		cancel:             cancel,
		retryQueue:         []UpdateMetricsRequest{},
		maxRetries:         3,
		retryDelay:         5 * time.Second,
	}

	utils.LavaFormatInfo("Starting Consumer Kafka Client", utils.LogAttr("kafka_address", kafkaAddress), utils.LogAttr("topic", topic))
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

	if !cuc.isSendQueueRunning && len(cuc.addQueue) > 0 {
		sendQueue := cuc.addQueue
		cuc.addQueue = make([]UpdateMetricsRequest, 0)
		cuc.isSendQueueRunning = true
		cuc.sendID++
		utils.LavaFormatDebug("[CKC] Swapped queues", utils.LogAttr("sendQueue_length", len((sendQueue))), utils.LogAttr("send_id", cuc.sendID))

		sendID := cuc.sendID

		go func() {
			cuc.sendKafkaData(sendQueue, sendID)

			cuc.lock.Lock()
			cuc.isSendQueueRunning = false
			cuc.lock.Unlock()
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
		// Calculate timeout with exponential backoff
		timeout := time.Duration(attempt+1) * cuc.retryDelay
		if timeout > 30*time.Second {
			timeout = 30 * time.Second
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)

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

		// Log the error with attempt information
		if attempt < cuc.maxRetries {
			utils.LavaFormatWarning("Kafka send failed, will retry", err, utils.LogAttr("send_id", sendID), utils.LogAttr("attempt", attempt+1), utils.LogAttr("max_retries", cuc.maxRetries))

			// Wait before retry (exponential backoff)
			backoffDelay := time.Duration(attempt+1) * cuc.retryDelay
			time.Sleep(backoffDelay)
		} else {
			// Final attempt failed
			utils.LavaFormatError("Failed to send data to Kafka after all retries", err, utils.LogAttr("send_id", sendID), utils.LogAttr("attempts", cuc.maxRetries+1), utils.LogAttr("records", recordCount))
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
	cuc.retryQueue = append(cuc.retryQueue, requests...)
	utils.LavaFormatInfo("Added failed requests to retry queue", utils.LogAttr("retry_queue_length", len(cuc.retryQueue)))
}

func (cuc *ConsumerKafkaClient) aggregateRelayData(reqs []UpdateMetricsRequest) []UpdateMetricsRequest {
	if cuc == nil {
		return nil
	}

	// Create a map to hold the aggregated data
	aggregated := make(map[string]*UpdateMetricsRequest)

	for _, req := range reqs {
		// Use the combination of RecordDate, Hash, Chain, and ApiType as the key
		key := generateRequestArregatedCacheKey(req)

		// If the key doesn't exist in the map, add it
		if _, exists := aggregated[key]; !exists {
			aggregated[key] = &UpdateMetricsRequest{
				RecordDate: req.RecordDate,
				Hash:       req.Hash,
				Chain:      req.Chain,
				ApiType:    req.ApiType,
			}
		}

		// Aggregate the data
		aggregated[key].RelaysInc += req.RelaysInc
		aggregated[key].CuInc += req.CuInc
		aggregated[key].LatencyToAdd += req.LatencyToAdd
		aggregated[key].LatencyAvgCount += req.LatencyAvgCount
	}

	// Convert the map to a slice
	var aggregatedSlice []UpdateMetricsRequest
	for _, req := range aggregated {
		aggregatedSlice = append(aggregatedSlice, *req)
	}

	return aggregatedSlice
}

func (cuc *ConsumerKafkaClient) Close() {
	if cuc == nil {
		return
	}

	cuc.cancel()

	if cuc.ticker != nil {
		cuc.ticker.Stop()
	}

	if cuc.writer != nil {
		cuc.writer.Close()
	}

	utils.LavaFormatDebug("[CKC] Kafka client closed")
}
