//go:generate mockgen -source=project.go -destination=../../tests/mocks/project.go -package=mocks

package repository

import (
	"context"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type ProjectRepoClient interface {
	Create(ctx context.Context, project *models.Project) error
	Find(ctx context.Context, filter *models.Project) (*models.Project, error)
	FindMany(ctx context.Context, filter *models.Project, pagination *dto.Pagination) ([]*models.Project, error)
	Updates(ctx context.Context, filter *models.Project, updates map[string]interface{}) error
	Delete(ctx context.Context, filter *models.Project) error
}

type ProjectRepo struct {
	db *postgres.PostgresDB
}

func NewProjectRepo(db postgres.PostgresDB) ProjectRepoClient {
	return &ProjectRepo{
		db: &db,
	}
}

func (r *ProjectRepo) Create(ctx context.Context, project *models.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *ProjectRepo) Find(ctx context.Context, filter *models.Project) (*models.Project, error) {
	var project models.Project
	if err := r.db.WithContext(ctx).Where(filter).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepo) FindMany(ctx context.Context, filter *models.Project, pagination *dto.Pagination) ([]*models.Project, error) {
	var projects []*models.Project

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.Limit < 1 {
		pagination.Limit = 20
	}
	if pagination.Limit > 100 {
		pagination.Limit = 100
	}

	query := r.db.WithContext(ctx).Model(&models.Project{}).Where(filter)

	if err := query.Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	offset := pagination.Offset()
	// Use Preload to fetch services along with projects in a single query
	if err := query.
		Preload("Services"). // Preload services for each project
		Order("created_at DESC").
		Limit(pagination.Limit).
		Offset(offset).
		Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

func (r *ProjectRepo) Updates(ctx context.Context, filter *models.Project, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Project{}).Where(filter).Updates(updates).Error
}

func (r *ProjectRepo) Delete(ctx context.Context, filter *models.Project) error {
	return r.db.WithContext(ctx).Where(filter).Delete(&models.Project{}).Error
}
