package function

import "context"

// Storage defines function code storage operations
type Storage interface {
	Store(ctx context.Context, functionID string, code []byte) (string, error)
	Retrieve(ctx context.Context, location string) ([]byte, error)
	Delete(ctx context.Context, location string) error
}
