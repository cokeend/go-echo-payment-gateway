package main

import (
	"log/slog"
	"os"

	"go-payment-gateway/internal/config"
	"go-payment-gateway/internal/payment/delivery/http"
	"go-payment-gateway/internal/payment/gateway"
	"go-payment-gateway/internal/payment/repository"
	"go-payment-gateway/internal/payment/usecase"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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

	// Database (GORM)
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
	slog.Info("connected to database")

	// Auto-migrate
	repo := repository.NewPostgresPaymentRepository(db)
	if err := repo.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Asynq client (Redis queue)
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err)
		os.Exit(1)
	}
	queueClient := asynq.NewClient(redisOpt)
	defer queueClient.Close()
	slog.Info("connected to redis queue")

	// Dependency injection
	stripeGW := gateway.NewStripeGateway(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	paymentUC := usecase.NewPaymentUseCase(repo, stripeGW)
	paymentHandler := http.NewPaymentHandler(paymentUC, queueClient)

	// Echo setup
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	paymentHandler.RegisterRoutes(e, cfg.APIKey)

	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	slog.Info("starting server", "port", cfg.Port, "env", cfg.AppEnv)
	if err := e.Start(":" + cfg.Port); err != nil {
		slog.Error("server shutdown", "error", err)
	}
}
