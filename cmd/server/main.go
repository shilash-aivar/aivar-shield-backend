package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aivar-shield/backend/internal/api"
	"github.com/aivar-shield/backend/internal/api/handlers"
	"github.com/aivar-shield/backend/internal/config"
	"github.com/aivar-shield/backend/internal/db"
	"github.com/aivar-shield/backend/internal/notify"
	"github.com/aivar-shield/backend/internal/services/analytics"
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/aivar-shield/backend/internal/services/auth"
	"github.com/aivar-shield/backend/internal/services/infra"
	"github.com/aivar-shield/backend/internal/services/policy"
	"github.com/aivar-shield/backend/internal/services/repos"
	"github.com/aivar-shield/backend/internal/services/reports"
	"github.com/aivar-shield/backend/internal/services/rules"
	"github.com/aivar-shield/backend/internal/services/suppressions"
	"github.com/aivar-shield/backend/internal/services/tenants"
	"github.com/aivar-shield/backend/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	auditSvc := audit.NewService(pool, cfg.SuppressionSigningKey)
	slack := notify.NewSlack(cfg.SlackWebhookURL, cfg.UIURL)
	email := notify.NewEmail(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.EmailFrom, cfg.NotifyEmails, cfg.UIURL)
	webhook := notify.NewWebhook(cfg.WebhookURL, cfg.WebhookSecret)
	notifyHub := notify.NewHub(slack, email, webhook)
	reposSvc := repos.NewService(pool)
	suppressionsSvc := suppressions.NewService(pool, auditSvc, notifyHub)
	rulesSvc := rules.NewService(pool)
	tenantsSvc := tenants.NewService(pool)
	authSvc := auth.NewService(pool, cfg)
	reportsSvc := reports.NewService(pool, suppressionsSvc, auditSvc)
	analyticsSvc := analytics.NewService(pool)
	infraSvc := infra.NewService(pool, auditSvc)
	policySvc := policy.NewService(suppressionsSvc)
	artifacts := storage.NewArtifacts(cfg.ArtifactsDir, cfg.S3Bucket, cfg.S3Region)
	if err := tenantsSvc.SeedDefaults(ctx); err != nil {
		log.Printf("warning: seed tenants: %v", err)
	}
	if err := rulesSvc.SeedCatalog(ctx); err != nil {
		log.Printf("warning: seed catalog: %v", err)
	}

	h := handlers.New(reposSvc, suppressionsSvc, rulesSvc, auditSvc, tenantsSvc, authSvc, reportsSvc, analyticsSvc, infraSvc, policySvc, artifacts, cfg.SuppressionSigningKey)
	router := api.NewRouter(h)

	expiryCtx, expiryCancel := context.WithCancel(ctx)
	defer expiryCancel()
	go suppressions.RunExpiryWorker(expiryCtx, suppressionsSvc, 15*time.Minute)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("aivar-shield-backend listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
