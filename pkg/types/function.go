package types

import (
	"encoding/json"
	"time"
)

// Function represents a serverless function
type Function struct {
	ID        string            `json:"id" db:"id"`
	Name      string            `json:"name" db:"name"`
	Version   string            `json:"version" db:"version"`
	Runtime   RuntimeType       `json:"runtime" db:"runtime"`
	Handler   string            `json:"handler" db:"handler"`
	Code      FunctionCode      `json:"code"`
	Config    FunctionConfig    `json:"config"`
	Metadata  map[string]string `json:"metadata" db:"metadata"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

// FunctionCode represents function source code
type FunctionCode struct {
	Source     string `json:"source" db:"code_source"`           // Base64 encoded or storage URL
	SourceType string `json:"source_type" db:"code_source_type"` // "inline", "s3", "git"
	Checksum   string `json:"checksum" db:"code_checksum"`       // SHA256 hash
	Size       int64  `json:"size" db:"code_size"`               // Size in bytes
}

// FunctionConfig represents function configuration
type FunctionConfig struct {
	Timeout     time.Duration     `json:"timeout" db:"timeout_seconds"`
	Memory      int               `json:"memory_mb" db:"memory_mb"`
	Environment map[string]string `json:"environment" db:"environment"`
	Concurrency int               `json:"max_concurrency" db:"max_concurrency"`
}

// Invocation represents a function invocation request
type Invocation struct {
	ID          string           `json:"id" db:"id"`
	FunctionID  string           `json:"function_id" db:"function_id"`
	Payload     json.RawMessage  `json:"payload" db:"payload"`
	Headers     map[string]string `json:"headers" db:"headers"`
	Status      ExecutionStatus  `json:"status" db:"status"`
	Result      json.RawMessage  `json:"result,omitempty" db:"result"`
	Error       *ExecutionError  `json:"error,omitempty"`
	Metrics     *ExecutionMetrics `json:"metrics,omitempty"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	StartedAt   *time.Time       `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
}
