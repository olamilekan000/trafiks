package services

import (
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
	manager := source.NewServiceSourceManager()

	// Always register the core Trafiks source
	trafiksSource := source.NewTrafiksSource(serviceRepo)
	manager.Register(trafiksSource)

	// Conditionally register Docker source if socket path is configured
	if config.Docker.SocketPath != "" {
		dockerSource, err := source.NewDockerSource(
			serviceRepo,
			config.Docker.SocketPath,
			logger,
		)
		if err != nil {
			logger.Warnf("Failed to initialize Docker source: %v. Docker source will be disabled.", err)
		} else {
			manager.Register(dockerSource)
			logger.Infof("Docker source enabled (socket: %s)", config.Docker.SocketPath)
		}
	}

	return manager
}
