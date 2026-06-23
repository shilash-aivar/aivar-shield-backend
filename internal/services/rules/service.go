package rules

import (
	"context"
	"fmt"

	"github.com/aivar-shield/backend/internal/catalog"
	"github.com/aivar-shield/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) SeedCatalog(ctx context.Context) error {
	for _, rule := range catalog.AllRules() {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO rules (id, tool, rule_id, severity, title, description, fix_guide, docs_url, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
			ON CONFLICT (tool, rule_id) DO UPDATE SET
				severity = EXCLUDED.severity,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				fix_guide = EXCLUDED.fix_guide,
				docs_url = EXCLUDED.docs_url,
				active = true
		`, uuid.NewString(), rule.Tool, rule.RuleID, rule.Severity, rule.Title,
			rule.Description, rule.FixGuide, rule.DocsURL)
		if err != nil {
			return fmt.Errorf("seed rule %s: %w", rule.RuleID, err)
		}
	}
	return nil
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

	var rules = make([]models.Rule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(rules) > 0 {
		return rules, nil
	}
	return s.listFromCatalog(tool), nil
}

func (s *Service) GetByRuleID(ctx context.Context, ruleID string) (models.Rule, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, tool, rule_id, severity, title, description, fix_guide, docs_url, version, active
		FROM rules WHERE rule_id = $1 AND active = true LIMIT 1
	`, ruleID)

	rule, err := scanRule(row)
	if err == nil {
		return rule, nil
	}
	if cat, ok := catalog.LookupRule(ruleID); ok {
		return catalogToModel(cat), nil
	}
	return models.Rule{}, fmt.Errorf("get rule: %w", err)
}

func (s *Service) Explain(ctx context.Context, tool, ruleID string) (models.ExplainResponse, error) {
	explain, err := catalog.Explain(tool, ruleID)
	if err != nil {
		return models.ExplainResponse{}, err
	}

	dbRule, dbErr := s.GetByRuleID(ctx, ruleID)
	if dbErr == nil {
		explain.Rule.Description = firstNonEmpty(dbRule.Description, explain.Rule.Description)
		explain.Rule.FixGuide = firstNonEmpty(dbRule.FixGuide, explain.Rule.FixGuide)
		explain.Rule.DocsURL = firstNonEmpty(dbRule.DocsURL, explain.Rule.DocsURL)
	}

	return models.ExplainResponse{
		Rule:          catalogToModel(explain.Rule),
		Tool:          catalogToolToModel(explain.Tool),
		OfficialDocs:  explain.OfficialDocs,
		RuleIndex:     explain.RuleIndex,
		ResolvedDocs:  explain.ResolvedDocs,
		CatalogSource: true,
	}, nil
}

func (s *Service) ListTools() []models.Tool {
	tools := catalog.AllTools()
	out := make([]models.Tool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, catalogToolToModel(tool))
	}
	return out
}

func (s *Service) listFromCatalog(tool string) []models.Rule {
	var source []catalog.Rule
	if tool == "" {
		source = catalog.AllRules()
	} else {
		source = catalog.RulesByTool(tool)
	}
	out := make([]models.Rule, 0, len(source))
	for _, rule := range source {
		out = append(out, catalogToModel(rule))
	}
	return out
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRule(row scannable) (models.Rule, error) {
	var rule models.Rule
	var description, fixGuide, docsURL *string
	if err := row.Scan(
		&rule.ID, &rule.Tool, &rule.RuleID, &rule.Severity, &rule.Title,
		&description, &fixGuide, &docsURL, &rule.Version, &rule.Active,
	); err != nil {
		return models.Rule{}, err
	}
	rule.Description = deref(description)
	rule.FixGuide = deref(fixGuide)
	rule.DocsURL = deref(docsURL)
	return rule, nil
}

func catalogToModel(rule catalog.Rule) models.Rule {
	return models.Rule{
		ID:          rule.RuleID,
		Tool:        rule.Tool,
		RuleID:      rule.RuleID,
		Severity:    rule.Severity,
		Title:       rule.Title,
		Description: rule.Description,
		FixGuide:    rule.FixGuide,
		DocsURL:     rule.DocsURL,
		Active:      true,
	}
}

func catalogToolToModel(tool catalog.Tool) models.Tool {
	return models.Tool{
		ID:           tool.ID,
		Name:         tool.Name,
		Category:     tool.Category,
		Binary:       tool.Binary,
		OfficialDocs: tool.OfficialDocs,
		RuleIndex:    tool.RuleIndex,
		Description:  tool.Description,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
