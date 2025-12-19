package app

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"

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
	var servers []*http.Server

	lifecycle.Append(fx.Hook{
		OnStart: func(c context.Context) error {
			logger.Info("Starting Application")
			routes.Setup()

			conf := cfg.GetConf()

			// Connect to Redis
			if err := redisClient.Connect(context.Background(), cache.RedisConf{
				Host:     conf.Redis.Host,
				Username: conf.Redis.Username,
				Password: conf.Redis.Password,
				Db:       conf.Redis.Db,
			}); err != nil {
				logger.Panicf("error connecting to redis: %v", err)
			}

			// Start HTTP server
			httpServer := startHTTPServer(conf.ServerPort, router, logger)
			servers = append(servers, httpServer)

			// Load TLS certificates
			if err := tlsManager.LoadAllFromDatabase(c, serviceRepo); err != nil {
				logger.Warnf("Failed to load TLS certificates: %v", err)
			} else if count := tlsManager.GetDomainCount(); count > 0 {
				logger.Infof("Loaded %d TLS certificate(s) from database", count)
			}

			// Start HTTPS server if configured
			if conf.TLSPort != "" {
				httpsServer := startHTTPSServer(conf.TLSPort, router, tlsManager, logger)
				servers = append(servers, httpsServer)
			}

			return nil
		},

		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping Application")
			conf := cfg.GetConf()

			shutdownCtx, cancel := context.WithTimeout(ctx, conf.Proxy.Server.ShutdownTimeout)
			defer cancel()

			return shutdownServers(shutdownCtx, servers, logger)
		},
	})
}

func startHTTPServer(port string, router routes.Router, logger pkg.LoggerClient) *http.Server {
	logger.Infof("HTTP server listening on port: %q", port)

	server := router.CreateServer(":" + port)

	go func() {
		defer recoverFromPanic(logger, "HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Error running HTTP server: %v", err)
		}
	}()

	return server
}

func startHTTPSServer(port string, router routes.Router, tlsManager *services.TLSManager, logger pkg.LoggerClient) *http.Server {
	logger.Infof("HTTPS server listening on port: %q", port)

	tlsConfig := tlsManager.GetTLSConfig()
	server := router.CreateTLSServer(":"+port, tlsConfig)

	go func() {
		defer recoverFromPanic(logger, "HTTPS server")

		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Error running HTTPS server: %v", err)
		}
	}()

	return server
}

func shutdownServers(ctx context.Context, servers []*http.Server, logger pkg.LoggerClient) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(servers))

	for _, srv := range servers {
		if srv == nil {
			continue
		}

		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()

			logger.Infof("Shutting down server on %s...", s.Addr)
			if err := s.Shutdown(ctx); err != nil {
				logger.Errorf("Error shutting down server: %v", err)
				errChan <- err
			} else {
				logger.Infof("Server on %s shut down gracefully", s.Addr)
			}
		}(srv)
	}

	wg.Wait()
	close(errChan)

	if ctx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown timeout exceeded")
		return ctx.Err()
	}

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	logger.Info("Application stopped successfully")
	return nil
}

func recoverFromPanic(logger pkg.LoggerClient, serverType string) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		logger.Errorf("Panic in %s goroutine: %q\n%s", serverType, r, stack)
	}
}
