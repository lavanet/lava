package streamer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lavanet/lava/v5/utils"
)

// SubscriptionManager manages all active subscriptions
type SubscriptionManager struct {
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	metrics       *StreamerMetrics
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(metrics *StreamerMetrics) *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
		metrics:       metrics,
	}
}

// Subscribe creates a new subscription
func (sm *SubscriptionManager) Subscribe(clientID string, filters *EventFilter, webhook *WebhookConfig, mq *MessageQueueConfig) (*Subscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub := &Subscription{
		ID:           uuid.New().String(),
		ClientID:     clientID,
		Filters:      filters,
		Webhook:      webhook,
		MessageQueue: mq,
		Active:       true,
		CreatedAt:    time.Now(),
		EventCount:   0,
	}

	sm.subscriptions[sub.ID] = sub

	utils.LavaFormatInfo("New subscription created",
		utils.LogAttr("subscriptionID", sub.ID),
		utils.LogAttr("clientID", clientID),
		utils.LogAttr("filters", filters),
	)

	return sub, nil
}

// Unsubscribe removes a subscription
func (sm *SubscriptionManager) Unsubscribe(subscriptionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	sub.Active = false
	delete(sm.subscriptions, subscriptionID)

	utils.LavaFormatInfo("Subscription removed",
		utils.LogAttr("subscriptionID", subscriptionID),
	)

	return nil
}

// GetSubscription retrieves a subscription by ID
func (sm *SubscriptionManager) GetSubscription(subscriptionID string) (*Subscription, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sub, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	return sub, nil
}

// GetAllSubscriptions returns all active subscriptions
func (sm *SubscriptionManager) GetAllSubscriptions() []*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := make([]*Subscription, 0, len(sm.subscriptions))
	for _, sub := range sm.subscriptions {
		if sub.Active {
			subs = append(subs, sub)
		}
	}

	return subs
}

// GetSubscriptionsByClient returns all subscriptions for a client
func (sm *SubscriptionManager) GetSubscriptionsByClient(clientID string) []*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.ClientID == clientID && sub.Active {
			subs = append(subs, sub)
		}
	}

	return subs
}

// UpdateWebSocket updates the WebSocket connection for a subscription
func (sm *SubscriptionManager) UpdateWebSocket(subscriptionID string, ws interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	sub.WebSocket = ws
	return nil
}

// MatchEvent checks if an event matches any active subscriptions
func (sm *SubscriptionManager) MatchEvent(event *StreamEvent) []*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	matches := make([]*Subscription, 0)

	for _, sub := range sm.subscriptions {
		if !sub.Active {
			continue
		}

		if sm.eventMatchesFilter(event, sub.Filters) {
			matches = append(matches, sub)
		}
	}

	return matches
}

// eventMatchesFilter checks if an event matches a filter
func (sm *SubscriptionManager) eventMatchesFilter(event *StreamEvent, filter *EventFilter) bool {
	if filter == nil {
		return true // No filter = match all
	}

	// Check chain ID
	if filter.ChainID != "" && filter.ChainID != event.ChainID {
		return false
	}

	// Check event types
	if len(filter.EventTypes) > 0 {
		matched := false
		for _, et := range filter.EventTypes {
			if et == event.Type {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check block range
	if filter.FromBlock != nil && event.BlockNumber < *filter.FromBlock {
		return false
	}
	if filter.ToBlock != nil && event.BlockNumber > *filter.ToBlock {
		return false
	}

	// Check addresses (from tx data)
	if len(filter.Addresses) > 0 {
		if event.Data == nil {
			return false
		}

		matched := false

		// Check various address fields
		if from, ok := event.Data["from_address"].(string); ok {
			for _, addr := range filter.Addresses {
				if from == addr {
					matched = true
					break
				}
			}
		}

		if to, ok := event.Data["to_address"].(string); ok && !matched {
			for _, addr := range filter.Addresses {
				if to == addr {
					matched = true
					break
				}
			}
		}

		if contract, ok := event.Data["contract_address"].(string); ok && !matched {
			for _, addr := range filter.Addresses {
				if contract == addr {
					matched = true
					break
				}
			}
		}

		if !matched {
			return false
		}
	}

	// Check specific address filters
	if filter.FromAddress != nil {
		if from, ok := event.Data["from_address"].(string); !ok || from != *filter.FromAddress {
			return false
		}
	}

	if filter.ToAddress != nil {
		if to, ok := event.Data["to_address"].(string); !ok || to != *filter.ToAddress {
			return false
		}
	}

	if filter.ContractAddress != nil {
		if contract, ok := event.Data["contract_address"].(string); !ok || contract != *filter.ContractAddress {
			return false
		}
	}

	// Check topics (for log events)
	if len(filter.Topics) > 0 && event.Type == EventTypeLog {
		if topics, ok := event.Data["topics"].([]interface{}); ok {
			matched := false
			for _, filterTopic := range filter.Topics {
				for _, eventTopic := range topics {
					if eventTopic == filterTopic {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	// Check decoded event type
	if filter.DecodedEventType != nil && event.Type == EventTypeDecodedEvent {
		if eventType, ok := event.Data["event_type"].(string); !ok || eventType != *filter.DecodedEventType {
			return false
		}
	}

	return true
}

// IncrementEventCount increments the event count for a subscription
func (sm *SubscriptionManager) IncrementEventCount(subscriptionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sub, exists := sm.subscriptions[subscriptionID]; exists {
		sub.EventCount++
		now := time.Now()
		sub.LastEventAt = &now
	}
}

// CleanupInactive removes inactive subscriptions
func (sm *SubscriptionManager) CleanupInactive(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.mu.Lock()
			toRemove := make([]string, 0)

			for id, sub := range sm.subscriptions {
				if !sub.Active {
					toRemove = append(toRemove, id)
				}
			}

			for _, id := range toRemove {
				delete(sm.subscriptions, id)
			}

			sm.mu.Unlock()

			if len(toRemove) > 0 {
				utils.LavaFormatInfo("Cleaned up inactive subscriptions",
					utils.LogAttr("count", len(toRemove)),
				)
			}
		}
	}
}

// GetActiveCount returns the number of active subscriptions
func (sm *SubscriptionManager) GetActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, sub := range sm.subscriptions {
		if sub.Active {
			count++
		}
	}

	return count
}
