package repository

import (
	"context"

	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type APIKeyRepoClient interface {
	Create(ctx context.Context, apiKey *models.APIKey) error
	Find(ctx context.Context, filter *models.APIKey) (*models.APIKey, error)
	FindMany(ctx context.Context, filter *models.APIKey) ([]*models.APIKey, error)
	Updates(ctx context.Context, filter *models.APIKey, updates map[string]interface{}) error
	Delete(ctx context.Context, filter *models.APIKey) error
}

type APIKeyRepo struct {
	db *postgres.PostgresDB
}

func NewAPIKeyRepo(db postgres.PostgresDB) APIKeyRepoClient {
	return &APIKeyRepo{
		db: &db,
	}
}

func (r *APIKeyRepo) Create(ctx context.Context, apiKey *models.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

func (r *APIKeyRepo) Find(ctx context.Context, filter *models.APIKey) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.WithContext(ctx).Where(filter).First(&apiKey).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *APIKeyRepo) FindMany(ctx context.Context, filter *models.APIKey) ([]*models.APIKey, error) {
	var apiKeys []*models.APIKey
	query := r.db.WithContext(ctx).Where(filter)

	query = query.Order("created_at DESC")

	if err := query.Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepo) Updates(ctx context.Context, filter *models.APIKey, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.APIKey{}).Where(filter).Updates(updates).Error
}

func (r *APIKeyRepo) Delete(ctx context.Context, filter *models.APIKey) error {
	return r.db.WithContext(ctx).Where(filter).Delete(&models.APIKey{}).Error
}
