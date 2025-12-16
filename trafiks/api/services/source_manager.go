package services

import (
	"context"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/source"
)

// NewServiceSourceManager initializes and returns a ServiceSourceManager with all
// supported sources (trafiks, docker, etc.) registered.
//
// This centralizes source registration so it happens in one place during
// application startup, and callers just receive a ready-to-use manager.
func NewServiceSourceManager(
	logger pkg.LoggerClient,
	config *cfg.Config,
	serviceRepo repository.ServiceRepoClient,
) *source.ServiceSourceManager {
	ctx := context.Background()

	manager := source.NewServiceSourceManager()

	manager.Register(source.NewTrafiksSource(serviceRepo))

	registerDockerSource(ctx, manager, config, serviceRepo, logger)
	registerKubernetesSource(ctx, manager, config, serviceRepo, logger)

	return manager
}

// registerDockerSource attempts to register Docker as a service discovery source
func registerDockerSource(
	ctx context.Context,
	manager *source.ServiceSourceManager,
	config *cfg.Config,
	serviceRepo repository.ServiceRepoClient,
	logger pkg.LoggerClient,
) {
	if config.Docker.SocketPath == "" {
		logger.Info("Docker source disabled: no socket path configured")
		return
	}

	dockerSource, err := source.NewDockerSource(
		ctx,
		serviceRepo,
		config.Docker.SocketPath,
		logger,
	)
	if err != nil {
		logger.Warnf("failed to initialize Docker source: %v", err)
		return
	}

	manager.Register(dockerSource)

	logger.Infof("Docker source enabled socket: %s", config.Docker.SocketPath)
}

// registerKubernetesSource attempts to register Kubernetes as a service discovery source
func registerKubernetesSource(
	ctx context.Context,
	manager *source.ServiceSourceManager,
	config *cfg.Config,
	serviceRepo repository.ServiceRepoClient,
	logger pkg.LoggerClient,
) {
	k8sSource, err := source.NewKubernetesSource(
		ctx,
		serviceRepo,
		config.Kubernetes.KubeconfigPath,
		logger,
	)
	if err != nil {
		logger.Warnf("failed to initialize Kubernetes source: %v", err)
		return
	}

	manager.Register(k8sSource)
	logger.Infof("Kubernetes source enabled")
}
