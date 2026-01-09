package function

import (
	"time"

	"GoFaas/pkg/types"
)

// CreateFunctionRequest represents a function creation request
type CreateFunctionRequest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Runtime     types.RuntimeType `json:"runtime"`
	Handler     string            `json:"handler"`
	Code        string            `json:"code"` // Base64 encoded
	Timeout     time.Duration     `json:"timeout"`
	Memory      int               `json:"memory_mb"`
	Environment map[string]string `json:"environment"`
	Concurrency int               `json:"max_concurrency"`
	Metadata    map[string]string `json:"metadata"`
}

// UpdateFunctionRequest represents a function update request
type UpdateFunctionRequest struct {
	Handler     *string            `json:"handler,omitempty"`
	Code        *string            `json:"code,omitempty"` // Base64 encoded
	Timeout     *time.Duration     `json:"timeout,omitempty"`
	Memory      *int               `json:"memory_mb,omitempty"`
	Environment map[string]string  `json:"environment,omitempty"`
	Concurrency *int               `json:"max_concurrency,omitempty"`
}
