package invocation

import (
	"encoding/json"
	"time"

	"GoFaas/pkg/types"
)

// InvocationRequest represents a function invocation request
type InvocationRequest struct {
	FunctionID string                 `json:"function_id"`
	Payload    json.RawMessage        `json:"payload"`
	Headers    map[string]string      `json:"headers"`
	Timeout    *time.Duration         `json:"timeout,omitempty"`
}

// InvocationHandle represents an async invocation handle
type InvocationHandle struct {
	InvocationID string              `json:"invocation_id"`
	FunctionID   string              `json:"function_id"`
	Status       types.ExecutionStatus `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
}

// ExecutionRequest represents a function execution request (queued message)
type ExecutionRequest struct {
	InvocationID string            `json:"invocation_id"`
	FunctionID   string            `json:"function_id"`
	Payload      json.RawMessage   `json:"payload"`
	Headers      map[string]string `json:"headers"`
	Timeout      *time.Duration    `json:"timeout"`
}

// ExecutionResult represents a function execution result
type ExecutionResult struct {
	Status  types.ExecutionStatus  `json:"status"`
	Result  json.RawMessage        `json:"result,omitempty"`
	Error   *types.ExecutionError  `json:"error,omitempty"`
	Metrics *types.ExecutionMetrics `json:"metrics,omitempty"`
}
