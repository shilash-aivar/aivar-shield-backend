package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/policy"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) EvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	var req policy.EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Repo == "" {
		writeError(w, http.StatusBadRequest, "repo is required")
		return
	}
	result, err := h.policy.Evaluate(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) PublishDeliveryBundle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	repo := q.Get("repo")
	orgID := q.Get("org_id")
	projectID := q.Get("project_id")

	data, manifest, err := h.reports.BuildBundle(r.Context(), repo, projectID, orgID, h.signingKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.artifacts == nil || !h.artifacts.Enabled() {
		writeError(w, http.StatusBadRequest, "artifact storage not configured")
		return
	}
	prefix := "delivery"
	if repo != "" {
		prefix = strings.ReplaceAll(repo, "/", "-")
	}
	stored, err := h.artifacts.Put(r.Context(), prefix, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id := uuid.NewString()
	_ = h.reports.RecordArtifact(r.Context(), id, repo, orgID, projectID, stored.Key, manifest.Signature, stored.ByteSize)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         id,
		"url":        stored.URL,
		"storage":    stored.Storage,
		"byte_size":  stored.ByteSize,
		"signature":  manifest.Signature,
		"created_at": stored.CreatedAt,
	})
}

func (h *Handler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if h.artifacts == nil {
		writeError(w, http.StatusNotFound, "artifacts not configured")
		return
	}
	data, err := h.artifacts.GetLocal(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	_, _ = w.Write(data)
}

func (h *Handler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	member, err := h.tenants.AddTeamMember(r.Context(), teamID, req.GitHubLogin, req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamID")
	orgID := chi.URLParam(r, "orgID")
	if !h.requireOrgAdmin(w, r, orgID) {
		return
	}
	items, err := h.tenants.ListTeamMembers(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateOrganizationUI(w http.ResponseWriter, r *http.Request) {
	h.CreateOrganization(w, r)
}
