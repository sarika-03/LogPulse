package plugin

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// WebhookConfig holds configuration for a webhook
// Example: { "url": "https://hooks.slack.com/services/...", "events": ["alert", "log"] }
type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// WebhookNotifier sends events to configured webhooks
// Usage: notifier.Notify("alert", map[string]interface{}{...})
type WebhookNotifier struct {
	Webhooks []WebhookConfig
}

func NewWebhookNotifier(cfgs []WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{Webhooks: cfgs}
}

func (w *WebhookNotifier) Notify(event string, payload map[string]interface{}) {
	for _, wh := range w.Webhooks {
		if !contains(wh.Events, event) {
			continue
		}
		go func(url string) {
			b, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Webhook error: %v", err)
				return
			}
			defer resp.Body.Close()
		}(wh.URL)
	}
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
