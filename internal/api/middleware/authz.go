package middleware

import (
	"GoFaas/internal/observability/logging"
	"GoFaas/pkg/errors"
	"context"
	"net/http"
)

// Permission represents a system permission
type Permission string

const (
	PermissionFunctionCreate Permission = "function:create"
	PermissionFunctionRead   Permission = "function:read"
	PermissionFunctionUpdate Permission = "function:update"
	PermissionFunctionDelete Permission = "function:delete"
	PermissionFunctionInvoke Permission = "function:invoke"
	PermissionInvocationRead Permission = "invocation:read"
	PermissionAdminAll       Permission = "admin:*"
)

// AuthzMiddleware handles authorization checks
type AuthzMiddleware struct {
	logger logging.Logger
}

// NewAuthzMiddleware creates a new authorization middleware
func NewAuthzMiddleware(logger logging.Logger) *AuthzMiddleware {
	return &AuthzMiddleware{
		logger: logger,
	}
}

// RequirePermission returns middleware that checks for required permission
func (a *AuthzMiddleware) RequirePermission(required Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get permissions from context (set by auth middleware)
			permissions, ok := r.Context().Value("permissions").([]string)
			if !ok {
				a.respondError(w, errors.NewAppError(
					errors.ErrCodeForbidden,
					"No permissions found in context",
					"",
				))
				return
			}

			// Check if user has required permission
			if !a.hasPermission(permissions, required) {
				userID, _ := r.Context().Value("user_id").(string)
				a.logger.Warn("Permission denied",
					logging.F("user_id", userID),
					logging.F("required", required),
					logging.F("has", permissions),
				)

				a.respondError(w, errors.NewAppError(
					errors.ErrCodeForbidden,
					"Insufficient permissions",
					string(required),
				))
				return
			}

			// Permission granted
			next.ServeHTTP(w, r)
		})
	}
}

// hasPermission checks if user has required permission
func (a *AuthzMiddleware) hasPermission(userPerms []string, required Permission) bool {
	requiredStr := string(required)

	for _, perm := range userPerms {
		// Exact match
		if perm == requiredStr {
			return true
		}

		// Admin wildcard
		if perm == string(PermissionAdminAll) {
			return true
		}

		// Wildcard match (e.g., "function:*" matches "function:read")
		if len(perm) > 2 && perm[len(perm)-1] == '*' {
			prefix := perm[:len(perm)-1]
			if len(requiredStr) >= len(prefix) && requiredStr[:len(prefix)] == prefix {
				return true
			}
		}
	}

	return false
}

// respondError writes error response
func (a *AuthzMiddleware) respondError(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	w.Write([]byte(`{"error":"` + err.Message + `"}`))
}

// GetUserID extracts user ID from context (helper)
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetPermissions extracts permissions from context (helper)
func GetPermissions(ctx context.Context) ([]string, bool) {
	perms, ok := ctx.Value("permissions").([]string)
	return perms, ok
}
