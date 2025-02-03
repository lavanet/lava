package monitoring_test

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/monitoring"
	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/assert"
)

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
