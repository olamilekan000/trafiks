package dto

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/trafiks/trafiks/models"
)

func deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return result
}

type CreateServiceRequest struct {
	Source           string                `json:"Source"`                      // Service source (trafiks, kubernetes, docker, etc.)
	Scheme           string                `json:"Scheme"`                      // Proxy URL scheme (http, https)
	TargetBackendURL string                `json:"TargetBackendURL"`            // Required for trafiks source, optional for others
	ProxyURL         string                `json:"ProxyURL" binding:"required"` // Domain only, no scheme
	CacheEnabled     bool                  `json:"CacheEnabled"`
	CacheTTL         int                   `json:"CacheTTL"`
	Configuration    *ServiceConfigRequest `json:"Configuration,omitempty"`
	TLSCertificate   string                `json:"TLSCertificate,omitempty"`  // TLS certificate PEM (for letsencrypt or manual)
	TLSKey           string                `json:"TLSKey,omitempty"`          // TLS private key PEM
	TLSCertResolver  string                `json:"TLSCertResolver,omitempty"` // Certificate resolver type (letsencrypt, selfsigned, manual)
}

type UpdateServiceRequest struct {
	Source           string                `json:"Source,omitempty"`
	Scheme           string                `json:"Scheme,omitempty"`
	TargetBackendURL string                `json:"TargetBackendURL,omitempty"`
	ProxyURL         string                `json:"ProxyURL,omitempty"` // Domain only, no scheme
	CacheEnabled     *bool                 `json:"CacheEnabled,omitempty"`
	CacheTTL         *int                  `json:"CacheTTL,omitempty"`
	Configuration    *ServiceConfigRequest `json:"Configuration,omitempty"`
	TLSCertificate   string                `json:"TLSCertificate,omitempty"`  // TLS certificate PEM (for letsencrypt or manual)
	TLSKey           string                `json:"TLSKey,omitempty"`          // TLS private key PEM
	TLSCertResolver  string                `json:"TLSCertResolver,omitempty"` // Certificate resolver type (letsencrypt, selfsigned, manual)
}

type ServiceConfigRequest struct {
	Headers       *HeadersConfigRequest     `json:"headers,omitempty"`
	QueryParams   *QueryParamsConfigRequest `json:"query_params,omitempty"`
	HTTPSRedirect *bool                     `json:"https_redirect,omitempty"`
	Docker        *DockerConfigRequest      `json:"docker,omitempty"`
	Kubernetes    *KubernetesConfigRequest  `json:"kubernetes,omitempty"`
}

// DockerConfigRequest represents Docker-specific configuration
type DockerConfigRequest struct {
	Labels  map[string]string `json:"labels,omitempty"`  // Docker labels to match containers
	Network string            `json:"network,omitempty"` // Optional: Docker network name
	Port    string            `json:"port,omitempty"`    // Container port to use
}

type KubernetesConfigRequest struct {
	Namespace       string            `json:"namespace,omitempty"`
	ServiceName     string            `json:"service_name,omitempty"`
	ServicePortName string            `json:"service_port_name,omitempty"`
	Selector        map[string]string `json:"selector,omitempty"`
}

type HeadersConfigRequest struct {
	Remove []string          `json:"remove,omitempty"`
	Add    map[string]string `json:"add,omitempty"`
}

type QueryParamsConfigRequest struct {
	Remove []string `json:"remove,omitempty"`
}

func (c *CreateServiceRequest) Validate() error {
	c.Source = strings.TrimSpace(c.Source)
	c.Scheme = strings.TrimSpace(c.Scheme)
	c.TargetBackendURL = strings.TrimSpace(c.TargetBackendURL)
	c.ProxyURL = strings.TrimSpace(c.ProxyURL)

	if c.Source == "" {
		c.Source = models.SourceTrafiks
	}

	if c.Scheme == "" {
		c.Scheme = models.SchemeHTTP
	}

	if c.Scheme != models.SchemeHTTP && c.Scheme != models.SchemeHTTPS {
		return fmt.Errorf("scheme must be http or https")
	}

	if c.ProxyURL == "" {
		return fmt.Errorf("proxy URL is required")
	}

	c.ProxyURL = models.TrimScheme(c.ProxyURL)
	if c.ProxyURL == "" {
		return fmt.Errorf("proxy URL must include a domain")
	}

	if strings.Contains(c.ProxyURL, "://") {
		return fmt.Errorf("proxy URL should not include a scheme (http:// or https://)")
	}

	if c.Source == models.SourceTrafiks {
		if c.TargetBackendURL == "" {
			return fmt.Errorf("target backend URL is required for trafiks source")
		}

		parsedURL, err := url.Parse(c.TargetBackendURL)
		if err != nil {
			return fmt.Errorf("invalid target backend URL format: %v", err)
		}

		if parsedURL.Scheme != models.SchemeHTTP && parsedURL.Scheme != models.SchemeHTTPS {
			return fmt.Errorf("target backend URL must use http or https scheme")
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("target backend URL must include a host")
		}
	}

	if c.CacheTTL < 0 {
		return fmt.Errorf("cache TTL must be a positive number")
	}
	if c.CacheTTL > 86400 {
		return fmt.Errorf("cache TTL cannot exceed 86400 seconds (24 hours)")
	}

	if c.CacheTTL == 0 {
		c.CacheTTL = 300
	}

	return nil
}

