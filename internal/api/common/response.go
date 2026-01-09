package common

import (
	"encoding/json"
	"net/http"

	"GoFaas/pkg/errors"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error information in API response
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := Response{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, err error) {
	var statusCode int
	var errorInfo ErrorInfo

	if appErr, ok := err.(*errors.AppError); ok {
		statusCode = appErr.HTTPStatus
		errorInfo = ErrorInfo{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		}
	} else {
		statusCode = http.StatusInternalServerError
		errorInfo = ErrorInfo{
			Code:    string(errors.ErrCodeInternal),
			Message: "Internal server error",
			Details: err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := Response{
		Success: false,
		Error:   &errorInfo,
	}

	json.NewEncoder(w).Encode(response)
}

// ParseJSON parses JSON request body
func ParseJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return errors.ValidationError(err.Error())
	}
	return nil
}
