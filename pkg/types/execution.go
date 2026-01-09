package types

import "time"

// ExecutionStatus represents the status of function execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusTimeout   ExecutionStatus = "timeout"
)

// IsTerminal returns true if the status represents a terminal state
func (s ExecutionStatus) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusTimeout:
		return true
	default:
		return false
	}
}

// ExecutionMetrics represents execution metrics
type ExecutionMetrics struct {
	Duration   time.Duration `json:"duration" db:"duration"`
	CPUTime    time.Duration `json:"cpu_time" db:"cpu_time"`
	MemoryPeak int64         `json:"memory_peak" db:"memory_peak"`
	NetworkIn  int64         `json:"network_in" db:"network_in"`
	NetworkOut int64         `json:"network_out" db:"network_out"`
}

// ExecutionError represents an execution error
type ExecutionError struct {
	Type    string `json:"type" db:"error_type"`
	Message string `json:"message" db:"error_message"`
	Stack   string `json:"stack,omitempty" db:"error_stack"`
}
