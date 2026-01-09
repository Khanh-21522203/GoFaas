package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"GoFaas/pkg/types"
)

// SimpleRuntime implements Runtime using direct process execution
// This is a simplified implementation without container isolation
type SimpleRuntime struct {
	workDir string
}

// NewSimpleRuntime creates a new simple runtime
func NewSimpleRuntime(workDir string) (*SimpleRuntime, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	return &SimpleRuntime{
		workDir: workDir,
	}, nil
}

// Execute runs a function using direct process execution
func (r *SimpleRuntime) Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error) {
	startTime := time.Now()

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	// Create temporary directory for this execution
	execDir := filepath.Join(r.workDir, spec.FunctionID)
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}
	defer os.RemoveAll(execDir) // Cleanup after execution

	// Write function code to file
	var codePath string
	var cmd *exec.Cmd

	switch spec.Runtime {
	case types.RuntimeGo:
		codePath = filepath.Join(execDir, "main.go")
		if err := os.WriteFile(codePath, spec.Code, 0644); err != nil {
			return nil, fmt.Errorf("failed to write function code: %w", err)
		}
		cmd = exec.CommandContext(execCtx, "go", "run", codePath)

	case types.RuntimePython:
		codePath = filepath.Join(execDir, "main.py")
		if err := os.WriteFile(codePath, spec.Code, 0644); err != nil {
			return nil, fmt.Errorf("failed to write function code: %w", err)
		}
		cmd = exec.CommandContext(execCtx, "python3", codePath)

	case types.RuntimeNodeJS:
		codePath = filepath.Join(execDir, "main.js")
		if err := os.WriteFile(codePath, spec.Code, 0644); err != nil {
			return nil, fmt.Errorf("failed to write function code: %w", err)
		}
		cmd = exec.CommandContext(execCtx, "node", codePath)

	default:
		return &ExecutionResult{
			Status: types.StatusFailed,
			Error: &types.ExecutionError{
				Type:    "UnsupportedRuntime",
				Message: fmt.Sprintf("unsupported runtime: %s", spec.Runtime),
			},
			Metrics: types.ExecutionMetrics{
				Duration: time.Since(startTime),
			},
		}, nil
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range spec.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Pass payload as environment variable (simplified approach)
	cmd.Env = append(cmd.Env, fmt.Sprintf("FUNCTION_PAYLOAD=%s", string(spec.Payload)))
	cmd.Env = append(cmd.Env, fmt.Sprintf("FUNCTION_HANDLER=%s", spec.Handler))

	cmd.Dir = execDir

	// Execute function
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result := &ExecutionResult{
		Metrics: types.ExecutionMetrics{
			Duration: endTime.Sub(startTime),
		},
	}

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		result.Status = types.StatusTimeout
		result.Error = &types.ExecutionError{
			Type:    "TimeoutError",
			Message: "Function execution timed out",
		}
		return result, nil
	}

	// Check for execution error
	if err != nil {
		result.Status = types.StatusFailed
		result.Error = &types.ExecutionError{
			Type:    "RuntimeError",
			Message: fmt.Sprintf("Function execution failed: %v", err),
			Stack:   string(output),
		}
		return result, nil
	}

	// Success
	result.Status = types.StatusCompleted
	result.Result = output

	return result, nil
}

// GetCapabilities returns runtime capabilities
func (r *SimpleRuntime) GetCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		Language:   "multi",
		Version:    "1.0.0",
		MaxTimeout: 5 * time.Minute,
		MaxMemory:  512 * 1024 * 1024, // 512 MB
	}
}
