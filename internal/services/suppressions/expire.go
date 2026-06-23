package suppressions

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/notify"
)

// ExpireDue marks approved suppressions past expires_at as expired and notifies subscribers.
func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE suppressions s
		SET status = 'expired', updated_at = now()
		WHERE s.status = 'approved'
		  AND s.expires_at IS NOT NULL
		  AND s.expires_at < now()
		RETURNING s.id, s.platform_ref, s.tool, s.rule_id, s.requested_by, s.reason,
		          (SELECT r.full_name FROM repos r WHERE r.id = s.repo_id)
	`)
	if err != nil {
		return 0, fmt.Errorf("expire suppressions: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, platformRef, tool, ruleID, requestedBy, reason, repo string
		if err := rows.Scan(&id, &platformRef, &tool, &ruleID, &requestedBy, &reason, &repo); err != nil {
			return count, err
		}
		count++
		_, _ = s.audit.Write(ctx, models.AuditEntry{
			Actor: "system", ActorType: "system", Surface: "platform",
			Action: "exception_expired", ResourceType: "suppression", ResourceID: id,
			Repo: repo, RuleID: ruleID, Tool: tool,
		})
		_ = s.notify.ExceptionExpired(ctx, notify.SuppressionEvent{
			PlatformRef: platformRef,
			Repo:        repo,
			Tool:        tool,
			RuleID:      ruleID,
			Status:      "expired",
			RequestedBy: requestedBy,
			Reason:      reason,
		})
	}
	return count, rows.Err()
}

// RunExpiryWorker polls for due suppressions until ctx is cancelled.
func RunExpiryWorker(ctx context.Context, svc *Service, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		n, err := svc.ExpireDue(ctx)
		if err != nil {
			log.Printf("expiry worker: %v", err)
			return
		}
		if n > 0 {
			log.Printf("expiry worker: marked %d suppression(s) expired", n)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
