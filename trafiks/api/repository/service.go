package repository

import (
	"context"

	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type ServiceRepoClient interface {
	Create(ctx context.Context, service *models.Service) error
	Find(ctx context.Context, filter *models.Service) (*models.Service, error)
	FindMany(ctx context.Context, filter *models.Service) ([]*models.Service, error)
	FindByProjectSlug(ctx context.Context, projectSlug string) (*models.Service, error)
	Updates(ctx context.Context, filter *models.Service, updates map[string]interface{}) error
	Delete(ctx context.Context, filter *models.Service) error
}

type ServiceRepo struct {
	db *postgres.PostgresDB
}

func NewServiceRepo(db postgres.PostgresDB) ServiceRepoClient {
	return &ServiceRepo{
		db: &db,
	}
}

func (r *ServiceRepo) Create(ctx context.Context, service *models.Service) error {
	return r.db.WithContext(ctx).Create(service).Error
}

func (r *ServiceRepo) Find(ctx context.Context, filter *models.Service) (*models.Service, error) {
	var service models.Service
	query := r.db.WithContext(ctx).Where(filter)

	query = query.Preload("Project")

	if err := query.First(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

func (r *ServiceRepo) FindMany(ctx context.Context, filter *models.Service) ([]*models.Service, error) {
	var services []*models.Service
	query := r.db.WithContext(ctx).Where(filter)

	query = query.Preload("Project")
	query = query.Order("created_at DESC")

	if err := query.Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

func (r *ServiceRepo) FindByProjectSlug(ctx context.Context, projectSlug string) (*models.Service, error) {
	var service models.Service
	if err := r.db.WithContext(ctx).Joins("JOIN projects ON projects.id = services.project_id").
		Preload("Project").
		Where("projects.slug = ?", projectSlug).
		First(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

func (r *ServiceRepo) Updates(ctx context.Context, filter *models.Service, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Service{}).Where(filter).Updates(updates).Error
}

func (r *ServiceRepo) Delete(ctx context.Context, filter *models.Service) error {
	return r.db.WithContext(ctx).Where(filter).Delete(&models.Service{}).Error
}
