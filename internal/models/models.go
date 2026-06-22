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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
}

type RegisterRepoRequest struct {
	FullName string   `json:"full_name"`
	Type     []string `json:"type"`
	Owner    string   `json:"owner"`
	TFOwner  string   `json:"tf_owner,omitempty"`
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
