package streamer

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// WebhookSender handles webhook delivery
type WebhookSender struct {
	config          *StreamerConfig
	subscriptionMgr *SubscriptionManager
	metrics         *StreamerMetrics
	eventQueue      chan *webhookJob
	wg              sync.WaitGroup
	httpClient      *http.Client
}

type webhookJob struct {
	event        *StreamEvent
	subscription *Subscription
	attempt      int
}

// NewWebhookSender creates a new webhook sender
func NewWebhookSender(config *StreamerConfig, subMgr *SubscriptionManager, metrics *StreamerMetrics) *WebhookSender {
	return &WebhookSender{
		config:          config,
		subscriptionMgr: subMgr,
		metrics:         metrics,
		eventQueue:      make(chan *webhookJob, config.WebhookQueueSize),
		httpClient: &http.Client{
			Timeout: config.WebhookTimeout,
		},
	}
}

// Start starts webhook workers
func (wh *WebhookSender) Start(ctx context.Context) {
	if !wh.config.EnableWebhooks {
		utils.LavaFormatInfo("Webhook delivery disabled")
		return
	}

	utils.LavaFormatInfo("Starting webhook workers",
		utils.LogAttr("workers", wh.config.WebhookWorkers),
	)

	for i := 0; i < wh.config.WebhookWorkers; i++ {
		wh.wg.Add(1)
		go wh.worker(ctx, i)
	}
}

// Stop stops all webhook workers
func (wh *WebhookSender) Stop() {
	close(wh.eventQueue)
	wh.wg.Wait()
	utils.LavaFormatInfo("Webhook sender stopped")
}

// SendEvent queues an event for webhook delivery
func (wh *WebhookSender) SendEvent(event *StreamEvent, subscription *Subscription) {
	if subscription.Webhook == nil {
		return
	}

	select {
	case wh.eventQueue <- &webhookJob{
		event:        event,
		subscription: subscription,
		attempt:      0,
	}:
	default:
		utils.LavaFormatWarning("Webhook queue full, dropping event")
		wh.metrics.WebhooksFailed++
	}
}

// worker processes webhook deliveries
func (wh *WebhookSender) worker(ctx context.Context, workerID int) {
	defer wh.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wh.eventQueue:
			if !ok {
				return
			}
			wh.deliverWebhook(job)
		}
	}
}

// deliverWebhook delivers a webhook with retries
func (wh *WebhookSender) deliverWebhook(job *webhookJob) {
	config := job.subscription.Webhook

	// Prepare payload
	payload, err := json.Marshal(job.event)
	if err != nil {
		utils.LavaFormatError("Failed to marshal webhook payload", err)
		wh.metrics.WebhooksFailed++
		return
	}

	// Create request
	req, err := http.NewRequest("POST", config.URL, bytes.NewReader(payload))
	if err != nil {
		utils.LavaFormatError("Failed to create webhook request", err)
		wh.metrics.WebhooksFailed++
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lava-Streamer/1.0")
	req.Header.Set("X-Lava-Event-Type", string(job.event.Type))
	req.Header.Set("X-Lava-Event-ID", job.event.ID)
	req.Header.Set("X-Lava-Subscription-ID", job.subscription.ID)

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature if secret is configured
	if config.Secret != "" {
		signature := wh.generateHMAC(payload, config.Secret)
		req.Header.Set("X-Lava-Signature", signature)
	}

	// Send request
	resp, err := wh.httpClient.Do(req)
	if err != nil {
		wh.handleWebhookError(job, err)
		return
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		wh.metrics.WebhooksDelivered++
		wh.subscriptionMgr.IncrementEventCount(job.subscription.ID)
		return
	}

	// Retry on error
	err = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	wh.handleWebhookError(job, err)
}

// handleWebhookError handles webhook delivery errors with retry logic
func (wh *WebhookSender) handleWebhookError(job *webhookJob, err error) {
	job.attempt++

	maxRetries := job.subscription.Webhook.RetryAttempts
	if maxRetries == 0 {
		maxRetries = wh.config.WebhookMaxRetries
	}

	if job.attempt < maxRetries {
		// Retry with exponential backoff
		delay := time.Duration(job.attempt) * wh.config.WebhookRetryDelay

		utils.LavaFormatWarning("Webhook delivery failed, retrying", err,
			utils.LogAttr("attempt", job.attempt),
			utils.LogAttr("maxRetries", maxRetries),
			utils.LogAttr("delay", delay),
		)

		time.AfterFunc(delay, func() {
			wh.eventQueue <- job
		})
	} else {
		utils.LavaFormatError("Webhook delivery failed after max retries", err,
			utils.LogAttr("url", job.subscription.Webhook.URL),
			utils.LogAttr("attempts", job.attempt),
		)
		wh.metrics.WebhooksFailed++
	}
}

// generateHMAC generates HMAC-SHA256 signature
func (wh *WebhookSender) generateHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
