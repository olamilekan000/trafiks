package source

import (
	"context"
	"errors"

	"github.com/trafiks/trafiks/models"
)

// ServiceSourceManager manages multiple service sources and provides a fluent API
type ServiceSourceManager struct {
	sources map[string]ServiceSource
	current ServiceSource
}

// NewServiceSourceManager creates a new ServiceSourceManager
func NewServiceSourceManager() *ServiceSourceManager {
	return &ServiceSourceManager{
		sources: make(map[string]ServiceSource),
	}
}

// Register registers a service source with the manager
func (m *ServiceSourceManager) Register(source ServiceSource) {
	m.sources[source.Name()] = source
}

// Use selects a source by name and returns the manager for method chaining
func (m *ServiceSourceManager) Use(sourceName string) *ServiceSourceManager {
	if source, ok := m.sources[sourceName]; ok {
		m.current = source
	} else {
		m.current = nil
	}
	return m
}

// Get retrieves a service using the currently selected source
func (m *ServiceSourceManager) Get(ctx context.Context, proxyURL string) (*models.Service, error) {
	if m.current == nil {
		return nil, errors.New("no source selected")
	}
	if !m.current.Enabled() {
		return nil, errors.New("selected source is not enabled")
	}
	return m.current.Get(ctx, proxyURL)
}

// GetSource returns the currently selected source (or nil if none selected)
func (m *ServiceSourceManager) GetSource() ServiceSource {
	return m.current
}
