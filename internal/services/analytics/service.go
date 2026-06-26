package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Scope struct {
	OrgID     string
	TeamID    string
	ProjectID string
}

type Summary struct {
	PendingSuppressions   int            `json:"pending_suppressions"`
	ApprovedSuppressions  int            `json:"approved_suppressions"`
	RejectedSuppressions  int            `json:"rejected_suppressions"`
	ExpiredSuppressions   int            `json:"expired_suppressions"`
	AuditEvents30d        int            `json:"audit_events_30d"`
	ReposRegistered       int            `json:"repos_registered"`
	PendingInfraReviews   int            `json:"pending_infra_reviews"`
	SuppressionsByTool    []ToolCount    `json:"suppressions_by_tool"`
	ApprovalsLast7Days    []DayCount     `json:"approvals_last_7_days"`
	AvgApprovalHours      *float64       `json:"avg_approval_hours,omitempty"`
}

type ToolCount struct {
	Tool  string `json:"tool"`
	Count int    `json:"count"`
}

type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Summary(ctx context.Context, scope Scope) (Summary, error) {
	var out Summary

	scopeSQL := `
		AND ($1 = '' OR COALESCE(p.organization_id, p2.organization_id)::text = $1)
		AND ($2 = '' OR COALESCE(p.team_id, p2.team_id)::text = $2)
		AND ($3 = '' OR r.project_id::text = $3 OR rp.project_id::text = $3)
	`

	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE s.status = 'pending'),
			COUNT(*) FILTER (WHERE s.status = 'approved'),
			COUNT(*) FILTER (WHERE s.status = 'rejected'),
			COUNT(*) FILTER (WHERE s.status = 'expired')
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE true `+scopeSQL,
		scope.OrgID, scope.TeamID, scope.ProjectID,
	).Scan(&out.PendingSuppressions, &out.ApprovedSuppressions, &out.RejectedSuppressions, &out.ExpiredSuppressions)
	if err != nil {
		return out, fmt.Errorf("suppression counts: %w", err)
	}

	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_log a
		LEFT JOIN repos r ON r.full_name = a.repo
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE a.timestamp > now() - interval '30 days'
		AND ($1 = '' OR COALESCE(p.organization_id, p2.organization_id)::text = $1 OR a.repo = '')
		AND ($2 = '' OR COALESCE(p.team_id, p2.team_id)::text = $2 OR a.repo = '')
		AND ($3 = '' OR r.project_id::text = $3 OR rp.project_id::text = $3 OR a.repo = '')
	`, scope.OrgID, scope.TeamID, scope.ProjectID).Scan(&out.AuditEvents30d)

	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT r.id) FROM repos r
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE ($1 = '' OR COALESCE(p.organization_id, p2.organization_id)::text = $1)
		AND ($2 = '' OR COALESCE(p.team_id, p2.team_id)::text = $2)
		AND ($3 = '' OR r.project_id::text = $3 OR rp.project_id::text = $3)
	`, scope.OrgID, scope.TeamID, scope.ProjectID).Scan(&out.ReposRegistered)

	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM infra_reviews i
		WHERE i.status = 'pending'
		AND ($1 = '' OR i.org_id::text = $1)
		AND ($3 = '' OR i.project_id::text = $3)
	`, scope.OrgID, scope.TeamID, scope.ProjectID).Scan(&out.PendingInfraReviews)

	rows, err := s.pool.Query(ctx, `
		SELECT s.tool, COUNT(*) FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE s.status IN ('approved', 'pending') `+scopeSQL+`
		GROUP BY s.tool ORDER BY COUNT(*) DESC LIMIT 10
	`, scope.OrgID, scope.TeamID, scope.ProjectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tc ToolCount
			if err := rows.Scan(&tc.Tool, &tc.Count); err == nil {
				out.SuppressionsByTool = append(out.SuppressionsByTool, tc)
			}
		}
	}

	dayRows, err := s.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', s.updated_at), 'YYYY-MM-DD'), COUNT(*)
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE s.status = 'approved'
		AND s.updated_at > now() - interval '7 days' `+scopeSQL+`
		GROUP BY 1 ORDER BY 1
	`, scope.OrgID, scope.TeamID, scope.ProjectID)
	if err == nil {
		defer dayRows.Close()
		for dayRows.Next() {
			var dc DayCount
			if err := dayRows.Scan(&dc.Day, &dc.Count); err == nil {
				out.ApprovalsLast7Days = append(out.ApprovalsLast7Days, dc)
			}
		}
	}

	var avgHours float64
	if err := s.pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (s.updated_at - s.created_at)) / 3600.0)
		FROM suppressions s
		JOIN repos r ON r.id = s.repo_id
		LEFT JOIN projects p ON p.id = r.project_id
		LEFT JOIN repo_projects rp ON rp.repo_id = r.id
		LEFT JOIN projects p2 ON p2.id = rp.project_id
		WHERE s.status = 'approved' `+scopeSQL,
		scope.OrgID, scope.TeamID, scope.ProjectID,
	).Scan(&avgHours); err == nil && avgHours > 0 {
		out.AvgApprovalHours = &avgHours
	}

	return out, nil
}
