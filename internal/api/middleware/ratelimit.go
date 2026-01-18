package middleware

import (
	"GoFaas/internal/observability/logging"
	"GoFaas/pkg/errors"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RedisClient *redis.Client
	Logger      logging.Logger
	// Requests per window
	RequestsPerWindow int
	// Window duration
	WindowDuration time.Duration
}

// RateLimitMiddleware handles rate limiting
type RateLimitMiddleware struct {
	redis             *redis.Client
	logger            logging.Logger
	requestsPerWindow int
	windowDuration    time.Duration
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(cfg RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redis:             cfg.RedisClient,
		logger:            cfg.Logger,
		requestsPerWindow: cfg.RequestsPerWindow,
		windowDuration:    cfg.WindowDuration,
	}
}

// Middleware returns HTTP middleware that enforces rate limits
func (rl *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get rate limit key (user ID or IP)
		key := rl.getRateLimitKey(r)

		// Check rate limit
		allowed, remaining, resetTime, err := rl.checkRateLimit(r.Context(), key)
		if err != nil {
			rl.logger.Error("Rate limit check failed",
				logging.F("error", err),
				logging.F("key", key),
			)
			// Fail open - allow request if rate limit check fails
			next.ServeHTTP(w, r)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.requestsPerWindow))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			rl.logger.Warn("Rate limit exceeded",
				logging.F("key", key),
				logging.F("reset_time", resetTime),
			)

			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			rl.respondError(w, errors.NewAppError(
				errors.ErrCodeRateLimitExceeded,
				"Rate limit exceeded",
				fmt.Sprintf("Try again after %s", resetTime.Format(time.RFC3339)),
			))
			return
		}

		// Allow request
		next.ServeHTTP(w, r)
	})
}

// getRateLimitKey determines the key for rate limiting
func (rl *RateLimitMiddleware) getRateLimitKey(r *http.Request) string {
	// Prefer user ID if authenticated
	if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
		return fmt.Sprintf("ratelimit:user:%s", userID)
	}

	// Fall back to IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	return fmt.Sprintf("ratelimit:ip:%s", ip)
}

// checkRateLimit checks if request is within rate limit using sliding window
func (rl *RateLimitMiddleware) checkRateLimit(ctx context.Context, key string) (allowed bool, remaining int, resetTime time.Time, err error) {
	now := time.Now()
	windowStart := now.Add(-rl.windowDuration)

	pipe := rl.redis.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))

	// Count requests in current window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration
	pipe.Expire(ctx, key, rl.windowDuration+time.Minute)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("redis pipeline failed: %w", err)
	}

	// Get count
	count := int(countCmd.Val())

	// Check if within limit
	allowed = count < rl.requestsPerWindow
	remaining = rl.requestsPerWindow - count - 1
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (end of current window)
	resetTime = now.Add(rl.windowDuration)

	return allowed, remaining, resetTime, nil
}

// respondError writes error response
func (rl *RateLimitMiddleware) respondError(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	w.Write([]byte(`{"error":"` + err.Message + `","details":"` + err.Details + `"}`))
}
