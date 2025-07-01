package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// DiscordNotifier posts events to a Discord webhook URL.
type DiscordNotifier struct {
	WebhookURL string
}

func (n *DiscordNotifier) Notify(event MigrationEvent) error {
	if n.WebhookURL == "" {
		return nil
	}
	msg := formatMessage(event)
	payload := map[string]string{"content": msg}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(n.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook status %s", resp.Status)
	}
	return nil
}
