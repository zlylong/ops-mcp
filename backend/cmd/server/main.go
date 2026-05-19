package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zlylong/ops-mcp/backend/internal/adapters/kubernetes"
	"github.com/zlylong/ops-mcp/backend/internal/adapters/prometheus"
	"github.com/zlylong/ops-mcp/backend/internal/api"
	"github.com/zlylong/ops-mcp/backend/internal/app"
	"github.com/zlylong/ops-mcp/backend/internal/audit"
	"github.com/zlylong/ops-mcp/backend/internal/config"
	"github.com/zlylong/ops-mcp/backend/internal/policy"
	"github.com/zlylong/ops-mcp/backend/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	if cfg.DatabaseURL != "" && cfg.Mode != "mock" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db, err := storage.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("postgres unavailable", "error", err)
			os.Exit(1)
		}
		defer db.Close()
	}
	auditor := audit.NewStore(logger)
	registry := app.NewRegistry(policy.NewEngine(), auditor, storage.NewExecutionStore(), storage.NewApprovalStore(), cfg.Environment)
	if err := app.RegisterMockTools(registry, kubernetes.NewMockAdapter(), prometheus.NewMockAdapter()); err != nil {
		logger.Error("register tools", "error", err)
		os.Exit(1)
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: api.NewRouter(cfg, registry, auditor, logger), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("ops-mcp backend starting", "addr", cfg.Addr, "mode", cfg.Mode, "environment", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