func (u *UpdateServiceRequest) Validate() error {
	if u.Source != "" {
		u.Source = strings.TrimSpace(u.Source)
	}

	if u.Scheme != "" {
		u.Scheme = strings.TrimSpace(u.Scheme)
		if u.Scheme != models.SchemeHTTP && u.Scheme != models.SchemeHTTPS {
			return fmt.Errorf("scheme must be http or https")
		}
	}

	if u.TargetBackendURL != "" {
		u.TargetBackendURL = strings.TrimSpace(u.TargetBackendURL)

		parsedURL, err := url.Parse(u.TargetBackendURL)
		if err != nil {
			return fmt.Errorf("invalid target backend URL format: %v", err)
		}

		if parsedURL.Scheme != models.SchemeHTTP && parsedURL.Scheme != models.SchemeHTTPS {
			return fmt.Errorf("target backend URL must use http or https scheme")
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("target backend URL must include a host")
		}
	}

	if u.ProxyURL != "" {
		u.ProxyURL = strings.TrimSpace(u.ProxyURL)

		u.ProxyURL = models.TrimScheme(u.ProxyURL)

		if u.ProxyURL == "" {
			return fmt.Errorf("proxy URL must include a domain")
		}

		if strings.Contains(u.ProxyURL, "://") {
			return fmt.Errorf("proxy URL should not include a scheme (http:// or https://)")
		}
	}

	if u.CacheTTL != nil {
		if *u.CacheTTL < 0 {
			return fmt.Errorf("cache TTL must be a positive number")
		}
		if *u.CacheTTL > 86400 {
			return fmt.Errorf("cache TTL cannot exceed 86400 seconds (24 hours)")
		}
	}

	return nil
}

func (s *ServiceConfigRequest) ToServiceConfig() *models.ServiceConfig {
	if s == nil {
		return &models.ServiceConfig{}
	}

	config := &models.ServiceConfig{}

	if s.Headers != nil {
		removeHeaders := deduplicateStrings(s.Headers.Remove)

		addHeaders := make(map[string]string)
		for key, value := range s.Headers.Add {
			trimmedKey := strings.TrimSpace(key)
			trimmedValue := strings.TrimSpace(value)
			if trimmedKey != "" && trimmedValue != "" {
				addHeaders[trimmedKey] = trimmedValue
			}
		}

		config.Headers = models.HeadersConfig{
			Remove: removeHeaders,
			Add:    addHeaders,
		}
	}

	if s.QueryParams != nil {
		removeParams := deduplicateStrings(s.QueryParams.Remove)
		config.QueryParams = models.QueryParamsConfig{
			Remove: removeParams,
		}
	}

	if s.HTTPSRedirect != nil {
		config.HTTPSRedirect = s.HTTPSRedirect
	}

	if s.Docker != nil {
		config.Docker = &models.DockerConfig{
			Labels:  s.Docker.Labels,
			Network: s.Docker.Network,
			Port:    s.Docker.Port,
		}
	}

	if s.Kubernetes != nil {
		config.Kubernetes = &models.KubernetesConfig{
			Namespace:       s.Kubernetes.Namespace,
			ServiceName:     s.Kubernetes.ServiceName,
			ServicePortName: s.Kubernetes.ServicePortName,
			Selector:        s.Kubernetes.Selector,
		}
	}

	return config
}

func (s *ServiceConfigRequest) ToJSON() ([]byte, error) {
	config := s.ToServiceConfig()
	return json.Marshal(config)
}
