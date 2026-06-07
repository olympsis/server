package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
)

// BotInterface is the main server's client for the bots microservice (Telegram/Discord
// "bot father"). It mirrors NotificationInterface: a thin typed HTTP client that fires
// reminder requests at the external service. A shared secret authenticates the call.
//
// When ServiceURL is empty the integration is treated as disabled and all calls no-op,
// so the server runs unchanged in deployments without the bots service.
type BotInterface struct {
	ServiceURL string
	Secret     string
	Client     http.Client
	Logger     *logrus.Logger
}

func NewBotInterface(url string, secret string, logger *logrus.Logger) *BotInterface {
	return &BotInterface{
		ServiceURL: url,
		Secret:     secret,
		Client:     http.Client{},
		Logger:     logger,
	}
}

// Enabled reports whether the bots integration is configured.
func (b *BotInterface) Enabled() bool {
	return b != nil && b.ServiceURL != ""
}

// SendReminder asks the bots service to post an event reminder into a linked chat. The
// request is fired asynchronously (best-effort), matching how notifications are sent.
func (b *BotInterface) SendReminder(reminder models.BotReminderRequest) error {
	if !b.Enabled() {
		return nil
	}

	data, err := json.Marshal(reminder)
	if err != nil {
		b.Logger.Errorf("Failed to marshal bot reminder: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/bots/reminders", b.ServiceURL), bytes.NewBuffer(data))
	if err != nil {
		b.Logger.Errorf("Failed to create bot reminder request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.Secret != "" {
		req.Header.Set("X-Internal-Secret", b.Secret)
	}

	go func() {
		resp, err := b.Client.Do(req)
		if err != nil {
			b.Logger.Errorf("Failed to send bot reminder: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			b.Logger.Errorf("Bots service returned non-success status code: %d", resp.StatusCode)
		}
	}()

	return nil
}
