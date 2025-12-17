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
	HTTPSRedirect *bool             `json:"https_redirect,omitempty"`
	Docker        *DockerConfig     `json:"docker,omitempty"`
	Kubernetes    *KubernetesConfig `json:"kubernetes,omitempty"`
	// Future: RateLimit, Retry, Timeouts, etc.
}

// DockerConfig contains Docker-specific service discovery configuration
type DockerConfig struct {
	Labels  map[string]string `json:"labels"`
	Network string            `json:"network"`
	Port    string            `json:"port"`
}

// KubernetesConfig contains Kubernetes-specific service discovery configuration
type KubernetesConfig struct {
	Namespace       string            `json:"namespace"`
	ServiceName     string            `json:"service_name"`
	ServicePortName string            `json:"service_port_name"`
	Selector        map[string]string `json:"selector,omitempty"`
}

type HeadersConfig struct {
	Remove []string          `json:"remove,omitempty"`
	Add    map[string]string `json:"add,omitempty"`
}

func (h HeadersConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

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
	Remove []string `json:"remove,omitempty"`
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
	Source           string         `gorm:"default:'trafiks'"`
	Scheme           string         `gorm:"default:'http'"`
	TargetBackendURL string         `gorm:"not null"`
	ProxyURL         string         `gorm:"not null"`
	TLSCertificate   string         `gorm:"type:text"`
	TLSKey           string         `gorm:"type:text"`
	TLSCertResolver  string         `gorm:"default:'selfsigned'"`
	CacheEnabled     bool           `gorm:"default:false"`
	CacheTTL         int            `gorm:"default:300"`
	Configuration    datatypes.JSON `gorm:"type:jsonb"`
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
