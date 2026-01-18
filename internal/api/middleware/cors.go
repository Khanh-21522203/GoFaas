package middleware

import (
	"GoFaas/internal/observability/logging"
	"fmt"
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
	Logger           logging.Logger
}

// CORSMiddleware handles CORS
type CORSMiddleware struct {
	config CORSConfig
	logger logging.Logger
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(cfg CORSConfig) *CORSMiddleware {
	// Set defaults
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 3600 // 1 hour
	}

	return &CORSMiddleware{
		config: cfg,
		logger: cfg.Logger,
	}
}

// Middleware returns HTTP middleware that handles CORS
func (c *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && c.isOriginAllowed(origin) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)

			if c.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if len(c.config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.config.ExposedHeaders, ", "))
			}

			// Handle preflight request
			if r.Method == "OPTIONS" {
				c.handlePreflight(w, r)
				return
			}
		}

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if origin is in allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range c.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}

		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

// handlePreflight handles OPTIONS preflight requests
func (c *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request) {
	// Set allowed methods
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowedMethods, ", "))

	// Set allowed headers
	requestHeaders := r.Header.Get("Access-Control-Request-Headers")
	if requestHeaders != "" {
		// Validate requested headers
		if c.areHeadersAllowed(requestHeaders) {
			w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
		}
	} else {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
	}

	// Set max age
	w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", c.config.MaxAge))

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// areHeadersAllowed checks if requested headers are allowed
func (c *CORSMiddleware) areHeadersAllowed(requestHeaders string) bool {
	requested := strings.Split(requestHeaders, ",")
	for _, header := range requested {
		header = strings.TrimSpace(header)
		if !c.isHeaderAllowed(header) {
			return false
		}
	}
	return true
}

// isHeaderAllowed checks if a single header is allowed
func (c *CORSMiddleware) isHeaderAllowed(header string) bool {
	header = strings.ToLower(header)
	for _, allowed := range c.config.AllowedHeaders {
		if strings.ToLower(allowed) == header {
			return true
		}
	}
	return false
}
