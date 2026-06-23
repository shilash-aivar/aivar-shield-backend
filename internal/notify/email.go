package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type Email struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	To       []string
	UIURL    string
}

func NewEmail(host, port, user, pass, from string, to []string, uiURL string) *Email {
	if port == "" {
		port = "587"
	}
	return &Email{
		Host: host, Port: port, User: user, Password: pass,
		From: from, To: to, UIURL: uiURL,
	}
}

func (e *Email) Enabled() bool {
	return e != nil && e.Host != "" && e.From != "" && len(e.To) > 0
}

func (e *Email) SuppressionStatusChanged(_ context.Context, ev SuppressionEvent) error {
	if !e.Enabled() {
		return nil
	}
	subject := fmt.Sprintf("[Aivar Shield] Exception %s — %s", ev.Status, ev.PlatformRef)
	body := formatSuppressionText(ev, e.UIURL)
	return e.send(subject, body, ev.RequestedBy)
}

func (e *Email) ExceptionExpired(_ context.Context, ev SuppressionEvent) error {
	if !e.Enabled() {
		return nil
	}
	subject := fmt.Sprintf("[Aivar Shield] Exception expired — %s", ev.PlatformRef)
	body := fmt.Sprintf(
		"Exception %s has expired and no longer suppresses findings.\n\nRepo: %s\nRule: %s / %s\nOriginally filed by: %s\n\nRun aivar sync in the repo to refresh local suppressions.yaml.",
		ev.PlatformRef, ev.Repo, ev.RuleID, ev.Tool, ev.RequestedBy,
	)
	if e.UIURL != "" {
		body += fmt.Sprintf("\n\nPortal: %s/exceptions/all", e.UIURL)
	}
	return e.send(subject, body, ev.RequestedBy)
}

func formatSuppressionText(ev SuppressionEvent, uiURL string) string {
	title := "Exception filed"
	if ev.Status == "approved" {
		title = "Exception approved"
	} else if ev.Status == "rejected" {
		title = "Exception rejected"
	}
	text := fmt.Sprintf(
		"%s — %s (%s)\n\nRepo: %s\nRule: %s / %s\nFiled by: %s",
		title, ev.PlatformRef, ev.Status, ev.Repo, ev.RuleID, ev.Tool, ev.RequestedBy,
	)
	if ev.ApprovedBy != "" {
		text += fmt.Sprintf("\nActor: %s", ev.ApprovedBy)
	}
	if ev.Reason != "" {
		text += fmt.Sprintf("\nReason: %s", ev.Reason)
	}
	if ev.Status == "approved" {
		text += "\n\nNext step: run `aivar sync` in the repo."
	}
	if uiURL != "" {
		text += fmt.Sprintf("\n\nPortal: %s/exceptions/pending", uiURL)
	}
	return text
}

func (e *Email) send(subject, body, extraRecipient string) error {
	recipients := append([]string{}, e.To...)
	if strings.Contains(extraRecipient, "@") {
		recipients = appendUnique(recipients, strings.TrimSpace(extraRecipient))
	}
	if len(recipients) == 0 {
		return nil
	}

	msg := strings.Join([]string{
		"From: " + e.From,
		"To: " + strings.Join(recipients, ", "),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := e.Host + ":" + e.Port
	var auth smtp.Auth
	if e.User != "" {
		auth = smtp.PlainAuth("", e.User, e.Password, e.Host)
	}
	return smtp.SendMail(addr, auth, e.From, recipients, []byte(msg))
}

func appendUnique(items []string, v string) []string {
	for _, item := range items {
		if strings.EqualFold(item, v) {
			return items
		}
	}
	return append(items, v)
}
