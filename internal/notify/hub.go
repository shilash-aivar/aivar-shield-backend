package notify

import "context"

// Hub fans out suppression events to configured channels (Slack, email, …).
type Hub struct {
	Slack *Slack
	Email *Email
}

func NewHub(slack *Slack, email *Email) *Hub {
	return &Hub{Slack: slack, Email: email}
}

func (h *Hub) SuppressionStatusChanged(ctx context.Context, ev SuppressionEvent) error {
	if h == nil {
		return nil
	}
	if h.Slack != nil {
		_ = h.Slack.SuppressionStatusChanged(ctx, ev)
	}
	if h.Email != nil {
		_ = h.Email.SuppressionStatusChanged(ctx, ev)
	}
	return nil
}

func (h *Hub) ExceptionExpired(ctx context.Context, ev SuppressionEvent) error {
	if h == nil {
		return nil
	}
	if h.Slack != nil {
		_ = h.Slack.ExceptionExpired(ctx, ev)
	}
	if h.Email != nil {
		_ = h.Email.ExceptionExpired(ctx, ev)
	}
	return nil
}
