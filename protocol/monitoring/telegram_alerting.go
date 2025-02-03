package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lavanet/lava/v5/utils"
)

type TelegramAlertingOptions struct {
	TelegramBotToken  string
	TelegramChannelID string
}

const TELEGRAM_URL = "https://api.telegram.org"

func NewTelegramAlerting(options TelegramAlertingOptions) *TelegramAlertingOptions {
	return &TelegramAlertingOptions{
		TelegramBotToken:  options.TelegramBotToken,
		TelegramChannelID: options.TelegramChannelID,
	}
}

func (al *Alerting) SendTelegramAlert(alert string, attrs []utils.Attribute) error {
	if al.TelegramAlerting.TelegramBotToken == "" && al.TelegramAlerting.TelegramChannelID == "" {
		return nil
	} else if al.TelegramAlerting.TelegramBotToken == "" || al.TelegramAlerting.TelegramChannelID == "" {
		return fmt.Errorf("telegram configuration missing")
	}

	message := fmt.Sprintf("%s\n", alert)
	for _, attr := range attrs {
		message += fmt.Sprintf("%s: %v\n", attr.Key, attr.Value)
	}

	payload := map[string]string{
		"chat_id": al.TelegramAlerting.TelegramChannelID,
		"text":    message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", TELEGRAM_URL, al.TelegramAlerting.TelegramBotToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
