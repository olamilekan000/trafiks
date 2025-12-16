package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/pkg"
)

// Service scheme constants
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

// Service source constants
const (
	SourceTrafiks    = "trafiks"
	SourceDocker     = "docker"
	SourceKubernetes = "kubernetes"
)

// TrimScheme removes http://, https://, or // prefix from a URL/domain string
func TrimScheme(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "//")
	return url
}

// ServiceConfig represents the JSON structure stored in Configuration field
type ServiceConfig struct {
	Headers       HeadersConfig     `json:"headers,omitempty" gorm:"type:jsonb"`
	QueryParams   QueryParamsConfig `json:"query_params,omitempty" gorm:"type:jsonb"`
	HTTPSRedirect *bool             `json:"https_redirect,omitempty"` // If true, redirect HTTP to HTTPS; if false, reject; if nil, default to true
	Docker        *DockerConfig     `json:"docker,omitempty"`         // Docker-specific configuration
	Kubernetes    *KubernetesConfig `json:"kubernetes,omitempty"`     // Kubernetes-specific configuration
	// Future: RateLimit, Retry, Timeouts, etc.
}

// DockerConfig contains Docker-specific service discovery configuration
type DockerConfig struct {
	Labels  map[string]string `json:"labels"`  // Docker labels to match containers (e.g., {"trafiks.service": "my-api"})
	Network string            `json:"network"` // Optional: Docker network name (e.g., "bridge", "custom-network")
	Port    string            `json:"port"`    // Container port to use (e.g., "8080")
}

// KubernetesConfig contains Kubernetes-specific service discovery configuration
type KubernetesConfig struct {
	Namespace   string            `json:"namespace"`          // K8s namespace (e.g., "default")
	ServiceName string            `json:"service_name"`       // K8s Service name
	ServicePort string            `json:"service_port"`       // Port name or number (e.g., "http", "8080")
	Selector    map[string]string `json:"selector,omitempty"` // Optional: label selector (e.g., {"app": "my-api"})
}

type HeadersConfig struct {
	Remove []string          `json:"remove,omitempty"` // Headers to remove
	Add    map[string]string `json:"add,omitempty"`    // Headers to add
}

func (h HeadersConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Unmarshal from DB
func (h *HeadersConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal HeadersConfig: %v", value)
	}

	return json.Unmarshal(bytes, h)
}

type QueryParamsConfig struct {
	Remove []string `json:"remove,omitempty"` // Query params to remove
}

func (q QueryParamsConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (q *QueryParamsConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal QueryParamsConfig: %v", value)
	}

	return json.Unmarshal(bytes, q)
}

type Service struct {
	gorm.Model
	ID               uint           `json:"-" gorm:"primaryKey;unique"`
	UID              string         `gorm:"not null;uniqueIndex"`
	ProjectID        uint           `gorm:"not null;index"`
	Source           string         `gorm:"default:'trafiks'"`    // Service source/provider (trafiks, kubernetes, docker, etc.)
	Scheme           string         `gorm:"default:'http'"`       // Proxy URL scheme (http, https)
	TargetBackendURL string         `gorm:"not null"`             // Required for trafiks source, optional for others
	ProxyURL         string         `gorm:"not null;uniqueIndex"` // Domain only, no scheme
	TLSCertificate   string         `gorm:"type:text"`            // PEM encoded TLS certificate (for HTTPS)
	TLSKey           string         `gorm:"type:text"`            // PEM encoded TLS private key (for HTTPS)
	TLSCertResolver  string         `gorm:"default:'selfsigned'"` // Certificate resolver: selfsigned, letsencrypt, manual
	CacheEnabled     bool           `gorm:"default:false"`        // For easy querying
	CacheTTL         int            `gorm:"default:300"`          // seconds, for easy querying
	Configuration    datatypes.JSON `gorm:"type:jsonb"`           // For complex nested config
	CreatedAt        time.Time      `json:"CreatedAt"`
	UpdatedAt        time.Time      `json:"UpdatedAt"`

	Project Project `json:"-" gorm:"foreignKey:ProjectID"`
}

func (s *Service) GetConfig() (*ServiceConfig, error) {
	if len(s.Configuration) == 0 {
		return &ServiceConfig{}, nil
	}

	var config ServiceConfig
	if err := json.Unmarshal(s.Configuration, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *Service) SetConfig(config *ServiceConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.Configuration = datatypes.JSON(data)
	return nil
}

func (s *Service) BeforeCreate(tx *gorm.DB) (err error) {
	s.UID = pkg.GenerateUUIDV7()
	s.CreatedAt = time.Now().Local()
	s.UpdatedAt = time.Now().Local()
	return
}

func (s *Service) BeforeUpdate(tx *gorm.DB) (err error) {
	s.UpdatedAt = time.Now().Local()
	return
}
