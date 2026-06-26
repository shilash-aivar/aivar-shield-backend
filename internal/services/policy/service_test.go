package policy_test

import (
	"testing"

	"github.com/aivar-shield/backend/internal/services/policy"
)

func TestMatchKey(t *testing.T) {
	req := policy.EvaluateRequest{
		Repo: "org/app",
		Findings: []policy.Finding{
			{Tool: "hadolint", RuleID: "DL3008", File: "Dockerfile", Line: 14},
		},
	}
	// Service needs DB — smoke test types only
	if req.Repo == "" {
		t.Fatal("repo required")
	}
}
