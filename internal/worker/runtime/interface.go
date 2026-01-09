package runtime

import (
	"context"
	"time"

	"GoFaas/pkg/types"
)

// Runtime defines function execution interface
type Runtime interface {
	Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error)
	GetCapabilities() RuntimeCapabilities
}

// ExecutionSpec defines function execution parameters
type ExecutionSpec struct {
	FunctionID  string                 `json:"function_id"`
	Code        []byte                 `json:"code"`
	Runtime     types.RuntimeType      `json:"runtime"`
	Handler     string                 `json:"handler"`
	Payload     []byte                 `json:"payload"`
	Environment map[string]string      `json:"environment"`
	Timeout     time.Duration          `json:"timeout"`
	Limits      ResourceLimits         `json:"limits"`
}

// ExecutionResult represents function execution result
type ExecutionResult struct {
	Status  types.ExecutionStatus   `json:"status"`
	Result  []byte                  `json:"result,omitempty"`
	Error   *types.ExecutionError   `json:"error,omitempty"`
	Metrics types.ExecutionMetrics  `json:"metrics"`
	Logs    []LogEntry              `json:"logs,omitempty"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MemoryBytes int64         `json:"memory_bytes"`
	CPUShares   int64         `json:"cpu_shares"`
	Timeout     time.Duration `json:"timeout"`
}

// RuntimeCapabilities describes runtime capabilities
type RuntimeCapabilities struct {
	Language   string        `json:"language"`
	Version    string        `json:"version"`
	MaxTimeout time.Duration `json:"max_timeout"`
	MaxMemory  int64         `json:"max_memory"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}
