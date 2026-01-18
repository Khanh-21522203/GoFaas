package controller

import (
	"net/http"
	"time"

	"GoFaas/internal/api/common"
	"GoFaas/internal/api/middleware"
	"GoFaas/internal/observability/logging"
	"GoFaas/pkg/errors"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authMiddleware *middleware.AuthMiddleware
	logger         logging.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authMiddleware *middleware.AuthMiddleware, logger logging.Logger) *AuthHandler {
	return &AuthHandler{
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token      string   `json:"token"`
	UserID     string   `json:"user_id"`
	ExpiresAt  string   `json:"expires_at"`
	Permission []string `json:"permissions"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	// TODO: Implement actual authentication logic
	// For now, accept any username/password and return a mock token
	if req.Username == "" || req.Password == "" {
		common.WriteError(w, errors.ValidationError("username and password are required"))
		return
	}

	// Mock user data - in real implementation, validate against database
	userID := req.Username
	permissions := []string{
		string(middleware.PermissionFunctionCreate),
		string(middleware.PermissionFunctionRead),
		string(middleware.PermissionFunctionUpdate),
		string(middleware.PermissionFunctionDelete),
		string(middleware.PermissionFunctionInvoke),
	}

	// Generate JWT token
	token, err := h.authMiddleware.GenerateToken(userID, permissions, 24*time.Hour)
	if err != nil {
		h.logger.Error("Failed to generate token", logging.F("error", err))
		common.WriteError(w, errors.InternalError("failed to generate token"))
		return
	}

	response := LoginResponse{
		Token:      token,
		UserID:     userID,
		ExpiresAt:  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		Permission: permissions,
	}

	common.WriteJSON(w, http.StatusOK, response)
}
