package source

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

// DockerSource implements ServiceSource for Docker container discovery
type DockerSource struct {
	serviceRepo repository.ServiceRepoClient
	dockerCli   *client.Client
	logger      pkg.LoggerClient
	socketPath  string
}

// NewDockerSource creates a new DockerSource instance
// If socketPath is empty, Docker source will be disabled
func NewDockerSource(
	ctx context.Context,
	serviceRepo repository.ServiceRepoClient,
	socketPath string,
	logger pkg.LoggerClient,
) (ServiceSource, error) {
	if socketPath == "" {
		return &DockerSource{
			serviceRepo: serviceRepo,
			logger:      logger,
		}, nil
	}

	dockerCli, err := client.New(
		client.WithHost(socketPath),
		client.WithAPIVersion("1.41"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	_, err = dockerCli.Ping(ctx, client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	logger.Infof("Docker source initialized (socket: %s)", socketPath)

	return &DockerSource{
		serviceRepo: serviceRepo,
		dockerCli:   dockerCli,
		logger:      logger,
		socketPath:  socketPath,
	}, nil
}

func (d *DockerSource) Name() string {
	return models.SourceDocker
}

func (d *DockerSource) Get(ctx context.Context, proxyURL string) (*models.Service, error) {
	if !d.Enabled() {
		return nil, fmt.Errorf("docker source is not enabled")
	}

	service, err := d.serviceRepo.Find(ctx, &models.Service{ProxyURL: proxyURL})
	if err != nil {
		return nil, err
	}

	config, err := service.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse service config: %w", err)
	}

	if config.Docker == nil {
		return nil, fmt.Errorf("docker configuration not found for service")
	}

	dockerConfig := config.Docker

	filter := client.Filters{}
	for key, value := range dockerConfig.Labels {
		filter.Add("label", fmt.Sprintf("%s=%s", key, value))
	}

	containers, err := d.dockerCli.ContainerList(ctx, client.ContainerListOptions{
		Filters: filter,
		All:     false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers.Items) == 0 {
		return nil, fmt.Errorf("no Docker containers found matching labels: %v", dockerConfig.Labels)
	}

	container := containers.Items[0]

	targetURL := d.resolveContainerEndpoint(container, dockerConfig)
	if targetURL == "" {
		return nil, fmt.Errorf("failed to resolve container endpoint")
	}

	service.TargetBackendURL = targetURL

	d.logger.Infof("Resolved Docker container %s to %s for proxy URL %s", container.ID[:12], targetURL, proxyURL)

	return service, nil
}

// Enabled returns whether the Docker source is enabled
func (d *DockerSource) Enabled() bool {
	return d.dockerCli != nil
}

// resolveContainerEndpoint determines the target URL for a container
func (d *DockerSource) resolveContainerEndpoint(
	container container.Summary,
	dockerConfig *models.DockerConfig,
) string {
	var address string
	var port string

	isRunningInContainer := d.isRunningInContainer()
	isUnixSocket := strings.HasPrefix(d.socketPath, "unix://")

	if isUnixSocket && isRunningInContainer {
		address = d.getContainerNetworkAddress(container, dockerConfig)
		port = dockerConfig.Port
	} else {
		address, port = d.getExposedPortAddress(container, dockerConfig)
	}

	if address == "" || port == "" {
		return ""
	}

	return fmt.Sprintf("http://%s:%s", address, port)
}

// isRunningInContainer checks if Trafiks is running inside a Docker container
func (d *DockerSource) isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	if cgroup, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		content := string(cgroup)

		if strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd") ||
			strings.Contains(content, "kubepods") {
			return true
		}
	}

	return false
}

// getContainerNetworkAddress gets the address when using container networking
func (d *DockerSource) getContainerNetworkAddress(
	container container.Summary,
	dockerConfig *models.DockerConfig,
) string {
	if containerName, ok := container.Labels["com.docker.compose.service"]; ok {
		return containerName
	}

	if len(container.Names) > 0 {
		name := strings.TrimPrefix(container.Names[0], "/")
		if idx := strings.LastIndex(name, "_"); idx > 0 {
			name = name[idx+1:]
		}
		return name
	}

	if len(container.NetworkSettings.Networks) > 0 {
		if dockerConfig.Network != "" {
			if network, exists := container.NetworkSettings.Networks[dockerConfig.Network]; exists {
				if network.IPAddress.IsValid() {
					return network.IPAddress.String()
				}
			}
			d.logger.Warnf("Docker network '%s' not found for container %s, auto-detecting network", dockerConfig.Network, container.ID[:12])
		}

		for networkName, network := range container.NetworkSettings.Networks {
			if network.IPAddress.IsValid() {
				d.logger.Infof("Auto-detected Docker network '%s' for container %s", networkName, container.ID[:12])
				return network.IPAddress.String()
			}
		}
	}

	return ""
}

// getExposedPortAddress gets the address when using exposed ports (host mode)
func (d *DockerSource) getExposedPortAddress(
	container container.Summary,
	dockerConfig *models.DockerConfig,
) (string, string) {
	configuredPort := dockerConfig.Port

	for _, portInfo := range container.Ports {
		if fmt.Sprintf("%d", portInfo.PrivatePort) == configuredPort {
			address := portInfo.IP.String()
			if address == "0.0.0.0" || address == "" {
				address = "localhost"
			}
			return address, fmt.Sprintf("%d", portInfo.PublicPort)
		}
	}

	if len(container.Ports) > 0 {
		portInfo := container.Ports[0]
		address := portInfo.IP.String()
		if address == "0.0.0.0" || address == "" {
			address = "localhost"
		}
		return address, fmt.Sprintf("%d", portInfo.PublicPort)
	}

	if len(container.NetworkSettings.Networks) > 0 {
		for _, network := range container.NetworkSettings.Networks {
			if network.IPAddress.IsValid() {
				return network.IPAddress.String(), configuredPort
			}
		}
	}

	return "", ""
}
