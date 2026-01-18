package middleware

import (
	"GoFaas/internal/observability/logging"
	"GoFaas/pkg/errors"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string
	TokenDuration time.Duration
	Logger        logging.Logger
}

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	secret []byte
	logger logging.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		secret: []byte(cfg.JWTSecret),
		logger: cfg.Logger,
	}
}

// Claims represents JWT token claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// Middleware returns HTTP middleware that validates JWT tokens
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.respondError(w, errors.NewAppError(
				errors.ErrCodeUnauthorized,
				"Missing authorization header",
				"",
			))
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			a.respondError(w, errors.NewAppError(
				errors.ErrCodeUnauthorized,
				"Invalid authorization header format",
				"Expected: Bearer <token>",
			))
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims, err := a.validateToken(tokenString)
		if err != nil {
			//a.respondError(w, err)
			a.respondError(w, errors.NewAppError(
				errors.ErrCodeUnauthorized,
				"Invalid authotization token",
				"",
			))
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "permissions", claims.Permissions)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken parses and validates a JWT token
func (a *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.NewAppError(
				errors.ErrCodeUnauthorized,
				"Invalid signing method",
				"",
			)
		}
		return a.secret, nil
	})

	if err != nil {
		a.logger.Warn("Token validation failed", logging.F("error", err))
		return nil, errors.NewAppError(
			errors.ErrCodeUnauthorized,
			"Invalid or expired token",
			err.Error(),
		)
	}

	if !token.Valid {
		return nil, errors.NewAppError(
			errors.ErrCodeUnauthorized,
			"Token is not valid",
			"",
		)
	}

	return claims, nil
}

// respondError writes error response
func (a *AuthMiddleware) respondError(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	// Write error response (simplified)
	w.Write([]byte(`{"error":"` + err.Message + `"}`))
}

// GenerateToken generates a new JWT token (helper for auth service)
func (a *AuthMiddleware) GenerateToken(userID string, permissions []string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID:      userID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}
