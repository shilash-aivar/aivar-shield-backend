package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool  *pgxpool.Pool
	audit *audit.Service
}

type ListFilter struct {
	Repo      string
	Status    string
	OrgID     string
	ProjectID string
}

func NewService(pool *pgxpool.Pool, auditSvc *audit.Service) *Service {
	return &Service{pool: pool, audit: auditSvc}
}

func (s *Service) Submit(ctx context.Context, req models.SubmitInfraReviewRequest) (models.InfraReview, error) {
	summary, add, change, destroy, cost := summarizePlan(req.PlanJSON)

	var repoID *string
	if req.Repo != "" {
		var id string
		if err := s.pool.QueryRow(ctx, `SELECT id FROM repos WHERE full_name = $1`, req.Repo).Scan(&id); err == nil {
			repoID = &id
		}
	}

	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO infra_reviews (
			id, repo_id, repo_full_name, org_id, project_id, plan_summary,
			changes_add, changes_change, changes_destroy, estimated_monthly_delta, status, submitted_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'pending',$11)
		RETURNING id, repo_full_name, status, submitted_by, approved_by, notes,
			changes_add, changes_change, changes_destroy, estimated_monthly_delta, plan_summary, created_at, updated_at
	`, id, repoID, req.Repo, nullUUID(req.OrgID), nullUUID(req.ProjectID), summary,
		add, change, destroy, cost, req.SubmittedBy)

	review, err := scanReview(row)
	if err != nil {
		return models.InfraReview{}, err
	}

	_, _ = s.audit.Write(ctx, models.AuditEntry{
		Actor: req.SubmittedBy, ActorType: "developer", Surface: "api",
		Action: "infra_plan_submitted", ResourceType: "infra_review", ResourceID: review.ID,
		Repo: req.Repo,
	})
	return review, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]models.InfraReview, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, repo_full_name, status, submitted_by, approved_by, notes,
			changes_add, changes_change, changes_destroy, estimated_monthly_delta, plan_summary, created_at, updated_at
		FROM infra_reviews
		WHERE ($1 = '' OR repo_full_name = $1)
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR org_id::text = $3)
		  AND ($4 = '' OR project_id::text = $4)
		ORDER BY created_at DESC
		LIMIT 100
	`, filter.Repo, filter.Status, filter.OrgID, filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.InfraReview, 0)
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, review)
	}
	return out, rows.Err()
}

func (s *Service) UpdateStatus(ctx context.Context, id string, req models.UpdateInfraReviewRequest) (models.InfraReview, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE infra_reviews
		SET status = $2, approved_by = $3, notes = COALESCE($4, notes), updated_at = now()
		WHERE id = $1
		RETURNING id, repo_full_name, status, submitted_by, approved_by, notes,
			changes_add, changes_change, changes_destroy, estimated_monthly_delta, plan_summary, created_at, updated_at
	`, id, req.Status, nullString(req.ApprovedBy), nullString(req.Notes))

	review, err := scanReview(row)
	if err != nil {
		return models.InfraReview{}, fmt.Errorf("update infra review: %w", err)
	}

	action := "infra_plan_rejected"
	if req.Status == "approved" {
		action = "infra_plan_approved"
	}
	_, _ = s.audit.Write(ctx, models.AuditEntry{
		Actor: req.ApprovedBy, ActorType: "approver", Surface: "portal",
		Action: action, ResourceType: "infra_review", ResourceID: id,
		Repo: review.Repo,
	})
	return review, nil
}

func summarizePlan(raw json.RawMessage) (json.RawMessage, int, int, int, *float64) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), 0, 0, 0, nil
	}
	var plan struct {
		ResourceChanges []struct {
			Change struct {
				Actions []string `json:"actions"`
			} `json:"change"`
			Type string `json:"type"`
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return raw, 0, 0, 0, nil
	}
	add, change, destroy := 0, 0, 0
	cost := 0.0
	for _, rc := range plan.ResourceChanges {
		for _, a := range rc.Change.Actions {
			switch strings.ToLower(a) {
			case "create":
				add++
				cost += estimateResourceCost(rc.Type)
			case "update":
				change++
				cost += estimateResourceCost(rc.Type) * 0.25
			case "delete":
				destroy++
				cost -= estimateResourceCost(rc.Type) * 0.5
			}
		}
	}
	summary, _ := json.Marshal(map[string]any{
		"resource_changes": len(plan.ResourceChanges),
		"add":              add,
		"change":           change,
		"destroy":          destroy,
		"estimated_monthly_delta_usd": cost,
	})
	var costPtr *float64
	if cost != 0 {
		costPtr = &cost
	}
	return summary, add, change, destroy, costPtr
}

func estimateResourceCost(resourceType string) float64 {
	switch {
	case strings.Contains(resourceType, "aws_instance"):
		return 80
	case strings.Contains(resourceType, "aws_db"):
		return 120
	case strings.Contains(resourceType, "aws_rds"):
		return 120
	case strings.Contains(resourceType, "aws_lb"):
		return 25
	case strings.Contains(resourceType, "aws_s3"):
		return 5
	default:
		return 15
	}
}

type scannable interface {
	Scan(dest ...any) error
}

func scanReview(row scannable) (models.InfraReview, error) {
	var r models.InfraReview
	var approvedBy, notes *string
	var cost *float64
	if err := row.Scan(
		&r.ID, &r.Repo, &r.Status, &r.SubmittedBy, &approvedBy, &notes,
		&r.ChangesAdd, &r.ChangesChange, &r.ChangesDestroy, &cost, &r.PlanSummary,
		&r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return models.InfraReview{}, err
	}
	r.ApprovedBy = deref(approvedBy)
	r.Notes = deref(notes)
	r.EstimatedMonthlyDelta = cost
	return r, nil
}

func nullUUID(v string) *string {
	if v == "" {
		return nil
	}
	return &v
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
