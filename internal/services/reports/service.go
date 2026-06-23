package reports

import (
	"context"
	"time"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/aivar-shield/backend/internal/services/suppressions"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool         *pgxpool.Pool
	suppressions *suppressions.Service
	audit        *audit.Service
}

func NewService(pool *pgxpool.Pool, supSvc *suppressions.Service, auditSvc *audit.Service) *Service {
	return &Service{pool: pool, suppressions: supSvc, audit: auditSvc}
}

type DeliveryReport struct {
	GeneratedAt  time.Time            `json:"generated_at"`
	Repo         string               `json:"repo,omitempty"`
	ProjectID    string               `json:"project_id,omitempty"`
	Summary      DeliverySummary      `json:"summary"`
	Suppressions []models.Suppression `json:"suppressions"`
	Audit        []models.AuditEntry  `json:"audit"`
}

type DeliverySummary struct {
	TotalSuppressions int `json:"total_suppressions"`
	Approved          int `json:"approved"`
	Pending           int `json:"pending"`
	Rejected          int `json:"rejected"`
	AuditEvents       int `json:"audit_events"`
}

func (s *Service) Delivery(ctx context.Context, repo, projectID, orgID string) (DeliveryReport, error) {
	filter := suppressions.ListFilter{Repo: repo, OrgID: orgID, ProjectID: projectID}
	items, err := s.suppressions.List(ctx, filter)
	if err != nil {
		return DeliveryReport{}, err
	}

	auditItems, err := s.audit.List(ctx, audit.ListFilter{
		Repo:      repo,
		OrgID:     orgID,
		ProjectID: projectID,
		Limit:     500,
	})
	if err != nil {
		return DeliveryReport{}, err
	}

	summary := DeliverySummary{TotalSuppressions: len(items), AuditEvents: len(auditItems)}
	for _, item := range items {
		switch item.Status {
		case "approved":
			summary.Approved++
		case "pending":
			summary.Pending++
		case "rejected":
			summary.Rejected++
		}
	}

	return DeliveryReport{
		GeneratedAt:  time.Now().UTC(),
		Repo:         repo,
		ProjectID:    projectID,
		Summary:      summary,
		Suppressions: items,
		Audit:        auditItems,
	}, nil
}
