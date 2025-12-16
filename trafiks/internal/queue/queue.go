package queue

import (
	"context"
)

// Queue interface for message queuing
type Queue interface {
	// Push adds a message to the queue
	Push(ctx context.Context, stream string, data []byte) error

	// Pop removes and returns a message from the queue (blocking)
	Pop(ctx context.Context, stream string) ([]byte, error)

	// PopWithTimeout removes and returns a message with timeout
	PopWithTimeout(ctx context.Context, stream string, timeoutSeconds int) ([]byte, error)

	// GetQueueLength returns the number of pending messages
	GetQueueLength(ctx context.Context, stream string) (int64, error)
}
