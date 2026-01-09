package messaging

import (
	"context"
	"time"
)

// Message represents a queue message
type Message struct {
	ID         string            `json:"id"`
	Queue      string            `json:"queue"`
	Payload    []byte            `json:"payload"`
	Headers    map[string]string `json:"headers"`
	Attempts   int               `json:"attempts"`
	EnqueuedAt time.Time         `json:"enqueued_at"`
}

// Queue defines message queue operations
type Queue interface {
	Enqueue(ctx context.Context, queue string, payload []byte, headers map[string]string) error
	Dequeue(ctx context.Context, queue string, timeout time.Duration) (*Message, error)
	Ack(ctx context.Context, message *Message) error
	Nack(ctx context.Context, message *Message) error
	DeadLetter(ctx context.Context, message *Message, reason string) error
	GetStats(ctx context.Context, queue string) (*QueueStats, error)
}

// QueueStats represents queue statistics
type QueueStats struct {
	Name        string  `json:"name"`
	Size        int64   `json:"size"`
	Consumers   int     `json:"consumers"`
	EnqueueRate float64 `json:"enqueue_rate"`
	DequeueRate float64 `json:"dequeue_rate"`
}
