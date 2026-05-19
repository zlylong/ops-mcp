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

	"github.com/example/ops-mcp/backend/internal/api"
	"github.com/example/ops-mcp/backend/internal/audit"
	"github.com/example/ops-mcp/backend/internal/config"
	"github.com/example/ops-mcp/backend/internal/ops"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.FromEnv()
	auditor := audit.NewLogger(logger)
	svc := ops.NewService(cfg.Mode, auditor)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(cfg, svc, auditor, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("ops-mcp backend starting", "addr", cfg.Addr, "mode", cfg.Mode)
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
