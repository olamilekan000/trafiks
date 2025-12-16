package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/trafiks/trafiks/pkg/cache"
)

type RedisQueue struct {
	client     cache.RedisClient
	consumerID string // Unique consumer ID for this queue instance
}

// NewRedisQueueFromClient creates a queue from a redis client
func NewRedisQueueFromClient(redisClient cache.RedisClient) Queue {
	return &RedisQueue{
		client:     redisClient,
		consumerID: fmt.Sprintf("consumer-%d", time.Now().UnixNano()),
	}
}

func (r *RedisQueue) Push(ctx context.Context, stream string, data []byte) error {
	if r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// Use Redis Streams for ordered processing
	args := &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}

	_, err := r.client.GetClient().XAdd(ctx, args).Result()
	return err
}

func (r *RedisQueue) Pop(ctx context.Context, stream string) ([]byte, error) {
	return r.PopWithTimeout(ctx, stream, 0) // 0 = blocking
}

func (r *RedisQueue) PopWithTimeout(ctx context.Context, stream string, timeoutSeconds int) ([]byte, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	client := r.client.GetClient()
	groupName := "webhook-workers"
	consumerName := r.consumerID // Use stable consumer ID for this queue instance

	// Create consumer group if it doesn't exist (ignore error if it already exists)
	client.XGroupCreateMkStream(ctx, stream, groupName, "0").Result()

	// Use XREADGROUP to read from the consumer group
	// This ensures each message is only delivered to one consumer
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: consumerName,
		Streams:  []string{stream, ">"}, // ">" means read new messages
		Count:    1,
		Block:    time.Duration(timeoutSeconds) * time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // No messages
		}
		return nil, err
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	message := streams[0].Messages[0]
	data, ok := message.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}

	// Acknowledge the message (mark as processed)
	// This removes it from the pending list and prevents re-delivery
	client.XAck(ctx, stream, groupName, message.ID).Result()

	return []byte(data), nil
}

func (r *RedisQueue) GetQueueLength(ctx context.Context, stream string) (int64, error) {
	if r.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	return r.client.GetClient().XLen(ctx, stream).Result()
}

// PushWebhookDelivery pushes a webhook delivery to the queue
func PushWebhookDelivery(q Queue, ctx context.Context, deliveryID uint, webhookID uint, projectID uint, eventType string, payload interface{}) error {
	data := map[string]interface{}{
		"delivery_id": deliveryID,
		"webhook_id":  webhookID,
		"project_id":  projectID,
		"event_type":  eventType,
		"payload":     payload,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery data: %w", err)
	}

	return q.Push(ctx, "webhook:deliveries", jsonData)
}
