package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"GoFaas/internal/observability/logging"
)

// Client wraps Docker client with FaaS-specific operations
type Client struct {
	cli    *client.Client
	logger logging.Logger
}

// NewClient creates a new Docker client wrapper
func NewClient(logger logging.Logger) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Client{
		cli:    cli,
		logger: logger,
	}, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.cli.Close()
}

// CreateContainer creates a new container with the specified configuration
func (c *Client) CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error) {
	// Prepare environment variables
	env := make([]string, 0, len(cfg.Environment)+2)
	env = append(env, fmt.Sprintf("FUNCTION_HANDLER=%s", cfg.Handler))
	env = append(env, fmt.Sprintf("FUNCTION_PAYLOAD=%s", string(cfg.Payload)))
	for key, value := range cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Env:          env,
		WorkingDir:   "/app",
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// Host configuration with resource limits
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   cfg.MemoryLimit,
			NanoCPUs: cfg.CPULimit,
		},
		AutoRemove: false, // We'll remove manually after getting logs
		NetworkMode: "bridge",
	}

	// Add volume mount for function code
	if cfg.CodePath != "" {
		hostConfig.Mounts = []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   cfg.CodePath,
				Target:   "/app/function",
				ReadOnly: true,
			},
		}
	}

	// Create container
	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	c.logger.Debug("Container created",
		logging.F("container_id", resp.ID),
		logging.F("image", cfg.Image),
	)

	return resp.ID, nil
}

// StartContainer starts a container
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	c.logger.Debug("Container started", logging.F("container_id", containerID))
	return nil
}

// WaitContainer waits for container to finish and returns exit code
func (c *Client) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return -1, fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		return status.StatusCode, nil
	case <-ctx.Done():
		// Timeout - kill container
		c.KillContainer(context.Background(), containerID)
		return -1, ctx.Err()
	}

	return -1, fmt.Errorf("unexpected wait completion")
}

// GetContainerLogs retrieves container logs
func (c *Client) GetContainerLogs(ctx context.Context, containerID string) ([]byte, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     false,
	}

	reader, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Read all logs
	logs, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read container logs: %w", err)
	}

	// Docker logs include 8-byte header per line, strip it
	return stripDockerLogHeaders(logs), nil
}

// KillContainer forcefully stops a container
func (c *Client) KillContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerKill(ctx, containerID, "SIGKILL"); err != nil {
		return fmt.Errorf("failed to kill container: %w", err)
	}

	c.logger.Debug("Container killed", logging.F("container_id", containerID))
	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	c.logger.Debug("Container removed", logging.F("container_id", containerID))
	return nil
}

// GetContainerStats retrieves container resource usage statistics
func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Parse stats (simplified - in production, parse full JSON)
	// For now, return empty stats
	return &ContainerStats{}, nil
}

// stripDockerLogHeaders removes Docker's 8-byte header from log output
func stripDockerLogHeaders(logs []byte) []byte {
	if len(logs) == 0 {
		return logs
	}

	// Docker multiplexed stream format: [8 bytes header][payload]
	// Header: [stream_type, 0, 0, 0, size1, size2, size3, size4]
	// For simplicity, we'll strip headers by looking for the pattern
	
	result := make([]byte, 0, len(logs))
	i := 0
	
	for i < len(logs) {
		// Check if we have at least 8 bytes for header
		if i+8 > len(logs) {
			// Remaining bytes without header
			result = append(result, logs[i:]...)
			break
		}

		// Read payload size from header (big-endian uint32 at offset 4)
		size := int(logs[i+4])<<24 | int(logs[i+5])<<16 | int(logs[i+6])<<8 | int(logs[i+7])
		
		// Skip header (8 bytes)
		i += 8
		
		// Append payload
		if i+size <= len(logs) {
			result = append(result, logs[i:i+size]...)
			i += size
		} else {
			// Malformed, append rest
			result = append(result, logs[i:]...)
			break
		}
	}

	return result
}

// ContainerConfig holds container creation configuration
type ContainerConfig struct {
	Image       string
	Handler     string
	Payload     []byte
	Environment map[string]string
	MemoryLimit int64
	CPULimit    int64
	CodePath    string // Path to function code on host
}

// ContainerStats holds container resource usage statistics
type ContainerStats struct {
	CPUUsage    int64
	MemoryUsage int64
	NetworkIn   int64
	NetworkOut  int64
}
