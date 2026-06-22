package rules

import (
	"context"
	"fmt"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) List(ctx context.Context, tool string) ([]models.Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tool, rule_id, severity, title, description, fix_guide, docs_url, version, active
		FROM rules
		WHERE active = true
		  AND ($1 = '' OR tool = $1)
		ORDER BY tool, rule_id
	`, tool)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []models.Rule
	for rows.Next() {
		var rule models.Rule
		var description, fixGuide, docsURL *string
		if err := rows.Scan(
			&rule.ID, &rule.Tool, &rule.RuleID, &rule.Severity, &rule.Title,
			&description, &fixGuide, &docsURL, &rule.Version, &rule.Active,
		); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rule.Description = deref(description)
		rule.FixGuide = deref(fixGuide)
		rule.DocsURL = deref(docsURL)
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Service) GetByRuleID(ctx context.Context, ruleID string) (models.Rule, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, tool, rule_id, severity, title, description, fix_guide, docs_url, version, active
		FROM rules WHERE rule_id = $1 AND active = true LIMIT 1
	`, ruleID)

	var rule models.Rule
	var description, fixGuide, docsURL *string
	if err := row.Scan(
		&rule.ID, &rule.Tool, &rule.RuleID, &rule.Severity, &rule.Title,
		&description, &fixGuide, &docsURL, &rule.Version, &rule.Active,
	); err != nil {
		return models.Rule{}, fmt.Errorf("get rule: %w", err)
	}
	rule.Description = deref(description)
	rule.FixGuide = deref(fixGuide)
	rule.DocsURL = deref(docsURL)
	return rule, nil
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
