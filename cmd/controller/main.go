package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	"GoFaas/internal/api/controller"
	"GoFaas/internal/api/middleware"
	"GoFaas/internal/config"
	"GoFaas/internal/core/function"
	"GoFaas/internal/core/invocation"
	"GoFaas/internal/messaging"
	"GoFaas/internal/observability/logging"
	functionStorage "GoFaas/internal/storage/function"
	"GoFaas/internal/storage/metadata"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logging.NewSimpleLogger()
	logger.Info("Starting FaaS Controller")

	// Initialize database
	db, err := sql.Open("postgres", cfg.Database.GetDSN())
	if err != nil {
		logger.Error("Failed to connect to database", logging.F("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", logging.F("error", err))
		os.Exit(1)
	}
	logger.Info("Database connection established")

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", logging.F("error", err))
		os.Exit(1)
	}
	logger.Info("Redis connection established")

	// Initialize repositories
	metadataRepo := metadata.NewPostgresRepository(db)

	// Initialize function storage
	funcStorage, err := functionStorage.NewLocalStorage(cfg.Storage.BaseDir)
	if err != nil {
		logger.Error("Failed to initialize function storage", logging.F("error", err))
		os.Exit(1)
	}

	// Initialize message queue
	queue := messaging.NewRedisQueue(redisClient, "faas")

	// Initialize services
	functionService := function.NewService(metadataRepo, funcStorage, logger)
	invocationService := invocation.NewService(metadataRepo, metadataRepo, queue, logger)

	// Initialize HTTP handlers
	functionHandler := controller.NewFunctionHandler(functionService, logger)
	invocationHandler := controller.NewInvocationHandler(invocationService, logger)

	// Initialize auth middleware and handlers
	authMiddleware := middleware.NewAuthMiddleware(middleware.AuthConfig{
		JWTSecret:     "change-me-in-production", // TODO: Load from config
		TokenDuration: 24 * time.Hour,
		Logger:        logger,
	})
	authHandler := controller.NewAuthHandler(authMiddleware, logger)
	authzMiddleware := middleware.NewAuthzMiddleware(logger)

	// Initialize HTTP server
	server := controller.NewServer(controller.Config{
		Addr:              cfg.Server.Addr,
		FunctionHandler:   functionHandler,
		InvocationHandler: invocationHandler,
		AuthHandler:       authHandler,
		AuthMiddleware:    authMiddleware,
		AuthzMiddleware:   authzMiddleware,
		RedisClient:       redisClient,
		Logger:            logger,
	})

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Server error", logging.F("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down gracefully...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Failed to stop server gracefully", logging.F("error", err))
	}

	logger.Info("Controller stopped")
}
