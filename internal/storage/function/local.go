package function

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalStorage implements Storage using local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

// Store saves function code to local filesystem
func (s *LocalStorage) Store(ctx context.Context, functionID string, code []byte) (string, error) {
	// Create function-specific directory
	functionDir := filepath.Join(s.basePath, functionID)
	if err := os.MkdirAll(functionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create function directory: %w", err)
	}

	// Write code to file
	codePath := filepath.Join(functionDir, "code")
	if err := os.WriteFile(codePath, code, 0644); err != nil {
		return "", fmt.Errorf("failed to write function code: %w", err)
	}

	// Return relative path as location
	return filepath.Join(functionID, "code"), nil
}

// Retrieve reads function code from local filesystem
func (s *LocalStorage) Retrieve(ctx context.Context, location string) ([]byte, error) {
	codePath := filepath.Join(s.basePath, location)

	code, err := os.ReadFile(codePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("function code not found: %s", location)
		}
		return nil, fmt.Errorf("failed to read function code: %w", err)
	}

	return code, nil
}

// Delete removes function code from local filesystem
func (s *LocalStorage) Delete(ctx context.Context, location string) error {
	codePath := filepath.Join(s.basePath, location)

	// Remove the entire function directory
	functionDir := filepath.Dir(codePath)
	if err := os.RemoveAll(functionDir); err != nil {
		return fmt.Errorf("failed to delete function code: %w", err)
	}

	return nil
}
