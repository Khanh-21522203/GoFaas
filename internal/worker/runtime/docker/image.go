package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"

	"GoFaas/internal/observability/logging"
	pkgTypes "GoFaas/pkg/types"
)

// ImageManager manages Docker images for function runtimes
type ImageManager struct {
	client *Client
	logger logging.Logger
}

// NewImageManager creates a new image manager
func NewImageManager(client *Client, logger logging.Logger) *ImageManager {
	return &ImageManager{
		client: client,
		logger: logger,
	}
}

// GetRuntimeImage returns the Docker image name for a runtime
func (m *ImageManager) GetRuntimeImage(runtime pkgTypes.RuntimeType) string {
	switch runtime {
	case pkgTypes.RuntimeGo:
		return "faas-runtime-go:latest"
	case pkgTypes.RuntimePython:
		return "faas-runtime-python:latest"
	case pkgTypes.RuntimeNodeJS:
		return "faas-runtime-nodejs:latest"
	default:
		return ""
	}
}

// EnsureImage ensures the runtime image exists, pulling if necessary
func (m *ImageManager) EnsureImage(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, err := m.client.cli.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image exists
		m.logger.Debug("Runtime image exists", logging.F("image", imageName))
		return nil
	}

	// Image doesn't exist, try to pull
	m.logger.Info("Pulling runtime image", logging.F("image", imageName))

	reader, err := m.client.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to complete image pull: %w", err)
	}

	m.logger.Info("Runtime image pulled successfully", logging.F("image", imageName))
	return nil
}

// BuildRuntimeImages builds all runtime base images
func (m *ImageManager) BuildRuntimeImages(ctx context.Context) error {
	runtimes := []pkgTypes.RuntimeType{
		pkgTypes.RuntimeGo,
		pkgTypes.RuntimePython,
		pkgTypes.RuntimeNodeJS,
	}

	for _, runtime := range runtimes {
		image := m.GetRuntimeImage(runtime)
		if err := m.EnsureImage(ctx, image); err != nil {
			m.logger.Warn("Failed to ensure runtime image",
				logging.F("runtime", runtime),
				logging.F("image", image),
				logging.F("error", err),
			)
			// Continue with other runtimes
		}
	}

	return nil
}
