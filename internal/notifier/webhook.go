package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookNotifier sends the raw event as JSON to an HTTP endpoint.
type WebhookNotifier struct {
	URL     string
	Headers map[string]string
}

func (n *WebhookNotifier) Notify(event MigrationEvent) error {
	if n.URL == "" {
		return nil
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", n.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook status %s", resp.Status)
	}
	return nil
}
