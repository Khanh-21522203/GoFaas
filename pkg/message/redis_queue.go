package messaging

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisQueue struct {
	client *redis.Client
	queue  string
}

func NewRedisQueue(queue string) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default Redis address
	})
	return &RedisQueue{
		client: client,
		queue:  queue,
	}
}

func (rq *RedisQueue) Subscribe(handler func(string)) {
	ctx := context.Background()
	for {
		result, err := rq.client.BLPop(ctx, 0, rq.queue).Result()
		if err != nil {
			log.Printf("Error subscribing to queue: %v\n", err)
			continue
		}
		if len(result) > 1 {
			handler(result[1])
		}
	}
}
