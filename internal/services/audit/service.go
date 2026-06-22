package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool       *pgxpool.Pool
	signingKey string
}

func NewService(pool *pgxpool.Pool, signingKey string) *Service {
	return &Service{pool: pool, signingKey: signingKey}
}

func (s *Service) Write(ctx context.Context, entry models.AuditEntry) (models.AuditEntry, error) {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if len(entry.Details) == 0 {
		entry.Details = json.RawMessage(`{}`)
	}

	entry.Signature = s.sign(entry)

	row := s.pool.QueryRow(ctx, `
		INSERT INTO audit_log (
			id, timestamp, actor, actor_type, surface, action,
			resource_type, resource_id, repo, rule_id, tool, severity, details, signature
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, timestamp, actor, actor_type, surface, action,
			resource_type, resource_id, repo, rule_id, tool, severity, details, signature
	`, entry.ID, entry.Timestamp, entry.Actor, entry.ActorType, entry.Surface, entry.Action,
		entry.ResourceType, entry.ResourceID, nullString(entry.Repo), nullString(entry.RuleID),
		nullString(entry.Tool), nullString(entry.Severity), entry.Details, entry.Signature)

	var out models.AuditEntry
	var repo, ruleID, tool, severity *string
	if err := row.Scan(
		&out.ID, &out.Timestamp, &out.Actor, &out.ActorType, &out.Surface, &out.Action,
		&out.ResourceType, &out.ResourceID, &repo, &ruleID, &tool, &severity, &out.Details, &out.Signature,
	); err != nil {
		return models.AuditEntry{}, fmt.Errorf("insert audit log: %w", err)
	}
	out.Repo = deref(repo)
	out.RuleID = deref(ruleID)
	out.Tool = deref(tool)
	out.Severity = deref(severity)
	return out, nil
}

func (s *Service) List(ctx context.Context, repo, action string, limit int) ([]models.AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, timestamp, actor, actor_type, surface, action,
			resource_type, resource_id, repo, rule_id, tool, severity, details, signature
		FROM audit_log
		WHERE ($1 = '' OR repo = $1)
		  AND ($2 = '' OR action = $2)
		ORDER BY timestamp DESC
		LIMIT $3
	`, repo, action, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []models.AuditEntry
	for rows.Next() {
		var entry models.AuditEntry
		var repoVal, ruleID, tool, severity *string
		if err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Actor, &entry.ActorType, &entry.Surface, &entry.Action,
			&entry.ResourceType, &entry.ResourceID, &repoVal, &ruleID, &tool, &severity,
			&entry.Details, &entry.Signature,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entry.Repo = deref(repoVal)
		entry.RuleID = deref(ruleID)
		entry.Tool = deref(tool)
		entry.Severity = deref(severity)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (s *Service) sign(entry models.AuditEntry) string {
	payload, _ := json.Marshal(struct {
		ID           string          `json:"id"`
		Timestamp    time.Time       `json:"timestamp"`
		Actor        string          `json:"actor"`
		Action       string          `json:"action"`
		ResourceType string          `json:"resource_type"`
		ResourceID   string          `json:"resource_id"`
		Details      json.RawMessage `json:"details"`
	}{
		ID: entry.ID, Timestamp: entry.Timestamp, Actor: entry.Actor, Action: entry.Action,
		ResourceType: entry.ResourceType, ResourceID: entry.ResourceID, Details: entry.Details,
	})
	mac := hmac.New(sha256.New, []byte(s.signingKey))
	mac.Write(payload)
	return "sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func nullString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
