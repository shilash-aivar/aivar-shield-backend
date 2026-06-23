package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalog.json
var rawCatalog []byte

type Tool struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Binary       string   `json:"binary"`
	Aliases      []string `json:"aliases,omitempty"`
	OfficialDocs string   `json:"official_docs"`
	RuleIndex    string   `json:"rule_index"`
	Description  string   `json:"description"`
}

type Rule struct {
	Tool        string `json:"tool"`
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FixGuide    string `json:"fix_guide"`
	DocsURL     string `json:"docs_url"`
}

type Data struct {
	Tools []Tool `json:"tools"`
	Rules []Rule `json:"rules"`
}

var loaded Data

func init() {
	if err := json.Unmarshal(rawCatalog, &loaded); err != nil {
		panic(fmt.Sprintf("catalog: %v", err))
	}
}

func AllTools() []Tool {
	return loaded.Tools
}

func AllRules() []Rule {
	return loaded.Rules
}

func GetTool(id string) (Tool, bool) {
	normalized := NormalizeTool(id)
	for _, tool := range loaded.Tools {
		if tool.ID == normalized {
			return tool, true
		}
		for _, alias := range tool.Aliases {
			if alias == normalized {
				return tool, true
			}
		}
	}
	return Tool{}, false
}

func NormalizeTool(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func GetRule(tool, ruleID string) (Rule, bool) {
	normalizedTool := NormalizeTool(tool)
	normalizedRule := strings.TrimSpace(ruleID)
	for _, rule := range loaded.Rules {
		if rule.RuleID == normalizedRule && (normalizedTool == "" || NormalizeTool(rule.Tool) == normalizedTool) {
			return enrichRule(rule), true
		}
	}
	return Rule{}, false
}

func LookupRule(ruleID string) (Rule, bool) {
	normalizedRule := strings.TrimSpace(ruleID)
	for _, rule := range loaded.Rules {
		if rule.RuleID == normalizedRule {
			return enrichRule(rule), true
		}
	}
	if inferredTool := InferTool(ruleID); inferredTool != "" {
		return Rule{
			Tool:     inferredTool,
			RuleID:   ruleID,
			Severity: "unknown",
			Title:    fmt.Sprintf("%s rule %s", inferredTool, ruleID),
			DocsURL:  ResolveDocsURL(inferredTool, ruleID),
		}, true
	}
	return Rule{}, false
}

func RulesByTool(tool string) []Rule {
	normalized := NormalizeTool(tool)
	var out []Rule
	for _, rule := range loaded.Rules {
		if NormalizeTool(rule.Tool) == normalized {
			out = append(out, enrichRule(rule))
		}
	}
	return out
}

func enrichRule(rule Rule) Rule {
	if rule.DocsURL == "" {
		rule.DocsURL = ResolveDocsURL(rule.Tool, rule.RuleID)
	}
	return rule
}

func InferTool(ruleID string) string {
	switch {
	case strings.HasPrefix(ruleID, "DL"):
		return "hadolint"
	case strings.HasPrefix(ruleID, "CKV_"):
		return "checkov"
	case strings.HasPrefix(ruleID, "AWS") || strings.HasPrefix(ruleID, "GCP") || strings.HasPrefix(ruleID, "AZU"):
		return "tfsec"
	case strings.HasPrefix(ruleID, "LICENSE-"):
		return "grant"
	case strings.HasPrefix(ruleID, "GHSA-"):
		return "grype"
	case strings.HasPrefix(ruleID, "CVE-"):
		return "trivy"
	case strings.HasPrefix(ruleID, "CIS-DI-"):
		return "dockle"
	case strings.HasPrefix(ruleID, "AVD-"), strings.HasPrefix(ruleID, "DS-"):
		return "trivy"
	case strings.HasPrefix(ruleID, "SBOM-"):
		return "sbom"
	case strings.Contains(ruleID, ":"):
		return "sonarqube"
	case strings.Contains(ruleID, "."):
		return "semgrep"
	default:
		return ""
	}
}

func ResolveDocsURL(tool, ruleID string) string {
	tool = NormalizeTool(tool)
	switch tool {
	case "hadolint":
		return fmt.Sprintf("https://github.com/hadolint/hadolint/wiki/%s", ruleID)
	case "checkov":
		return "https://www.checkov.io/5.Policy%20Index/terraform.html"
	case "tfsec":
		return fmt.Sprintf("https://aquasecurity.github.io/tfsec/latest/checks/aws/%s/", strings.ToLower(ruleID))
	case "trivy":
		if strings.HasPrefix(ruleID, "CVE-") {
			return fmt.Sprintf("https://avd.aquasecurity.github.io/nvd/%s/", strings.ToLower(ruleID))
		}
		return fmt.Sprintf("https://avd.aquasecurity.github.io/misconfig/%s", strings.ToLower(ruleID))
	case "grype":
		return fmt.Sprintf("https://osv.dev/vulnerability/%s", ruleID)
	case "grant":
		if strings.HasPrefix(ruleID, "LICENSE-") {
			return "https://oss.anchore.com/docs/guides/license/policies/"
		}
		return "https://oss.anchore.com/docs/guides/license/"
	case "dockle":
		return "https://github.com/goodwithtech/dockle/blob/master/CHECKS.md"
	case "semgrep":
		return fmt.Sprintf("https://semgrep.dev/r/%s", ruleID)
	case "tflint":
		return "https://github.com/terraform-linters/tflint/tree/master/docs/rules"
	case "sonarqube":
		parts := strings.Split(ruleID, ":")
		if len(parts) == 2 {
			lang := parts[0]
			return fmt.Sprintf("https://rules.sonarsource.com/%s/RSPEC-%s/", lang, strings.TrimPrefix(parts[1], "S"))
		}
		return "https://rules.sonarsource.com/"
	case "sbom":
		return "https://cyclonedx.org/capabilities/sbom/"
	default:
		if t, ok := GetTool(tool); ok {
			return t.RuleIndex
		}
		return ""
	}
}

type ExplainResponse struct {
	Rule           Rule   `json:"rule"`
	Tool           Tool   `json:"tool"`
	OfficialDocs   string `json:"official_docs"`
	RuleIndex      string `json:"rule_index"`
	ResolvedDocs   string `json:"resolved_docs_url"`
	CatalogSource  bool   `json:"catalog_source"`
}

func Explain(tool, ruleID string) (ExplainResponse, error) {
	rule, ok := GetRule(tool, ruleID)
	if !ok {
		rule, ok = LookupRule(ruleID)
	}
	if !ok {
		return ExplainResponse{}, fmt.Errorf("rule not found: %s", ruleID)
	}

	toolMeta, ok := GetTool(rule.Tool)
	if !ok {
		toolMeta = Tool{ID: rule.Tool, Name: rule.Tool, RuleIndex: ResolveDocsURL(rule.Tool, ruleID)}
	}

	return ExplainResponse{
		Rule:          rule,
		Tool:          toolMeta,
		OfficialDocs:  toolMeta.OfficialDocs,
		RuleIndex:     toolMeta.RuleIndex,
		ResolvedDocs:  rule.DocsURL,
		CatalogSource: ok,
	}, nil
}
