package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/aivar-shield/backend/internal/services/repos"
	"github.com/aivar-shield/backend/internal/services/rules"
	"github.com/aivar-shield/backend/internal/services/suppressions"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repos        *repos.Service
	suppressions *suppressions.Service
	rules        *rules.Service
	audit        *audit.Service
}

func New(
	reposSvc *repos.Service,
	suppressionsSvc *suppressions.Service,
	rulesSvc *rules.Service,
	auditSvc *audit.Service,
) *Handler {
	return &Handler{
		repos:        reposSvc,
		suppressions: suppressionsSvc,
		rules:        rulesSvc,
		audit:        auditSvc,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	items, err := h.suppressions.List(r.Context(), r.URL.Query().Get("repo"), r.URL.Query().Get("status"))
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

func (h *Handler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}

	items, err := h.audit.List(r.Context(), r.URL.Query().Get("repo"), r.URL.Query().Get("action"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
