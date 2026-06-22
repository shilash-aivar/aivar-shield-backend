package api

import (
	"net/http"
	"time"

	"github.com/aivar-shield/backend/internal/api/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(h *handlers.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/repos", h.RegisterRepo)
		r.Get("/repos/{fullName}", h.GetRepo)

		r.Get("/suppressions", h.ListSuppressions)
		r.Post("/suppressions", h.CreateSuppression)
		r.Patch("/suppressions/{id}/status", h.UpdateSuppressionStatus)

		r.Get("/rules", h.ListRules)
		r.Get("/rules/{ruleID}", h.GetRule)

		r.Get("/audit", h.ListAuditLog)
	})

	return r
}
