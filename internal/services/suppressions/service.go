package suppressions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/notify"
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	audit   *audit.Service
	notify  *notify.Hub
}

type ListFilter struct {
	Repo      string
	Status    string
	OrgID     string
	TeamID    string
	ProjectID string
}

func NewService(pool *pgxpool.Pool, auditSvc *audit.Service, hub *notify.Hub) *Service {
	return &Service{pool: pool, audit: auditSvc, notify: hub}
}

func (s *Service) Create(ctx context.Context, req models.CreateSuppressionRequest) (models.Suppression, error) {
	repoID, err := s.repoID(ctx, req.Repo)
	if err != nil {
		return models.Suppression{}, err
	}

	id := uuid.NewString()
	platformRef := fmt.Sprintf("EXC-%s", strings.ToUpper(id[:8]))
	scope := req.Scope
	if scope == "" {
		scope = "repo"
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return models.Suppression{}, fmt.Errorf("invalid expires_at: %w", err)
		}
		expiresAt = &t
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO suppressions (
			id, platform_ref, repo_id, tool, rule_id, suppression_type,
			file_path, line_number, reason, scope, status, requested_by, expires_at, native_comment
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'pending',$11,$12,$13)
		RETURNING id, platform_ref, repo_id, tool, rule_id, suppression_type,
			file_path, line_number, reason, scope, status, severity, requested_by,
			approved_by, expires_at, native_comment, created_at, updated_at
	`, id, platformRef, repoID, req.Tool, req.RuleID, req.Type, nullString(req.File),
		req.Line, req.Reason, scope, req.RequestedBy, expiresAt, nullString(req.NativeComment))

	sup, err := scanSuppression(row)
	if err != nil {
		return models.Suppression{}, err
	}
	sup.Repo = req.Repo

	_, _ = s.audit.Write(ctx, models.AuditEntry{
		Actor: req.RequestedBy, ActorType: "developer", Surface: "api",
		Action: "suppression_filed", ResourceType: "suppression", ResourceID: sup.ID,
		Repo: req.Repo, RuleID: req.RuleID, Tool: req.Tool,
	})

	_ = s.notify.SuppressionStatusChanged(ctx, notify.SuppressionEvent{
		PlatformRef: sup.PlatformRef,
		Repo:        req.Repo,
		Tool:        sup.Tool,
		RuleID:      sup.RuleID,
		Status:      sup.Status,
		RequestedBy: sup.RequestedBy,
		Reason:      sup.Reason,
	})

	return sup, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]models.Suppression, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.platform_ref, s.repo_id, r.full_name, s.tool, s.rule_id, s.suppression_type,
			s.file_path, s.line_number, s.reason, s.scope, s.status, s.severity, s.requested_by,
			s.approved_by, s.expires_at, s.native_comment, s.created_at, s.updated_at
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE ($1 = '' OR r.full_name = $1)
		  AND ($2 = '' OR s.status = $2)
		  AND ($3 = '' OR COALESCE(p.organization_id, p2.organization_id)::text = $3)
		  AND ($4 = '' OR COALESCE(p.team_id, p2.team_id)::text = $4)
		  AND ($5 = '' OR r.project_id::text = $5 OR rp.project_id::text = $5)
		ORDER BY s.created_at DESC
	`, filter.Repo, filter.Status, filter.OrgID, filter.TeamID, filter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("list suppressions: %w", err)
	}
	defer rows.Close()

	var out = make([]models.Suppression, 0)
	for rows.Next() {
		sup, err := scanSuppressionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sup)
	}
	return out, rows.Err()
}

func (s *Service) UpdateStatus(ctx context.Context, id string, req models.UpdateSuppressionStatusRequest) (models.Suppression, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return models.Suppression{}, fmt.Errorf("invalid expires_at: %w", err)
		}
		expiresAt = &t
	}

	row := s.pool.QueryRow(ctx, `
		UPDATE suppressions
		SET status = $2,
		    approved_by = $3,
		    scope = COALESCE(NULLIF($4, ''), scope),
		    expires_at = COALESCE($5, expires_at),
		    updated_at = now()
		WHERE id = $1
		RETURNING id, platform_ref, repo_id, tool, rule_id, suppression_type,
			file_path, line_number, reason, scope, status, severity, requested_by,
			approved_by, expires_at, native_comment, created_at, updated_at
	`, id, req.Status, nullString(req.ApprovedBy), req.Scope, expiresAt)

	sup, err := scanSuppression(row)
	if err != nil {
		return models.Suppression{}, fmt.Errorf("update suppression: %w", err)
	}

	action := "exception_rejected"
	if req.Status == "approved" {
		action = "exception_approved"
	} else if req.Status == "revoked" {
		action = "exception_revoked"
	} else if req.Status == "expired" {
		action = "exception_expired"
	}
	_, _ = s.audit.Write(ctx, models.AuditEntry{
		Actor: req.ApprovedBy, ActorType: "approver", Surface: "portal",
		Action: action, ResourceType: "suppression", ResourceID: sup.ID,
		RuleID: sup.RuleID, Tool: sup.Tool,
	})

	repo, _ := s.repoFullName(ctx, id)
	sup.Repo = repo
	_ = s.notify.SuppressionStatusChanged(ctx, notify.SuppressionEvent{
		PlatformRef: sup.PlatformRef,
		Repo:        repo,
		Tool:        sup.Tool,
		RuleID:      sup.RuleID,
		Status:      sup.Status,
		RequestedBy: sup.RequestedBy,
		ApprovedBy:  sup.ApprovedBy,
		Reason:      sup.Reason,
	})

	return sup, nil
}

func (s *Service) OrgIDForSuppression(ctx context.Context, id string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(
			p.organization_id,
			p2.organization_id,
			(SELECT id FROM organizations ORDER BY created_at LIMIT 1)
		)::text
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE s.id = $1
	`, id).Scan(&orgID)
	if err != nil {
		return "", fmt.Errorf("resolve suppression org: %w", err)
	}
	return orgID, nil
}

func (s *Service) TeamIDForSuppression(ctx context.Context, id string) (string, error) {
	var teamID string
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(p.team_id, p2.team_id)::text
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE s.id = $1
	`, id).Scan(&teamID)
	if err != nil {
		return "", nil
	}
	return teamID, nil
}

func (s *Service) repoFullName(ctx context.Context, suppressionID string) (string, error) {
	var fullName string
	err := s.pool.QueryRow(ctx, `
		SELECT r.full_name FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		WHERE s.id = $1
	`, suppressionID).Scan(&fullName)
	return fullName, err
}

func (s *Service) repoID(ctx context.Context, fullName string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM repos WHERE full_name = $1`, fullName).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("repo not found: %s", fullName)
	}
	return id, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSuppression(row scannable) (models.Suppression, error) {
	return scanSuppressionRow(row)
}

func scanSuppressionRow(row scannable) (models.Suppression, error) {
	var sup models.Suppression
	var file, severity, approvedBy, native *string
	err := row.Scan(
		&sup.ID, &sup.PlatformRef, &sup.RepoID, &sup.Repo, &sup.Tool, &sup.RuleID, &sup.Type,
		&file, &sup.Line, &sup.Reason, &sup.Scope, &sup.Status, &severity, &sup.RequestedBy,
		&approvedBy, &sup.ExpiresAt, &native, &sup.CreatedAt, &sup.UpdatedAt,
	)
	if err != nil {
		return models.Suppression{}, err
	}
	sup.File = deref(file)
	sup.Severity = deref(severity)
	sup.ApprovedBy = deref(approvedBy)
	sup.NativeComment = deref(native)
	return sup, nil
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
