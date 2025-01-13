package monitoring_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lavanet/lava/v4/protocol/monitoring"
	"github.com/lavanet/lava/v4/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramAlerting_SendTelegramAlert(t *testing.T) {
	tests := []struct {
		name          string
		botToken      string
		channelID     string
		alert         string
		attrs         []utils.Attribute
		mockResponse  string
		mockStatus    int
		expectedError bool
		checkRequest  func(*testing.T, *http.Request)
	}{
		{
			name:      "successful alert",
			botToken:  "test_token",
			channelID: "test_channel",
			alert:     "Test Alert",
			attrs: []utils.Attribute{
				{Key: "severity", Value: "high"},
				{Key: "service", Value: "test-service"},
			},
			mockResponse:  `{"ok":true}`,
			mockStatus:    http.StatusOK,
			expectedError: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				// Check method and content type
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Read and verify request body
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				// Check if body contains expected content
				bodyStr := string(body)
				assert.Contains(t, bodyStr, "Test Alert")
				assert.Contains(t, bodyStr, "severity")
				assert.Contains(t, bodyStr, "high")
				assert.Contains(t, bodyStr, "service")
				assert.Contains(t, bodyStr, "test-service")
			},
		},
		{
			name:          "missing configuration",
			botToken:      "",
			channelID:     "",
			alert:         "Test Alert",
			attrs:         []utils.Attribute{},
			expectedError: true,
		},
		{
			name:          "server error",
			botToken:      "test_token",
			channelID:     "test_channel",
			alert:         "Test Alert",
			attrs:         []utils.Attribute{},
			mockResponse:  `{"ok":false}`,
			mockStatus:    http.StatusInternalServerError,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server if mockResponse is provided
			var ts *httptest.Server
			if tt.mockResponse != "" {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.checkRequest != nil {
						tt.checkRequest(t, r)
					}
					w.WriteHeader(tt.mockStatus)
					w.Write([]byte(tt.mockResponse))
				}))
				defer ts.Close()
			}

			// Initialize TelegramAlerting
			options := monitoring.TelegramAlertingOptions{
				TelegramBotToken:  tt.botToken,
				TelegramChannelID: tt.channelID,
			}
			alerting := monitoring.NewTelegramAlerting(options)

			// Send alert
			al := &monitoring.Alerting{TelegramAlerting: *alerting}
			err := al.SendTelegramAlert(tt.alert, tt.attrs)

			// Check error
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Optional: Integration test (disabled by default)
func TestTelegramAlerting_Integration(t *testing.T) {
	t.Skip("Integration test - run manually with valid credentials")

	options := monitoring.TelegramAlertingOptions{
		TelegramBotToken:  "YOUR_BOT_TOKEN",  // Replace with actual token
		TelegramChannelID: "YOUR_CHANNEL_ID", // Replace with actual channel ID
	}

	alerting := monitoring.NewTelegramAlerting(options)

	al := &monitoring.Alerting{TelegramAlerting: *alerting}
	err := al.SendTelegramAlert(
		"Integration Test Alert",
		[]utils.Attribute{
			{Key: "test_key", Value: "test_value"},
			{Key: "timestamp", Value: "2024-03-14 12:00:00"},
		},
	)

	assert.NoError(t, err)
}
