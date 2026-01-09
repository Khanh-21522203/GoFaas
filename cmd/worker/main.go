package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	"GoFaas/internal/config"
	"GoFaas/internal/core/invocation"
	"GoFaas/internal/messaging"
	"GoFaas/internal/observability/logging"
	functionStorage "GoFaas/internal/storage/function"
	"GoFaas/internal/storage/metadata"
	"GoFaas/internal/worker"
	"GoFaas/internal/worker/runtime"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logging.NewSimpleLogger()
	logger.Info("Starting FaaS Worker", logging.F("worker_id", cfg.Worker.ID))

	// Initialize database
	db, err := sql.Open("postgres", cfg.Database.GetDSN())
	if err != nil {
		logger.Error("Failed to connect to database", logging.F("error", err))
		os.Exit(1)
	}
	defer db.Close()

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

	// Initialize runtime based on configuration
	var rt runtime.Runtime

	if cfg.Worker.UseContainer {
		logger.Info("Initializing container-based runtime")
		rt, err = runtime.NewContainerRuntime(cfg.Worker.WorkDir, logger)
		if err != nil {
			logger.Error("Failed to initialize container runtime", logging.F("error", err))
			logger.Info("Falling back to simple runtime")
			rt, err = runtime.NewSimpleRuntime(cfg.Worker.WorkDir)
			if err != nil {
				logger.Error("Failed to initialize simple runtime", logging.F("error", err))
				os.Exit(1)
			}
		}
	} else {
		logger.Info("Initializing simple runtime")
		rt, err = runtime.NewSimpleRuntime(cfg.Worker.WorkDir)
		if err != nil {
			logger.Error("Failed to initialize runtime", logging.F("error", err))
			os.Exit(1)
		}
	}

	// Initialize invocation service
	invocationService := invocation.NewService(metadataRepo, metadataRepo, queue, logger)

	// Initialize worker
	w := worker.NewWorker(worker.Config{
		ID:             cfg.Worker.ID,
		Queue:          queue,
		FunctionRepo:   metadataRepo,
		InvocationRepo: metadataRepo,
		FunctionStore:  funcStorage,
		Runtime:        rt,
		InvocationSvc:  invocationService,
		Logger:         logger,
	})

	// Start worker in goroutine
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := w.Start(workerCtx); err != nil {
			logger.Error("Worker error", logging.F("error", err))
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down gracefully...")

	// Stop worker
	w.Stop()
	cancel()

	logger.Info("Worker stopped")
}
