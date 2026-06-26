package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aivar-shield/backend/internal/gitconfig"
	"github.com/aivar-shield/backend/internal/models"
	"github.com/aivar-shield/backend/internal/services/analytics"
	"github.com/aivar-shield/backend/internal/services/infra"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.repos.List(r.Context(), q.Get("org_id"), q.Get("project_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	summary, err := h.analytics.Summary(r.Context(), analytics.Scope{
		OrgID: q.Get("org_id"), TeamID: q.Get("team_id"), ProjectID: q.Get("project_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) VerifyAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, h.audit.Verify(r.Context(), limit))
}

func (h *Handler) DeliveryBundle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	data, manifest, err := h.reports.BuildBundle(r.Context(), q.Get("repo"), q.Get("project_id"), q.Get("org_id"), h.signingKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	filename := fmt.Sprintf("aivar-delivery-%s.zip", manifest.GeneratedAt.Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("X-Aivar-Manifest-Signature", manifest.Signature)
	_, _ = w.Write(data)
}

func (h *Handler) ListInfraReviews(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.infra.List(r.Context(), infra.ListFilter{
		Repo: q.Get("repo"), Status: q.Get("status"),
		OrgID: q.Get("org_id"), ProjectID: q.Get("project_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) SubmitInfraReview(w http.ResponseWriter, r *http.Request) {
	var req models.SubmitInfraReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Repo == "" || len(req.PlanJSON) == 0 {
		writeError(w, http.StatusBadRequest, "repo and plan_json are required")
		return
	}
	if req.SubmittedBy == "" {
		req.SubmittedBy = gitconfig.Current().Email
		if req.SubmittedBy == "" {
			req.SubmittedBy = "unknown"
		}
	}
	review, err := h.infra.Submit(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func (h *Handler) UpdateInfraReviewStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.UpdateInfraReviewRequest
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
		me, err := h.auth.Me(r.Context(), r)
		if err != nil || !me.CanApprove {
			writeError(w, http.StatusForbidden, "approver role required")
			return
		}
		if user.Email != "" {
			req.ApprovedBy = user.Email
		} else {
			req.ApprovedBy = user.Login
		}
	}
	review, err := h.infra.UpdateStatus(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, review)
}
