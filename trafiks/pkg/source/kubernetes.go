package source

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type KubernetesSource struct {
	serviceRepo repository.ServiceRepoClient
	k8sClient   *kubernetes.Clientset
	logger      pkg.LoggerClient
}

func NewKubernetesSource(
	ctx context.Context,
	serviceRepo repository.ServiceRepoClient,
	kubeconfigPath string,
	logger pkg.LoggerClient,
) (ServiceSource, error) {
	config, err := loadKubeConfig(kubeconfigPath, logger)
	if err != nil {
		if kubeconfigPath != "" {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
		}

		logger.Warnf("Kubernetes not available: %v", err)
		return &KubernetesSource{
			serviceRepo: serviceRepo,
			logger:      logger,
		}, nil
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		logger.Warnf("Kubernetes cluster unreachable: %v", err)
		return &KubernetesSource{
			serviceRepo: serviceRepo,
			logger:      logger,
		}, nil
	}

	logger.Infof("Kubernetes source initialized and validated")
	return &KubernetesSource{
		serviceRepo: serviceRepo,
		k8sClient:   clientset,
		logger:      logger,
	}, nil
}

func (k *KubernetesSource) Name() string {
	return models.SourceKubernetes
}

func (k *KubernetesSource) Get(ctx context.Context, proxyURL string) (*models.Service, error) {
	if !k.Enabled() {
		return nil, fmt.Errorf("kubernetes source is not enabled")
	}

	service, err := k.serviceRepo.Find(ctx, &models.Service{ProxyURL: proxyURL})
	if err != nil {
		return nil, err
	}

	config, err := service.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse service config: %w", err)
	}

	if config.Kubernetes == nil {
		return nil, fmt.Errorf("kubernetes configuration not found for service")
	}

	k8sConfig := config.Kubernetes

	if k8sConfig.Namespace == "" {
		return nil, fmt.Errorf("kubernetes namespace is required")
	}
	if k8sConfig.ServiceName == "" {
		return nil, fmt.Errorf("kubernetes service name is required")
	}
	if k8sConfig.ServicePortName == "" {
		return nil, fmt.Errorf("kubernetes service port name is required")
	}

	targetURL, err := k.resolveServiceEndpoint(ctx, k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve k8s service endpoint: %w", err)
	}

	service.TargetBackendURL = targetURL
	k.logger.Infof("Resolved K8s service %s/%s to %s for proxy URL %s",
		k8sConfig.Namespace, k8sConfig.ServiceName, targetURL, proxyURL)

	return service, nil
}

func (k *KubernetesSource) Enabled() bool {
	return k.k8sClient != nil
}

func (k *KubernetesSource) resolveServiceEndpoint(ctx context.Context, config *models.KubernetesConfig) (string, error) {
	svc, err := k.k8sClient.CoreV1().Services(config.Namespace).Get(ctx, config.ServiceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service %s/%s: %w", config.Namespace, config.ServiceName, err)
	}

	port, err := k.resolvePort(svc, config.ServicePortName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve port %s: %w", config.ServicePortName, err)
	}

	targetURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		config.ServiceName, config.Namespace, port)

	return targetURL, nil
}

func (k *KubernetesSource) resolvePort(svc *corev1.Service, portName string) (int32, error) {
	for _, port := range svc.Spec.Ports {
		if port.Name == portName {
			return port.Port, nil
		}
	}

	return 0, fmt.Errorf("port name %s not found in service %s/%s", portName, svc.Namespace, svc.Name)
}

// loadKubeConfig attempts to load Kubernetes configuration in the following order:
// 1. Provided kubeconfig path
// 2. In-cluster config
// 3. Default kubeconfig location (~/.kube/config)
func loadKubeConfig(kubeconfigPath string, logger pkg.LoggerClient) (*rest.Config, error) {
	if kubeconfigPath != "" {
		if config, err := tryLoadKubeconfig(kubeconfigPath); err == nil {
			logger.Infof("Loaded kubeconfig from: %s", kubeconfigPath)
			return config, nil
		}
		logger.Infof("Failed to load kubeconfig from %s, trying alternatives", kubeconfigPath)
	}

	if config, err := rest.InClusterConfig(); err == nil {
		logger.Infof("Using in-cluster Kubernetes configuration")
		return config, nil
	}

	defaultPath := clientcmd.RecommendedHomeFile
	if config, err := tryLoadKubeconfig(defaultPath); err == nil {
		logger.Infof("Loaded kubeconfig from default location: %s", defaultPath)
		return config, nil
	}

	return nil, fmt.Errorf("no valid kubeconfig found")
}

func tryLoadKubeconfig(path string) (*rest.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found: %s", path)
	}

	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from %s: %w", path, err)
	}

	return config, nil
}
