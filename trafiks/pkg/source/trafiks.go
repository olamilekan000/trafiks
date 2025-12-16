package source

import (
	"context"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
)

type TrafiksSource struct {
	serviceRepo repository.ServiceRepoClient
}

func NewTrafiksSource(serviceRepo repository.ServiceRepoClient) ServiceSource {
	return &TrafiksSource{
		serviceRepo: serviceRepo,
	}
}

func (t *TrafiksSource) Name() string {
	return models.SourceTrafiks
}

func (t *TrafiksSource) Get(ctx context.Context, proxyURL string) (*models.Service, error) {
	return t.serviceRepo.Find(ctx, &models.Service{ProxyURL: proxyURL})
}

func (t *TrafiksSource) Enabled() bool {
	return true
}
