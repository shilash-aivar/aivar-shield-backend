package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Webhook struct {
	URL    string
	Secret string
	client *http.Client
}

func NewWebhook(url, secret string) *Webhook {
	return &Webhook{
		URL: url, Secret: secret,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *Webhook) Enabled() bool {
	return w != nil && w.URL != ""
}

func (w *Webhook) SuppressionStatusChanged(_ context.Context, ev SuppressionEvent) error {
	return w.post(map[string]any{
		"event": "suppression.status_changed", "payload": ev,
	})
}

func (w *Webhook) ExceptionExpired(_ context.Context, ev SuppressionEvent) error {
	return w.post(map[string]any{
		"event": "suppression.expired", "payload": ev,
	})
}

func (w *Webhook) post(body map[string]any) error {
	if !w.Enabled() {
		return nil
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.Secret != "" {
		req.Header.Set("X-Aivar-Secret", w.Secret)
	}
	res, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("webhook: %s", res.Status)
	}
	return nil
}
