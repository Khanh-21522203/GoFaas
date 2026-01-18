package controller

import (
	"GoFaas/internal/api/middleware"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"

	"GoFaas/internal/observability/logging"
)

// Server represents the HTTP API server
type Server struct {
	addr              string
	functionHandler   *FunctionHandler
	invocationHandler *InvocationHandler
	authHandler       *AuthHandler
	authMiddleware    *middleware.AuthMiddleware
	authzMiddleware   *middleware.AuthzMiddleware
	redisClient       *redis.Client
	logger            logging.Logger
	server            *http.Server
}

// Config holds server configuration
type Config struct {
	Addr              string
	FunctionHandler   *FunctionHandler
	InvocationHandler *InvocationHandler
	AuthHandler       *AuthHandler
	AuthMiddleware    *middleware.AuthMiddleware
	AuthzMiddleware   *middleware.AuthzMiddleware
	RedisClient       *redis.Client
	Logger            logging.Logger
}

// NewServer creates a new API server
func NewServer(cfg Config) *Server {
	return &Server{
		addr:              cfg.Addr,
		functionHandler:   cfg.FunctionHandler,
		invocationHandler: cfg.InvocationHandler,
		authHandler:       cfg.AuthHandler,
		authMiddleware:    cfg.AuthMiddleware,
		authzMiddleware:   cfg.AuthzMiddleware,
		redisClient:       cfg.RedisClient,
		logger:            cfg.Logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := s.setupRoutes()

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting API server", logging.F("addr", s.addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", s.healthCheck).Methods("GET")
	router.HandleFunc("/auth/login", s.authHandler.Login).Methods("POST")

	// Protected routes (auth required)
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(s.authMiddleware.Middleware)

	// Function management routes
	protected.Handle("/functions",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionCreate)(
			http.HandlerFunc(s.functionHandler.CreateFunction),
		)).Methods("POST")

	protected.Handle("/functions",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionRead)(
			http.HandlerFunc(s.functionHandler.ListFunctions),
		)).Methods("GET")

	protected.Handle("/functions/{id}",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionRead)(
			http.HandlerFunc(s.functionHandler.GetFunction),
		)).Methods("GET")

	protected.Handle("/functions/{id}",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionUpdate)(
			http.HandlerFunc(s.functionHandler.UpdateFunction),
		)).Methods("PUT")

	protected.Handle("/functions/{id}",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionDelete)(
			http.HandlerFunc(s.functionHandler.DeleteFunction),
		)).Methods("DELETE")

	// Invocation routes
	protected.Handle("/invoke",
		s.authzMiddleware.RequirePermission(middleware.PermissionFunctionInvoke)(
			http.HandlerFunc(s.invocationHandler.InvokeFunction),
		)).Methods("POST")
	router.HandleFunc("/invocations/{id}", s.invocationHandler.GetInvocationResult).Methods("GET")
	router.HandleFunc("/invocations", s.invocationHandler.ListInvocations).Methods("GET")

	corsMiddleware := middleware.NewCORSMiddleware(middleware.CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           3600,
		Logger:           s.logger,
	})
	router.Use(corsMiddleware.Middleware)

	standardLimiter := middleware.NewRateLimitMiddleware(middleware.RateLimitConfig{
		RedisClient:       s.redisClient,
		Logger:            s.logger,
		RequestsPerWindow: 100,
		WindowDuration:    time.Minute,
	})

	router.Use(standardLimiter.Middleware)

	// Add logging middleware
	router.Use(s.loggingMiddleware)

	return router
}

// healthCheck handles health check requests
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call next handler
		next.ServeHTTP(w, r)

		// Log request
		s.logger.Info("HTTP request",
			logging.F("method", r.Method),
			logging.F("path", r.URL.Path),
			logging.F("duration", time.Since(start)),
		)
	})
}
