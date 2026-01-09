package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// RedisQueue implements Queue using Redis
type RedisQueue struct {
	client *redis.Client
	prefix string
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(client *redis.Client, prefix string) *RedisQueue {
	return &RedisQueue{
		client: client,
		prefix: prefix,
	}
}

// Enqueue adds a message to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, queue string, payload []byte, headers map[string]string) error {
	message := Message{
		ID:         uuid.New().String(),
		Queue:      queue,
		Payload:    payload,
		Headers:    headers,
		Attempts:   0,
		EnqueuedAt: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	queueKey := q.queueKey(queue)
	return q.client.LPush(ctx, queueKey, data).Err()
}

// Dequeue removes and returns a message from the queue
func (q *RedisQueue) Dequeue(ctx context.Context, queue string, timeout time.Duration) (*Message, error) {
	queueKey := q.queueKey(queue)
	processingKey := q.processingKey(queue)

	// Use BRPOPLPUSH for reliable message processing
	result, err := q.client.BRPopLPush(ctx, queueKey, processingKey, timeout).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No message available
		}
		return nil, fmt.Errorf("failed to dequeue message: %w", err)
	}

	var message Message
	if err := json.Unmarshal([]byte(result), &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	message.Attempts++
	return &message, nil
}

// Ack acknowledges successful message processing
func (q *RedisQueue) Ack(ctx context.Context, message *Message) error {
	processingKey := q.processingKey(message.Queue)
	data, _ := json.Marshal(message)

	return q.client.LRem(ctx, processingKey, 1, string(data)).Err()
}

// Nack rejects a message and requeues it
func (q *RedisQueue) Nack(ctx context.Context, message *Message) error {
	processingKey := q.processingKey(message.Queue)
	queueKey := q.queueKey(message.Queue)

	data, _ := json.Marshal(message)

	// Remove from processing queue and add back to main queue
	pipe := q.client.Pipeline()
	pipe.LRem(ctx, processingKey, 1, string(data))
	pipe.LPush(ctx, queueKey, data)

	_, err := pipe.Exec(ctx)
	return err
}

// DeadLetter moves a message to the dead letter queue
func (q *RedisQueue) DeadLetter(ctx context.Context, message *Message, reason string) error {
	processingKey := q.processingKey(message.Queue)
	deadLetterKey := q.deadLetterKey(message.Queue)

	// Add reason to message headers
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}
	message.Headers["dead_letter_reason"] = reason
	message.Headers["dead_lettered_at"] = time.Now().Format(time.RFC3339)

	data, _ := json.Marshal(message)

	pipe := q.client.Pipeline()
	pipe.LRem(ctx, processingKey, 1, string(data))
	pipe.LPush(ctx, deadLetterKey, data)

	_, err := pipe.Exec(ctx)
	return err
}

// GetStats returns queue statistics
func (q *RedisQueue) GetStats(ctx context.Context, queue string) (*QueueStats, error) {
	queueKey := q.queueKey(queue)

	size, err := q.client.LLen(ctx, queueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue size: %w", err)
	}

	return &QueueStats{
		Name:        queue,
		Size:        size,
		Consumers:   0, // Would need additional tracking
		EnqueueRate: 0, // Would need time-series data
		DequeueRate: 0, // Would need time-series data
	}, nil
}

func (q *RedisQueue) queueKey(queue string) string {
	return fmt.Sprintf("%s:queue:%s", q.prefix, queue)
}

func (q *RedisQueue) processingKey(queue string) string {
	return fmt.Sprintf("%s:processing:%s", q.prefix, queue)
}

func (q *RedisQueue) deadLetterKey(queue string) string {
	return fmt.Sprintf("%s:dead_letter:%s", q.prefix, queue)
}
