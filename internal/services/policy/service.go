package policy

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/aivar-shield/backend/internal/services/suppressions"
)

type Finding struct {
	Tool     string `json:"tool"`
	RuleID   string `json:"rule_id"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
}

type EvaluateRequest struct {
	Repo      string    `json:"repo"`
	OrgID     string    `json:"org_id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	Findings  []Finding `json:"findings"`
}

type EvaluateResult struct {
	Repo       string    `json:"repo"`
	Blocking   []Finding `json:"blocking"`
	Suppressed []Finding `json:"suppressed"`
	Approved   int       `json:"approved_suppressions"`
	Pending    int       `json:"pending_suppressions"`
}

type Service struct {
	suppressions *suppressions.Service
}

func NewService(supSvc *suppressions.Service) *Service {
	return &Service{suppressions: supSvc}
}

func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (EvaluateResult, error) {
	items, err := s.suppressions.List(ctx, suppressions.ListFilter{
		Repo: req.Repo, OrgID: req.OrgID, ProjectID: req.ProjectID,
	})
	if err != nil {
		return EvaluateResult{}, err
	}

	approved := map[string]bool{}
	pending := 0
	approvedCount := 0
	now := time.Now().UTC()
	for _, item := range items {
		if item.Status == "pending" {
			pending++
		}
		if item.Status != "approved" {
			continue
		}
		if item.ExpiresAt != nil && item.ExpiresAt.Before(now) {
			continue
		}
		approvedCount++
		file := item.File
		line := 0
		if item.Line != nil {
			line = *item.Line
		}
		approved[matchKey(item.Tool, item.RuleID, file, line)] = true
		if item.Scope == "global" || item.Scope == "repo" {
			approved[matchKey(item.Tool, item.RuleID, "", 0)] = true
		}
	}

	result := EvaluateResult{
		Repo: req.Repo, Approved: approvedCount, Pending: pending,
	}
	for _, f := range req.Findings {
		key := matchKey(f.Tool, f.RuleID, f.File, f.Line)
		loose := matchKey(f.Tool, f.RuleID, "", 0)
		if approved[key] || approved[loose] {
			result.Suppressed = append(result.Suppressed, f)
		} else {
			result.Blocking = append(result.Blocking, f)
		}
	}
	return result, nil
}

func matchKey(tool, rule, file string, line int) string {
	return strings.ToLower(tool) + "|" + rule + "|" + file + "|" + strconv.Itoa(line)
}
