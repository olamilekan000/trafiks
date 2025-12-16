package processor

import (
	"context"
	"sync"

	"github.com/trafiks/trafiks/internal/agent/webhook"
	"github.com/trafiks/trafiks/pkg"
)

type ProcessorClient interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Processor struct {
	logger           pkg.LoggerClient
	webhookProcessor webhook.WebhookProcessorClient
	// Future: logProcessor *log.Processor
	// Future: metricsProcessor *metrics.Processor

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewProcessor(
	logger pkg.LoggerClient,
	webhookProcessor webhook.WebhookProcessorClient,
) ProcessorClient {
	return &Processor{
		logger:           logger,
		webhookProcessor: webhookProcessor,
	}
}

func (p *Processor) Start(ctx context.Context) error {
	// Create a cancellable context from the parent context
	// This allows us to cancel all processors when Stop() is called
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.logger.Info("Starting all processors...")

	// Start webhook processor
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.webhookProcessor.Start(ctx); err != nil {
			p.logger.Errorf("Webhook processor error: %v", err)
		}
	}()

	// Future: Start log processor
	// p.wg.Add(1)
	// go func() {
	//     defer p.wg.Done()
	//     p.logProcessor.Start(ctx)
	// }()

	// DON'T wait here! Start() should return immediately after starting goroutines
	// The goroutines will run in the background
	// We'll wait for them in Stop() for graceful shutdown
	return nil
}

func (p *Processor) Stop(ctx context.Context) error {
	p.logger.Info("Stopping all processors...")
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	return nil
}
