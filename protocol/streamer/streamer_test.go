package streamer

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.RPCEndpoint == "" {
		t.Error("RPCEndpoint should not be empty")
	}

	if config.ChainID == "" {
		t.Error("ChainID should not be empty")
	}

	if config.PollingInterval <= 0 {
		t.Error("PollingInterval should be positive")
	}

	if !config.StreamTransactions && !config.StreamLogs {
		t.Error("At least one stream type should be enabled")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *StreamerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing rpc endpoint",
			config: &StreamerConfig{
				ChainID: "ETH1",
			},
			wantErr: true,
		},
		{
			name: "missing chain id",
			config: &StreamerConfig{
				RPCEndpoint: "http://localhost:3333",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventFilter(t *testing.T) {
	filter := &EventFilter{
		EventTypes:      []StreamEventType{EventTypeTokenTransfer},
		ChainID:         "ETH1",
		ContractAddress: stringPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
	}

	if filter.ChainID != "ETH1" {
		t.Error("ChainID mismatch")
	}

	if len(filter.EventTypes) != 1 {
		t.Error("Should have one event type")
	}

	if filter.EventTypes[0] != EventTypeTokenTransfer {
		t.Error("Event type should be token_transfer")
	}
}

func TestSubscription(t *testing.T) {
	sub := &Subscription{
		ID:       "test-123",
		ClientID: "client-1",
		Filters: &EventFilter{
			EventTypes: []StreamEventType{EventTypeTransaction},
		},
		Active:    true,
		CreatedAt: time.Now(),
	}

	if sub.ID != "test-123" {
		t.Error("ID mismatch")
	}

	if !sub.Active {
		t.Error("Subscription should be active")
	}
}

func TestStreamEvent(t *testing.T) {
	event := &StreamEvent{
		ID:          "evt-123",
		Type:        EventTypeTokenTransfer,
		ChainID:     "ETH1",
		BlockNumber: 12345,
		Data: map[string]interface{}{
			"from":  "0xabc",
			"to":    "0xdef",
			"value": "1000",
		},
		EmittedAt: time.Now(),
	}

	if event.Type != EventTypeTokenTransfer {
		t.Error("Event type mismatch")
	}

	if event.BlockNumber != 12345 {
		t.Error("Block number mismatch")
	}

	if len(event.Data) != 3 {
		t.Error("Should have 3 data fields")
	}
}

func TestNewStreamerMetrics(t *testing.T) {
	metrics := NewStreamerMetrics()

	if metrics == nil {
		t.Fatal("NewStreamerMetrics returned nil")
	}

	stats := metrics.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if _, ok := stats["blocks_processed"]; !ok {
		t.Error("Stats should include blocks_processed")
	}
}

func stringPtr(s string) *string {
	return &s
}

