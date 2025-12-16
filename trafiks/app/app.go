package app

import (
	"context"
	"runtime/debug"

	"go.uber.org/fx"

	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/database"
	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/internal"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/cache"
	"github.com/trafiks/trafiks/routes"
)

var Module = fx.Options(
	cfg.Module,
	pkg.Module,
	routes.Module,
	database.Module,
	controller.Module,
	services.Module,
	repository.Module,
	middleware.Module,
	internal.Module,
	fx.Invoke(App),
)

func App(
	lifecycle fx.Lifecycle,
	routes routes.Routes,
	router routes.Router,
	db postgres.PostgresDB,
	logger pkg.LoggerClient,
	redisClient cache.RedisClient,
	serviceRepo repository.ServiceRepoClient,
	tlsManager *services.TLSManager,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(c context.Context) error {
			logger.Info("Starting Application")
			routes.Setup()

			conf := cfg.GetConf()

			// Redis connection commented out for now
			if err := redisClient.Connect(context.Background(), cache.RedisConf{
				Host:     conf.Redis.Host,
				Username: conf.Redis.Username,
				Password: conf.Redis.Password,
				Db:       conf.Redis.Db,
			}); err != nil {
				logger.Panicf("error connecting to redis: %v", err)
			}

			// Start HTTP server
			serverPort := conf.ServerPort
			logger.Infof("HTTP server listening on port: %q", serverPort)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						stack := string(debug.Stack())
						logger.Errorf("Panic in HTTP server goroutine: %q\n%s", r, stack)
					}
				}()

				err := router.Run(":" + serverPort)
				if err != nil {
					logger.Errorf("Error running HTTP server: %v", err)
				}
			}()

			// Load certificates into TLS manager
			if err := tlsManager.LoadAllFromDatabase(c, serviceRepo); err != nil {
				logger.Warnf("Failed to load TLS certificates: %v", err)
			} else {
				loadedCount := tlsManager.GetDomainCount()
				if loadedCount > 0 {
					logger.Infof("Loaded %d TLS certificate(s) from database", loadedCount)
				}
			}

			// Start HTTPS server (always start if TLSPort is configured, even if no certificates yet)
			// Certificates can be added dynamically via SNI
			tlsPort := conf.TLSPort
			if tlsPort != "" {
				logger.Infof("HTTPS server listening on port: %q", tlsPort)
				go func() {
					defer func() {
						if r := recover(); r != nil {
							stack := string(debug.Stack())
							logger.Errorf("Panic in HTTPS server goroutine: %q\n%s", r, stack)
						}
					}()

					tlsConfig := tlsManager.GetTLSConfig()
					err := router.RunTLS(":"+tlsPort, tlsConfig)
					if err != nil {
						logger.Errorf("Error running HTTPS server: %v", err)
					}
				}()
			}

			return nil
		},

		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping Application")
			return nil
		},
	})
}
