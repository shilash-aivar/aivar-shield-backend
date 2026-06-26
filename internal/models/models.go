package models

import (
	"encoding/json"
	"time"
)

type Repo struct {
	ID        string    `json:"id"`
	Org       string    `json:"org"`
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	RepoType  []string  `json:"type"`
	Owner     string    `json:"owner,omitempty"`
	TFOwner   string    `json:"tf_owner,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Organization struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	GitHubOrgSlug string    `json:"github_org_slug,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Team struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
}

type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	TeamID         string    `json:"team_id,omitempty"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
}

type Member struct {
	UserID    string `json:"user_id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	Role      string `json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type AddMemberRequest struct {
	GitHubLogin string `json:"github_login"`
	Role        string `json:"role"`
}

type Rule struct {
	ID          string `json:"id"`
	Tool        string `json:"tool"`
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	FixGuide    string `json:"fix_guide,omitempty"`
	DocsURL     string `json:"docs_url,omitempty"`
	Version     int    `json:"version"`
	Active      bool   `json:"active"`
}

type Tool struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Binary       string `json:"binary"`
	OfficialDocs string `json:"official_docs"`
	RuleIndex    string `json:"rule_index"`
	Description  string `json:"description"`
}

type ExplainResponse struct {
	Rule          Rule   `json:"rule"`
	Tool          Tool   `json:"tool"`
	OfficialDocs  string `json:"official_docs"`
	RuleIndex     string `json:"rule_index"`
	ResolvedDocs  string `json:"resolved_docs_url"`
	CatalogSource bool   `json:"catalog_source"`
}

type Suppression struct {
	ID              string     `json:"id"`
	PlatformRef     string     `json:"platform_ref"`
	RepoID          string     `json:"repo_id"`
	Repo            string     `json:"repo,omitempty"`
	Tool            string     `json:"tool"`
	RuleID          string     `json:"rule"`
	Type            string     `json:"type"`
	File            string     `json:"file,omitempty"`
	Line            *int       `json:"line,omitempty"`
	Reason          string     `json:"reason"`
	Scope           string     `json:"scope"`
	Status          string     `json:"status"`
	Severity        string     `json:"severity,omitempty"`
	RequestedBy     string     `json:"requested_by"`
	ApprovedBy      string     `json:"approved_by,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	NativeComment   string     `json:"native_comment,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AuditEntry struct {
	ID           string          `json:"id"`
	Timestamp    time.Time       `json:"timestamp"`
	Actor        string          `json:"actor"`
	ActorType    string          `json:"actor_type"`
	Surface      string          `json:"surface"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Repo         string          `json:"repo,omitempty"`
	RuleID       string          `json:"rule,omitempty"`
	Tool         string          `json:"tool,omitempty"`
	Severity     string          `json:"severity,omitempty"`
	Details      json.RawMessage `json:"details"`
	Signature    string          `json:"signature"`
	PrevHash     string          `json:"prev_hash,omitempty"`
}

type RegisterRepoRequest struct {
	FullName  string   `json:"full_name"`
	Type      []string `json:"type"`
	Owner     string   `json:"owner"`
	TFOwner   string   `json:"tf_owner,omitempty"`
	ProjectID string   `json:"project_id,omitempty"`
}

type CreateOrganizationRequest struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	GitHubOrgSlug string `json:"github_org_slug,omitempty"`
}

type CreateTeamRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type CreateProjectRequest struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	TeamID string `json:"team_id,omitempty"`
}

type CreateSuppressionRequest struct {
	Repo          string `json:"repo"`
	Tool          string `json:"tool"`
	RuleID        string `json:"rule"`
	Type          string `json:"type"`
	File          string `json:"file,omitempty"`
	Line          *int   `json:"line,omitempty"`
	Reason        string `json:"reason"`
	Scope         string `json:"scope,omitempty"`
	RequestedBy   string `json:"requested_by"`
	NativeComment string `json:"native_comment,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type UpdateSuppressionStatusRequest struct {
	Status     string `json:"status"`
	ApprovedBy string `json:"approved_by"`
	Scope      string `json:"scope,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

type InfraReview struct {
	ID            string          `json:"id"`
	Repo          string          `json:"repo"`
	Status        string          `json:"status"`
	SubmittedBy   string          `json:"submitted_by"`
	ApprovedBy    string          `json:"approved_by,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	ChangesAdd    int             `json:"changes_add"`
	ChangesChange int             `json:"changes_change"`
	ChangesDestroy         int             `json:"changes_destroy"`
	EstimatedMonthlyDelta  *float64        `json:"estimated_monthly_delta,omitempty"`
	PlanSummary            json.RawMessage `json:"plan_summary"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SubmitInfraReviewRequest struct {
	Repo        string          `json:"repo"`
	OrgID       string          `json:"org_id,omitempty"`
	ProjectID   string          `json:"project_id,omitempty"`
	SubmittedBy string          `json:"submitted_by"`
	PlanJSON    json.RawMessage `json:"plan_json"`
}

type UpdateInfraReviewRequest struct {
	Status     string `json:"status"`
	ApprovedBy string `json:"approved_by"`
	Notes      string `json:"notes,omitempty"`
}
