package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"GoFaas/internal/observability/logging"
	"GoFaas/internal/worker/runtime/docker"
	"GoFaas/pkg/types"
)

// ContainerRuntime implements Runtime using Docker containers
type ContainerRuntime struct {
	dockerClient *docker.Client
	imageManager *docker.ImageManager
	workDir      string
	logger       logging.Logger
}

// NewContainerRuntime creates a new container-based runtime
func NewContainerRuntime(workDir string, logger logging.Logger) (*ContainerRuntime, error) {
	// Create Docker client
	dockerClient, err := docker.NewClient(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Create image manager
	imageManager := docker.NewImageManager(dockerClient, logger)

	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	return &ContainerRuntime{
		dockerClient: dockerClient,
		imageManager: imageManager,
		workDir:      workDir,
		logger:       logger,
	}, nil
}

// Execute runs a function inside a Docker container
func (r *ContainerRuntime) Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error) {
	startTime := time.Now()

	// Get runtime image
	imageName := r.imageManager.GetRuntimeImage(spec.Runtime)
	if imageName == "" {
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

	// Ensure image exists
	if err := r.imageManager.EnsureImage(ctx, imageName); err != nil {
		return &ExecutionResult{
			Status: types.StatusFailed,
			Error: &types.ExecutionError{
				Type:    "ImageError",
				Message: fmt.Sprintf("failed to ensure runtime image: %v", err),
			},
			Metrics: types.ExecutionMetrics{
				Duration: time.Since(startTime),
			},
		}, nil
	}

	// Create temporary directory for function code
	execDir := filepath.Join(r.workDir, spec.FunctionID, fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}
	defer os.RemoveAll(execDir) // Cleanup after execution

	// Write function code to file
	_, err := r.writeCodeToFile(execDir, spec.Runtime, spec.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to write function code: %w", err)
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()

	// Prepare container configuration
	containerCfg := docker.ContainerConfig{
		Image:       imageName,
		Handler:     spec.Handler,
		Payload:     spec.Payload,
		Environment: spec.Environment,
		MemoryLimit: spec.Limits.MemoryBytes,
		CPULimit:    spec.Limits.CPUShares * 1000000, // Convert to nanocpus
		CodePath:    execDir,
	}

	// Create container
	containerID, err := r.dockerClient.CreateContainer(execCtx, containerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Ensure container cleanup
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		r.dockerClient.RemoveContainer(cleanupCtx, containerID)
	}()

	// Start container
	if err := r.dockerClient.StartContainer(execCtx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	r.logger.Debug("Container started",
		logging.F("container_id", containerID),
		logging.F("function_id", spec.FunctionID),
		logging.F("runtime", spec.Runtime),
	)

	// Wait for container to finish
	exitCode, err := r.dockerClient.WaitContainer(execCtx, containerID)
	endTime := time.Now()

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		r.logger.Warn("Container execution timed out",
			logging.F("container_id", containerID),
			logging.F("function_id", spec.FunctionID),
		)

		return &ExecutionResult{
			Status: types.StatusTimeout,
			Error: &types.ExecutionError{
				Type:    "TimeoutError",
				Message: "Function execution timed out",
			},
			Metrics: types.ExecutionMetrics{
				Duration: endTime.Sub(startTime),
			},
		}, nil
	}

	// Get container logs
	logs, err := r.dockerClient.GetContainerLogs(context.Background(), containerID)
	if err != nil {
		r.logger.Error("Failed to get container logs",
			logging.F("container_id", containerID),
			logging.F("error", err),
		)
		logs = []byte{}
	}

	// Get container stats (best effort)
	stats, _ := r.dockerClient.GetContainerStats(context.Background(), containerID)

	// Build execution result
	result := &ExecutionResult{
		Metrics: types.ExecutionMetrics{
			Duration:   endTime.Sub(startTime),
			MemoryPeak: stats.MemoryUsage,
			NetworkIn:  stats.NetworkIn,
			NetworkOut: stats.NetworkOut,
		},
	}

	// Check exit code
	if exitCode == 0 {
		result.Status = types.StatusCompleted
		result.Result = logs
		r.logger.Info("Container execution completed successfully",
			logging.F("container_id", containerID),
			logging.F("function_id", spec.FunctionID),
			logging.F("duration", result.Metrics.Duration),
		)
	} else {
		result.Status = types.StatusFailed
		result.Error = &types.ExecutionError{
			Type:    "RuntimeError",
			Message: fmt.Sprintf("Function exited with code %d", exitCode),
			Stack:   string(logs),
		}
		r.logger.Warn("Container execution failed",
			logging.F("container_id", containerID),
			logging.F("function_id", spec.FunctionID),
			logging.F("exit_code", exitCode),
		)
	}

	return result, nil
}

// GetCapabilities returns runtime capabilities
func (r *ContainerRuntime) GetCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		Language:   "multi",
		Version:    "1.0.0-container",
		MaxTimeout: 5 * time.Minute,
		MaxMemory:  2 * 1024 * 1024 * 1024, // 2 GB
	}
}

// Close closes the container runtime and releases resources
func (r *ContainerRuntime) Close() error {
	return r.dockerClient.Close()
}

// writeCodeToFile writes function code to a file based on runtime
func (r *ContainerRuntime) writeCodeToFile(dir string, runtime types.RuntimeType, code []byte) (string, error) {
	var filename string

	switch runtime {
	case types.RuntimeGo:
		filename = "main.go"
	case types.RuntimePython:
		filename = "main.py"
	case types.RuntimeNodeJS:
		filename = "main.js"
	default:
		return "", fmt.Errorf("unsupported runtime: %s", runtime)
	}

	codePath := filepath.Join(dir, filename)
	if err := os.WriteFile(codePath, code, 0644); err != nil {
		return "", fmt.Errorf("failed to write code file: %w", err)
	}

	return codePath, nil
}
