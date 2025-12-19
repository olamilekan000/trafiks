//go:generate mockgen -source=source.go -destination=../../tests/mocks/source.go -package=mocks

package source

import (
	"context"

	"github.com/trafiks/trafiks/models"
)

// ServiceSource defines the interface for service discovery sources
// Each source (trafiks, kubernetes, docker, etc.) implements this interface
type ServiceSource interface {
	// Name returns the identifier for this source (e.g., "trafiks", "kubernetes")
	Name() string

	// Get retrieves a service by its proxy URL
	Get(ctx context.Context, proxyURL string) (*models.Service, error)

	// Enabled returns whether this source is currently enabled
	Enabled() bool
}
