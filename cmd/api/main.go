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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
	"github.com/mulki214/gdcpay_test_project/internal/config"
	"github.com/mulki214/gdcpay_test_project/internal/database"
	"github.com/mulki214/gdcpay_test_project/internal/httpapi"
	"github.com/mulki214/gdcpay_test_project/internal/repository/postgres"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("configure database connection", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := database.Migrate(migrationCtx, cfg.DatabaseURL); err != nil {
		migrationCancel()
		logger.Error("apply database migrations", "error", err)
		os.Exit(1)
	}
	migrationCancel()

	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		logger.Error("configure token manager", "error", err)
		os.Exit(1)
	}
	authService := auth.NewService(postgres.NewAuthRepository(pool), auth.BcryptHasher{}, tokenManager)
	taskService := task.NewService(postgres.NewTaskRepository(pool))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(pool, authService, taskService, tokenManager, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	}()

	logger.Info("http server starting", "address", cfg.HTTPAddr, "environment", cfg.AppEnv)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}
