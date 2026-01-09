package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents application error codes
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrCodeConflict            ErrorCode = "CONFLICT"
	ErrCodeValidation          ErrorCode = "VALIDATION_ERROR"
	ErrCodeRateLimitExceeded   ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Server errors (5xx)
	ErrCodeInternal            ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout             ErrorCode = "TIMEOUT"
	ErrCodeStorageError        ErrorCode = "STORAGE_ERROR"
	ErrCodeExecutionError      ErrorCode = "EXECUTION_ERROR"
)

// AppError represents an application error with code and HTTP status
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, details string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: getHTTPStatus(code),
	}
}

// getHTTPStatus maps error codes to HTTP status codes
func getHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeBadRequest, ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case ErrCodeTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Common error constructors
func NotFound(resource, id string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), id)
}

func ValidationError(message string) *AppError {
	return NewAppError(ErrCodeValidation, "Validation failed", message)
}

func InternalError(details string) *AppError {
	return NewAppError(ErrCodeInternal, "Internal server error", details)
}

func Conflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, "Resource conflict", message)
}
