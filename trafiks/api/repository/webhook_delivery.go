//go:generate mockgen -source=webhook_delivery.go -destination=../../tests/mocks/webhook_delivery.go -package=mocks

package repository

import (
	"context"
	"time"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type WebhookDeliveryFilters struct {
	Status    string
	EventType string
	StartTime *time.Time
	EndTime   *time.Time
}

type WebhookDeliveryRepoClient interface {
	Create(ctx context.Context, delivery *models.WebhookDelivery) error
	Find(ctx context.Context, filter *models.WebhookDelivery) (*models.WebhookDelivery, error)
	FindMany(
		ctx context.Context,
		filter *models.WebhookDelivery,
		pagination *dto.Pagination,
		filters *WebhookDeliveryFilters,
	) ([]*models.WebhookDelivery, error)
	FindRetryable(
		ctx context.Context,
		limit int,
	) ([]*models.WebhookDelivery, error)
	FindPending(ctx context.Context, filter *models.WebhookDelivery, limit int) ([]*models.WebhookDelivery, error)
	Count(ctx context.Context, filter *models.WebhookDelivery, filters *WebhookDeliveryFilters) (int64, error)
	Update(ctx context.Context, delivery *models.WebhookDelivery) error
	Updates(ctx context.Context, delivery *models.WebhookDelivery, updates map[string]interface{}) error
}

type WebhookDeliveryRepo struct {
	db *postgres.PostgresDB
}

func NewWebhookDeliveryRepo(db postgres.PostgresDB) WebhookDeliveryRepoClient {
	return &WebhookDeliveryRepo{db: &db}
}

func (r *WebhookDeliveryRepo) Create(ctx context.Context, delivery *models.WebhookDelivery) error {
	return r.db.WithContext(ctx).Create(delivery).Error
}

func (r *WebhookDeliveryRepo) Find(ctx context.Context, filter *models.WebhookDelivery) (*models.WebhookDelivery, error) {
	var delivery models.WebhookDelivery
	if err := r.db.WithContext(ctx).Where(filter).First(&delivery).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *WebhookDeliveryRepo) FindPending(ctx context.Context, filter *models.WebhookDelivery, limit int) ([]*models.WebhookDelivery, error) {
	var deliveries []*models.WebhookDelivery
	if err := r.db.WithContext(ctx).Where(filter).Where("status = ?", models.DeliveryStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *WebhookDeliveryRepo) FindRetryable(ctx context.Context, limit int) ([]*models.WebhookDelivery, error) {
	var deliveries []*models.WebhookDelivery
	now := time.Now()
	if err := r.db.WithContext(ctx).Where("status = ? AND retry_count < ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		models.DeliveryStatusFailed, 5, now).
		Order("next_retry_at ASC NULLS FIRST").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *WebhookDeliveryRepo) Update(ctx context.Context, delivery *models.WebhookDelivery) error {
	return r.db.WithContext(ctx).Save(delivery).Error
}

func (r *WebhookDeliveryRepo) Updates(ctx context.Context, delivery *models.WebhookDelivery, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(delivery).Updates(updates).Error
}

func (r *WebhookDeliveryRepo) FindMany(ctx context.Context, filter *models.WebhookDelivery, pagination *dto.Pagination, filters *WebhookDeliveryFilters) ([]*models.WebhookDelivery, error) {
	var deliveries []*models.WebhookDelivery
	query := r.db.WithContext(ctx).Where(filter)

	if filters != nil {
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.EventType != "" {
			query = query.Where("event_type = ?", filters.EventType)
		}
		if filters.StartTime != nil {
			query = query.Where("created_at >= ?", *filters.StartTime)
		}
		if filters.EndTime != nil {
			query = query.Where("created_at <= ?", *filters.EndTime)
		}
	}

	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.Limit
		query = query.Offset(offset).Limit(pagination.Limit)
	}

	if err := query.Order("created_at DESC").Find(&deliveries).Error; err != nil {
		return nil, err
	}

	return deliveries, nil
}

func (r *WebhookDeliveryRepo) Count(ctx context.Context, filter *models.WebhookDelivery, filters *WebhookDeliveryFilters) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.WebhookDelivery{}).Where(filter)

	if filters != nil {
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.EventType != "" {
			query = query.Where("event_type = ?", filters.EventType)
		}
		if filters.StartTime != nil {
			query = query.Where("created_at >= ?", *filters.StartTime)
		}
		if filters.EndTime != nil {
			query = query.Where("created_at <= ?", *filters.EndTime)
		}
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
