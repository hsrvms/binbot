package execution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookClient struct {
	URL        string
	HTTPClient *http.Client
}

func NewWebhookClient(url string) *WebhookClient {
	return &WebhookClient{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (w *WebhookClient) Broadcast(message string) error {
	if w.URL == "" {
		return ErrMissingWebhookConfig
	}

	payload, err := json.Marshal(map[string]string{
		"content": message,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to marshal payload: %v", ErrWebhookBroadcast, err)
	}

	resp, err := w.HTTPClient.Post(w.URL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("%w: http post failed: %v", ErrWebhookBroadcast, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: returned non-200 status: %d", ErrWebhookBroadcast, resp.StatusCode)
	}

	return nil
}
