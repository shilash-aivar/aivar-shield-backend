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
	"github.com/aivar-shield/backend/internal/services/audit"
	"github.com/aivar-shield/backend/internal/services/repos"
	"github.com/aivar-shield/backend/internal/services/rules"
	"github.com/aivar-shield/backend/internal/services/suppressions"
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
	reposSvc := repos.NewService(pool)
	suppressionsSvc := suppressions.NewService(pool, auditSvc)
	rulesSvc := rules.NewService(pool)

	h := handlers.New(reposSvc, suppressionsSvc, rulesSvc, auditSvc)
	router := api.NewRouter(h)

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
