package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-payment-gateway/internal/config"
	"go-payment-gateway/internal/payment/gateway"
	"go-payment-gateway/internal/payment/repository"
	"go-payment-gateway/internal/payment/usecase"
	"go-payment-gateway/internal/worker"

	"github.com/hibiken/asynq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database
	logLevel := logger.Info
	if cfg.IsProduction() {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("worker connected to database")

	// Dependencies
	repo := repository.NewPostgresPaymentRepository(db)
	stripeGW := gateway.NewStripeGateway(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	paymentUC := usecase.NewPaymentUseCase(repo, stripeGW)

	// Asynq server
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err)
		os.Exit(1)
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			slog.Error("task failed",
				"type", task.Type(),
				"error", err,
			)
		}),
	})

	// Register task handlers
	mux := asynq.NewServeMux()
	taskHandler := worker.NewTaskHandler(paymentUC)
	taskHandler.RegisterHandlers(mux)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down worker...")
		srv.Shutdown()
	}()

	slog.Info("starting worker", "concurrency", 10, "env", cfg.AppEnv)
	if err := srv.Run(mux); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
