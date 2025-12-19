//go:generate mockgen -source=webhook.go -destination=../../tests/mocks/webhook_repo.go -package=mocks

package repository

import (
	"context"

	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
)

type WebhookRepoClient interface {
	Create(ctx context.Context, webhook *models.Webhook) error
	Find(ctx context.Context, filter *models.Webhook) (*models.Webhook, error)
	FindMany(ctx context.Context, filter *models.Webhook) ([]*models.Webhook, error)
	Update(ctx context.Context, webhook *models.Webhook) error
	Delete(ctx context.Context, webhook *models.Webhook) error
}

type WebhookRepo struct {
	db *postgres.PostgresDB
}

func NewWebhookRepo(db postgres.PostgresDB) WebhookRepoClient {
	return &WebhookRepo{db: &db}
}

func (r *WebhookRepo) Create(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Create(webhook).Error
}

func (r *WebhookRepo) Find(ctx context.Context, filter *models.Webhook) (*models.Webhook, error) {
	var webhook models.Webhook
	if err := r.db.WithContext(ctx).Where(filter).First(&webhook).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (r *WebhookRepo) FindMany(ctx context.Context, filter *models.Webhook) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	if err := r.db.WithContext(ctx).Where(filter).Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

func (r *WebhookRepo) Update(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Save(webhook).Error
}

func (r *WebhookRepo) Delete(ctx context.Context, webhook *models.Webhook) error {
	return r.db.WithContext(ctx).Delete(webhook).Error
}
