package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aivar-shield/backend/internal/gitconfig"
	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/audit"
	authsvc "github.com/aivar-shield/backend/internal/services/auth"
	"github.com/aivar-shield/backend/internal/services/repos"
	"github.com/aivar-shield/backend/internal/services/reports"
	"github.com/aivar-shield/backend/internal/services/rules"
	"github.com/aivar-shield/backend/internal/services/suppressions"
	"github.com/aivar-shield/backend/internal/services/tenants"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repos        *repos.Service
	suppressions *suppressions.Service
	rules        *rules.Service
	audit        *audit.Service
	tenants      *tenants.Service
	auth         *authsvc.Service
	reports      *reports.Service
}

func New(
	reposSvc *repos.Service,
	suppressionsSvc *suppressions.Service,
	rulesSvc *rules.Service,
	auditSvc *audit.Service,
	tenantsSvc *tenants.Service,
	authSvc *authsvc.Service,
	reportsSvc *reports.Service,
) *Handler {
	return &Handler{
		repos:        reposSvc,
		suppressions: suppressionsSvc,
		rules:        rulesSvc,
		audit:        auditSvc,
		tenants:      tenantsSvc,
		auth:         authSvc,
		reports:      reportsSvc,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Identity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, gitconfig.Current())
}

func (h *Handler) RegisterRepo(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	repo, err := h.repos.Register(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, repo)
}

func (h *Handler) GetRepo(w http.ResponseWriter, r *http.Request) {
	fullName := chi.URLParam(r, "fullName")
	repo, err := h.repos.GetByFullName(r.Context(), fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (h *Handler) ListSuppressions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.suppressions.List(r.Context(), suppressions.ListFilter{
		Repo:      q.Get("repo"),
		Status:    q.Get("status"),
		OrgID:     q.Get("org_id"),
		TeamID:    q.Get("team_id"),
		ProjectID: q.Get("project_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateSuppression(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSuppressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sup, err := h.suppressions.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sup)
}

func (h *Handler) UpdateSuppressionStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdateSuppressionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if h.auth.Enabled() {
		user, err := h.auth.UserFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		orgID, err := h.suppressions.OrgIDForSuppression(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ok, err := h.auth.CanApprove(r.Context(), user.ID, orgID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, "approver role required")
			return
		}
		if user.Email != "" {
			req.ApprovedBy = user.Email
		} else {
			req.ApprovedBy = user.Login
		}
	}

	sup, err := h.suppressions.UpdateStatus(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sup)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	items, err := h.rules.List(r.Context(), r.URL.Query().Get("tool"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")
	rule, err := h.rules.GetByRuleID(r.Context(), ruleID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) ExplainRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")
	tool := r.URL.Query().Get("tool")
	explain, err := h.rules.Explain(r.Context(), tool, ruleID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, explain)
}

func (h *Handler) ListTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.rules.ListTools())
}

func (h *Handler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}

	items, err := h.audit.List(r.Context(), audit.ListFilter{
		Repo:      q.Get("repo"),
		Action:    q.Get("action"),
		OrgID:     q.Get("org_id"),
		TeamID:    q.Get("team_id"),
		ProjectID: q.Get("project_id"),
		Limit:     limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	items, err := h.tenants.ListOrganizations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	items, err := h.tenants.ListTeams(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	items, err := h.tenants.ListProjects(r.Context(), orgID, r.URL.Query().Get("team_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	if h.auth.Enabled() {
		user, err := h.auth.UserFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		if !h.auth.IsPlatformAdmin(r.Context(), user) {
			writeError(w, http.StatusForbidden, "platform admin required")
			return
		}
	}
	var req models.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	org, err := h.tenants.CreateOrganization(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	var req models.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	team, err := h.tenants.CreateTeam(r.Context(), orgID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	project, err := h.tenants.CreateProject(r.Context(), orgID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) AuthConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.auth.ConfigPayload())
}

func (h *Handler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.StartLogin(w, r); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (h *Handler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.HandleCallback(w, r); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	me, err := h.auth.Me(r.Context(), r)
	if err != nil {
		if h.auth.Enabled() {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
	}
	writeJSON(w, http.StatusOK, me)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	items, err := h.tenants.ListMembers(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	member, err := h.tenants.AddMember(r.Context(), orgID, req.GitHubLogin, req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	userID := chi.URLParam(r, "userID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	var req models.UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.tenants.UpdateMemberRole(r.Context(), orgID, userID, req.Role); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) DeliveryReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	report, err := h.reports.Delivery(r.Context(), q.Get("repo"), q.Get("project_id"), q.Get("org_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if q.Get("format") == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderDeliveryHTML(report)))
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) requireOrgAdmin(w http.ResponseWriter, r *http.Request, orgID string) bool {
	if !h.auth.Enabled() {
		return true
	}
	user, err := h.auth.UserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "login required")
		return false
	}
	ok, err := h.auth.IsOrgAdmin(r.Context(), user.ID, orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !ok {
		writeError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
