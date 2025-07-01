package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackNotifier posts events to a Slack webhook URL.
type SlackNotifier struct {
	WebhookURL string
}

func (n *SlackNotifier) Notify(event MigrationEvent) error {
	if n.WebhookURL == "" {
		return nil
	}
	msg := formatMessage(event)
	payload := map[string]string{"text": msg}
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
		return fmt.Errorf("slack webhook status %s", resp.Status)
	}
	return nil
}
