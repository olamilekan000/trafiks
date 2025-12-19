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

			conf := cfg.GetConf()
			if err := redisClient.Connect(startCtx, cache.RedisConf{
				Host:     conf.Redis.Host,
				Username: conf.Redis.Username,
				Password: conf.Redis.Password,
				Db:       conf.Redis.Db,
			}); err != nil {
				logger.Panicf("error connecting to redis: %v", err)
			}

			go func() {
				processorCtx := context.Background()
				if err := processor.Start(processorCtx); err != nil {
					logger.Errorf("Processor error: %v", err)
				}
			}()

			return nil
		},

		OnStop: func(stopCtx context.Context) error {
			logger.Info("Stopping Agent Service")
			return processor.Stop(stopCtx)
		},
	})
}
