package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"GoFaas/internal/observability/logging"
)

// Server represents the HTTP API server
type Server struct {
	addr              string
	functionHandler   *FunctionHandler
	invocationHandler *InvocationHandler
	logger            logging.Logger
	server            *http.Server
}

// Config holds server configuration
type Config struct {
	Addr              string
	FunctionHandler   *FunctionHandler
	InvocationHandler *InvocationHandler
	Logger            logging.Logger
}

// NewServer creates a new API server
func NewServer(cfg Config) *Server {
	return &Server{
		addr:              cfg.Addr,
		functionHandler:   cfg.FunctionHandler,
		invocationHandler: cfg.InvocationHandler,
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

	// Function management routes
	router.HandleFunc("/functions", s.functionHandler.CreateFunction).Methods("POST")
	router.HandleFunc("/functions", s.functionHandler.ListFunctions).Methods("GET")
	router.HandleFunc("/functions/{id}", s.functionHandler.GetFunction).Methods("GET")
	router.HandleFunc("/functions/{id}", s.functionHandler.UpdateFunction).Methods("PUT")
	router.HandleFunc("/functions/{id}", s.functionHandler.DeleteFunction).Methods("DELETE")

	// Invocation routes
	router.HandleFunc("/invoke", s.invocationHandler.InvokeFunction).Methods("POST")
	router.HandleFunc("/invocations/{id}", s.invocationHandler.GetInvocationResult).Methods("GET")
	router.HandleFunc("/invocations", s.invocationHandler.ListInvocations).Methods("GET")

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
