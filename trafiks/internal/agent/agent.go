package agent

import (
	"context"

	"go.uber.org/fx"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/database"
	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/internal"
	"github.com/trafiks/trafiks/internal/agent/processor"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/cache"
)

var Module = fx.Options(
	cfg.Module,
	pkg.Module,
	database.Module,
	repository.Module,
	services.Module,
	internal.Module,
	// fx.Provide(NewQueue),
	// fx.Provide(webhook.NewProcessor),
	fx.Invoke(Agent),
)

func Agent(
	lifecycle fx.Lifecycle,
	processor processor.ProcessorClient,
	logger pkg.LoggerClient,
	db postgres.PostgresDB,
	redisClient cache.RedisClient,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("Starting Agent Service")

			// Connect to Redis (use startup context for connection)
			conf := cfg.GetConf()
			if err := redisClient.Connect(startCtx, cache.RedisConf{
				Host:     conf.Redis.Host,
				Username: conf.Redis.Username,
				Password: conf.Redis.Password,
				Db:       conf.Redis.Db,
			}); err != nil {
				logger.Panicf("error connecting to redis: %v", err)
			}

			// Start main processor (which starts all sub-processors)
			// Use context.Background() for long-running operations, not the startup context
			// The startup context is only valid during OnStart execution
			go func() {
				// Create a background context for the long-running processor
				// This won't be cancelled when OnStart returns
				processorCtx := context.Background()
				if err := processor.Start(processorCtx); err != nil {
					logger.Errorf("Processor error: %v", err)
				}
			}()

			// Returning nil means startup succeeded
			// The goroutine will continue running independently
			return nil
		},

		OnStop: func(stopCtx context.Context) error {
			logger.Info("Stopping Agent Service")
			// Use the stop context for graceful shutdown
			return processor.Stop(stopCtx)
		},
	})
}
