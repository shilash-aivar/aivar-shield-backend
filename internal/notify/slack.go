package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Slack struct {
	WebhookURL string
	UIURL      string
	client     *http.Client
}

func NewSlack(webhookURL, uiURL string) *Slack {
	return &Slack{
		WebhookURL: webhookURL,
		UIURL:      uiURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Slack) Enabled() bool {
	return s.WebhookURL != ""
}

type SuppressionEvent struct {
	PlatformRef string
	Repo        string
	Tool        string
	RuleID      string
	Status      string
	RequestedBy string
	ApprovedBy  string
	Reason      string
}

func (s *Slack) SuppressionStatusChanged(_ context.Context, ev SuppressionEvent) error {
	if !s.Enabled() {
		return nil
	}

	emoji := "✅"
	title := "Exception approved"
	if ev.Status == "rejected" {
		emoji = "❌"
		title = "Exception rejected"
	} else if ev.Status == "pending" {
		emoji = "⏳"
		title = "Exception filed"
	}

	text := fmt.Sprintf("%s *%s* — `%s` (%s)\n• Repo: `%s`\n• Rule: `%s` / %s\n• Filed by: %s",
		emoji, title, ev.PlatformRef, ev.Status, ev.Repo, ev.RuleID, ev.Tool, ev.RequestedBy)
	if ev.ApprovedBy != "" {
		text += fmt.Sprintf("\n• Actor: %s", ev.ApprovedBy)
	}
	if ev.Reason != "" {
		text += fmt.Sprintf("\n• Reason: %s", ev.Reason)
	}
	if s.UIURL != "" {
		text += fmt.Sprintf("\n• Portal: %s/exceptions/pending", s.UIURL)
	}
	if ev.Status == "approved" {
		text += "\n• Next step: run `aivar sync` in the repo"
	}

	payload := map[string]any{
		"text": text,
		"blocks": []map[string]any{
			{"type": "section", "text": map[string]string{"type": "mrkdwn", "text": text}},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("slack webhook: %s", res.Status)
	}
	return nil
}

func (s *Slack) ExceptionExpired(_ context.Context, ev SuppressionEvent) error {
	if !s.Enabled() {
		return nil
	}
	text := fmt.Sprintf("⏰ *Exception expired* — `%s`\n• Repo: `%s`\n• Rule: `%s` / %s\n• Filed by: %s\n• Run `aivar sync` to refresh local suppressions",
		ev.PlatformRef, ev.Repo, ev.RuleID, ev.Tool, ev.RequestedBy)
	if s.UIURL != "" {
		text += fmt.Sprintf("\n• Portal: %s/exceptions/all", s.UIURL)
	}
	payload := map[string]any{"text": text}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("slack webhook: %s", res.Status)
	}
	return nil
}
