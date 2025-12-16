package internal

import (
	"go.uber.org/fx"

	"github.com/trafiks/trafiks/internal/agent/processor"
	"github.com/trafiks/trafiks/internal/agent/webhook"
	"github.com/trafiks/trafiks/internal/queue"
)

var Module = fx.Options(
	fx.Provide(queue.NewRedisQueueFromClient),
	fx.Provide(webhook.NewWebhookProcessor),
	fx.Provide(processor.NewProcessor),
)
